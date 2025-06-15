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

## 📋 Complete Command Reference

### Frontmatter Operations

#### `mdnotes frontmatter ensure`
Add missing frontmatter fields with default values and template support.

**Basic Usage:**
```bash
# Add missing tags field with empty array default
mdnotes frontmatter ensure --field tags --default "[]" /path/to/vault

# Add created field with current date
mdnotes frontmatter ensure --field created --default "{{current_date}}" /path/to/vault

# Multiple fields at once
mdnotes frontmatter ensure \
  --field tags --default "[]" \
  --field priority --default "3" \
  --field status --default "draft" \
  /path/to/vault
```

**Template Variables:**
```bash
# Use template variables for dynamic defaults
mdnotes frontmatter ensure --field id --default "{{filename|slug}}" /path/to/vault
mdnotes frontmatter ensure --field modified --default "{{file_mtime}}" /path/to/vault
mdnotes frontmatter ensure --field title --default "{{filename}}" /path/to/vault
```

**Available template variables:**
- `{{current_date}}` - Current date (YYYY-MM-DD)
- `{{current_datetime}}` - Current datetime (ISO format)
- `{{filename}}` - Base filename without extension
- `{{title}}` - Value from title frontmatter field
- `{{file_mtime}}` - File modification date
- `{{relative_path}}` - Relative path from vault root
- `{{parent_dir}}` - Parent directory name
- `{{uuid}}` - Generate random UUID v4

**Template filters:**
- `{{filename|upper}}` - Uppercase transformation
- `{{filename|lower}}` - Lowercase transformation
- `{{title|slug}}` - Convert to URL-friendly slug
- `{{file_mtime|date:Jan 2, 2006}}` - Custom date formatting

**Options:**
- `--dry-run` - Preview changes without applying
- `--verbose` - Show detailed output
- `--include-pattern` - Only process files matching pattern
- `--exclude-pattern` - Skip files matching pattern

#### `mdnotes frontmatter validate`
Validate frontmatter fields for completeness and type correctness.

**Basic Usage:**
```bash
# Check for required fields
mdnotes frontmatter validate --required title --required tags /path/to/vault

# Validate field types
mdnotes frontmatter validate --type tags:array --type priority:number /path/to/vault

# Combined validation
mdnotes frontmatter validate \
  --required title --required created \
  --type tags:array --type priority:number --type published:boolean \
  /path/to/vault
```

**Supported types:**
- `string` - Text values
- `number` - Integer or float values
- `boolean` - true/false values
- `array` - List of values
- `date` - Date values
- `null` - Empty/null values

**Options:**
- `--verbose` - Show detailed validation results
- `--format json` - Output results in JSON format

#### `mdnotes frontmatter cast`
Convert frontmatter field types with auto-detection.

**Basic Usage:**
```bash
# Auto-detect and cast all fields
mdnotes frontmatter cast --auto-detect /path/to/vault

# Cast specific fields to specific types  
mdnotes frontmatter cast \
  --field created --type created:date \
  --field priority --type priority:number \
  --field published --type published:boolean \
  /path/to/vault

# Preview casting changes
mdnotes frontmatter cast --auto-detect --dry-run /path/to/vault
```

**Type casting examples:**
```bash
# String "2023-01-15" → Date 2023-01-15 (no quotes)
mdnotes frontmatter cast --field start --type start:date /path/to/vault

# String "true" → Boolean true
mdnotes frontmatter cast --field published --type published:boolean /path/to/vault

# String "tag1,tag2" → Array ["tag1", "tag2"]
mdnotes frontmatter cast --field tags --type tags:array /path/to/vault

# String "42" → Number 42
mdnotes frontmatter cast --field priority --type priority:number /path/to/vault
```

#### `mdnotes frontmatter sync`
Synchronize frontmatter fields with file system metadata.

**Basic Usage:**
```bash
# Sync modification time
mdnotes frontmatter sync --field modified --source file-mtime /path/to/vault

# Extract from filename patterns
mdnotes frontmatter sync \
  --field date --source "filename:pattern:^(\\d{8})" \
  /path/to/vault

# Sync directory structure  
mdnotes frontmatter sync --field category --source "path:dir" /path/to/vault
```

**Sync sources:**
- `file-mtime` - File modification time
- `filename` - Base filename
- `filename:pattern:REGEX` - Extract from filename using regex
- `path:dir` - Parent directory name
- `path:full` - Full relative path

**Options:**
- `--overwrite` - Overwrite existing field values
- `--dry-run` - Preview sync operations

### Content Operations

#### `mdnotes headings analyze`
Analyze heading structure and report issues.

**Basic Usage:**
```bash
# Analyze all heading issues
mdnotes headings analyze /path/to/vault

# Focus on specific issues
mdnotes headings analyze --check-h1-title --check-sequence /path/to/vault

# JSON output for automation
mdnotes headings analyze --format json /path/to/vault
```

**Detected issues:**
- Multiple H1 headings in a file
- H1 heading doesn't match title in frontmatter
- Skipped heading levels (H1 → H3)
- Missing H1 when title exists in frontmatter

#### `mdnotes headings fix`
Fix heading structure issues automatically.

**Basic Usage:**
```bash
# Ensure H1 matches title from frontmatter
mdnotes headings fix --ensure-h1-title /path/to/vault

# Convert multiple H1s to H2s (keep first as H1)
mdnotes headings fix --single-h1 /path/to/vault

# Fix heading level sequences
mdnotes headings fix --fix-sequence /path/to/vault

# Apply all fixes
mdnotes headings fix --ensure-h1-title --single-h1 --fix-sequence /path/to/vault

# Preview fixes first
mdnotes headings fix --ensure-h1-title --dry-run /path/to/vault
```

