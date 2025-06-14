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

### Available Commands
```bash
# Ensure frontmatter fields exist
mdnotes frontmatter ensure --field tags --default "[]" --field created --default "{{filename}}" /path/to/vault

# Dry run to preview changes
mdnotes frontmatter ensure --field tags --default "[]" --dry-run /path/to/vault

# Verbose output
mdnotes frontmatter ensure --field tags --default "[]" --verbose /path/to/vault
```

### Template Variables
Currently supported in default values:
- `{{filename}}`: Base filename without extension  
- `{{title}}`: Value from title field if it exists

## Key Features to Implement

### Frontmatter Management

- **Ensure**: Add missing fields with default values (supports templates like `{{current_date}}`)
- **Validate**: Check against rules for required fields and types
- **Cast**: Convert string values to proper types (date, number, boolean, array)
- **Sync**: Update fields from file system data (mtime, filename patterns)

### Link Processing

- Parse wiki links: `[[note]]`, `[[note|alias]]`
- Parse markdown links: `[text](file.md)`
- Parse embeds: `![[image.png]]`
- Convert between formats while preserving aliases
- Update references when files are moved/renamed
- Always prefer markdown links, and convert wiki links to markdown links where possible.

### Safety Features

- Dry-run mode for previewing changes
- Automatic backups before modifications
- Transaction support with rollback capability
- File locking during operations

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

