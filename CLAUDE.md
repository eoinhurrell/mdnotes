# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is **mdnotes** - a Go CLI tool for managing Obsidian markdown notes. The tool provides batch operations for frontmatter management, heading fixes, link conversions, and external service integrations like Linkding.

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
- `LinkProcessor`: Parses and converts between wiki/markdown link formats
- `BatchProcessor`: Executes multiple operations with transaction support

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

### Cycle 3: Content Operations (Planned)
- **Heading Management**: Fix H1 issues, ensure title consistency
- **Link Parsing**: Parse wiki links, markdown links, and embeds
- **Link Conversion**: Convert between wiki and markdown formats
- **File Organization**: Rename and move files with link updates

### Cycle 4: External Integration (Planned)
- **Linkding API**: Sync URLs with Linkding bookmarking service
- **Batch Operations**: Multi-step operations with transaction support
- **Progress Reporting**: Live progress bars and status updates

### Completed Safety Features
- ✅ **Dry-run mode**: Preview changes without applying them
- ✅ **Verbose output**: Detailed operation feedback
- ✅ **Error handling**: Clear error messages with suggestions
- ✅ **Atomic operations**: File changes are atomic

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

