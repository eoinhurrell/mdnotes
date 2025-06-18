# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is **mdnotes** - a Go CLI tool for managing Obsidian markdown notes. The tool provides operations for frontmatter management, heading fixes, link conversions, and external service integrations like Linkding.

## Development Commands

### Building and Testing

Use the provided Makefile for common tasks:

```bash
# Build the project
make build

# Run tests
make test

# Run tests with coverage
make test-coverage

# Clean and rebuild
make clean build

# Format and check code
make check

# Install dependencies
make deps
```

Or use Go commands directly:

```bash
# Build the project
go build -o mdnotes ./cmd

# Run tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run linter (once golangci-lint is configured)
golangci-lint run

# Run benchmarks
go test -bench=. ./...
```

### Development Workflow

The project follows Test-Driven Development (TDD) with Jujutsu VCS as outlined in plan.md:

- Write failing tests first
- Implement minimal code to pass
- Refactor and improve
- Use `jj` for commit management with logical, atomic commits

## Architecture

### Project Structure

```
mdnotes/
├── cmd/                    # Command line interface definitions
├── internal/               # Private application code
│   ├── vault/             # Obsidian vault operations
│   ├── processor/         # File processing logic
│   ├── linkding/          # External service integration
│   └── config/            # Configuration management
├── pkg/                   # Public/reusable packages
│   ├── markdown/          # Markdown parsing utilities
│   └── template/          # Template processing
└── test/                  # Test fixtures and integration tests
```

### Core Components

**Vault Operations** (`internal/vault/`):

- `VaultFile`: Represents a markdown file with frontmatter, body, links, and headings
- `Scanner`: Walks directory trees and finds markdown files
- Supports ignore patterns like `.obsidian/*`, `*.tmp`

**Processors** (`internal/processor/`):

- `FrontmatterProcessor`: Ensures fields, validates, type casts
- `HeadingProcessor`: Fixes H1 issues, ensures title consistency
- `LinkParser`: Parses wiki links, markdown links, and embeds
- `LinkConverter`: Converts between wiki/markdown formats with alias preservation
- `LinkUpdater`: Updates link references when files are moved
- `Organizer`: Renames and moves files with pattern-based organization
- `LinkdingSync`: Synchronizes URLs with Linkding bookmarking service
- `BatchProcessor`: Executes multiple operations with transactional support
- `ProgressReporter`: Terminal, JSON, and silent progress reporting modes

**File Processing**:

- Files are processed with frontmatter separated from body content
- Supports YAML frontmatter between `---` delimiters
- Preserves file structure and line endings
- Atomic operations with backup/rollback capability

## Implemented Features

### Core Foundation (Cycle 1 - Complete)
- ✅ **VaultFile**: Parses and serializes markdown with YAML frontmatter
- ✅ **Scanner**: Walks directories with ignore patterns support  
- ✅ **FrontmatterProcessor**: Ensures fields with template variable support
- ✅ **CLI Structure**: Cobra-based CLI with `frontmatter ensure` command

### Frontmatter Features (Cycle 2 - Complete)
- ✅ **Validation**: Check required fields and type constraints
- ✅ **Type Casting**: Convert strings to proper types with auto-detection
- ✅ **Field Sync**: Synchronize with file system metadata
- ✅ **Enhanced Templates**: Rich template engine with filters and variables

### Content Operations (Cycle 3 - Complete)
- ✅ **Heading Analysis & Fixing**: H1 title synchronization with frontmatter
- ✅ **Link Parsing & Management**: Comprehensive parsing for wiki links, markdown links, and embeds
- ✅ **Link Format Conversion**: Bidirectional conversion between wiki and markdown formats
- ✅ **File Organization**: Pattern-based file renaming with template support
- ✅ **Link Update Tracking**: Automatic link updates when files are moved

### External Integration (Cycle 4 - Complete)
- ✅ **Linkding API Client**: Full REST API integration with rate limiting
- ✅ **Linkding Sync Processor**: Synchronize vault URLs with bookmarks
- ✅ **Batch Operations Framework**: Multi-step operations with transactional support
- ✅ **Progress Reporting**: Terminal, JSON, and silent progress modes

