# mdnotes

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)](https://github.com/eoinhurrell/mdnotes)

A powerful CLI tool for managing Obsidian markdown note vaults with automated batch operations, frontmatter management, and external service integrations.

## ✨ Features

- **🔧 Frontmatter Management**: Ensure, validate, cast, and sync frontmatter fields
- **📝 Content Operations**: Fix headings, parse links, and organize files  
- **🔗 Link Management**: Convert between wiki/markdown links and check integrity
- **📊 Vault Analysis**: Generate statistics, find duplicates, and assess quality
- **📤 Export & Backup**: Export filtered files with link processing and asset copying
- **⚡ Batch Operations**: Execute multiple operations with progress tracking
- **🔄 External Integrations**: Sync with Linkding and other services
- **🚀 Performance**: Parallel processing and memory optimization for large vaults
- **👁️ File Watching**: Automated processing on file changes with configurable rules
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

# Analyze content quality scores
mdnotes analyze content --scores /path/to/vault

# Export filtered files with link processing
mdnotes export ./backup --query "tags contains 'published'" --include-assets

# Start file watching for automated processing
mdnotes watch --config .obsidian-admin.yaml

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

### Content Export & Publishing
- Export filtered collections for publishing workflows
- Process links for external compatibility
- Include referenced assets and backlinks
- Optimize for performance with large vault exports

### Vault Analysis & Quality Assessment
- Generate comprehensive statistics and health reports
- Find duplicate content and sync conflicts
- Assess content quality with Zettelkasten scoring
- Monitor vault trends and growth patterns

## 📋 Complete Command Reference

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

# Mix null and regular defaults
mdnotes frontmatter ensure \
  --field optional_field --default null \
  --field required_field --default "default_value" \
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

**Flags:**
- `--field` (required, multiple): Field name to ensure
- `--default` (required, multiple): Default value for field (must match number of --field flags)
- `--type` (optional, multiple): Type rules in format field:type
- `--recursive` (bool): Process subdirectories [default: true]
- `--ignore` (multiple): Ignore patterns [default: [".obsidian/*", "*.tmp"]]
- `--dry-run`: Preview changes without applying
- `--verbose`: Show detailed output
- `--quiet`: Only show errors and final summary

#### `mdnotes frontmatter set` (alias: `s`)
Set frontmatter fields to specific values (always overwrites existing values).

**Basic Usage:**
```bash
# Set status field to 'published' for all files
mdnotes frontmatter set --field status --value "published" /path/to/vault

# Set multiple fields at once
mdnotes frontmatter set \
  --field status --value "published" \
  --field modified --value "{{current_date}}" \
  /path/to/vault
```

**Flags:**
- `--field` (required, multiple): Field name to set
- `--value` (required, multiple): Value for field (must match number of --field flags)
- `--type` (optional, multiple): Type rules in format field:type
- `--recursive` (bool): Process subdirectories [default: true]
- `--ignore` (multiple): Ignore patterns [default: [".obsidian/*", "*.tmp"]]
- Standard global flags (--dry-run, --verbose, --quiet)

#### `mdnotes frontmatter check` (alias: `ch`)
Validate frontmatter fields for completeness and type correctness.

**Basic Usage:**
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

# Only check for YAML parsing issues
mdnotes frontmatter check --parsing-only /path/to/vault
```

**Supported types:**
- `string` - Text values
- `number` - Integer or float values
- `boolean` - true/false values
- `array` - List of values
- `date` - Date values
- `null` - Empty/null values

**Flags:**
- `--required` (multiple): Required field names
- `--type` (multiple): Type rules in format field:type
- `--parsing-only` (bool): Only check for YAML parsing issues, skip validation rules
- `--ignore` (multiple): Ignore patterns [default: [".obsidian/*", "*.tmp"]]
- Standard global flags (--dry-run, --verbose, --quiet)

#### `mdnotes frontmatter query` (alias: `q`)
Query and filter frontmatter fields using advanced query language.

**Basic Usage:**
```bash
# Find files with specific field values
mdnotes frontmatter query --where "status = 'draft'" /path/to/vault

# Complex queries with logical operators
mdnotes frontmatter query --where "priority > 3 AND status != 'done'" /path/to/vault

# Find files missing specific fields
mdnotes frontmatter query --missing "created" /path/to/vault

# Find duplicate values
mdnotes frontmatter query --duplicates "title" /path/to/vault

# Auto-fix missing fields
mdnotes frontmatter query --missing "tags" --fix-with "[]" /path/to/vault
```

**Enhanced Query Language:**
```bash
# Simple comparisons
--where "status = 'draft'"
--where "priority > 3"
--where "status != 'done'"

# Contains operator
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

**Flags:**
- **Query Criteria (mutually exclusive):**
  - `--where` (string): Filter expression with enhanced query language
  - `--missing` (string): Find files missing this field
  - `--duplicates` (string): Find files with duplicate values for this field
- **Output Control:**
  - `--field` (multiple): Select specific fields to display
  - `--format` (string): Output format: table, json, csv, yaml [default: table]
  - `--count` (bool): Show only the count of matching files
- **Auto-fix:**
  - `--fix-with` (string): Auto-fix missing fields with this value (only with --missing)
- `--ignore` (multiple): Ignore patterns [default: [".obsidian/*", "*.tmp"]]
- Standard global flags (--dry-run, --verbose, --quiet)

#### `mdnotes frontmatter cast` (alias: `c`)
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

**Smart Date/DateTime Formatting:**
mdnotes automatically detects and formats date fields intelligently:
- **Dates at midnight** (00:00:00) → `YYYY-MM-DD` format (e.g., `2023-01-15`)
- **Dates with time** → `YYYY-MM-DD HH:mm:ss` format (e.g., `2023-01-15 14:30:00`)

```bash
# Input: start: "2023-01-15"         → Output: start: 2023-01-15
# Input: meeting: "2023-01-15 14:30" → Output: meeting: 2023-01-15 14:30:00
# Input: created: 2023-01-15T10:30:45Z → Output: created: 2023-01-15 10:30:45
```

**Supported Types:**
- `date`: ISO date strings to time.Time objects
- `number`: String numbers to int/float
- `boolean`: String booleans ("true"/"false") to bool
- `array`: Comma-separated strings to []string
- `null`: Empty strings to nil

**Flags:**
- `--field` (multiple): Field names to cast
- `--type` (multiple): Target types for fields (field:type format)
- `--auto-detect` (bool): Automatically detect and cast types
- `--ignore` (multiple): Ignore patterns [default: [".obsidian/*", "*.tmp"]]
- Standard global flags (--dry-run, --verbose, --quiet)

#### `mdnotes frontmatter sync` (alias: `sy`)
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

**Flags:**
- `--field` (required, multiple): Field names to sync
- `--source` (required, multiple): Data sources for fields (must match number of --field flags)
- `--ignore` (multiple): Ignore patterns [default: [".obsidian/*", "*.tmp"]]
- Standard global flags (--dry-run, --verbose, --quiet)

#### `mdnotes frontmatter download` (alias: `d`)
Download web resources from frontmatter fields and replace URLs with local file links.

**Basic Usage:**
```bash
# Download from all URL fields
mdnotes frontmatter download /path/to/vault

# Download from specific fields only
mdnotes frontmatter download --field cover_image --field attachment /path/to/vault
```

**Behavior:**
1. Scans frontmatter for HTTP/HTTPS URLs
2. Downloads resources to configured attachments directory
3. Renames original field to `<field>-original`
4. Replaces field value with wiki link to downloaded file

**Flags:**
- `--field` (multiple): Only download specific fields (default: all URL fields)
- `--ignore` (multiple): Ignore patterns [default: [".obsidian/*", "*.tmp"]]
- `--config` (string): Config file path
- Standard global flags (--dry-run, --verbose, --quiet)

### Heading Operations

#### `mdnotes headings analyze`
Analyze heading structure and report issues.

**Basic Usage:**
```bash
# Analyze all heading issues
mdnotes headings analyze /path/to/vault
```

**Detected issues:**
- Multiple H1 headings in a file
- H1 heading doesn't match title in frontmatter
- Skipped heading levels (H1 → H3)
- Missing H1 when title exists in frontmatter

**Flags:**
- `--ignore` (multiple): Ignore patterns [default: [".obsidian/*", "*.tmp"]]
- Standard global flags (--verbose, --quiet)

#### `mdnotes headings fix` (alias: `f`)
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

**Flags:**
- `--ensure-h1-title` (bool): Ensure H1 matches title field [default: true]
- `--single-h1` (bool): Convert extra H1s to H2s [default: true]
- `--fix-sequence` (bool): Fix skipped heading levels [default: false]
- `--ignore` (multiple): Ignore patterns [default: [".obsidian/*", "*.tmp"]]
- Standard global flags (--dry-run, --verbose, --quiet)

### Link Operations

#### `mdnotes links check` (alias: `c`)
Verify internal link integrity and find broken links.

**Basic Usage:**
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

**Link Resolution Behavior:**
- **Wiki links and embeds**: Always relative to vault root (Obsidian behavior)
- **Markdown links**: Default to vault root, or file-relative with `--file-relative`

**Flags:**
- `--file-relative` (bool): Check markdown links relative to each file's directory instead of vault root
- `--ignore` (multiple): Ignore patterns [default: [".obsidian/*", "*.tmp"]]
- Standard global flags (--verbose, --quiet)

#### `mdnotes links convert` (alias: `co`)
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

**Supported Formats:**
- **Wiki format**: `[[note]]` or `[[note|alias]]`
- **Markdown format**: `[text](note.md)`

**Flags:**
- `--from` (string): Source format (wiki, markdown) [default: wiki]
- `--to` (string): Target format (wiki, markdown) [default: markdown]
- `--ignore` (multiple): Ignore patterns [default: [".obsidian/*", "*.tmp"]]
- Standard global flags (--dry-run, --verbose, --quiet)

### Analysis & Reporting

#### `mdnotes analyze stats` (alias: `a`)
Generate comprehensive vault statistics.

**Basic Usage:**
```bash
# Basic statistics
mdnotes analyze stats /path/to/vault

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

**Flags:**
- `--format` (string): Output format (text, json) [default: text]
- `--output` (string): Output file (default: stdout)
- Standard global flags (--verbose, --quiet)

#### `mdnotes analyze duplicates`
Find duplicate content and similar files.

**Basic Usage:**
```bash
# Find exact duplicates
mdnotes analyze duplicates /path/to/vault

# Find similar content (fuzzy matching)
mdnotes analyze duplicates --similarity 0.8 /path/to/vault

# Focus on specific duplicate types
mdnotes analyze duplicates --type content /path/to/vault

# Output detailed results
mdnotes analyze duplicates --format json /path/to/vault
```

**Duplicate types:**
- `all`: All types of duplicates (default)
- `obsidian`: Obsidian sync conflicts
- `sync-conflicts`: General sync conflicts
- `content`: Content-based duplicates

**Flags:**
- `--format` (string): Output format (text, json) [default: text]
- `--similarity` (float64): Minimum similarity threshold (0.0-1.0) [default: 0.8]
- `--type` (string): Type of duplicates to find [default: all]
- Standard global flags (--verbose, --quiet)

#### `mdnotes analyze health`
Assess overall vault health and generate recommendations.

**Basic Usage:**
```bash
# Overall health assessment
mdnotes analyze health /path/to/vault

# JSON output for automation
mdnotes analyze health --format json /path/to/vault
```

**Health metrics:**
- Frontmatter completeness score
- Link integrity percentage
- Heading structure quality
- Content organization score
- Overall health rating (0-100)

**Flags:**
- `--format` (string): Output format (text, json) [default: text]
- Standard global flags (--verbose, --quiet)

#### `mdnotes analyze links` (alias: `l`)
Analyze link structure and connectivity patterns.

**Basic Usage:**
```bash
# Analyze link structure
mdnotes analyze links /path/to/vault

# Show text-based link graph
mdnotes analyze links --graph /path/to/vault

# Customize graph depth and connections
mdnotes analyze links --graph --depth 2 --min-connections 3 /path/to/vault
```

**Flags:**
- `--format` (string): Output format (text, json) [default: text]
- `--graph` (bool): Show text-based link graph visualization
- `--depth` (int): Maximum depth for graph visualization [default: 3]
- `--min-connections` (int): Minimum connections to show in graph [default: 1]
- Standard global flags (--verbose, --quiet)

#### `mdnotes analyze content` (alias: `c`)
Analyze content quality using Zettelkasten principles.

**Basic Usage:**
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

**Enhanced Features:**
- **Worst-scoring files** shown in summary for immediate attention
- **Verbose mode** displays individual metrics breakdown in tabular format
- **Actionable suggestions** provided for each low-scoring file
- **Score distribution** shows vault-wide quality patterns

**Output Examples:**
```bash
# Summary shows worst files needing attention:
⚠️  Files Needing Attention (lowest scores):
  1. 42.1  drafts/incomplete-note.md
      → Add more links to related concepts
  2. 45.8  notes/stub-article.md  
      → Expand content - add more detail

# Verbose mode shows detailed breakdown:
📊 Individual File Scores (showing files >= 0.0):
Score  File                          Read Link Comp Atom Rec
52.4   poor-note.md                   77    0   10   75  100
       Improvements: Add title; Link to related concepts
```

**Flags:**
- `--format` (string): Output format (text, json) [default: text]
- `--scores` (bool): Include individual file quality scores
- `--min-score` (float64): Minimum quality score to display (0.0-100) [default: 0.0]
- `--verbose` (global): Show detailed metric breakdown for each file
- Standard global flags (--quiet)

#### `mdnotes analyze trends` (alias: `t`)
Analyze vault growth trends and patterns.

**Basic Usage:**
```bash
# Analyze last year trends
mdnotes analyze trends /path/to/vault

# Focus on last 3 months with weekly granularity
mdnotes analyze trends --timespan 3m --granularity week /path/to/vault

# All-time trends with monthly granularity
mdnotes analyze trends --timespan all --granularity month /path/to/vault
```

**Flags:**
- `--format` (string): Output format (text, json) [default: text]
- `--timespan` (string): Time span to analyze (1w, 1m, 3m, 6m, 1y, all) [default: 1y]
- `--granularity` (string): Time granularity (day, week, month, quarter) [default: month]
- Standard global flags (--verbose, --quiet)

### File Operations

#### `mdnotes rename` (alias: `r`)
Rename a file and update all references throughout the vault.

**Basic Usage:**
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

**Template Variables:**
- `{{created|date:20060102150405}}`: Formatted creation time from frontmatter
- `{{filename|slug}}`: Slugified filename
- `{{filename}}`: Original filename
- `{{file_mtime}}`: File modification time
- `{{current_date}}`: Current date

**Behavior:**
1. Renames the source file to the target name
2. Uses ripgrep (if available) to quickly find files containing references
3. Updates all wiki links and markdown links pointing to the renamed file
4. Creates target directories if needed
5. Falls back to full vault scan if ripgrep is unavailable

**Performance Optimization:**
- **Ripgrep Integration**: Uses ripgrep for ultra-fast file discovery, processing only files that contain references
- **Smart Fallback**: If ripgrep isn't available, gracefully falls back to comprehensive vault scanning
- **Typical Speedup**: 10x-100x faster than traditional approaches, especially for large vaults

**Flags:**
- `--template` (string): Template for default rename target [default: "{{created|date:20060102150405}}-{{filename|slug}}.md"]
- `--vault` (string): Vault root directory for link updates [default: "."]
- `--ignore` (multiple): Ignore patterns for scanning vault [default: [".obsidian/*", "*.tmp"]]
- Standard global flags (--dry-run, --verbose, --quiet)

#### `mdnotes export` (alias: `e`)
Export markdown files from vault to another location with filtering and processing options.

**Basic Usage:**
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

**Preview and Debugging:**
```bash
# Preview what would be exported without copying
mdnotes export ./output --dry-run

# Show detailed progress information
mdnotes export ./output --verbose

# Minimize output (errors only)
mdnotes export ./output --quiet
```

**Behavior:**
1. Scans vault for markdown files
2. Filters files based on query (if provided)
3. Optionally expands selection with backlinks
4. Normalizes filenames (if requested)
5. Copies files while preserving directory structure
6. Processes links according to strategy
7. Copies referenced assets (if requested)

**Error Handling:**
The export command provides clear error messages for common issues:
- Invalid query syntax with suggestions
- Missing vault directories with helpful paths
- Permission errors with recommended fixes
- Output directory conflicts with resolution options

**Performance Guidelines:**
- For vaults with <100 files: ~1 second processing time
- For vaults with <1000 files: ~10 seconds processing time
- Use `--parallel` flag for vaults with >50 files
- Use `--optimize-memory` for vaults with >1000 files
- Large vaults benefit from SSD storage and adequate RAM

**Flags:**
- **Query & Filtering:**
  - `--query` (string): Query to filter which files are exported (uses frontmatter query syntax)
  - `--ignore` (multiple): Ignore patterns for scanning vault [default: [".obsidian/*", "*.tmp"]]
- **Link Processing:**
  - `--process-links` (bool): Process and rewrite links in exported files [default: true]
  - `--link-strategy` (string): Strategy for handling external links: 'remove' (convert to plain text) or 'url' (use frontmatter URL field) [default: remove]
- **Advanced Features:**
  - `--include-assets` (bool): Copy referenced assets (images, PDFs, etc.) to output directory
  - `--with-backlinks` (bool): Include files that link to exported files (recursive)
  - `--slugify` (bool): Convert filenames to URL-safe slugs
  - `--flatten` (bool): Put all files in a single directory
- **Performance:**
  - `--parallel` (int): Number of parallel workers for file processing (0 = auto-detect) [default: 0]
  - `--optimize-memory` (bool): Use memory-optimized processing for large vaults
  - `--timeout` (duration): Maximum time to wait for export to complete [default: 10m]
- Standard global flags (--dry-run, --verbose, --quiet)

#### `mdnotes watch`
Monitor file system for changes and automatically execute mdnotes commands.

**Basic Usage:**
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

**Watch Events:**
- `create`: File created
- `write`: File modified
- `remove`: File deleted
- `rename`: File moved/renamed
- `chmod`: File permissions changed

**Action Placeholders:**
- `{{file}}`: Full file path
- `{{dir}}`: Directory containing the file
- `{{basename}}`: Filename only

**Behavior:**
1. Monitors specified paths for markdown file changes
2. Debounces rapid events to avoid duplicate processing
3. Executes configured actions when events match rules
4. Runs in foreground by default, or background with `--daemon`

**Flags:**
- `--config` (string): Path to configuration file
- `--daemon` (bool): Run in daemon mode (background)
- Standard global flags (--verbose, --quiet)

### External Integrations

#### `mdnotes linkding sync` (alias: `s`)
Synchronize URLs from vault files to Linkding bookmarks.

**Basic Usage:**
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

**Flags:**
- `--url-field` (string): Frontmatter field containing the URL [default: "url"]
- `--title-field` (string): Frontmatter field containing the title [default: "title"]
- `--tags-field` (string): Frontmatter field containing tags [default: "tags"]
- `--sync-title` (bool): Sync title to Linkding [default: false]
- `--sync-tags` (bool): Sync tags to Linkding [default: false]
- Standard global flags (--dry-run, --verbose, --quiet)

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

**Flags:**
- Standard global flags (--verbose, --quiet)

### Global Commands and Options

#### Command Aliases
mdnotes provides convenient aliases for frequently used commands:
- `e` → `frontmatter ensure`
- `s` → `frontmatter set`
- `f` → `headings fix`
- `c` → `links check`
- `q` → `frontmatter query`

#### Shell Completion

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
- **Sync source completion**: File metadata sources (file-mtime, filename patterns, path components)
- **Query filter completion**: Pre-built filter expressions for complex queries
- **Global shortcut support**: Full completion for ultra-short commands (e, s, f, c, q)

**Persistent Flags (available for all commands):**
- `--dry-run`: Preview changes without applying them
- `--verbose`: Enable detailed output showing every file examined and actions taken
- `--quiet`: Suppress all output except errors and final summary (overrides --verbose)
- `--config` (string): Config file path [default: .obsidian-admin.yaml]
- `--help`: Show command help

**Common Command-Specific Flags:**
- `--format` (string): Output format (text, json) [available on analysis commands]
- `--output` (string): Output file path [available on analysis commands]
- `--ignore` (multiple): Ignore patterns [default: [".obsidian/*", "*.tmp"]] [available on most commands]
- `--recursive` (bool): Process subdirectories [default: true] [available on frontmatter commands]

### Development and Profiling

#### `mdnotes profile` (hidden command)
Performance profiling and benchmarking tools.

**CPU Profiling:**
```bash
# Generate CPU profile
mdnotes profile cpu --duration 30s --output cpu.prof /path/to/vault
```

**Memory Profiling:**
```bash
# Generate memory profile
mdnotes profile memory --output memory.prof /path/to/vault
```

**Benchmarking:**
```bash
# Run performance benchmarks
mdnotes profile benchmark --iterations 5 /path/to/vault

# Generate test vault and benchmark
mdnotes profile benchmark --generate 1000 --workers 8 --parallel /path/to/vault
```

**Profile Flags:**
- **CPU profiling:**
  - `--output` (string): Output file for CPU profile [default: "cpu.prof"]
  - `--duration` (duration): Duration to run profile [default: 30s]
- **Memory profiling:**
  - `--output` (string): Output file for memory profile [default: "memory.prof"]
- **Benchmarking:**
  - `--iterations` (int): Number of benchmark iterations [default: 5]
  - `--generate` (int): Generate test vault with N files [default: 0]
  - `--workers` (int): Number of workers for parallel tests [default: runtime.NumCPU()]
  - `--parallel` (bool): Enable parallel processing tests [default: true]

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
