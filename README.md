# mdnotes

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)](https://github.com/eoinhurrell/mdnotes)

A powerful CLI tool for managing Obsidian markdown note vaults with automated operations, frontmatter management, and external service integrations.

## ✨ Features

- **🔧 Frontmatter Management**: Ensure, validate, cast, and sync frontmatter fields
- **📝 Content Operations**: Fix headings, parse links, and organize files  
- **🔗 Link Management**: Convert between wiki/markdown links and check integrity
- **📊 Vault Analysis**: Generate statistics, find duplicates, and assess quality
- **📤 Export & Backup**: Export filtered files with link processing and asset copying
- **🔄 External Integrations**: Sync with Linkding and other services
- **⚡ Performance**: Parallel processing and memory optimization for large vaults
- **👁️ File Watching**: Automated processing on file changes with configurable rules
- **🛡️ Safety**: Dry-run mode, backups, atomic operations, and security hardening
- **🔌 Plugin System**: Extensible architecture with hook-based plugin support
- **🏆 Enterprise Ready**: Production-grade error handling, validation, and monitoring

## 🚀 Quick Start

### Installation

```bash
# From source
git clone https://github.com/eoinhurrell/mdnotes.git
cd mdnotes
go build -o mdnotes ./cmd

# Or using make
make build
```

### Basic Usage

```bash
# Ensure all notes have required frontmatter
mdnotes frontmatter ensure --field tags --default "[]" /path/to/vault

# Validate frontmatter consistency  
mdnotes frontmatter check --required title --required tags /path/to/vault

# Fix heading structure
mdnotes headings fix --ensure-h1-title /path/to/vault

# Analyze content quality scores
mdnotes analyze content --scores /path/to/vault

# Export filtered files with link processing
mdnotes export ./backup --query "tags contains 'published'" --include-assets

# Start file watching for automated processing
mdnotes watch --config .obsidian-admin.yaml

# Always preview changes first!
mdnotes frontmatter ensure --field created --default "{{current_date}}" --dry-run /path/to/vault
```

## 📚 Complete Command Reference

### Global Shortcuts

mdnotes provides convenient shortcuts for frequently used commands:

```bash
mdnotes e [path]    # Shortcut for: frontmatter ensure
mdnotes s [path]    # Shortcut for: frontmatter set  
mdnotes f [path]    # Shortcut for: headings fix
mdnotes c [path]    # Shortcut for: links check
mdnotes q [path]    # Shortcut for: frontmatter query
```

### Frontmatter Operations

#### `mdnotes frontmatter ensure` (alias: `e`)
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

**Special Values:**
```bash
# Set field to null (not the string "null")
mdnotes frontmatter ensure --field optional_field --default null /path/to/vault
```

**Template Variables:**
- `{{current_date}}` - Current date (YYYY-MM-DD)
- `{{current_datetime}}` - Current datetime (ISO format)
- `{{filename}}` - Base filename without extension
- `{{title}}` - Value from title frontmatter field
- `{{file_mtime}}` - File modification date
- `{{relative_path}}` - Relative path from vault root
- `{{parent_dir}}` - Parent directory name
- `{{uuid}}` - Generate random UUID v4

**Template Filters:**
- `{{filename|upper}}` - Uppercase transformation  
- `{{filename|lower}}` - Lowercase transformation
- `{{title|slug}}` - Convert to URL-friendly slug
- `{{file_mtime|date:Jan 2, 2006}}` - Custom date formatting

#### `mdnotes frontmatter set` (alias: `s`)
Set frontmatter fields to specific values (always overwrites existing values).

```bash
# Set status field to 'published' for all files
mdnotes frontmatter set --field status --value "published" /path/to/vault

# Set multiple fields at once
mdnotes frontmatter set \
  --field status --value "published" \
  --field modified --value "{{current_date}}" \
  /path/to/vault
```

#### `mdnotes frontmatter check`
Validate frontmatter fields for completeness and type correctness.

```bash
# Check for required fields
mdnotes frontmatter check --required title --required tags /path/to/vault

# Validate field types
mdnotes frontmatter check --type tags:array --type priority:number /path/to/vault

# Combined validation
mdnotes frontmatter check \
  --required title --required created \
  --type tags:array --type priority:number --type published:boolean \
  /path/to/vault
```

**Supported types:** `string`, `number`, `boolean`, `array`, `date`, `null`

#### `mdnotes frontmatter query` (alias: `q`)
Query and filter frontmatter fields using advanced query language.

```bash
# Find files with specific field values
mdnotes frontmatter query --where "status = 'draft'" /path/to/vault

# Complex queries with logical operators
mdnotes frontmatter query --where "priority > 3 AND status != 'done'" /path/to/vault

# Find files missing specific fields
mdnotes frontmatter query --missing "created" /path/to/vault

# Find duplicate values
mdnotes frontmatter query --duplicates "title" /path/to/vault
```

**Enhanced Query Language:**
```bash
# Simple comparisons
--where "status = 'draft'"
--where "priority > 3"
--where "status != 'done'"

# Contains operator for exact substring matching
--where "tags contains 'urgent'"
--where "title contains 'project'"

# Date comparisons
--where "created after '2024-01-01'"
--where "modified before '2024-12-01'"
--where "updated within '7 days'"

# Logical operators
--where "priority > 3 AND status != 'done'"
--where "tags contains 'work' OR tags contains 'project'"
--where "(priority > 5 OR status = 'urgent') AND tags contains 'active'"
```

**Array-Specific Queries:**
```bash
# Exact element matching (recommended for arrays)
--where "tags has 'learning'"                    # Matches ['learning', 'study']
--where "tags has 'machine_learning'"            # Matches ['ai', 'machine_learning']

# Multiple exact matches
--where "tags has 'learning' AND tags has 'ai'"  # Both tags must exist
--where "tags has 'learning' OR tags has 'study'" # Either tag exists

# Complex array filtering
--where "tags has 'work' AND NOT (tags has 'archive')"
--where "count(tags) > 2 AND tags has 'priority'"
```

#### `mdnotes frontmatter cast` (alias: `c`)
Convert frontmatter field types with auto-detection.

```bash
# Auto-detect and cast all fields
mdnotes frontmatter cast --auto-detect /path/to/vault

# Cast specific fields to specific types  
mdnotes frontmatter cast \
  --field created --type created:date \
  --field priority --type priority:number \
  --field published --type published:boolean \
  /path/to/vault
```

**Smart Date/DateTime Formatting:**
- **Dates at midnight** (00:00:00) → `YYYY-MM-DD` format (e.g., `2023-01-15`)
- **Dates with time** → `YYYY-MM-DD HH:mm:ss` format (e.g., `2023-01-15 14:30:00`)

#### `mdnotes frontmatter sync` (alias: `sy`)
Synchronize frontmatter fields with file system metadata.

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

#### `mdnotes frontmatter download` (alias: `d`)
Download web resources from frontmatter fields and replace URLs with local file links.

```bash
# Download from all URL fields
mdnotes frontmatter download /path/to/vault

# Download from specific fields only
mdnotes frontmatter download --field cover_image --field attachment /path/to/vault
```

### Heading Operations

#### `mdnotes headings analyze`
Analyze heading structure and report issues.

```bash
# Analyze all heading issues
mdnotes headings analyze /path/to/vault
```

**Detected issues:**
- Multiple H1 headings in a file
- H1 heading doesn't match title in frontmatter
- Skipped heading levels (H1 → H3)
- Missing H1 when title exists in frontmatter

#### `mdnotes headings fix` (alias: `f`)
Fix heading structure issues automatically.

```bash
# Ensure H1 matches title from frontmatter
mdnotes headings fix --ensure-h1-title /path/to/vault

# Convert multiple H1s to H2s (keep first as H1)
mdnotes headings fix --single-h1 /path/to/vault

# Fix heading level sequences
mdnotes headings fix --fix-sequence /path/to/vault

# Apply all fixes
mdnotes headings fix --ensure-h1-title --single-h1 --fix-sequence /path/to/vault
```

### Link Operations

#### `mdnotes links check` (alias: `c`)
Verify internal link integrity and find broken links.

```bash
# Check all internal links
mdnotes links check /path/to/vault

# Check with file-relative markdown links
mdnotes links check --file-relative /path/to/vault
```

**Link types checked:**
- Wiki links: `[[Note Name]]`, `[[Note Name|Alias]]`
- Markdown links: `[text](note.md)`, `[text](path/note.md)`
- Embed links: `![[image.png]]`, `![[note.md]]`

#### `mdnotes links convert` (alias: `co`)
Convert between wiki and markdown link formats.

```bash
# Convert wiki links to markdown format
mdnotes links convert --from wiki --to markdown /path/to/vault

# Convert markdown links to wiki format  
mdnotes links convert --from markdown --to wiki /path/to/vault

# Preview conversions
mdnotes links convert --from wiki --to markdown --dry-run /path/to/vault
```

### Analysis & Reporting

#### `mdnotes analyze stats`
Generate comprehensive vault statistics.

```bash
# Basic statistics
mdnotes analyze stats /path/to/vault

# JSON output for automation
mdnotes analyze stats --format json /path/to/vault

# Save to file
mdnotes analyze stats --output stats.json --format json /path/to/vault
```

#### `mdnotes analyze content`
Analyze content quality using Zettelkasten principles.

```bash
# Analyze content quality with summary
mdnotes analyze content /path/to/vault

# Include individual file scores  
mdnotes analyze content --scores /path/to/vault

# Show detailed metric breakdown in verbose mode
mdnotes analyze content --scores --verbose /path/to/vault

# Filter by minimum quality score
mdnotes analyze content --scores --min-score 75 /path/to/vault
```

**Quality Scoring (0-100 scale):**
The analysis evaluates content based on five Zettelkasten principles:

1. **Readability** (Flesch-Kincaid Reading Ease) - Clear, simple language
2. **Link Density** - Outbound links per 100 words (optimal: 2-4 links)
3. **Completeness** - Title, summary, and adequate word count
4. **Atomicity** - One concept per note, appropriate length
5. **Recency** - Recently modified content scores higher

#### `mdnotes analyze duplicates`
Find duplicate content and similar files.

```bash
# Find exact duplicates
mdnotes analyze duplicates /path/to/vault

# Find similar content (fuzzy matching)
mdnotes analyze duplicates --similarity 0.8 /path/to/vault

# Focus on specific duplicate types
mdnotes analyze duplicates --type content /path/to/vault
```

#### `mdnotes analyze health`
Assess overall vault health and generate recommendations.

```bash
# Overall health assessment
mdnotes analyze health /path/to/vault

# JSON output for automation
mdnotes analyze health --format json /path/to/vault
```

#### `mdnotes analyze links`
Analyze link structure and connectivity patterns.

```bash
# Analyze link structure
mdnotes analyze links /path/to/vault

# Show text-based link graph
mdnotes analyze links --graph /path/to/vault

# Customize graph depth and connections
mdnotes analyze links --graph --depth 2 --min-connections 3 /path/to/vault
```

#### `mdnotes analyze trends`
Analyze vault growth trends and patterns.

```bash
# Analyze last year trends
mdnotes analyze trends /path/to/vault

# Focus on last 3 months with weekly granularity
mdnotes analyze trends --timespan 3m --granularity week /path/to/vault

# All-time trends with monthly granularity
mdnotes analyze trends --timespan all --granularity month /path/to/vault
```

### File Operations

#### `mdnotes rename` (alias: `r`)
Rename a file and update all references throughout the vault.

```bash
# Rename with explicit new name
mdnotes rename "old-note.md" "new-note.md"

# Rename using template (if no new name provided)
mdnotes rename "messy filename.md"

# Rename with custom template
mdnotes rename "note.md" --template "{{created|date:20060102150405}}-{{filename|slug}}.md"

# Specify vault root for link updates
mdnotes rename "note.md" "better-name.md" --vault "/path/to/vault"
```

**Performance Optimization:**
- **Ripgrep Integration**: Uses ripgrep for ultra-fast file discovery, processing only files that contain references
- **Smart Fallback**: If ripgrep isn't available, gracefully falls back to comprehensive vault scanning
- **Typical Speedup**: 10x-100x faster than traditional approaches, especially for large vaults

#### `mdnotes export` (alias: `e`)
Export markdown files from vault to another location with filtering and processing options.

```bash
# Export entire vault to backup folder
mdnotes export ./backup

# Export specific vault to output folder
mdnotes export ./output /path/to/vault

# Export files matching query criteria
mdnotes export ./blog --query "tags contains 'published'"
mdnotes export ./work --query "folder = 'projects/' AND status = 'active'"
mdnotes export ./recent --query "created >= '2024-01-01'"
```

**Link Processing:**
```bash
# Convert external links to plain text (default)
mdnotes export ./output --link-strategy remove

# Use frontmatter URLs for external links
mdnotes export ./output --link-strategy url

# Skip link processing entirely
mdnotes export ./output --process-links=false
```

**Advanced Features:**
```bash
# Include referenced assets (images, PDFs, etc.)
mdnotes export ./complete --include-assets

# Include files that link to exported files (recursive)
mdnotes export ./network --with-backlinks

# Normalize filenames for web compatibility
mdnotes export ./web --slugify --flatten
```

**Performance Options:**
```bash
# Use parallel processing (auto-detects CPU count)
mdnotes export ./output --parallel 0

# Use specific number of workers
mdnotes export ./output --parallel 4

# Optimize memory usage for large vaults
mdnotes export ./large-vault --optimize-memory

# Set timeout for large exports
mdnotes export ./huge-vault --timeout 30m
```

#### `mdnotes watch`
Monitor file system for changes and automatically execute mdnotes commands.

```bash
# Start watching with default config
mdnotes watch

# Start watching with specific config file
mdnotes watch --config .obsidian-admin.yaml

# Run in daemon mode (background)
mdnotes watch --daemon
```

**Configuration Example:**
Watch rules are configured in the YAML configuration file:

```yaml
watch:
  enabled: true
  debounce_timeout: "2s"
  ignore_patterns:
    - ".obsidian/*"
    - ".git/*"
    - "*.tmp"
    - "*.bak"
  rules:
    - name: "Auto-ensure frontmatter"
      paths: ["./notes/", "./inbox/"]
      events: ["create", "write"]
      actions: ["mdnotes frontmatter ensure {{file}}"]
    - name: "Sync with Linkding"
      paths: ["./notes/"]
      events: ["write"]
      actions: ["mdnotes linkding sync {{file}}"]
```

### External Integrations

#### `mdnotes linkding sync` (alias: `s`)
Synchronize URLs from vault files to Linkding bookmarks.

```bash
# Sync all files with URLs to Linkding
mdnotes linkding sync /path/to/vault

# Dry run to preview sync
mdnotes linkding sync --dry-run /path/to/vault

# Sync with custom field names
mdnotes linkding sync \
  --url-field "link" \
  --title-field "name" \
  --tags-field "categories" \
  /path/to/vault

# Enable title and tag syncing
mdnotes linkding sync --sync-title --sync-tags /path/to/vault
```

**Requirements:**
Files must have a `url` frontmatter field to be synced:
```yaml
---
title: Example Article
url: https://example.com/article
tags: [programming, tools]
---
```

After syncing, the `linkding_id` field is added:
```yaml
---
title: Example Article
url: https://example.com/article
tags: [programming, tools]
linkding_id: 123
---
```

#### `mdnotes linkding list` (alias: `l`)
List vault files containing URLs and their sync status.

```bash
# List all files with URLs
mdnotes linkding list /path/to/vault

# Example output:
# File                               Status      URL
# ----                               ------      ---
# articles/example.md               synced #123  https://example.com
# bookmarks/todo.md                 unsynced     https://todo.com
```

#### `mdnotes linkding get` (alias: `g`)
Retrieve HTML content from a note's Linkding bookmark snapshot or live URL.

```bash
# Get content from snapshot or live URL
mdnotes linkding get note.md

# Use the power alias
mdnotes ld get note.md

# Preview what would be retrieved without fetching
mdnotes linkding get note.md --dry-run

# Custom timeout and size limits
mdnotes linkding get note.md --timeout 30s --max-size 5000000
```

**How it works:**
1. **Snapshot Priority**: If the note has a `linkding_id` field, queries Linkding API for HTML snapshots
2. **Latest Selection**: Automatically selects the most recent complete snapshot
3. **Live Fallback**: If no snapshots exist, fetches content from the `url` field
4. **Text Extraction**: Strips HTML tags and returns clean text to stdout
5. **Smart Cleanup**: Automatically removes temporary files

### Shell Completion

mdnotes provides comprehensive shell completion that's dynamically generated for all commands, subcommands, and flags.

```bash
# Generate completion scripts for your shell
mdnotes completion bash > /etc/bash_completion.d/mdnotes        # Linux
mdnotes completion bash > /usr/local/etc/bash_completion.d/mdnotes  # macOS

mdnotes completion zsh > "${fpath[1]}/_mdnotes"                # zsh
mdnotes completion fish > ~/.config/fish/completions/mdnotes.fish  # fish
mdnotes completion powershell > mdnotes.ps1                    # PowerShell

# Quick setup for current session
source <(mdnotes completion bash)  # bash
mdnotes completion zsh | source    # zsh  
mdnotes completion fish | source   # fish
```

**Enhanced Completion Features:**
- **Smart path completion**: Automatically completes vault directories and markdown files
- **Frontmatter field suggestions**: Common field names (title, tags, created, modified, priority, etc.)
- **Type completion**: Field types with both `field:type` and standalone formats (string, number, boolean, array, date, null)
- **Default value templates**: Template variables (`{{current_date}}`, `{{filename}}`, `{{uuid}}`) and common values
- **Output format completion**: Valid formats (text, json, csv, yaml, table) for all analyze commands
- **Link format completion**: Wiki and markdown formats for link conversion commands
- **Query filter completion**: Pre-built filter expressions for complex queries
- **Global shortcut support**: Full completion for ultra-short commands (e, s, f, c, q)

### Global Flags

**Persistent Flags (available for all commands):**
- `--dry-run`: Preview changes without applying them
- `--verbose`: Enable detailed output showing every file examined and actions taken
- `--quiet`: Suppress all output except errors and final summary (overrides --verbose)
- `--config` (string): Config file path [default: .obsidian-admin.yaml]
- `--query` (string): Filter files using query expression (e.g., "tags contains 'published'")
- `--from-file` (string): Read file list from specified file (one file path per line)
- `--from-stdin`: Read file list from stdin (one file path per line)
- `--ignore` (multiple): Ignore patterns [default: [".obsidian/*", "*.tmp"]]

**Common Command-Specific Flags:**
- `--format` (string): Output format (text, json) [available on analysis commands]
- `--output` (string): Output file path [available on analysis commands]
- `--recursive` (bool): Process subdirectories [default: true] [available on frontmatter commands]

## ⚙️ Configuration

mdnotes uses YAML configuration files for advanced settings. The configuration file is automatically loaded from:

1. `./.obsidian-admin.yaml` (current directory)
2. `./obsidian-admin.yaml` (current directory)
3. `~/.config/obsidian-admin/config.yaml`
4. `~/.obsidian-admin.yaml`
5. `/etc/obsidian-admin/config.yaml`

### Basic Configuration

**Sample `.obsidian-admin.yaml`:**
```yaml
version: "1.0"

vault:
  path: "."
  ignore_patterns:
    - ".obsidian/*"
    - "*.tmp"
    - "templates/*"

frontmatter:
  required_fields:
    - title
    - tags
    - created
  type_rules:
    fields:
      created: date
      modified: date
      priority: number
      published: boolean
      tags: array

linkding:
  api_url: "${LINKDING_URL}"
  api_token: "${LINKDING_TOKEN}"
  sync_title: true
  sync_tags: true

batch:
  stop_on_error: false
  create_backup: true
  max_workers: 4

safety:
  backup_retention: "7d"
  max_backups: 10

watch:
  enabled: true
  debounce_timeout: "2s"
  ignore_patterns:
    - ".obsidian/*"
    - ".git/*"
    - "*.tmp"
    - "*.bak"
  rules:
    - name: "Auto-ensure frontmatter"
      paths: ["./notes/", "./inbox/"]
      events: ["create", "write"]
      actions: ["mdnotes frontmatter ensure {{file}}"]
```

### Linkding Integration Setup

**1. Set Environment Variables:**
```bash
export LINKDING_URL="https://your-linkding-instance.com"
export LINKDING_TOKEN="your-api-token-here"
```

**2. Configure `.obsidian-admin.yaml`:**
```yaml
linkding:
  api_url: "${LINKDING_URL}"
  api_token: "${LINKDING_TOKEN}"
  sync_title: true      # Sync note title to bookmark title
  sync_tags: true       # Sync note tags to bookmark tags
```

**3. Prepare vault files:**
```yaml
---
title: "Useful Tool"
url: "https://example.com/tool"
tags: [productivity, tools]
description: "A helpful productivity tool"
---

# Content about the tool
```

**4. Sync to Linkding:**
```bash
# Preview what will be synced
mdnotes linkding list /path/to/vault

# Sync URLs to bookmarks
mdnotes linkding sync /path/to/vault
```

**Environment Variable Support:**
Configuration values support environment variable expansion using `${VARIABLE_NAME}` syntax. This is recommended for sensitive values like API tokens.

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