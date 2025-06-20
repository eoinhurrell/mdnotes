# mdnotes CLI - Complete Development Plan

## Executive Summary

The mdnotes CLI has **significantly exceeded** its original 6-cycle development plan, achieving 100% of planned features plus substantial additional functionality. This plan provides a clear roadmap for future development while documenting current status and outstanding tasks.

## Current Implementation Status

### ✅ **Fully Completed Features**

**Core Foundation**

- ✅ Vault file parsing and frontmatter handling
- ✅ Directory scanning with ignore patterns
- ✅ CLI structure with Cobra framework
- ✅ Configuration system with environment variables

**Frontmatter Management**

- ✅ Frontmatter ensure command with templates
- ✅ Validation with rules and type constraints
- ✅ Type casting system with auto-detection
- ✅ Field synchronization with file system
- ✅ Template engine with variables and filters

**Content Operations**

- ✅ Heading analysis and fixing
- ✅ **Heading cleaning for Obsidian compatibility** (Task 2.2b - COMPLETED)
- ✅ Comprehensive link parsing (wiki, markdown, embed)
- ✅ Link format conversion (bidirectional)
- ✅ File organization with pattern-based renaming
- ✅ Link update tracking for file moves

**External Integration**

- ✅ Complete Linkding API client with rate limiting
- ✅ Linkding sync processor
- ✅ Web resource downloader with automatic link conversion

**Analysis and Safety**

- ✅ Vault statistics and health analysis
- ✅ Content quality scoring with actionable suggestions
- ✅ **INBOX triage analysis for pending content** (NEW!)
- ✅ Link graph analysis with centrality scoring
- ✅ Vault trends and growth pattern analysis
- ✅ Comprehensive duplicate detection (content, Obsidian copies, sync conflicts)
- ✅ Backup and restore functionality
- ✅ Dry-run mode with detailed reporting
- ✅ Batch operations framework
- ✅ Progress reporting (terminal, JSON, silent modes)

**Polish and Quality**

- ✅ Performance optimization and benchmarking
- ✅ User-friendly error messages
- ✅ Comprehensive shell completion
- ✅ Cross-platform build support
- ✅ File rename command with link reference updates
- ✅ Performance profiling tools
- ✅ Advanced error handling with suggestions

### 🔄 **In Progress / Planned Tasks**

**Phase 1: CLI Restructure** ✅ **COMPLETED**

- ✅ **Task 1.1**: Remove Batch Command
- ✅ **Task 1.2**: Add Command Aliases (fm, a, etc.)
- ✅ **Task 1.3**: Implement Frontmatter Query Command
- ✅ **Task 1.4**: Standardize Flag Behavior

**Phase 2: Enhanced Features** ✅ **COMPLETED**

- ✅ **Task 2.1**: Enhanced Query Language with Complex Expressions
- ✅ **Task 2.2**: Performance and Smart Organization Features  
- ✅ **Task 2.3**: Enhanced Analysis Commands (Link Graph, Content Quality, INBOX Triage, Trends, Health)

**Phase 3: Polish and Future-Proofing**

- ⏳ **Task 3.1**: Comprehensive Testing
- ⏳ **Task 3.2**: Documentation and Examples
- ⏳ **Task 3.3**: Future Command Framework

## Current Command Structure

### Available Commands

```bash
# Frontmatter Management
mdnotes frontmatter ensure [path]     # Add/ensure frontmatter fields
mdnotes frontmatter validate [path]   # Validate frontmatter rules
mdnotes frontmatter cast [path]       # Type cast frontmatter fields
mdnotes frontmatter sync [path]       # Sync with file system metadata

# Content Operations
mdnotes headings analyze [path]       # Analyze heading structure
mdnotes headings fix [path]           # Fix heading issues
mdnotes headings clean [path]         # Clean headings for Obsidian (NEW!)
mdnotes links check [path]            # Check for broken links
mdnotes links convert [path]          # Convert link formats

# File Operations
mdnotes rename <file> [new]           # Rename with link updates
mdnotes organize [path]               # Pattern-based organization

# External Integration
mdnotes linkding sync [path]          # Sync URLs to Linkding
mdnotes linkding list [path]          # List vault URLs

# Analysis & Utilities
mdnotes analyze health [path]         # Vault health report
mdnotes analyze content [path]        # Content quality scoring
mdnotes analyze inbox [path]          # INBOX triage analysis
mdnotes analyze links [path]          # Link graph analysis
mdnotes analyze trends [path]         # Vault growth trends
mdnotes analyze duplicates [path]     # Find duplicate files
```

### Global Flags (Consistent Across All Commands)

```bash
--dry-run, -n     # Preview changes without applying
--verbose, -v     # Detailed output with file-by-file progress
--quiet, -q       # Only show errors and final summary
--config, -c      # Specify configuration file path
```

