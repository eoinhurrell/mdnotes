# mdnotes CLI - Complete Audit and Enhancement Plan

## Executive Summary

The mdnotes CLI has **significantly exceeded** its original 6-cycle development plan, achieving 100% of planned features plus substantial additional functionality. This plan focuses on refining the tool for excellent local usage and redesigning the CLI interface for maximum usability while maintaining logical command groupings.

## Current Implementation Status

### ✅ Completed Features (100% of Original Plan + Extras)

**Cycle 1 - Foundation (Complete)**

- Core vault file parsing and frontmatter handling
- Directory scanning with ignore patterns
- CLI structure with Cobra framework
- Frontmatter ensure command

**Cycle 2 - Frontmatter Features (Complete)**

- Frontmatter validation with rules
- Type casting system with auto-detection
- Field synchronization with file system
- Template engine with variables and filters

**Cycle 3 - Content Operations (Complete)**

- Heading analysis and fixing
- Comprehensive link parsing (wiki, markdown, embed)
- Link format conversion (bidirectional)
- File organization with pattern-based renaming
- Link update tracking for file moves

**Cycle 4 - External Integration (Complete)**

- Complete Linkding API client with rate limiting
- Linkding sync processor
- Batch operations framework
- Progress reporting (terminal, JSON, silent modes)

**Cycle 5 - Analysis and Safety (Complete)**

- Vault statistics and health analysis
- Duplicate detection
- Backup and restore functionality
- Configuration system with environment variables
- Dry-run mode with detailed reporting

**Cycle 6 - Polish and Release (Complete)**

- Performance optimization and benchmarking
- User-friendly error messages
- Comprehensive shell completion
- Cross-platform build support

**Beyond Original Plan (Bonus Features)**

- Web resource downloader with automatic link conversion
- File rename command with link reference updates
- Performance profiling tools
- Enhanced configuration system
- Advanced error handling with suggestions

## CLI Interface Redesign Plan

### Current Problems with CLI Structure

1. **Redundant batch command**: Every command already works on folders, batch command is unnecessary
2. **Verbose command names**: Commands are too long for frequent daily use
3. **Missing short aliases**: No quick access to common operations
4. **Inconsistent flag usage**: Verbose and other flags behave differently across commands
5. **Limited query capabilities**: No way to search or filter based on frontmatter

### New Improved CLI Structure

#### Core Command Groups (with aliases and shortcuts)

```bash
# Frontmatter Management (alias: fm)
mdnotes frontmatter ensure [path]     # mdnotes fm ensure / mdnotes fm e
mdnotes frontmatter set [path]        # mdnotes fm set / mdnotes fm s
mdnotes frontmatter cast [path]       # mdnotes fm cast / mdnotes fm c
mdnotes frontmatter sync [path]       # mdnotes fm sync / mdnotes fm sy
mdnotes frontmatter check [path]      # mdnotes fm check / mdnotes fm ch
mdnotes frontmatter query [path]      # mdnotes fm query / mdnotes fm q
mdnotes frontmatter download [path]   # mdnotes fm download / mdnotes fm d

# Content Operations
mdnotes headings fix [path]           # mdnotes headings f
mdnotes links check [path]            # mdnotes links c
mdnotes links convert [path]          # mdnotes links co
mdnotes links graph [path]            # mdnotes links g

# File Operations
mdnotes rename <file> <new>           # mdnotes r
mdnotes organize [path]               # mdnotes o (future: smart file organization)

# External Integration
mdnotes linkding sync [path]          # mdnotes ld sync / mdnotes ld s
mdnotes linkding list [path]          # mdnotes ld list / mdnotes ld l

# Analysis & Utilities
mdnotes analyze health [path]         # mdnotes a health / mdnotes a h
mdnotes analyze links [path]          # mdnotes a links / mdnotes a l
mdnotes analyze content [path]        # mdnotes a content / mdnotes a c
mdnotes analyze trends [path]         # mdnotes a trends / mdnotes a t
mdnotes profile [command]             # mdnotes p
```

#### Quick Access Shortcuts (for power users)

```bash
# Ultra-short aliases for most common operations
mdnotes e [path]     # Shortcut for: frontmatter ensure
mdnotes s [path]     # Shortcut for: frontmatter set
mdnotes q [path]     # Shortcut for: frontmatter query
mdnotes f [path]     # Shortcut for: headings fix
mdnotes c [path]     # Shortcut for: links check
mdnotes r <file>     # Shortcut for: rename
```

### New Commands and Enhanced Functionality

#### Frontmatter Query Command (NEW)

