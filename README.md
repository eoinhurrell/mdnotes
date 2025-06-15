# mdnotes

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)](https://github.com/eoinhurrell/mdnotes)

A powerful CLI tool for managing Obsidian markdown note vaults with automated batch operations, frontmatter management, and external service integrations.

## ✨ Features

- **🔧 Frontmatter Management**: Ensure, validate, cast, and sync frontmatter fields
- **📝 Content Operations**: Fix headings, parse links, and organize files  
- **🔗 Link Management**: Convert between wiki/markdown links and check integrity
- **📊 Vault Analysis**: Generate statistics, find duplicates, and assess health
- **⚡ Batch Operations**: Execute multiple operations with progress tracking
- **🔄 External Integrations**: Sync with Linkding and other services
- **🚀 Performance**: Parallel processing and memory optimization for large vaults
- **🛡️ Safety**: Dry-run mode, backups, and atomic operations

## 🚀 Quick Start

### Installation

```bash
# From source
git clone https://github.com/eoinhurrell/mdnotes.git
cd mdnotes
go build -o mdnotes ./cmd
```

### Basic Usage

```bash
# Ensure all notes have required frontmatter
mdnotes frontmatter ensure --field tags --default "[]" /path/to/vault

# Validate frontmatter consistency  
mdnotes frontmatter validate --required title --required tags /path/to/vault

# Fix heading structure
mdnotes headings fix --ensure-h1-title /path/to/vault

# Check vault health
mdnotes analyze health /path/to/vault

# Always preview changes first!
mdnotes frontmatter ensure --field created --default "{{current_date}}" --dry-run /path/to/vault
```

## 📚 Documentation

- **[User Guide](docs/USER_GUIDE.md)** - Comprehensive usage guide with examples
- **[Development Guide](CLAUDE.md)** - Developer documentation and architecture

## 🎯 Use Cases

### Daily Vault Maintenance
- Ensure consistent frontmatter across all notes
- Validate field types and required fields
- Fix heading structure issues
- Check for broken internal links

### Bulk Import Processing
- Add missing frontmatter to imported files
- Standardize field formats and types
- Convert link formats for consistency

### Vault Analysis
- Generate comprehensive statistics
- Find duplicate content
- Assess vault health over time

## 📋 Commands Reference

### Frontmatter Operations
```bash
mdnotes frontmatter ensure     # Add missing fields
mdnotes frontmatter validate   # Check field requirements  
mdnotes frontmatter cast       # Convert field types
mdnotes frontmatter sync       # Sync with file system
```

### Content Operations  
```bash
mdnotes headings analyze       # Check heading structure
mdnotes headings fix           # Fix heading issues
mdnotes links check            # Verify link integrity
mdnotes links convert          # Convert link formats
```

### Analysis & Reporting
```bash
mdnotes analyze stats          # Generate vault statistics
mdnotes analyze duplicates     # Find duplicate content
mdnotes analyze health         # Assess vault health
```

### Batch Operations
```bash
mdnotes batch execute          # Run batch operations
mdnotes batch validate         # Validate batch config
```

## 🚀 Performance

Optimized for large vaults with thousands of files:

| Vault Size | Processing Time | Memory Usage |
|------------|----------------|--------------|
| 100 files | < 50ms | < 10MB |
| 1,000 files | < 500ms | < 50MB |
| 10,000 files | < 5s | < 200MB |

Performance features include parallel processing, memory management, and smart batching.

## 🛡️ Safety Features

- **Dry Run Mode**: Preview all changes before applying
- **Atomic Operations**: All-or-nothing file modifications
- **Backup Management**: Automatic backups with rollback capability
- **Progress Tracking**: Real-time progress with cancellation support

## 🔧 Development

### Prerequisites
- Go 1.21 or higher
- Git

### Building from Source
```bash
git clone https://github.com/eoinhurrell/mdnotes.git
cd mdnotes
go mod download
make build
```

### Running Tests
```bash
make test              # Unit tests
make test-coverage     # Tests with coverage
make bench            # Benchmarks
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

**Made with ❤️ for the Obsidian community**
