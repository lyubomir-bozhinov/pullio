# pullio

A cross-platform Go application that updates all Git repositories in a directory tree. It automatically locates Git repositories, detects their default branches, and performs a fast-forward pull to update them.

## Features

- Finds all Git repositories in a directory tree
- Automatically sets up SSH agent and adds your SSH key if needed
- Detects the default branch of each repository
- Performs a fast-forward only pull to avoid merge conflicts
- Processes repositories concurrently for better performance
- Works on Linux, macOS, and Windows
- Provides clear, color-coded output with success/failure status

## Requirements

- Git must be installed and available in your PATH
- SSH key for repositories that require authentication

## Installation

### Using pre-built binaries

Download the appropriate binary for your platform from the [releases page](https://github.com/lyubomir-bozhinov/pullio/releases).

### Building from source

```bash
# Clone the repository
git clone https://github.com/lyubomir-bozhinov/pullio.git
cd pullio

# Build
go build -o bin/pullio cmd/pullio/main.go

# Or build for a specific platform
GOOS=windows GOARCH=amd64 go build -o pullio.exe cmd/pullio/main.go
```

## Usage

```bash
# Basic usage (updates all repositories in current directory)
./pullio

# Specify a different SSH key
./pullio -key ~/.ssh/my_custom_key

# Specify different default branches to try
./pullio -branches "dev,main,master"

# Set the number of concurrent operations
./pullio -concurrent 8

# Enable verbose output
./pullio -verbose

# Start from a specific directory
./pullio -path /path/to/repositories
```
user
## Command-line Options

| Option | Default | Description |
|--------|---------|-------------|
| `-key` | `~/.ssh/id_ed25519` | Path to SSH private key |
| `-branches` | `main,master` | Comma-separated list of default branch names to try |
| `-concurrent` | `4` | Number of repositories to process concurrently |
| `-verbose` | `false` | Enable verbose output |
| `-path` | `.` | Starting path to search for repositories |

## Example Output

```
ℹ️ Initializing SSH agent...
ℹ️ Finding Git repositories from .

📁 ./my-project
✅ Pulled main

📁 ./another-repo
❌ Failed to pull: git command failed: exit status 1: fatal: Not possible to fast-forward, aborting.

📦 Done. 1 updated, 1 failed.

Successfully updated repositories:
✅ ./my-project (branch: main)

Failed repositories:
❌ ./another-repo (reason: Failed to pull: git command failed: exit status 1: fatal: Not possible to fast-forward, aborting.)
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