```bash
# Find files based on frontmatter criteria
mdnotes frontmatter query [path] --where "tags contains 'project'"
mdnotes fm q [path] --where "priority > 3"
mdnotes fm q [path] --where "created after 2024-01-01"
mdnotes fm q [path] --missing "tags"
mdnotes fm q [path] --field "title,tags,priority" --format table

# Examples:
mdnotes fm q . --where "status = 'draft'" --count
mdnotes fm q . --missing "created" --fix-with "{{current_date}}"
mdnotes fm q . --duplicates "title"
```

#### Enhanced Organize Command (FUTURE)

```bash
# Smart file organization based on content/metadata
mdnotes organize [path] --by "date"           # Organize by creation date
mdnotes organize [path] --by "tags"           # Organize by tag structure
mdnotes organize [path] --by "template"       # Use custom organization template
mdnotes organize [path] --auto                # AI-powered organization suggestions
```

#### Content Generation Commands (FUTURE)

```bash
# Generate content from templates
mdnotes generate note --template "daily" --date "2024-01-15"
mdnotes generate index --for "projects"
mdnotes generate summary --from "meeting-notes"
```

### Global Flag Guidelines

#### Consistent Flag Behavior Across All Commands

**Required Flags (same behavior everywhere):**

```bash
--dry-run, -n
  # Preview changes without applying them
  # Shows exactly what would be changed
  # For query commands: shows what would be found
  # ALWAYS safe to use

--verbose, -v
  # Detailed output with progress information
  # Prints filepath of EVERY file examined
  # Shows what action was taken (or skipped) for each file
  # Includes timing information for operations
  # Example output:
  #   "Examining: /vault/note1.md - Added field 'tags' = []"
  #   "Examining: /vault/note2.md - Skipped (field exists)"
  #   "Examining: /vault/note3.md - Fixed heading structure"

--quiet, -q
  # Suppress all output except errors and final summary
  # Overrides --verbose if both specified
  # Only shows critical errors and final counts

--config, -c <path>
  # Specify configuration file path
  # Overrides default config search locations
```

**Optional Flags (command-specific but consistent naming):**

```bash
--recursive, -R
  # Process subdirectories (default: true for directories)
  # Can be disabled with --recursive=false

--ignore <patterns>
  # File/directory patterns to ignore
  # Default: [".obsidian/*", "*.tmp"]
  # Can specify multiple times

--format <type>
  # Output format for query/analysis commands
  # Options: table, json, csv, yaml
  # Default: table for terminal, json for scripts

--limit, -l <number>
  # Limit number of results (for query commands)
  # Default: unlimited

--sort <field>
  # Sort results by field (for query commands)
  # Options: name, date, size, etc.

--filter <expression>
  # Advanced filtering (for query commands)
  # Uses simple expression language
```

#### Flag Usage Examples

```bash
# Verbose mode shows every file examined
mdnotes fm ensure /vault --field tags --default "[]" --verbose
# Output:
#   Examining: /vault/project1.md - Added field 'tags' = []
#   Examining: /vault/project2.md - Skipped (field exists)
#   Examining: /vault/notes/daily.md - Added field 'tags' = []
#   Summary: 3 files examined, 2 modified

# Dry run shows what would happen
mdnotes headings fix /vault --dry-run --verbose
# Output:
#   Would examine: /vault/project1.md - Would fix H1 title mismatch
#   Would examine: /vault/project2.md - Would skip (headings OK)
#   Dry run summary: 2 files would be examined, 1 would be modified

# Quiet mode only shows summary
mdnotes fm check /vault --quiet
# Output:
#   Validation passed: 150 files validated

# Query with formatting
mdnotes fm query /vault --where "tags contains 'urgent'" --format table --verbose
# Output:
#   Examining: /vault/project1.md - Matches query
#   Examining: /vault/project2.md - No match (no 'urgent' tag)
#   ┌──────────────┬─────────┬──────────┐
#   │ File         │ Title   │ Tags     │
#   ├──────────────┼─────────┼──────────┤
#   │ project1.md  │ Project │ urgent   │
#   └──────────────┴─────────┴──────────┘
```

### Key Improvements

1. **Preserved Command Scopes**: Frontmatter, headings, links, etc. remain logically grouped
2. **Multiple Access Patterns**:
   - Full commands: `mdnotes frontmatter ensure`
   - Group aliases: `mdnotes fm ensure`
   - Subcommand aliases: `mdnotes fm e`
   - Ultra-short: `mdnotes e`
3. **Automatic Batch Processing**: All commands work on files/folders without separate batch command
4. **Consistent Flag Behavior**: Same flags work the same way across all commands
5. **Extensible Design**: Easy to add new commands in existing groups

### Command Consolidation Mapping

**Batch Command Elimination:**

```bash
# Before: Complex batch configuration
batch execute --config batch.yaml /vault

# After: Direct command usage (automatic batch on folders)
mdnotes fm ensure /vault --field tags --default "[]"
mdnotes headings fix /vault
mdnotes links check /vault
```

