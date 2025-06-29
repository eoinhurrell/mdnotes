# Installation Guide

This guide provides comprehensive installation instructions for mdnotes across different platforms and use cases.

## Quick Start

### Download Pre-built Binary (Recommended)

1. Go to the [releases page](https://github.com/eoinhurrell/mdnotes/releases)
2. Download the appropriate binary for your platform:
   - **Linux AMD64**: `mdnotes_Linux_x86_64.tar.gz`
   - **Linux ARM64**: `mdnotes_Linux_arm64.tar.gz`
   - **macOS Intel**: `mdnotes_Darwin_x86_64.tar.gz`
   - **macOS Apple Silicon**: `mdnotes_Darwin_arm64.tar.gz`
   - **Windows**: `mdnotes_Windows_x86_64.zip`

3. Extract the archive and copy the binary to your PATH:

```bash
# Linux/macOS example
tar -xzf mdnotes_Linux_x86_64.tar.gz
sudo cp mdnotes /usr/local/bin/

# Verify installation
mdnotes --version
```

## Platform-Specific Installation

### macOS

#### Homebrew (Recommended)
```bash
# Add the tap
brew tap eoinhurrell/tap

# Install mdnotes
brew install mdnotes

# Verify installation
mdnotes --version
```

#### Manual Installation
```bash
# Download and extract
curl -L https://github.com/eoinhurrell/mdnotes/releases/latest/download/mdnotes_Darwin_x86_64.tar.gz | tar -xz

# Make executable and move to PATH
chmod +x mdnotes
sudo mv mdnotes /usr/local/bin/
```

### Linux

#### Package Managers

**Debian/Ubuntu (.deb)**:
```bash
# Download the .deb package
wget https://github.com/eoinhurrell/mdnotes/releases/latest/download/mdnotes_1.0.0_linux_amd64.deb

# Install
sudo dpkg -i mdnotes_1.0.0_linux_amd64.deb

# Fix dependencies if needed
sudo apt-get install -f
```

**RedHat/CentOS/Fedora (.rpm)**:
```bash
# Download the .rpm package
wget https://github.com/eoinhurrell/mdnotes/releases/latest/download/mdnotes_1.0.0_linux_amd64.rpm

# Install
sudo rpm -i mdnotes_1.0.0_linux_amd64.rpm
```

**Alpine (.apk)**:
```bash
# Download the .apk package
wget https://github.com/eoinhurrell/mdnotes/releases/latest/download/mdnotes_1.0.0_linux_amd64.apk

# Install
sudo apk add --allow-untrusted mdnotes_1.0.0_linux_amd64.apk
```

**Arch Linux**:
```bash
# Install from AUR (if available)
yay -S mdnotes

# Or download and install manually
wget https://github.com/eoinhurrell/mdnotes/releases/latest/download/mdnotes_1.0.0_linux_amd64.pkg.tar.xz
sudo pacman -U mdnotes_1.0.0_linux_amd64.pkg.tar.xz
```

#### Manual Installation
```bash
# Download and extract
curl -L https://github.com/eoinhurrell/mdnotes/releases/latest/download/mdnotes_Linux_x86_64.tar.gz | tar -xz

# Make executable and move to PATH
chmod +x mdnotes
sudo mv mdnotes /usr/local/bin/
```

### Windows

#### PowerShell (Recommended)
```powershell
# Download the latest release
$url = "https://github.com/eoinhurrell/mdnotes/releases/latest/download/mdnotes_Windows_x86_64.zip"
$output = "$env:TEMP\mdnotes.zip"
Invoke-WebRequest -Uri $url -OutFile $output

# Extract
Expand-Archive -Path $output -DestinationPath "$env:TEMP\mdnotes"

# Copy to a directory in your PATH (e.g., C:\Windows\System32 or create a local bin directory)
Copy-Item "$env:TEMP\mdnotes\mdnotes.exe" "C:\Windows\System32\"

# Verify installation
mdnotes --version
```

#### Manual Installation
1. Download `mdnotes_Windows_x86_64.zip` from the releases page
2. Extract the zip file
3. Copy `mdnotes.exe` to a directory in your PATH
4. Open Command Prompt or PowerShell and run `mdnotes --version`

## Advanced Installation Methods

### From Source

#### Prerequisites
- Go 1.21 or higher
- Git

#### Build and Install
```bash
# Clone the repository
git clone https://github.com/eoinhurrell/mdnotes.git
cd mdnotes

# Build
go build -o mdnotes ./cmd

# Install to GOPATH/bin
go install ./cmd

# Or copy to system PATH
sudo cp mdnotes /usr/local/bin/
```

### Docker

#### Pull and Run
```bash
# Pull the latest image
docker pull ghcr.io/eoinhurrell/mdnotes:latest

# Run with current directory as vault
docker run --rm -v $(pwd):/vault ghcr.io/eoinhurrell/mdnotes:latest analyze stats /vault

# Create an alias for convenience
echo 'alias mdnotes="docker run --rm -v $(pwd):/vault ghcr.io/eoinhurrell/mdnotes:latest"' >> ~/.bashrc
source ~/.bashrc
```

#### Docker Compose
```yaml
# docker-compose.yml
version: '3.8'
services:
  mdnotes:
    image: ghcr.io/eoinhurrell/mdnotes:latest
    volumes:
      - ./vault:/vault
    working_dir: /vault
    command: ["--help"]
```

### Development Installation

#### Using Go Install
```bash
# Install the latest version from source
go install github.com/eoinhurrell/mdnotes/cmd@latest

# Install a specific version
go install github.com/eoinhurrell/mdnotes/cmd@v1.2.3
```

#### Using Make
```bash
# Clone and build using Makefile
git clone https://github.com/eoinhurrell/mdnotes.git
cd mdnotes

# Install dependencies and build
make deps build

# Run tests
make test

# Install to system
make install
```

## Shell Completions

mdnotes provides comprehensive shell completions for all commands and flags.

### Bash

#### System-wide Installation
```bash
# Linux
sudo mdnotes completion bash > /etc/bash_completion.d/mdnotes

# macOS (Homebrew)
mdnotes completion bash > /usr/local/etc/bash_completion.d/mdnotes
```

#### User Installation
```bash
# Create completion directory if it doesn't exist
mkdir -p ~/.local/share/bash-completion/completions

# Install completion
mdnotes completion bash > ~/.local/share/bash-completion/completions/mdnotes
```

#### Session-only
```bash
# Load for current session
source <(mdnotes completion bash)
```

### Zsh

#### System-wide Installation
```bash
# Find zsh completion directory
echo $fpath

# Install completion (replace with actual path from fpath)
sudo mdnotes completion zsh > /usr/local/share/zsh/site-functions/_mdnotes
```

#### User Installation
```bash
# Create completion directory
mkdir -p ~/.local/share/zsh/completions

# Install completion
mdnotes completion zsh > ~/.local/share/zsh/completions/_mdnotes

# Add to .zshrc (if not already present)
echo 'fpath=(~/.local/share/zsh/completions $fpath)' >> ~/.zshrc
echo 'autoload -U compinit && compinit' >> ~/.zshrc
```

#### Session-only
```bash
# Load for current session
source <(mdnotes completion zsh)
```

### Fish

#### Installation
```bash
# Create completion directory if it doesn't exist
mkdir -p ~/.config/fish/completions

# Install completion
mdnotes completion fish > ~/.config/fish/completions/mdnotes.fish
```

#### Session-only
```bash
# Load for current session
mdnotes completion fish | source
```

### PowerShell

#### Installation
```powershell
# Create completion script
mdnotes completion powershell | Out-String | Invoke-Expression

# To persist across sessions, add to your PowerShell profile
mdnotes completion powershell >> $PROFILE
```

## Verification

### Binary Verification

#### Checksum Verification
```bash
# Download checksums file
wget https://github.com/eoinhurrell/mdnotes/releases/latest/download/checksums.txt

# Verify downloaded binary
sha256sum -c checksums.txt --ignore-missing
```

#### Version Verification
```bash
# Check version
mdnotes --version

# Check build information
mdnotes version
```

### Functionality Testing
```bash
# Test basic functionality
mdnotes --help

# Test completion generation
mdnotes completion bash > /dev/null && echo "Completions work"

# Test with a small vault (if you have one)
mdnotes analyze stats /path/to/your/vault
```

## Troubleshooting

### Common Issues

#### "mdnotes: command not found"
- Ensure the binary is in your PATH
- Check that the binary has execute permissions: `chmod +x mdnotes`
- Verify installation location: `which mdnotes`

#### Permission Denied
```bash
# Make binary executable
chmod +x mdnotes

# If installing system-wide, ensure you have sudo access
sudo cp mdnotes /usr/local/bin/
```

#### Completion Not Working
- Restart your shell after installing completions
- Check that completion files are in the correct location
- For bash, ensure bash-completion is installed:
  ```bash
  # Ubuntu/Debian
  sudo apt-get install bash-completion
  
  # macOS
  brew install bash-completion
  ```

#### Docker Issues
```bash
# Check if Docker is running
docker ps

# Pull the latest image
docker pull ghcr.io/eoinhurrell/mdnotes:latest

# Test with simple command
docker run --rm ghcr.io/eoinhurrell/mdnotes:latest --version
```

### Platform-Specific Issues

#### macOS Gatekeeper
If you get a security warning on macOS:
```bash
# Remove quarantine attribute
xattr -d com.apple.quarantine mdnotes

# Or allow in System Preferences > Security & Privacy
```

#### Windows Execution Policy
If PowerShell blocks execution:
```powershell
# Check current policy
Get-ExecutionPolicy

# Allow local scripts (if needed)
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

#### Linux Library Dependencies
If you get library errors:
```bash
# Check dependencies
ldd mdnotes

# Install missing libraries (example for Ubuntu)
sudo apt-get update
sudo apt-get install libc6
```

## Updating

### Homebrew
```bash
brew update
brew upgrade mdnotes
```

### Manual Updates
1. Download the new version from releases
2. Replace the existing binary
3. Verify the new version: `mdnotes --version`

### Docker
```bash
# Pull latest image
docker pull ghcr.io/eoinhurrell/mdnotes:latest

# Or pull specific version
docker pull ghcr.io/eoinhurrell/mdnotes:v1.2.3
```

## Uninstallation

### Binary Installation
```bash
# Remove binary
sudo rm /usr/local/bin/mdnotes

# Remove completions
sudo rm /etc/bash_completion.d/mdnotes  # Linux
sudo rm /usr/local/etc/bash_completion.d/mdnotes  # macOS
rm ~/.config/fish/completions/mdnotes.fish  # Fish
```

### Homebrew
```bash
brew uninstall mdnotes
brew untap eoinhurrell/tap
```

### Package Managers
```bash
# Debian/Ubuntu
sudo apt-get remove mdnotes

# RedHat/CentOS/Fedora
sudo rpm -e mdnotes

# Alpine
sudo apk del mdnotes
```

### Docker
```bash
# Remove images
docker rmi ghcr.io/eoinhurrell/mdnotes:latest
docker rmi ghcr.io/eoinhurrell/mdnotes:v1.2.3
```

## Next Steps

After installation:
1. Read the [User Guide](../README.md) for usage instructions
2. Set up shell completions for better CLI experience
3. Configure mdnotes with `.obsidian-admin.yaml` if needed
4. Try the quick start examples in the main README

For development or contributing:
1. See [CLAUDE.md](../CLAUDE.md) for development setup
2. Check out the [Release Process](RELEASES.md) documentation