### Available Commands

#### Ensure Fields
```bash
# Ensure frontmatter fields exist
mdnotes frontmatter ensure --field tags --default "[]" --field created --default "{{current_date}}" /path/to/vault

# With template variables and filters
mdnotes frontmatter ensure --field id --default "{{filename|slug}}" --field modified --default "{{file_mtime}}" /path/to/vault
```

#### Validate Frontmatter
```bash
# Check required fields and types
mdnotes frontmatter validate --required title --required tags --type tags:array --type priority:number /path/to/vault

# Verbose validation output
mdnotes frontmatter validate --required title --verbose /path/to/vault
```

#### Type Casting
```bash
# Auto-detect and cast all fields
mdnotes frontmatter cast --auto-detect /path/to/vault

# Cast specific fields to specific types
mdnotes frontmatter cast --field created --type created:date --field priority --type priority:number /path/to/vault

# Preview changes with dry-run
mdnotes frontmatter cast --auto-detect --dry-run /path/to/vault
```

#### Sync with File System
```bash
# Sync modification time
mdnotes frontmatter sync --field modified --source file-mtime /path/to/vault

# Extract from filename patterns
mdnotes frontmatter sync --field date --source "filename:pattern:^(\\d{8})" /path/to/vault

# Sync directory structure
mdnotes frontmatter sync --field category --source "path:dir" /path/to/vault
```

#### Heading Management
```bash
# Analyze heading structure and report issues
mdnotes headings analyze /path/to/vault

# Fix heading structure issues
mdnotes headings fix --ensure-h1-title --single-h1 /path/to/vault

# Preview changes with dry-run
mdnotes headings fix --fix-sequence --dry-run /path/to/vault
```

#### Link Management
```bash
# Check for broken internal links
mdnotes links check /path/to/vault

# Convert between link formats
mdnotes links convert --from wiki --to markdown /path/to/vault
mdnotes links convert --from markdown --to wiki /path/to/vault

# Preview conversions
mdnotes links convert --from wiki --to markdown --dry-run /path/to/vault
```


#### Linkding Integration
```bash
# List vault files with URLs and sync status
mdnotes linkding list /path/to/vault

# Sync URLs to Linkding bookmarks
mdnotes linkding sync /path/to/vault

# Preview sync without making changes
mdnotes linkding sync --dry-run --verbose /path/to/vault

# Sync with custom field names
mdnotes linkding sync --url-field "link" --title-field "name" /path/to/vault
```

### Template Variables
Supported in default values and templates:
- `{{current_date}}`: Current date (YYYY-MM-DD)
- `{{current_datetime}}`: Current datetime (ISO format)
- `{{filename}}`: Base filename without extension
- `{{title}}`: Value from title frontmatter field
- `{{file_mtime}}`: File modification date
- `{{relative_path}}`: Relative path from vault root
- `{{parent_dir}}`: Parent directory name
- `{{uuid}}`: Generate random UUID v4

### Template Filters
Apply filters using pipe syntax:
- `{{filename|upper}}`: Uppercase transformation
- `{{filename|lower}}`: Lowercase transformation
- `{{title|slug}}`: Convert to URL-friendly slug
- `{{file_mtime|date:Jan 2, 2006}}`: Custom date formatting

### Type Casting
Supported type conversions:
- **date**: ISO date strings to time.Time objects
- **number**: String numbers to int/float
- **boolean**: String booleans ("true"/"false") to bool
- **array**: Comma-separated strings to []string
- **null**: Empty strings to nil

## Next Development Phases

### Cycle 5: Analysis and Safety (Future)
- **Vault Analysis**: Generate statistics and reports on vault health
- **Safety Features**: Enhanced backup/restore and rollback functionality  
- **Configuration System**: YAML-based configuration with environment variable support
- **Duplicate Detection**: Find and resolve duplicate content and metadata

