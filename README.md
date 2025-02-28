# Jumper

Directory jumper

## Install

```bash
go install github.com/s3bw/jumper@latest

jumper setup

# Source your updated shell configuration
source ~/.bashrc  # or source ~/.bash_aliases
```

## Usage

Add the current directory to your jump list:

```bash
jumper add
```

List all available directories:

```bash
jumper list
# or
jp  # with no arguments
```

Jump to a directory (with tab completion):

```bash
jp <folder-name>  # Jump using folder name
jp 2              # Jump using folder number
```

## How it Works

- Directories are stored in `~/.jumper/folders`
- Shell configuration is stored in `~/.jumper/jumper.sh`
- The `jp` shell function provides directory changing functionality
- Bash completion is automatically configured for folder names