**Enhanced Access Patterns:**

```bash
# Verbose traditional approach
mdnotes frontmatter ensure /vault --field tags --default "[]"

# Moderate shortcut
mdnotes fm ensure /vault --field tags --default "[]"

# Power user shortcut
mdnotes e /vault --field tags --default "[]"
```

### Implementation Plan

#### Phase 1: CLI Restructure (Week 1)

**Goal**: Implement improved command structure while preserving logical groupings

##### Task 1.1: Remove Batch Command

- Remove `cmd/batch/` directory entirely
- Update root command to remove batch registration
- Document migration path for existing batch users

##### Task 1.2: Add Command Aliases

- Add group aliases: `fm` for frontmatter, `a` for analyze, etc.
- Add subcommand aliases: `e` for ensure, `s` for set, etc.
- Add ultra-short global shortcuts for most common commands
- Update shell completion for all alias variations

##### Task 1.3: Implement Frontmatter Query Command

```go
// New query command structure
mdnotes frontmatter query [path] [flags]
  --where <expression>     # Filter criteria
  --missing <field>        # Find files missing field
  --duplicates <field>     # Find duplicate values
  --field <list>           # Select specific fields to show
  --format <type>          # Output format
  --count                  # Just show count
  --fix-with <value>       # Auto-fix missing fields
```

##### Task 1.4: Standardize Flag Behavior

- Implement consistent --verbose behavior across all commands
- Ensure --dry-run works identically everywhere
- Add --format support to appropriate commands
- Update help text to show flag behavior clearly

#### Phase 2: Enhanced Features (Week 2)

**Goal**: Add powerful new functionality while maintaining simplicity

##### Task 2.1: Enhanced Query Language

```bash
# Simple comparisons
--where "priority > 3"
--where "status = 'draft'"
--where "tags contains 'urgent'"

# Date comparisons
--where "created after 2024-01-01"
--where "modified within 7 days"

# Complex expressions
--where "priority > 3 AND status != 'done'"
--where "tags contains 'project' OR tags contains 'work'"
```

##### Task 2.1b: Improved Enhanced Query Language

**Command‑Line Interface**

```bash
mdnotes frontmatter query [path] [flags]
mdnotes q [path] [flags]
```

Invokes the query engine on Markdown files with YAML frontmatter under `path` (default: current directory).

---

###### 1. Flags

| Flag                   | Description                                              | Example                                      |
| ---------------------- | -------------------------------------------------------- | -------------------------------------------- |
| `--where <expression>` | Filter files matching the boolean expression             | `--where "priority > 3 AND status = 'open'"` |
| `--missing <field>`    | List files where `<field>` is not present in frontmatter | `--missing due_date`                         |
| `--duplicates <field>` | Group files by `<field>` and list values occurring >1    | `--duplicates slug`                          |
| `--field <list>`       | Comma-separated list of frontmatter keys to display      | `--field title,created,tags`                 |
| `--format <type>`      | Output format: `table`, `json`, `yaml`, `csv`            | `--format json`                              |
| `--count`              | Only output the total number of matching files           | `--count`                                    |
| `--fix-with <value>`   | For `--missing`, auto-insert `<value>` for missing field | `--missing tags --fix-with 'misc'`           |

---

###### 2. Query Expressions

####### 2.1 Lexical Elements

- **Identifiers**: letters, digits, underscore; must start with letter or underscore.
- **String literals**: single-quoted (`'...'`) or double-quoted (`"..."`).
- **Numeric literals**: integer or floating point (e.g. `42`, `3.14`).
- **Date literals**: ISO-8601 date (`YYYY-MM-DD`).
- **Operators**: `=`, `!=`, `<`, `<=`, `>`, `>=`, `contains`, `not contains`, `in`, `not in`.
- **Logical**: `AND`, `OR`, `NOT` (case-insensitive).
- **Grouping**: parentheses `(` `)`.

####### 2.2 Grammar (EBNF)

```
<expr>      ::= <or_expr>
<or_expr>   ::= <and_expr> { ("OR" | "or") <and_expr> }
<and_expr>  ::= <not_expr> { ("AND" | "and") <not_expr> }
<not_expr>  ::= [ ("NOT"|"not") ] <cmp_expr>
<cmp_expr>  ::= <term> [ <cmp_op> <term> ]
<cmp_op>    ::= "=" | "!=" | ">" | ">=" | "<" | "<="
               | "contains" | "not contains" | "in" | "not in"
<term>      ::= <identifier> | <literal> | <function_call> | "(" <expr> ")"
<function_call> ::= <identifier> "(" [ <arg_list> ] ")"
<arg_list>  ::= <expr> {"," <expr> }
<identifier>::= letter { letter | digit | "_" }
<literal>   ::= <string> | <number> | <date> | <boolean>
```