### Cycle 6: Polish and Release (Future)
- **Performance Optimization**: Parallel processing and memory optimization
- **Enhanced Error Messages**: User-friendly error reporting with suggestions
- **Documentation**: Comprehensive user guides and API documentation
- **Release Preparation**: Cross-platform builds and distribution

### Completed Safety Features
- ✅ **Dry-run mode**: Preview changes without applying them
- ✅ **Verbose output**: Detailed operation feedback
- ✅ **Error handling**: Clear error messages with suggestions
- ✅ **Atomic operations**: File changes are atomic
- ✅ **Backup and rollback**: Transaction support for batch operations
- ✅ **Context cancellation**: Graceful interruption support
- ✅ **Rate limiting**: External API protection

## Configuration

Configuration uses YAML format in `.obsidian-admin.yaml`:

- Vault settings (ignore patterns)
- Frontmatter rules (required fields, types, defaults)
- Link processing options
- External service credentials (environment variables)
- Output formatting preferences

## External Integrations

### Linkding API

- Sync URLs from notes to Linkding bookmarks
- Rate limiting and retry logic
- Two-way sync capability
- Handle authentication via API tokens

## Performance Considerations

- Target: Process 100+ files/second for read operations
- Use worker pools for parallel processing
- Stream large files without full memory load
- Cache parsed frontmatter and links when possible
- Memory usage should be O(1) for individual file operations

## Testing Strategy

- **Unit Tests**: 80%+ coverage for core packages
- **Integration Tests**: Real file system operations
- **Benchmarks**: Performance testing with large vaults (10,000+ files)
- **Test Fixtures**: Multiple vault sizes (minimal, standard, large)
- Mock external APIs for consistent testing

## Error Handling

Provide clear, actionable error messages with:

- File path and line number context
- Specific error explanation
- Suggested fixes or alternatives
- Support for `--on-error` flags (skip, fail, backup)

## Development Notes

- Use interfaces extensively for testability
- Implement functional options pattern for configurability
- Propagate `context.Context` for cancellation support
- Follow Go error wrapping patterns with `fmt.Errorf` and `%w`
- Use structured logging for debugging
- Atomic file operations to prevent corruption

## CLI Changes and Migration Guide

### Batch Command Removal (v1.1+)

**REMOVED**: The `batch` command has been eliminated as redundant. All commands now automatically work in batch mode when processing directories.

#### Migration from Batch Commands

**Before (DEPRECATED):**
```bash
# Old batch configuration approach
mdnotes batch execute --config batch-config.yaml /vault/path

# Batch configuration file (batch-config.yaml):
operations:
  - name: "Ensure frontmatter fields"
    command: "frontmatter.ensure"
    parameters:
      fields: ["tags", "created"]
      defaults: ["[]", "{{current_date}}"]
```

**After (RECOMMENDED):**
```bash
# Direct command usage - automatic batch processing on directories
mdnotes frontmatter ensure /vault/path --field tags --default "[]" --field created --default "{{current_date}}"

# Individual operations (replace complex batch config with direct commands)
mdnotes frontmatter ensure /vault/path --field tags --default "[]"
mdnotes frontmatter cast /vault/path --auto-detect
mdnotes headings fix /vault/path
mdnotes links check /vault/path
```

#### Key Benefits of the New Approach

1. **Simpler**: No need for separate configuration files
2. **More Direct**: Clear command-to-action mapping
3. **Better Discoverability**: Commands are obvious and self-documenting
4. **Consistent**: All commands work the same way on files and directories
5. **Faster**: Less overhead and configuration parsing

#### Automatic Batch Processing

All commands now automatically detect whether the path is a file or directory:

- **File path**: Processes single file (e.g., `note.md`)
- **Directory path**: Recursively processes all `.md` files in directory
- **Same flags**: All flags work identically for both scenarios
- **Progress reporting**: Automatic progress indication for multiple files

#### Command Flag Standardization

All commands now support consistent flag behavior:

```bash
--dry-run, -n     # Preview changes (works on all commands)
--verbose, -v     # Show every file examined and action taken
--quiet, -q       # Only show errors and final summary
--config, -c      # Specify config file path
```