#### `mdnotes links check`
Verify internal link integrity and find broken links.

**Basic Usage:**
```bash
# Check all internal links
mdnotes links check /path/to/vault

# Show detailed information about broken links
mdnotes links check --verbose /path/to/vault

# Output results in JSON format
mdnotes links check --format json /path/to/vault
```

**Link types checked:**
- Wiki links: `[[Note Name]]`, `[[Note Name|Alias]]`
- Markdown links: `[text](note.md)`, `[text](path/note.md)`
- Embed links: `![[image.png]]`, `![[note.md]]`

#### `mdnotes links convert`
Convert between wiki and markdown link formats.

**Basic Usage:**
```bash
# Convert wiki links to markdown format
mdnotes links convert --from wiki --to markdown /path/to/vault

# Convert markdown links to wiki format  
mdnotes links convert --from markdown --to wiki /path/to/vault

# Preview conversions
mdnotes links convert --from wiki --to markdown --dry-run /path/to/vault
```

**Conversion examples:**
```bash
# Wiki to Markdown
# [[Note Name]] → [Note Name](Note%20Name.md)
# [[Note Name|Alias]] → [Alias](Note%20Name.md)

# Markdown to Wiki  
# [Note Name](Note%20Name.md) → [[Note Name]]
# [Alias](Note%20Name.md) → [[Note Name|Alias]]
```

### Analysis & Reporting

#### `mdnotes analyze stats`
Generate comprehensive vault statistics.

**Basic Usage:**
```bash
# Basic statistics
mdnotes analyze stats /path/to/vault

# Detailed output with field analysis
mdnotes analyze stats --detailed /path/to/vault

# JSON output for automation
mdnotes analyze stats --format json /path/to/vault

# Save to file
mdnotes analyze stats --output stats.json --format json /path/to/vault
```

**Statistics included:**
- Total files and size
- Frontmatter field usage
- Content metrics (word count, links, headings)
- File type distribution
- Creation and modification patterns

#### `mdnotes analyze duplicates`
Find duplicate content and similar files.

**Basic Usage:**
```bash
# Find exact duplicates
mdnotes analyze duplicates /path/to/vault

# Find similar content (fuzzy matching)
mdnotes analyze duplicates --similarity 0.8 /path/to/vault

# Include frontmatter in duplicate detection
mdnotes analyze duplicates --include-frontmatter /path/to/vault

# Output detailed results
mdnotes analyze duplicates --format json --output duplicates.json /path/to/vault
```

**Duplicate types:**
- Exact content matches
- Similar content (configurable threshold)
- Same title in frontmatter
- Identical frontmatter fields

#### `mdnotes analyze health`
Assess overall vault health and generate recommendations.

**Basic Usage:**
```bash
# Overall health assessment
mdnotes analyze health /path/to/vault

# Detailed health report
mdnotes analyze health --detailed /path/to/vault

# Focus on specific areas
mdnotes analyze health --check frontmatter --check links --check headings /path/to/vault
```

**Health metrics:**
- Frontmatter completeness score
- Link integrity percentage
- Heading structure quality
- Content organization score
- Overall health rating (0-100)

### Batch Operations

#### `mdnotes batch execute`
Execute multiple operations from a configuration file.

**Basic Usage:**
```bash
# Execute batch operations
mdnotes batch execute --config batch-config.yaml /path/to/vault

# With progress reporting
mdnotes batch execute --config batch-config.yaml --progress terminal /path/to/vault

# Dry run with JSON progress output
mdnotes batch execute --config batch-config.yaml --dry-run --progress json /path/to/vault
```

**Sample batch configuration (batch-config.yaml):**
```yaml
operations:
  - name: ensure-frontmatter
    type: frontmatter-ensure
    config:
      fields:
        - name: tags
          default: "[]"
        - name: created
          default: "{{current_date}}"
        - name: modified
          default: "{{file_mtime}}"
  
  - name: fix-headings
    type: headings-fix
    config:
      ensure_h1_title: true
      single_h1: true
  
  - name: validate-links
    type: links-check
    config:
      fix_broken: false

safety:
  dry_run: false
  backup_enabled: true
  continue_on_error: false

progress:
  mode: terminal
  verbose: true
```

**Progress modes:**
- `terminal` - Interactive progress bars
- `json` - JSON progress events
- `silent` - No progress output

#### `mdnotes batch validate`
Validate batch configuration without executing.

**Basic Usage:**
```bash
# Validate configuration file
mdnotes batch validate --config batch-config.yaml

# Verbose validation with suggestions
mdnotes batch validate --config batch-config.yaml --verbose
```

### Global Options

**Available for all commands:**
- `--help` - Show command help
- `--verbose` - Enable verbose output
- `--dry-run` - Preview changes without applying (where applicable)
- `--format` - Output format (text, json)
- `--output` - Output file path
- `--include-pattern` - Only process files matching pattern
- `--exclude-pattern` - Skip files matching pattern

**Pattern examples:**
```bash
# Process only markdown files
mdnotes frontmatter ensure --include-pattern "*.md" --field tags --default "[]" /vault

# Skip template files
mdnotes headings fix --exclude-pattern "templates/*" --ensure-h1-title /vault

# Process files in specific directory
mdnotes links check --include-pattern "notes/*" /vault
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