####### 2.3 Supported Types & Coercion

- **String**: YAML string values.
- **Number**: integers or floats.
- **Boolean**: `true`, `false` (case-insensitive).
- **Date**: literal or field parsed as date (fields ending in `_date` or configured schema).
- **List**: sequences in frontmatter (e.g. `tags`). `contains` & `in` apply to lists.

Type coercion is applied when reasonable (e.g. numeric string to number).

####### 2.4 Built‑in Functions

- `now()`: current timestamp.
- `date("YYYY-MM-DD")`: parse literal as date.
- `len(x)`: length of string or list.
- `lower(x)`, `upper(x)`: case manipulation.

---

###### 3. Flag Semantics & Output

1. **`--where`**: Evaluates expression for each file’s frontmatter. Files where expression is true are included.
2. **`--missing`**: Files where `<field>` is absent or null.
3. **`--duplicates`**: Collect values for `<field>` across files; print groups with count >1.
4. **Combining**: Multiple flags may be combined; evaluation order:

   1. `--missing`, `--duplicates` pre-filter
   2. `--where` on remaining
   3. Projection with `--field`
   4. Aggregation with `--count`
   5. Formatting with `--format`
   6. Auto‑fix when `--fix-with` used

5. **`--fix-with`**: For each file missing `<field>`, insert default value, update file on disk. Idempotent.
6. **`--count`**: Suppress record output; show count only.
7. **Output Formats**:

   - `table` (default): aligned columns to stdout
   - `json`: JSON array of objects
   - `yaml`: YAML sequence
   - `csv`: comma‑separated

---

###### 4. Error Handling

- **Parse errors** in expression: report location, expected tokens.
- **Type errors** (e.g. comparing date to string): clear message.
- **File I/O errors**: skip with warning
- **Invalid flags**: exit with usage.

---

###### 5. Examples

```bash
#### 1. All drafts with priority >3
mdnotes frontmatter query . \
  --where "status = 'draft' AND priority > 3"

#### 2. Notes missing due_date, auto‑fix with tomorrow’s date
mdnotes frontmatter query notes/ \
  --missing due_date --fix-with "$(date +%F --date='tomorrow')"

#### 3. Duplicate slugs in all notes
mdnotes frontmatter query . --duplicates slug --format table

#### 4. Count notes tagged 'urgent' or 'important'
mdnotes frontmatter query . \
  --where "tags contains 'urgent' OR tags contains 'important'" --count

#### 5. Output title and created date in CSV
mdnotes frontmatter query . \
  --where "created >= '2024-01-01'" \
  --field title,created --format csv
```

---

##### Task 2.2: Performance and Smart Organization Features

- Output format 'table' should align columns to be more legible
- The rename command should have a configurable default target that it renames to if a second parameter isn't passed, i.e. 'Case Closed.md' should be renamed by default to '{{file created time | YYYYMMDDHHmmss}}-{{filename all lowercase, with spaces replaced by underscores and non-alphanumeric characters removed}}.md' (i.e. 20250619121117-case_closed.md). This will let it work with batch mode.
- Duplicate detection improvements (find Obsidian copies (with ' 1' at end of filename), or syncthing sync-conflict files)

##### Task 2.2b: Headings Clean Command

**Goal**: Add `mdnotes headings clean` command to sanitize headings for Obsidian compatibility

**Functionality**:

1. **Square Bracket Replacement**: Convert `# [X] Git` → `# <X> Git`

   - Handles any content within square brackets in headings
   - Works with date stamps: `## [2019-11-28 04:56]` → `## <2019-11-28 04:56>`
   - Preserves content exactly, only changes bracket style

2. **Link Heading Conversion**: Convert headings containing links to list items
   - Wiki links: `# [[Some Link]]` → `- [[Some Link]]`
   - Markdown links: `# [Text](url)` → `- [Text](url)`
   - Mixed content: `# Project [[Link]] Notes` → `- Project [[Link]] Notes`

**Command Structure**:

```bash
mdnotes headings clean [path]           # mdnotes headings cl
  --dry-run, -n                         # Preview changes
  --verbose, -v                         # Show each file processed
  --quiet, -q                          # Only show summary
  --square-brackets                    # Enable square bracket cleaning (default: true)
  --link-headers                       # Enable link header conversion (default: true)
```

**Reporting**:

- **Summary**: Total count of each transformation type across all files
- **Verbose Mode**: Show per-file counts of each transformation type
- **Example Output**:

  ```
  Examining: notes/project.md - Fixed 2 square brackets, converted 1 link header
  Examining: daily/2024-01-15.md - Fixed 1 square bracket
  Summary: 150 files examined, 12 modified
    - Square brackets fixed: 15
    - Link headers converted: 8
  ```