## Outstanding Development Tasks

### Phase 1: CLI Restructure & Enhancement

#### Task 1.1: Remove Batch Command ✅

**Status**: Completed  
**Goal**: Eliminate redundant batch command since all commands work on directories

- ✅ Verified `cmd/batch/` directory doesn't exist (batch command already removed)
- ✅ Root command has no batch registration
- ✅ Internal batch processing infrastructure preserved for operation coordination

#### Task 1.2: Add Command Aliases ✅

**Status**: Completed  
**Goal**: Add convenient shortcuts for frequent operations

```bash
# Group aliases (implemented)
mdnotes fm ensure      # frontmatter ensure
mdnotes a health       # analyze health
mdnotes ld sync        # linkding sync

# Ultra-short global shortcuts (implemented)
mdnotes e [path]       # frontmatter ensure
mdnotes f [path]       # headings fix
mdnotes c [path]       # links check
mdnotes s [path]       # frontmatter set
mdnotes q [path]       # frontmatter query
```

#### Task 1.3: Implement Frontmatter Query Command ✅

**Status**: Completed  
**Goal**: Add powerful search and filter capabilities

```bash
mdnotes frontmatter query [path] --where "tags contains 'project'"
mdnotes fm q [path] --missing "created" --fix-with "{{current_date}}"
mdnotes fm q [path] --duplicates "title" --format table
```

**Features** (all implemented):

- ✅ Complex query expressions with AND/OR/NOT
- ✅ Field presence/absence checking
- ✅ Duplicate detection
- ✅ Auto-fix capabilities
- ✅ Multiple output formats (table, JSON, CSV, YAML)
- ✅ Enhanced query language with date comparisons and contains operator

#### Task 1.4: Standardize Flag Behavior ✅

**Status**: Completed  
**Goal**: Ensure consistent flag behavior across all commands

- ✅ Implement consistent `--verbose` behavior everywhere
- ✅ Ensure `--dry-run` works identically across commands
- ✅ Add `--format` support to analysis commands
- ✅ Update help text for clarity

### Phase 2: Enhanced Features

#### Task 2.1: Enhanced Query Language with Complex Expressions ✅

**Status**: Completed  
**Goal**: Advanced frontmatter query capabilities with complex logic

**Features** (all implemented):
- ✅ Complex query expressions with AND/OR/NOT operators
- ✅ Date comparisons with natural language (after, before, within)
- ✅ Contains operator for arrays and strings
- ✅ Numeric comparisons (>, >=, <, <=, =, !=)
- ✅ Field presence/absence checking
- ✅ Duplicate detection and auto-fix capabilities
- ✅ Multiple output formats (table, JSON, CSV, YAML)

#### Task 2.2: Performance and Smart Organization Features ✅

**Status**: Completed

- ✅ **Table Output**: Well-formatted columns with proper alignment
- ✅ **Rename Enhancement**: Configurable default patterns with `--template` flag
- ✅ **Duplicate Detection**: Comprehensive detection of Obsidian copies, sync conflicts, and content duplicates

#### Task 2.3: Enhanced Analysis Commands ✅

**Status**: Completed  
**Goal**: Comprehensive vault analysis tools

**Link Graph Analysis** (`mdnotes analyze links`) ✅:

- ✅ ASCII graph visualization of link relationships
- ✅ Hub detection and orphan analysis
- ✅ Connection statistics and centrality scoring

**Content Quality Scoring** (`mdnotes analyze content`) ✅:

- ✅ Multi-factor quality scoring (structure, completeness, complexity, density, recency)
- ✅ Actionable improvement suggestions for quality improvement
- ✅ Quality thresholds and score distribution analysis
- ✅ Individual file scores with customizable minimum thresholds

**INBOX Triage** (`mdnotes analyze inbox`) ✅ **NEW**:

- ✅ Find content under INBOX headings needing processing
- ✅ Sort by content volume, item count, or urgency for prioritization
- ✅ Action suggestions for common patterns (linkding sync, note conversion, etc.)
- ✅ Urgency assessment based on keywords and content analysis
- ✅ Comprehensive pattern matching for TODO, PENDING, INBOX, DRAFTS, etc.

**Vault Trends** (`mdnotes analyze trends`) ✅:

- ✅ File creation timeline analysis
- ✅ Content growth tracking with growth rate calculation
- ✅ Tag usage evolution and trend analysis
- ✅ Writing streak and activity percentage tracking

**Health Dashboard** (`mdnotes analyze health`) ✅:

- ✅ Comprehensive vault health checks
- ✅ Broken links, orphans, empty files detection
- ✅ Prioritized action recommendations with health scoring

### Phase 3: Polish and Future-Proofing

#### Task 3.1: Comprehensive Testing ⏳

- Test all alias combinations
- Performance testing with large vaults (10k+ files)
- User experience testing with real workflows

