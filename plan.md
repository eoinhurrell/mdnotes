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

##### Task 2.2: Smart Organization Features
- Pattern-based file organization
- Template-driven folder structures
- Automatic tagging suggestions
- Duplicate detection and resolution

##### Task 2.3: Enhanced Analysis
- Link graph visualization (text-based)
- Content quality scoring
- Vault growth trends
- Health monitoring dashboard

#### Phase 3: Polish and Future-Proofing (Week 3)

**Goal**: Ensure excellent user experience and extensibility

##### Task 3.1: Comprehensive Testing
- Test all alias combinations work correctly
- Verify flag consistency across commands
- Performance testing with large vaults
- User experience testing with real workflows

##### Task 3.2: Documentation and Examples
- Create comprehensive command reference
- Add practical workflow examples
- Document flag usage patterns
- Create migration guide from old CLI

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