**Implementation Details**:

- Extend existing `HeadingProcessor` with new cleaning methods
- Add `CleanRules` struct with toggle options for each cleaning type
- Integrate with existing heading infrastructure (dry-run, progress reporting)
- Use regex patterns for robust detection and replacement
- Maintain line number tracking for accurate reporting

**Integration**:

- Add to existing `cmd/headings/` command group
- Follow established patterns from `headings fix` and `headings analyze`
- Support all standard global flags (`--dry-run`, `--verbose`, `--config`)
- Include in batch operations framework for large vault processing

##### Task 2.3: Enhanced Analysis Commands

**Goal**: Comprehensive vault analysis and content quality tools for better knowledge management

#### 2.3.1: Link Graph Analysis (`mdnotes analyze links`)

**Command**: `mdnotes analyze links [path]` (alias: `mdnotes a links`, `mdnotes a l`)

**Features**:
- **Local Links Only** (default): Only analyze `[[internal links]]` within the vault
- **Text-based Graph Visualization**: ASCII art representation of link relationships
- **Connection Statistics**: Most linked, least linked, orphaned files
- **Hub Detection**: Files with unusually high in/out link counts

**Flags**:
```bash
--include-external          # Include external URLs and markdown links
--show-graph               # Display ASCII graph visualization  
--min-connections <n>      # Only show files with n+ connections (default: 1)
--format <type>           # Output: table, json, graph (default: table)
--depth <n>               # Graph traversal depth (default: 2)
--orphans-only            # Show only files with no inbound links
--hubs-only               # Show only files with 5+ connections
```

**Output Examples**:
```bash
# Table format (default)
┌─────────────────┬─────────┬──────────┬─────────────┐
│ File            │ In-Links│ Out-Links│ Connections │
├─────────────────┼─────────┼──────────┼─────────────┤
│ index.md        │    8    │    12    │     20      │
│ projects.md     │    5    │     7    │     12      │
│ orphan.md       │    0    │     2    │      2      │
└─────────────────┴─────────┴──────────┴─────────────┘

# Graph format (--show-graph)
index.md ──┬── projects.md
           ├── daily-notes.md
           └── archive.md ──── old-project.md
                          └── orphan.md (orphan)
```

#### 2.3.2: Content Quality Scoring (`mdnotes analyze content`)

**Command**: `mdnotes analyze content [path]` (alias: `mdnotes a content`, `mdnotes a c`)

**Quality Metrics**:
1. **Word Count**: Total words in content body (excluding frontmatter)
2. **Line Count**: Total lines in file
3. **Heading Count**: Number of headings (H1-H6)
4. **Link Density**: Links per 100 words
5. **Frontmatter Completeness**: Percentage of expected fields present
6. **Content Structure Score**: Based on heading hierarchy, paragraph length
7. **Recency Score**: How recently the file was modified

**Scoring Algorithm**:
```
Quality Score = (Structure * 0.3) + (Completeness * 0.25) + (Density * 0.2) + (Recency * 0.15) + (Length * 0.1)
- Structure: 0-100 based on proper heading sequence, paragraph balance
- Completeness: 0-100 based on frontmatter field presence  
- Density: 0-100 based on optimal link-to-word ratio (2-5 links per 100 words)
- Recency: 0-100 based on modification date (100 = today, decreases over time)
- Length: 0-100 based on word count (sweet spot: 200-2000 words)
```

**Flags**:
```bash
--min-score <n>           # Only show files with score >= n
--max-score <n>           # Only show files with score <= n  
--sort-by <metric>        # Sort by: score, words, lines, headings, links
--show-metrics           # Show detailed breakdown of all metrics
--quality-threshold <n>   # Mark files below threshold as "needs attention"
```

**Output Example**:
```bash
┌─────────────────┬───────┬───────┬───────┬──────────┬─────────┬─────────────┐
│ File            │ Score │ Words │ Lines │ Headings │ Links   │ Quality     │
├─────────────────┼───────┼───────┼───────┼──────────┼─────────┼─────────────┤
│ project-a.md    │  87   │  1,205│   89  │    6     │   12    │ Excellent   │
│ notes.md        │  65   │   234 │   45  │    2     │    3    │ Good        │
│ draft.md        │  32   │   89  │   12  │    1     │    0    │ Needs Work  │
└─────────────────┴───────┴───────┴───────┴──────────┴─────────┴─────────────┘
```

#### 2.3.3: INBOX Triage Commands (`mdnotes analyze inbox`)

**Command**: `mdnotes analyze inbox [path]` (alias: `mdnotes a inbox`, `mdnotes a i`)

**Purpose**: Find files with content under "INBOX" headings that need processing/organization

