package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	configDirName  = ".jumper"
	configFileName = "folders"
)

func main() {
	// Get home directory for config file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	configDir := filepath.Join(homeDir, configDirName)
	configPath := filepath.Join(configDir, configFileName)

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating config directory: %v\n", err)
		os.Exit(1)
	}

	// Ensure config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		file, err := os.Create(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating config file: %v\n", err)
			os.Exit(1)
		}
		file.Close()
	}

	// Process commands
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: jumper <command>\n")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "add":
		addFolder(configPath)
	case "list":
		listFolders(configPath)
	case "setup":
		setupJumper(configDir, homeDir)
	case "remove":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Usage: jumper remove <folder-name-or-number>\n")
			os.Exit(1)
		}
		removeFolder(configPath, os.Args[2])
	default:
		// Treat as a jump target
		jumpToFolder(configPath, os.Args[1])
	}
}

// addFolder adds the current directory to the jump list
func addFolder(configPath string) {
	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	// Check if folder already exists in the list
	folders, err := readFolderList(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading folder list: %v\n", err)
		os.Exit(1)
	}

	for _, folder := range folders {
		if folder == currentDir {
			fmt.Printf("Current folder already in the list: %s\n", currentDir)
			return
		}
	}

	// Add to config file
	file, err := os.OpenFile(configPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening config file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	if _, err := file.WriteString(currentDir + "\n"); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to config file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Added current folder to jump list: %s\n", currentDir)
}

// jumpToFolder prints the path to jump to for shell function to use
func jumpToFolder(configPath, arg string) {
	folders, err := readFolderList(configPath)
	if err != nil {
		os.Exit(1)  // Silent exit on error
	}

	// Check if argument is a number
	if num, err := strconv.Atoi(arg); err == nil && num > 0 && num <= len(folders) {
		fmt.Print(folders[num-1])
		return
	}

	// Check if argument matches a folder path
	for _, folder := range folders {
		if filepath.Base(folder) == arg || folder == arg {
			fmt.Print(folder)
			return
		}
	}

	os.Exit(1)  // Silent exit when folder not found
}

// listFolders displays all folders in the jump list
func listFolders(configPath string) {
	folders, err := readFolderList(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading folder list: %v\n", err)
		os.Exit(1)
	}

	if len(folders) == 0 {
		fmt.Println("No folders in jump list. Use 'jumper add' to add the current folder.")
		return
	}

	fmt.Println("Available folders:")
	for i, folder := range folders {
		fmt.Printf("%d. %s\n", i+1, folder)
	}
}

// readFolderList reads and returns the list of folders from the config file
func readFolderList(configPath string) ([]string, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var folders []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			folders = append(folders, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return folders, nil
}

// setupJumper creates the jumper.sh file and adds it to the shell configuration
func setupJumper(configDir, homeDir string) {
	jumperScript := `#!/bin/bash

# Function to jump to a folder
jp() {
    if [ -z "$1" ]; then
        jumper list
        return
    fi
    
    local target=$(jumper "$1")
    if [ $? -eq 0 ]; then
        cd "$target"
    fi
}

# Bash completion for jp
_jp_complete() {
    local cur prev
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    
    if [ "$prev" = "jumper" ]; then
        COMPREPLY=( $(compgen -W "add list remove setup" -- "$cur") )
    elif [ "$prev" = "remove" ]; then
        # Get folder names for remove command
        local folders=$(jumper list | grep -v "Available folders:" | sed 's/^[0-9]*\. \(.*\)$/\1/' | xargs -n1 basename)
        COMPREPLY=( $(compgen -W "$folders" -- "$cur") )
    elif [ "$prev" = "jp" ]; then
        # Get folder names for jp command
        local folders=$(jumper list | grep -v "Available folders:" | sed 's/^[0-9]*\. \(.*\)$/\1/' | xargs -n1 basename)
        COMPREPLY=( $(compgen -W "$folders" -- "$cur") )
    fi
    
    return 0
}

complete -F _jp_complete jp
complete -F _jp_complete jumper`

	// Write jumper.sh file (overwriting if it exists)
	scriptPath := filepath.Join(configDir, "jumper.sh")
	err := os.WriteFile(scriptPath, []byte(jumperScript), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating jumper.sh: %v\n", err)
		os.Exit(1)
	}

	// Check for .bashrc and .bash_aliases
	rcFiles := []string{".bashrc", ".bash_aliases"}
	sourceCmd := fmt.Sprintf("\n# Jumper configuration\nsource %s\n", scriptPath)

	for _, rcFile := range rcFiles {
		rcPath := filepath.Join(homeDir, rcFile)
		if _, err := os.Stat(rcPath); err == nil {
			// Read existing content
			content, err := os.ReadFile(rcPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", rcFile, err)
				continue
			}

			// Check if source command already exists
			if strings.Contains(string(content), scriptPath) {
				fmt.Printf("Jumper configuration already exists in %s\n", rcFile)
				break
			}

			// Append source command
			f, err := os.OpenFile(rcPath, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error opening %s: %v\n", rcFile, err)
				continue
			}
			
			if _, err := f.WriteString(sourceCmd); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing to %s: %v\n", rcFile, err)
				f.Close()
				continue
			}
			f.Close()
			
			fmt.Printf("Added jumper configuration to %s\n", rcFile)
			break  // Successfully added to one file, no need to continue
		}
	}

	fmt.Println("Setup complete! Please restart your shell or run:")
	fmt.Printf("source %s\n", scriptPath)
}

// removeFolder removes a folder from the jump list
func removeFolder(configPath, arg string) {
	folders, err := readFolderList(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading folder list: %v\n", err)
		os.Exit(1)
	}

	if len(folders) == 0 {
		fmt.Println("No folders in jump list.")
		return
	}

	var indexToRemove int = -1

	// Check if argument is a number
	if num, err := strconv.Atoi(arg); err == nil && num > 0 && num <= len(folders) {
		indexToRemove = num - 1
	} else {
		// Check if argument matches a folder name
		for i, folder := range folders {
			if filepath.Base(folder) == arg || folder == arg {
				indexToRemove = i
				break
			}
		}
	}

	if indexToRemove == -1 {
		fmt.Fprintf(os.Stderr, "Folder not found: %s\n", arg)
		os.Exit(1)
	}

	// Remove the folder and write back to file
	removedFolder := folders[indexToRemove]
	folders = append(folders[:indexToRemove], folders[indexToRemove+1:]...)

	// Write the updated list back to the file
	file, err := os.Create(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening config file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	for _, folder := range folders {
		if _, err := file.WriteString(folder + "\n"); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to config file: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("Removed folder: %s\n", removedFolder)
}