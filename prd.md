# Obsidian Admin CLI Tool - Product Requirements Document

**Version:** 1.0  
**Date:** June 2025  
**Product Name:** obsidian-admin

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Product Overview](#product-overview)
3. [Goals and Objectives](#goals-and-objectives)
4. [User Personas](#user-personas)
5. [Functional Requirements](#functional-requirements)
6. [Technical Requirements](#technical-requirements)
7. [User Interface Specifications](#user-interface-specifications)
8. [Data Models](#data-models)
9. [API Specifications](#api-specifications)
10. [Security and Privacy](#security-and-privacy)
11. [Performance Requirements](#performance-requirements)
12. [Testing Requirements](#testing-requirements)
13. [Implementation Plan](#implementation-plan)
14. [Success Metrics](#success-metrics)
15. [Risks and Mitigation](#risks-and-mitigation)

---

## Executive Summary

The Obsidian Admin CLI tool is a command-line utility designed to automate and standardize administrative tasks for Obsidian vaults. It provides powerful batch operations for managing frontmatter, headings, links, and file organization while maintaining the integrity of markdown files and preserving the user's organizational structure.

Key capabilities include:
- Automated frontmatter management with type casting
- Heading structure validation and correction
- Link format conversion and update tracking
- Integration with external services (Linkding)
- Batch operations with safety features

---

## Product Overview

### Problem Statement

Obsidian users managing large vaults face challenges:
- Inconsistent frontmatter across files
- Broken links after file reorganization
- Mixed link formats (wiki-style vs markdown)
- Manual maintenance of metadata
- No bulk operations for vault maintenance
- Difficulty integrating with external services

### Solution

A comprehensive CLI tool that:
- Automates repetitive vault maintenance tasks
- Ensures consistency across all notes
- Preserves data integrity with safety features
- Integrates with external services seamlessly
- Provides both interactive and scriptable interfaces

### Key Differentiators

- **Non-destructive**: All operations preserve original data
- **Vault-aware**: Understands Obsidian's specific conventions
- **Performance**: Handles vaults with 10,000+ files efficiently
- **Extensible**: Plugin architecture for custom operations
- **Safe**: Dry-run mode, backups, and undo functionality

---

## Goals and Objectives

### Primary Goals

1. **Standardization**: Ensure consistent structure across all vault files
2. **Automation**: Reduce manual maintenance time by 90%
3. **Safety**: Zero data loss during operations
4. **Integration**: Seamless connection with external tools

### Specific Objectives

- Process 1,000 files in under 10 seconds
- Support all common Obsidian conventions
- Provide clear, actionable error messages
- Enable both one-time and scheduled operations
- Maintain 100% compatibility with Obsidian's parser

### Success Criteria

- Adoption by 1,000+ Obsidian power users within 6 months
- 95% success rate for automated operations
- Zero reported data loss incidents
- 4.5+ star rating on package repositories

---

## User Personas

### Primary Persona: "Alex the Knowledge Worker"

- **Profile**: 30-45 years old, uses Obsidian for personal knowledge management
- **Vault Size**: 1,000-5,000 notes
- **Technical Skill**: Comfortable with command line
- **Pain Points**:
  - Inconsistent frontmatter from various import sources
  - Broken links after reorganizing folders
  - Time spent on manual maintenance
- **Goals**:
  - Automate repetitive tasks
  - Ensure vault consistency
  - Integrate with other tools in workflow

### Secondary Persona: "Sam the Researcher"

- **Profile**: Graduate student or academic researcher
- **Vault Size**: 5,000+ notes with complex linking
- **Technical Skill**: Basic command line usage
- **Pain Points**:
  - Managing citations and references
  - Keeping metadata synchronized
  - Batch operations on research notes
- **Goals**:
  - Maintain academic standards for organization
  - Automate bibliography management
  - Generate reports on vault structure

### Tertiary Persona: "Jordan the Developer"

- **Profile**: Software developer using Obsidian for documentation
- **Vault Size**: 500-2,000 technical notes
- **Technical Skill**: Advanced command line user
- **Pain Points**:
  - Integrating documentation workflow with code
  - Maintaining consistent formatting
  - Automating vault operations in CI/CD
- **Goals**:
  - Script vault maintenance
  - Integrate with development workflow
  - Ensure documentation standards

---

## Functional Requirements

### 1. Frontmatter Management

#### 1.1 Ensure Command
**Purpose**: Add missing frontmatter fields with default values

**Functionality**:
- Add specified fields if not present
- Preserve existing field order
- Support templated default values
- Handle nested frontmatter structures

**Command Structure**:
```bash
obsidian-admin frontmatter ensure [flags] <path>
```

**Flags**:
- `--field <name>`: Field name to ensure
- `--default <value>`: Default value (supports templates)
- `--type <type>`: Expected field type
- `--force`: Overwrite existing empty values
- `--recursive`: Process subdirectories

**Template Variables**:
- `{{current_date}}`: Current date in YYYY-MM-DD format
- `{{current_datetime}}`: Current datetime in ISO format
- `{{filename}}`: Base filename without extension
- `{{filepath}}`: Relative file path
- `{{uuid}}`: Generate UUID v4

#### 1.2 Validate Command
**Purpose**: Check frontmatter against defined rules

**Functionality**:
- Validate required fields presence
- Check field types
- Verify value constraints
- Report validation errors

**Command Structure**:
```bash
obsidian-admin frontmatter validate [flags] <path>
```

**Flags**:
- `--required <fields>`: Comma-separated required fields
- `--schema <file>`: JSON schema file for validation
- `--strict`: Fail on unknown fields

#### 1.3 Sync Command
**Purpose**: Synchronize frontmatter with file attributes

**Functionality**:
- Update fields based on file system data
- Generate IDs from filenames
- Sync dates with file timestamps
- Update computed fields

**Command Structure**:
```bash
obsidian-admin frontmatter sync [flags] <path>
```

**Flags**:
- `--field <name>`: Field to sync
- `--source <source>`: Data source (file-mtime, filename, etc.)
- `--pattern <pattern>`: Pattern for extraction

#### 1.4 Cast Command
**Purpose**: Convert frontmatter field types

**Functionality**:
- Cast string values to appropriate types
- Validate values before casting
- Handle casting errors gracefully
- Support batch type inference

**Command Structure**:
```bash
obsidian-admin frontmatter cast [flags] <path>
```

**Flags**:
- `--field <name>`: Field to cast
- `--type <type>`: Target type (date, number, boolean, array, null)
- `--auto-detect`: Automatically detect and cast types
- `--validate`: Validate values before casting
- `--on-error <action>`: Error handling (skip, fail, backup)

**Supported Type Conversions**:
- String to Date: `'2023-01-01'` → `2023-01-01`
- String to Number: `'42'` → `42`
- String to Boolean: `'true'` → `true`
- String to Array: `'tag1, tag2'` → `[tag1, tag2]`
- Empty to Null: `''` → `null`

### 2. Heading Management

#### 2.1 Fix Command
**Purpose**: Correct heading hierarchy issues

**Functionality**:
- Ensure first content line is H1
- Match H1 to title field
- Enforce single H1 rule
- Adjust heading levels

**Command Structure**:
```bash
obsidian-admin headings fix [flags] <path>
```

**Flags**:
- `--ensure-h1-title`: First line must be H1 matching title
- `--single-h1`: Only one H1 allowed
- `--min-level <n>`: Minimum heading level after H1
- `--fix-sequence`: Fix skipped heading levels

#### 2.2 Validate Command
**Purpose**: Check heading structure without modifications

**Functionality**:
- Report heading hierarchy issues
- Check for multiple H1s
- Verify heading sequence
- Validate title consistency

### 3. File Organization

#### 3.1 Rename Command
**Purpose**: Rename files based on frontmatter

**Functionality**:
- Generate filenames from templates
- Handle naming conflicts
- Update internal links
- Preserve file history

**Command Structure**:
```bash
obsidian-admin organize rename [flags] <path>
```

**Flags**:
- `--pattern <template>`: Naming pattern
- `--on-conflict <action>`: Conflict resolution (skip, number, ask)
- `--update-links`: Update references to renamed files

**Pattern Variables**:
- `{{field}}`: Frontmatter field value
- `{{field|slug}}`: Slugified field value
- `{{field|date:FORMAT}}`: Date formatting

#### 3.2 Move Command
**Purpose**: Organize files into directories

**Functionality**:
- Move based on frontmatter values
- Create directories as needed
- Update all references
- Support rule-based moving

**Command Structure**:
```bash
obsidian-admin organize move [flags] <path>
```

**Flags**:
- `--by <field>`: Field to organize by
- `--rule <expression>`: Custom rule for moving
- `--create-dirs`: Create missing directories

### 4. Link Management

#### 4.1 Check Command
**Purpose**: Find broken internal links

**Functionality**:
- Scan all link formats
- Verify target existence
- Check anchor validity
- Report ambiguous links

#### 4.2 Update Command
**Purpose**: Update links after file operations

**Functionality**:
- Track file movements
- Update all link formats
- Handle embedded content
- Preserve link text

**Command Structure**:
```bash
obsidian-admin links update [flags] <path>
```

**Flags**:
- `--moved <old> --to <new>`: Single file move
- `--from-log <file>`: Batch updates from log
- `--format <formats>`: Link formats to update

#### 4.3 Convert Command
**Purpose**: Convert between link formats

**Functionality**:
- Wiki to Markdown conversion
- Preserve aliases and text
- Handle relative paths
- Add file extensions

**Command Structure**:
```bash
obsidian-admin links convert [flags] <path>
```

**Flags**:
- `--from <format>`: Source format (wiki, markdown)
- `--to <format>`: Target format
- `--preserve-aliases`: Keep link aliases
- `--relative-paths`: Use relative paths
- `--add-extension`: Add .md extension

**Conversion Examples**:
- `[[note]]` → `[note](note.md)`
- `[[note|alias]]` → `[alias](note.md)`
- `![[image.png]]` → `![image](image.png)`

### 5. Linkding Integration

#### 5.1 Sync Command
**Purpose**: Sync URLs with Linkding bookmarking service

**Functionality**:
- Find URLs without Linkding IDs
- Create bookmarks via API
- Update frontmatter with IDs
- Handle rate limiting

**Command Structure**:
```bash
obsidian-admin linkding sync [flags] <path>
```

**Flags**:
- `--api-url <url>`: Linkding API endpoint
- `--api-token <token>`: Authentication token
- `--url-field <field>`: Field containing URL
- `--id-field <field>`: Field for Linkding ID
- `--sync-title`: Send note title
- `--sync-tags`: Synchronize tags

**API Integration**:
- POST to `/api/bookmarks/`
- Handle 429 rate limiting
- Retry failed requests
- Batch operations support

#### 5.2 Check Command
**Purpose**: Verify Linkding sync status

**Functionality**:
- List sync status
- Verify ID validity
- Find orphaned bookmarks
- Generate sync report

#### 5.3 Update Command
**Purpose**: Update existing Linkding bookmarks

**Functionality**:
- Sync changed metadata
- Update titles and tags
- Handle bookmark deletion
- Two-way sync option

### 6. Batch Operations

#### 6.1 Apply Command
**Purpose**: Run multiple operations sequentially

**Functionality**:
- Execute from config file
- Transaction support
- Progress reporting
- Error recovery

**Command Structure**:
```bash
obsidian-admin batch apply --config <file> <path>
```

**Config Format**:
```yaml
operations:
  - name: "Ensure standard frontmatter"
    command: frontmatter ensure
    parameters:
      fields:
        - name: tags
          default: []
        - name: created
          default: "{{current_date}}"
  - name: "Fix headings"
    command: headings fix
    parameters:
      ensure-h1-title: true
      single-h1: true
```

### 7. Analysis Commands

#### 7.1 Stats Command
**Purpose**: Generate vault statistics

**Functionality**:
- File counts and sizes
- Link graph analysis
- Tag distribution
- Frontmatter usage

**Output Formats**:
- Table (default)
- JSON
- CSV
- Markdown report

#### 7.2 Duplicates Command
**Purpose**: Find duplicate content

**Functionality**:
- Title matching
- Content similarity
- Fuzzy matching
- Duplicate reporting

---

## Technical Requirements

### Technology Stack

- **Language**: Go 1.21+
- **CLI Framework**: Cobra
- **Configuration**: Viper
- **Markdown Parsing**: goldmark with Obsidian extensions
- **HTTP Client**: Standard library with retry logic
- **Testing**: Go testing package + testify
- **Build System**: Makefile with goreleaser

### Architecture

```
obsidian-admin/
├── cmd/                    # Command definitions
│   ├── root.go
│   ├── frontmatter.go
│   ├── headings.go
│   ├── links.go
│   └── linkding.go
├── internal/
│   ├── vault/             # Vault operations
│   │   ├── file.go
│   │   ├── frontmatter.go
│   │   └── parser.go
│   ├── processor/         # File processors
│   │   ├── interface.go
│   │   └── batch.go
│   ├── linkding/         # Linkding client
│   │   ├── client.go
│   │   └── types.go
│   └── config/           # Configuration
│       └── config.go
├── pkg/                   # Public packages
│   ├── markdown/         # Markdown utilities
│   └── template/         # Template engine
└── test/                 # Test fixtures
```

### Performance Requirements

- **File Processing**: 100+ files/second for read operations
- **Memory Usage**: O(1) memory for file operations (streaming)
- **Startup Time**: < 100ms
- **Large Vaults**: Support 50,000+ files
- **Parallel Processing**: Utilize all CPU cores

### Dependencies

```go
module github.com/user/obsidian-admin

require (
    github.com/spf13/cobra v1.8.0
    github.com/spf13/viper v1.18.0
    github.com/yuin/goldmark v1.6.0
    github.com/schollz/progressbar/v3 v3.14.0
    github.com/fatih/color v1.16.0
    golang.org/x/sync v0.6.0
)
```

---

## User Interface Specifications

### Command Line Interface

#### Global Options
```bash
obsidian-admin [global-options] <command> [command-options] <path>

Global Options:
  -h, --help              Show help
  -v, --version           Show version
  --config <file>         Config file (default: .obsidian-admin.yaml)
  --dry-run              Preview changes without applying
  --verbose              Verbose output
  --quiet                Suppress non-error output
  --no-color             Disable colored output
  --backup <dir>         Backup directory
  --parallel <n>         Number of workers (default: CPU count)
  --format <fmt>         Output format (text, json, csv)
```

#### Output Formatting

**Standard Output**:
```
Processing vault at: /path/to/vault
Found 1,234 markdown files

✓ Processed: example-note.md
  - Added field: tags = []
  - Added field: created = 2025-06-14

⚠ Skipped: draft-note.md
  - Reason: File locked by another process

✗ Failed: broken-note.md
  - Error: Invalid frontmatter syntax at line 3

Summary:
- Processed: 1,230 files
- Skipped: 3 files  
- Failed: 1 file
- Time: 2.34s
```

**JSON Output**:
```json
{
  "operation": "frontmatter ensure",
  "vault": "/path/to/vault",
  "results": {
    "processed": 1230,
    "skipped": 3,
    "failed": 1,
    "duration": "2.34s"
  },
  "files": [
    {
      "path": "example-note.md",
      "status": "success",
      "changes": [
        {"action": "add_field", "field": "tags", "value": []},
        {"action": "add_field", "field": "created", "value": "2025-06-14"}
      ]
    }
  ]
}
```

### Progress Indicators

- **Spinner**: For indeterminate operations
- **Progress Bar**: For file processing with ETA
- **Live Updates**: Current file being processed
- **Summary Stats**: Real-time counts

### Error Messages

```
Error: Cannot cast value 'not-a-date' to date type
File: projects/example.md
Field: start_date
Line: 5

Suggestion: Value must be in YYYY-MM-DD format
Example: start_date: 2025-06-14

Use --on-error skip to ignore invalid values
```

---

## Data Models

### Vault File Model

```go
type VaultFile struct {
    Path         string
    RelativePath string
    Content      []byte
    Frontmatter  map[string]interface{}
    Body         string
    Links        []Link
    Headings     []Heading
    Modified     time.Time
}

type Link struct {
    Type     LinkType // wiki, markdown, embed
    Target   string
    Alias    string
    Position Position
}

type Heading struct {
    Level    int
    Text     string
    Position Position
}

type Position struct {
    Line   int
    Column int
}
```

### Configuration Model

```go
type Config struct {
    Vault      VaultConfig
    Frontmatter FrontmatterConfig
    Links      LinksConfig
    Linkding   LinkdingConfig
    Output     OutputConfig
}

type FrontmatterConfig struct {
    RequiredFields []string
    DefaultValues  map[string]interface{}
    TypeRules      TypeRules
    PreserveOrder  bool
}

type TypeRules struct {
    Fields   map[string]string
    Patterns []PatternRule
}
```

### Operation Result Model

```go
type OperationResult struct {
    Operation string
    File      string
    Status    Status
    Changes   []Change
    Error     error
    Duration  time.Duration
}

type Change struct {
    Type     ChangeType
    Field    string
    OldValue interface{}
    NewValue interface{}
    Position Position
}
```

---

## API Specifications

### Linkding API Integration

#### Create Bookmark
```http
POST /api/bookmarks/
Authorization: Token {api_token}
Content-Type: application/json

{
  "url": "https://example.com",
  "title": "Example Title",
  "description": "Note excerpt",
  "tag_names": ["obsidian", "article"],
  "is_archived": false
}

Response:
{
  "id": 123,
  "url": "https://example.com",
  "title": "Example Title",
  "tag_names": ["obsidian", "article"],
  "date_added": "2025-06-14T10:00:00Z"
}
```

#### Update Bookmark
```http
PATCH /api/bookmarks/{id}/
Authorization: Token {api_token}

{
  "title": "Updated Title",
  "tag_names": ["obsidian", "article", "updated"]
}
```

#### Check Bookmark
```http
GET /api/bookmarks/{id}/
Authorization: Token {api_token}

Response: 200 OK or 404 Not Found
```

### Rate Limiting

- Respect `X-RateLimit-*` headers
- Exponential backoff on 429 responses
- Configurable rate limiting
- Batch operations support

---

## Security and Privacy

### Authentication

- API tokens stored in environment variables
- Config file permissions check (600)
- No credentials in logs or output
- Secure token transmission

### Data Protection

- Local processing only
- No telemetry or analytics
- Backup files with same permissions
- Sanitized error messages

### Vault Integrity

- Atomic file operations
- Backup before modification
- Rollback on failure
- File locking during operations

---

## Performance Requirements

### Benchmarks

| Operation | Files | Target Time | Max Memory |
|-----------|-------|-------------|------------|
| Frontmatter Ensure | 1,000 | < 5s | 100MB |
| Link Check | 10,000 | < 30s | 500MB |
| Type Cast | 1,000 | < 3s | 50MB |
| Linkding Sync | 100 | < 60s | 50MB |

### Optimization Strategies

- **Parallel Processing**: Worker pool for file operations
- **Streaming**: Process large files without loading fully
- **Caching**: Cache parsed frontmatter and links
- **Lazy Loading**: Load file content only when needed
- **Batch Operations**: Group API calls

---

## Testing Requirements

### Unit Tests

- **Coverage**: Minimum 80% code coverage
- **Mocking**: Mock file system and HTTP clients
- **Edge Cases**: Empty files, malformed markdown, Unicode

### Integration Tests

- **File Operations**: Real file system operations
- **API Integration**: Mock Linkding server
- **Vault Scenarios**: Various vault structures

### End-to-End Tests

- **Command Execution**: Full command flows
- **Error Scenarios**: Network failures, invalid input
- **Performance Tests**: Large vault processing

### Test Fixtures

```
test/fixtures/
├── vaults/
│   ├── minimal/       # 10 files
│   ├── standard/      # 100 files  
│   └── large/         # 1000 files
├── configs/
│   └── *.yaml
└── markdown/
    ├── valid/
    └── invalid/
```

---

## Implementation Plan

### Phase 1: Core Foundation (Weeks 1-3)

**Week 1**:
- Project setup and structure
- Basic CLI framework
- File reading and parsing
- Frontmatter parser

**Week 2**:
- Frontmatter ensure command
- Frontmatter validate command
- Basic tests and documentation

**Week 3**:
- Configuration system
- Backup functionality
- Error handling framework

### Phase 2: Essential Features (Weeks 4-6)

**Week 4**:
- Heading management commands
- Link parsing and checking
- Dry-run mode

**Week 5**:
- Link update tracking
- File organization commands
- Parallel processing

**Week 6**:
- Type casting system
- Auto-detection logic
- Validation framework

### Phase 3: Advanced Features (Weeks 7-9)

**Week 7**:
- Link format conversion
- Wiki to Markdown converter
- Path resolution

**Week 8**:
- Linkding API client
- Sync command implementation
- Rate limiting

**Week 9**:
- Batch operations
- Transaction support
- Progress reporting

### Phase 4: Polish and Release (Weeks 10-12)

**Week 10**:
- Performance optimization
- Memory profiling
- Large vault testing

**Week 11**:
- Documentation completion
- Example configurations
- Video tutorials

**Week 12**:
- Release preparation
- Package distribution
- Community setup

---

## Success Metrics

### Adoption Metrics

- **Downloads**: 1,000+ in first month
- **GitHub Stars**: 100+ in three months
- **Active Users**: 500+ regular users
- **Community**: 50+ contributors

### Quality Metrics

- **Bug Reports**: < 5 critical bugs in first release
- **Performance**: Meet all benchmark targets
- **Reliability**: 99.9% success rate
- **User Satisfaction**: 4.5+ rating

### Feature Usage

- **Most Used**: Frontmatter ensure (80% of users)
- **Integration**: 30% use Linkding sync
- **Automation**: 40% use in scripts/cron

---

## Risks and Mitigation

### Technical Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Obsidian format changes | High | Low | Version detection, compatibility mode |
| Performance degradation | Medium | Medium | Profiling, optimization guidelines |
| Data corruption | Critical | Low | Extensive testing, atomic operations |
| API breaking changes | Medium | Medium | Version pinning, adapter pattern |

### Adoption Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Steep learning curve | High | Medium | Interactive mode, clear docs |
| Competition from GUI tools | Medium | High | Focus on automation use cases |
| Limited Go expertise | Medium | Low | Clear contribution guidelines |

### Mitigation Strategies

1. **Compatibility Testing**: Test against multiple Obsidian versions
2. **Community Engagement**: Regular feedback cycles
3. **Gradual Rollout**: Beta program before major releases
4. **Comprehensive Documentation**: Examples for all use cases
5. **Support Channels**: Discord, GitHub discussions

---

## Appendices

### A. Configuration File Schema

```yaml
# .obsidian-admin.yaml
version: "1.0"

vault:
  ignore_patterns:
    - "*.tmp"
    - ".obsidian/*"
    - "templates/*"
  
frontmatter:
  required_fields: ["id", "title", "tags"]
  preserve_order: true
  default_values:
    tags: []
    created: "{{current_date}}"
  type_rules:
    fields:
      created: date
      modified: date
      published: boolean
      priority: number
    patterns:
      - pattern: "*_date"
        type: date
      - pattern: "is_*"
        type: boolean

headings:
  ensure_h1_title: true
  single_h1: true
  min_level: 2
  fix_sequence: true

links:
  formats: ["wiki", "markdown"]
  update_embeds: true
  relative_paths: true
  add_extension: true

linkding:
  api_url: "${LINKDING_URL}"
  api_token: "${LINKDING_TOKEN}"
  field_mapping:
    url: "url"
    id: "linkding_id"
  sync_options:
    sync_title: true
    sync_tags: true
    tag_prefix: "obsidian/"
  rate_limit:
    requests_per_second: 2

output:
  format: "text"
  color: true
  verbose: false
```

### B. Example Commands

```bash
# Daily maintenance script
#!/bin/bash

# Ensure consistent frontmatter
obsidian-admin frontmatter ensure \
  --field "modified" --default "{{current_date}}" \
  --field "tags" --default "[]" \
  ./

# Fix any heading issues  
obsidian-admin headings fix \
  --ensure-h1-title \
  --single-h1 \
  ./

# Check for broken links
obsidian-admin links check ./

# Sync new URLs to Linkding
obsidian-admin linkding sync \
  --config .obsidian-admin.yaml \
  ./

# Generate weekly report
obsidian-admin analyze stats \
  --format markdown \
  > "reports/vault-stats-$(date +%Y-%m-%d).md"
```

### C. Error Code Reference

| Code | Description | Example |
|------|-------------|---------|
| 0 | Success | All operations completed |
| 1 | General error | Unknown failure |
| 2 | Invalid arguments | Missing required flag |
| 3 | File not found | Path does not exist |
| 4 | Parse error | Invalid markdown/frontmatter |
| 5 | Validation error | Failed validation rules |
| 6 | API error | Linkding connection failed |
| 7 | Permission error | Cannot write to file |
| 8 | User cancelled | Operation aborted |