**Features**:
1. **INBOX Content Detection**: Find all content under headings containing "INBOX" (case-insensitive)
2. **Content Size Analysis**: Measure lines/words under INBOX sections
3. **Triage Prioritization**: Sort by amount of content needing attention
4. **Quick Action Suggestions**: Common patterns and recommended actions

**Detection Patterns**:
- `# INBOX` or `## INBOX` or `### INBOX` (exact match)
- `# Inbox` or `## Inbox` (case variations)  
- `# 📥 INBOX` or `## 📥 Inbox` (with emoji)
- `# INBOX - Unsorted` (with descriptive text)

**Flags**:
```bash
--min-lines <n>           # Only show INBOX sections with n+ lines (default: 1)
--max-lines <n>           # Only show INBOX sections with <= n lines
--sort-by <metric>        # Sort by: lines, words, headings, age (default: lines desc)
--show-content           # Preview first few lines of INBOX content
--suggest-actions        # Show recommended actions for each file
--format <type>          # Output: table, json, summary (default: table)
```

**Output Example**:
```bash
# Default table output
┌─────────────────┬─────────┬───────┬──────────┬─────────────────────┐
│ File            │ Lines   │ Words │ Headings │ Last Modified       │
├─────────────────┼─────────┼───────┼──────────┼─────────────────────┤
│ daily-2024.md   │   47    │  312  │    3     │ 2024-12-18 09:30   │
│ meeting.md      │   23    │  156  │    1     │ 2024-12-17 14:22   │
│ ideas.md        │   12    │   89  │    2     │ 2024-12-16 11:45   │
└─────────────────┴─────────┴───────┴──────────┴─────────────────────┘

# With content preview (--show-content)  
daily-2024.md (47 lines under INBOX):
  ## INBOX
  - [ ] Follow up on project proposal
  - Review quarterly metrics
  - Schedule team meeting for next week
  ... (44 more lines)
  
  Suggested actions: Create separate task notes, move to projects folder

# Summary format (--format summary)
INBOX Triage Summary:
- 15 files contain INBOX sections
- Total unprocessed lines: 234
- Total unprocessed words: 1,567
- Oldest unprocessed content: 45 days (in archive/old-notes.md)
- Largest INBOX: daily-2024.md (47 lines)
```

#### 2.3.4: Vault Growth Trends (`mdnotes analyze trends`)

**Command**: `mdnotes analyze trends [path]` (alias: `mdnotes a trends`, `mdnotes a t`)

**Features**:
- **File Creation Timeline**: Files created per day/week/month
- **Content Growth**: Word count growth over time
- **Link Network Growth**: How connections between files evolve
- **Tag Usage Trends**: Most/least used tags over time
- **Activity Patterns**: When you're most productive

**Time Periods**:
```bash
--period <type>           # day, week, month, year (default: month)
--since <date>           # Only analyze from this date forward
--limit <n>              # Show last n periods (default: 12)
```

#### 2.3.5: Health Monitoring Dashboard (`mdnotes analyze health`)

**Command**: `mdnotes analyze health [path]` (alias: `mdnotes a health`, `mdnotes a h`)

**Health Checks**:
1. **Broken Links**: Internal links pointing to non-existent files
2. **Orphaned Files**: Files with no inbound links
3. **Empty Files**: Files with no substantial content
4. **Duplicate Content**: Files with similar/identical content
5. **Inconsistent Tags**: Tag variations that should be unified
6. **Missing Frontmatter**: Files lacking essential metadata
7. **Stale Content**: Files not modified in 90+ days

**Output**:
```bash
Vault Health Report:
✅ Total Files: 1,247
⚠️  Broken Links: 23 files
❌ Orphaned Files: 156 files  
⚠️  Empty Files: 12 files
✅ Average Quality Score: 73/100
❌ Files Needing Attention: 89 (7.1%)

Priority Actions:
1. Fix broken links in: project-archive.md, old-notes.md
2. Review orphaned files in: drafts/, archive/
3. Add content to empty files: placeholder.md, template.md
```

**Integration Points**:
- All analyze commands support standard global flags (`--dry-run`, `--verbose`, `--quiet`)
- Results can be exported in multiple formats (table, JSON, CSV)  
- INBOX triage integrates with task management workflows
- Health monitoring can trigger automated maintenance suggestions
- Content quality scoring helps identify notes ready for publication or needing improvement

#### Phase 3: Polish and Future-Proofing (Week 3)

**Goal**: Ensure excellent user experience and extensibility

- Simple automatic tagging suggestions

##### Task 3.1: Comprehensive Testing

- Test all alias combinations work correctly
- Verify flag consistency across commands
- Performance testing with large vaults
- User experience testing with real workflows

##### Task 3.2: Documentation and Examples

- Create comprehensive command reference
- Add practical workflow examples
- Document flag usage patterns