#### Task 3.2: Documentation and Examples ⏳

- Complete command reference with examples
- Workflow guides for common use cases
- Video tutorials for complex features

#### Task 3.3: Future Command Framework ⏳

- Design extensible command architecture
- Plugin system design
- Command development guidelines

## Future Work: Advanced Features

### Near-Term Enhancements

#### 1. **Watch Command** (`mdnotes watch`) - NEW

**Purpose**: Automated file monitoring with configurable actions per folder

**Core Functionality**:

- Watch vault directories for new file additions
- Execute specific commands based on folder-specific rules
- Configuration-driven automation for common workflows

**Command Structure**:

```bash
mdnotes watch [path]                    # Start watching with config
mdnotes watch --config watch.yaml      # Custom config file
mdnotes watch --daemon                 # Run as background service
```

**Configuration Example** (`.obsidian-admin.yaml`):

```yaml
watch:
  enabled: true
  rules:
    - path: "inbox/"
      recursive: true
      actions:
        - command: "linkding sync"
          args: ["--dry-run=false"]
        - command: "frontmatter ensure"
          args: ["--field", "created", "--default", "{{current_date}}"]

    - path: "resources/books/"
      recursive: false
      actions:
        - command: "frontmatter ensure"
          args: ["--field", "cover", "--default", "{{download_cover}}"]
        - command: "frontmatter ensure"
          args: ["--field", "type", "--default", "book"]

    - path: "projects/"
      recursive: true
      actions:
        - command: "headings fix"
        - command: "frontmatter ensure"
          args: ["--field", "status", "--default", "active"]

  # Global settings
  debounce: 2s # Wait 2 seconds after file changes
  ignore_patterns:
    - ".obsidian/*"
    - "*.tmp"
    - ".DS_Store"
  max_file_size: "10MB" # Skip files larger than 10MB
  log_level: "info" # info, debug, warn, error
```

**Advanced Features**:

- **Debouncing**: Wait for file operations to complete before acting
- **Conditional Execution**: Only run commands if certain conditions are met
- **Template Variables**: Support for dynamic values in commands
- **Error Handling**: Graceful failure with retry logic
- **Performance**: Efficient file system watching with minimal resource usage

**Example Use Cases**:

1. **Inbox Processing**: New files in inbox get frontmatter populated and synced to Linkding
2. **Book Management**: New book files get cover images downloaded automatically
3. **Project Organization**: Project files get standardized frontmatter and heading structure
4. **Daily Notes**: New daily notes get templates applied automatically

**Integration**:

- Uses existing command infrastructure
- Leverages current configuration system
- Supports all existing global flags for sub-commands
- Works with dry-run mode for testing configurations

### Long-Term Zettelkasten Features

#### 2. **Atomic Note Analysis** (`mdnotes analyze atomic`)

Ensure notes follow zettelkasten principles of one concept per note.

#### 3. **Cross-Reference Intelligence** (`mdnotes xref`)

Smart suggestion system for creating meaningful connections between notes.

#### 4. **Knowledge Gap Analysis** (`mdnotes gaps`)

Identify missing pieces in the knowledge graph.

#### 5. **Note Maturity Tracking** (`mdnotes maturity`)

Track development stages from fleeting to evergreen notes.

#### 6. **Spaced Review System** (`mdnotes review`)

Algorithm-based review scheduling for note maintenance.

_[Additional features 7-10 as previously detailed...]_

## Implementation Priority

**Immediate Focus** (Next 2-4 weeks):

1. ✅ Task 2.2b: Headings Clean Command (COMPLETED)
2. Task 1.2: Command Aliases
3. Task 1.3: Frontmatter Query Command

**Short Term** (1-3 months): 4. Watch Command implementation 5. Task 2.3: Enhanced Analysis Commands 6. Task 1.4: Standardize Flag Behavior

**Medium Term** (3-6 months): 7. Advanced Zettelkasten features (atomic analysis, cross-reference intelligence) 8. Performance optimizations for large vaults 9. Plugin architecture

**Long Term** (6+ months): 10. AI-powered features 11. Advanced template intelligence 12. Real-time collaboration features

## Success Metrics

**Usability**:

- ✅ Command efficiency (2-3 character shortcuts)
- ✅ Multiple access patterns for different user types
- ⏳ Clear help system with examples
- ⏳ Advanced query capabilities

**Technical Excellence**:

- ✅ Consistent flag behavior across commands
- ✅ No performance regression with new features
- ✅ Comprehensive error handling
- ⏳ Extensible command architecture

**User Experience**:

- ✅ Works for both beginners and power users
- ✅ Multiple ways to access functionality
- ✅ Transparent verbose mode
- ✅ Safe dry-run mode everywhere

This plan maintains mdnotes' philosophy of automation, consistency, and graph integrity while providing a clear roadmap for continued development and enhancement.