##### Task 3.3: Future Command Framework

- Design extensible command structure
- Plan for AI-powered features
- Consider plugin architecture
- Document command development guidelines

### Success Metrics

#### Usability Improvements

- **Command Efficiency**: Common tasks reduced to 2-3 characters (`mdnotes e`)
- **Learning Curve**: Multiple access patterns accommodate different user types
- **Discoverability**: Clear help system with examples and aliases
- **Power User Features**: Query and filter capabilities for advanced workflows

#### Technical Excellence

- **Consistency**: All flags behave identically across commands
- **Performance**: No regression in processing speed
- **Extensibility**: Easy to add new commands and features
- **Reliability**: Comprehensive error handling and recovery

#### User Experience Goals

- **Flexibility**: Works for both beginners and power users
- **Efficiency**: Multiple ways to access functionality
- **Clarity**: Verbose mode provides complete transparency
- **Safety**: Dry-run mode works everywhere

This redesigned CLI maintains logical command organization while dramatically improving usability through multiple access patterns, consistent flag behavior, and powerful new query capabilities.

## Future Work: Advanced Zettelkasten Features

The following features represent natural extensions of mdnotes' core mission to maintain graph consistency and support effective knowledge management. These ideas focus on zettelkasten-specific workflows while maintaining the tool's emphasis on automation, analysis, and file consistency.

### 1. Atomic Note Analysis (`mdnotes analyze atomic`)

**Purpose**: Ensure notes follow zettelkasten principles of atomicity - one concept per note.

**Features**:
- **Complexity Scoring**: Analyze notes for multiple distinct concepts that should be split
- **Topic Clustering**: Identify paragraphs/sections discussing different themes
- **Split Suggestions**: Recommend where to break large notes into atomic units
- **Concept Density**: Measure how focused each note is on a single idea

**Detection Criteria**:
- Multiple H2 headings suggesting different topics
- Word count above threshold (configurable, default: 800 words)
- Paragraph topic shifts detected through keyword analysis
- Multiple unrelated tag combinations

**Command**: `mdnotes analyze atomic [path] --suggest-splits --min-complexity 7`

### 2. Note Sequencing Management (`mdnotes sequence`)

**Purpose**: Manage Luhmann-style note sequences (1, 1a, 1b, 1a1, etc.) common in zettelkasten systems.

**Features**:
- **Sequence Validation**: Check for gaps or inconsistencies in numbering
- **Auto-numbering**: Generate next sequence number for branching thoughts
- **Sequence Visualization**: Show tree structure of note sequences
- **Branch Optimization**: Suggest sequence reorganization for better flow

**Commands**:
```bash
mdnotes sequence validate [path]        # Check sequence consistency
mdnotes sequence next 1a               # Get next number in sequence (1b)
mdnotes sequence tree [path]           # Visualize sequence hierarchy
mdnotes sequence rebalance [path]      # Optimize sequence structure
```

### 3. Literature Note Processing (`mdnotes literature`)

**Purpose**: Convert research highlights and citations into connected zettelkasten notes.

**Features**:
- **Highlight Extraction**: Parse highlights from imported sources (Kindle, Zotero, etc.)
- **Citation Linking**: Automatically create links between literature and permanent notes
- **Source Consolidation**: Group related excerpts from same source
- **Reference Formatting**: Ensure consistent citation formats across vault

**Workflow**:
1. Import highlights/excerpts with source metadata
2. Generate individual atomic notes from each highlight
3. Create connections to existing relevant notes
4. Maintain bibliography consistency

**Command**: `mdnotes literature process highlights.md --source "Author (2024)" --link-existing`

### 4. Concept Map Generation (`mdnotes concepts`)

**Purpose**: Discover and visualize recurring themes across the knowledge graph.

**Features**:
- **Theme Extraction**: Identify frequently co-occurring concepts
- **Concept Clustering**: Group related ideas across different notes
- **Missing Link Detection**: Suggest connections between conceptually related notes
- **Semantic Analysis**: Use keyword/tag patterns to find conceptual relationships

**Output**:
- Visual concept maps showing idea relationships
- Lists of notes that should be connected but aren't
- Concept evolution over time

**Command**: `mdnotes concepts map [path] --theme productivity --suggest-links`

### 5. Note Maturity Tracking (`mdnotes maturity`)

**Purpose**: Track the development and refinement of notes over time - key for zettelkasten evolution.

**Features**:
- **Development Stages**: Classify notes as fleeting, literature, permanent, evergreen
- **Refinement Metrics**: Track how notes improve through editing cycles
- **Review Scheduling**: Suggest notes due for review/development
- **Maturity Scoring**: Algorithm combining content quality, link density, and revision history

**Maturity Levels**:
1. **Fleeting**: Quick captures, minimal processing
2. **Literature**: Processed from sources but not fully integrated
3. **Permanent**: Well-developed, properly linked
4. **Evergreen**: Highly refined, frequently referenced

**Command**: `mdnotes maturity assess [path] --schedule-reviews --upgrade-candidates`

### 6. Cross-Reference Intelligence (`mdnotes xref`)

**Purpose**: Smart suggestion system for creating meaningful connections between notes.

**Features**:
- **Content Similarity**: Find notes with overlapping themes that should link
- **Citation Patterns**: Suggest links based on shared references
- **Tag Relationships**: Identify notes with complementary tag patterns
- **Context-Aware Linking**: Understand WHY notes should connect, not just that they could

**Smart Suggestions**:
- "Note A discusses productivity systems, Note B discusses time management - consider linking"
- "Both notes cite the same research paper but aren't connected"
- "Notes in sequence 1a-1c reference this concept but don't link to the main note"

**Command**: `mdnotes xref suggest [path] --confidence-threshold 0.7 --apply-suggestions`

### 7. Knowledge Gap Analysis (`mdnotes gaps`)

**Purpose**: Identify missing pieces in the knowledge graph - concepts mentioned but not developed.

**Features**:
- **Orphan Concept Detection**: Find frequently mentioned topics without dedicated notes
- **Link Stub Analysis**: Identify broken links that represent knowledge gaps
- **Research Direction Suggestions**: Highlight areas needing more development
- **Concept Coverage Mapping**: Show which domains are well/poorly covered

**Gap Types**:
- **Missing Definitions**: Terms used but never defined
- **Underdeveloped Concepts**: Single-mention topics that deserve expansion
- **Connection Voids**: Areas with few interconnections
- **Research Gaps**: Topics mentioned but lacking source material

**Command**: `mdnotes gaps identify [path] --suggest-research --min-mentions 3`

### 8. Note Consolidation Engine (`mdnotes consolidate`)

**Purpose**: Find and merge duplicate or highly overlapping notes while preserving unique insights.

**Features**:
- **Content Similarity Detection**: Identify notes with substantial overlap
- **Merge Conflict Resolution**: Smart merging of similar content
- **Link Preservation**: Maintain all inbound/outbound links during consolidation
- **Version History**: Track what was merged and allow rollback

**Consolidation Types**:
- **Exact Duplicates**: Identical content (easy merge)
- **Partial Overlap**: Some shared content, some unique (selective merge)
- **Theme Variants**: Same topic, different angles (careful consolidation)

**Command**: `mdnotes consolidate detect [path] --similarity-threshold 0.8 --preview-merges`

### 9. Template Intelligence (`mdnotes templates`)

**Purpose**: Advanced template system optimized for different zettelkasten note types.

**Features**:
- **Context-Aware Templates**: Different templates for literature, permanent, fleeting notes
- **Dynamic Field Population**: Auto-fill templates based on note context and connections
- **Template Evolution**: Learn from user patterns to improve templates
- **Consistency Enforcement**: Ensure all notes of same type follow template structure

**Template Types**:
- **Literature Note**: Citation, key points, personal insights, connections
- **Permanent Note**: Core concept, evidence, implications, related ideas
- **Meeting Note**: Attendees, decisions, action items, follow-ups
- **Daily Note**: Structured format for daily zettelkasten practice

**Command**: `mdnotes templates generate --type literature --source "Author (2024)" --connect-to productivity`

### 10. Spaced Review System (`mdnotes review`)

**Purpose**: Implement spaced repetition for note review - essential for zettelkasten maintenance.

**Features**:
- **Review Scheduling**: Algorithm-based review intervals (1 day, 3 days, 1 week, etc.)
- **Priority Weighting**: Review important/connected notes more frequently
- **Review Types**: Different review modes (quick scan, deep review, connection audit)
- **Progress Tracking**: Monitor which notes are being maintained vs. neglected

**Review Categories**:
- **New Notes**: Require quick review to establish connections
- **Developing Notes**: Need regular refinement
- **Mature Notes**: Periodic maintenance checks
- **Stale Notes**: Haven't been accessed recently, may need archiving

**Command**: `mdnotes review schedule [path] --due-today --type deep-review`

### Implementation Priority

These features should be implemented in order of impact on core zettelkasten workflows:

**Phase A (High Impact)**: Atomic Note Analysis, Cross-Reference Intelligence, Knowledge Gap Analysis
**Phase B (Medium Impact)**: Note Maturity Tracking, Spaced Review System, Concept Map Generation  
**Phase C (Nice-to-Have)**: Note Sequencing, Literature Processing, Template Intelligence, Consolidation Engine

Each feature maintains mdnotes' philosophy of automation, consistency, and graph integrity while adding powerful capabilities specific to zettelkasten methodology and knowledge work.
