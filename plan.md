# Implementation Plan: Link Updating System for mdnotes Rename

## Overview

This implementation plan focuses specifically on building the link updating functionality to integrate with the existing mdnotes rename command. The goal is to add comprehensive link detection and updating capabilities while leveraging the existing file renaming infrastructure.

---

## Architecture Overview

### Core Components

1. **Link Scanner**: Discovers all links in vault files
2. **Link Parser**: Analyzes and categorizes found links
3. **Link Updater**: Modifies links to point to new target
4. **File Processor**: Handles file I/O operations safely
5. **Progress Reporter**: Provides user feedback during operations

### Data Flow

```
Rename Request → Link Scanner → Link Parser → Link Updater → File Processor → Report
```

---

## Phase 1: Foundation (Week 1-2)

### 1.1 Link Detection Engine

**Goal**: Build robust link detection across all Obsidian link formats

#### Task 1.1.1: Regex Pattern Library

```javascript
// Core patterns to implement
const LINK_PATTERNS = {
  wikilink: /\[\[([^\]]+)\]\]/g,
  wikilink_with_alias: /\[\[([^|\]]+)\|([^\]]+)\]\]/g,
  wikilink_embed: /!\[\[([^\]]+)\]\]/g,
  markdown_link: /\[([^\]]+)\]\(([^)]+)\)/g,
  markdown_image: /!\[([^\]]*)\]\(([^)]+)\)/g,
};
```

**Deliverables**:

- [ ] Comprehensive regex patterns for all link types
- [ ] Pattern testing suite with 100+ test cases
- [ ] Link type classification system
- [ ] Fragment and block reference parsing

#### Task 1.1.2: File Content Scanner

```javascript
class LinkScanner {
  scanFile(filePath) {
    // Returns array of LinkMatch objects
  }

  scanVault(vaultPath, options = {}) {
    // Returns Map<filePath, LinkMatch[]>
  }
}
```

**Deliverables**:

- [ ] File system traversal with filtering
- [ ] Content reading with encoding detection
- [ ] Link extraction from file content
- [ ] Performance optimization for large files

### 1.2 Link Data Structure

#### Task 1.2.1: Link Model Definition

```javascript
class LinkMatch {
  constructor() {
    this.type = ""; // 'wikilink', 'markdown', 'embed'
    this.target = ""; // Target file/path
    this.fragment = ""; // #heading or #^blockid
    this.alias = ""; // Display text for wikilinks
    this.startPos = 0; // Character position in file
    this.endPos = 0; // End position
    this.rawText = ""; // Original link text
    this.encoding = ""; // URL encoding style if applicable
  }

  shouldUpdate(oldPath, newPath) {
    // Logic to determine if this link should be updated
  }

  generateUpdatedLink(newPath) {
    // Generate the new link text
  }
}
```

**Deliverables**:

- [ ] Complete LinkMatch class with all properties
- [ ] Path matching logic (exact vs fuzzy)
- [ ] Link reconstruction methods
- [ ] Validation and error checking

---

## Phase 2: Core Link Processing (Week 3-4)

### 2.1 Path Resolution System

#### Task 2.1.1: Vault-Relative Path Handler

```javascript
class PathResolver {
  constructor(vaultRoot) {
    this.vaultRoot = vaultRoot;
  }

  resolveTarget(linkTarget, contextFile) {
    // Convert link target to absolute vault path
  }

  shouldUpdateLink(linkTarget, oldPath, newPath) {
    // Determine if link points to renamed file
  }

  generateNewTarget(linkTarget, oldPath, newPath) {
    // Create updated link target
  }
}
```

**Implementation Details**:

- Handle filename-only links (`[[note]]`)
- Handle full path links (`[[folder/note]]`)
- Handle extension variants (`.md` implicit vs explicit)
- Case sensitivity handling per OS
- URL encoding/decoding

**Deliverables**:

- [ ] Path resolution algorithm
- [ ] Link target matching logic
- [ ] New path generation
- [ ] Cross-platform compatibility

### 2.2 Link Update Engine

#### Task 2.2.1: Content Modification System

```javascript
class LinkUpdater {
  updateLinksInFile(filePath, linkMatches, oldPath, newPath) {
    // Update all matching links in a single file
  }

  updateLinksInVault(vaultPath, oldPath, newPath, options = {}) {
    // Orchestrate updates across entire vault
  }

  previewUpdates(vaultPath, oldPath, newPath) {
    // Dry-run mode - show what would change
  }
}
```

**Implementation Strategy**:

- Process files in reverse character order to maintain positions
- Batch updates per file for efficiency
- Preserve original formatting and whitespace
- Handle encoding transformations

**Deliverables**:

- [ ] File content modification logic
- [ ] Position-aware text replacement
- [ ] Dry-run preview functionality
- [ ] Error recovery and rollback

---

## Phase 3: Advanced Link Types (Week 5-6)

### 3.1 Complex Link Format Support

#### Task 3.1.1: Fragment and Block References

```javascript
class FragmentHandler {
  parseFragment(linkTarget) {
    // Extract #heading or #^blockid
    return {
      path: "",
      heading: "",
      blockId: "",
      type: "heading" | "block" | "none",
    };
  }

  reconstructLink(basePath, fragment, linkType) {
    // Rebuild link with preserved fragment
  }
}
```

**Special Cases to Handle**:

- Heading links: `[[note#My Heading]]`
- Block references: `[[note#^abc123]]`
- Embedded content: `![[note#heading]]`
- Current file fragments: `[[#heading]]`

**Deliverables**:

- [ ] Fragment parsing and preservation
- [ ] Block reference handling
- [ ] Embed link updating
- [ ] Same-file fragment links

#### Task 3.1.2: Encoding and Special Characters

```javascript
class EncodingHandler {
  detectEncoding(linkText) {
    // Detect URL encoding, angle brackets, etc.
  }

  preserveEncodingStyle(originalLink, newTarget) {
    // Maintain original encoding approach
  }

  handleSpecialCharacters(filename) {
    // Process spaces, unicode, symbols
  }
}
```

**Edge Cases**:

- URL encoded spaces: `my%20note.md`
- Angle bracket escaping: `<my note.md>`
- Unicode characters in filenames
- Reserved characters: `|`, `#`, `^`

**Deliverables**:

- [ ] Encoding detection and preservation
- [ ] Special character handling
- [ ] Cross-encoding compatibility
- [ ] Validation for edge cases

---

## Phase 4: Performance and Safety (Week 7-8)

### 4.1 Performance Optimization

#### Task 4.1.1: Efficient Vault Processing

```javascript
class PerformanceOptimizer {
  async processVaultParallel(vaultPath, oldPath, newPath, options) {
    // Parallel file processing with worker threads
  }

  createFileCache(vaultPath) {
    // Cache file metadata for faster processing
  }

  streamLargeFiles(filePath) {
    // Handle very large files without memory issues
  }
}
```

**Optimization Strategies**:

- Parallel file processing (thread pool)
- File filtering to skip non-markdown files
- Memory-efficient streaming for large files
- Caching of file metadata

**Performance Targets**:

- 1,000 files: < 5 seconds
- 10,000 files: < 30 seconds
- Memory usage: < 100MB for 10,000 files

**Deliverables**:

- [ ] Parallel processing implementation
- [ ] Memory usage optimization
- [ ] Performance benchmarking
- [ ] Progress reporting system

### 4.2 Safety and Backup System

#### Task 4.2.1: File Safety Framework

```javascript
class SafetyManager {
  createBackup(filePath) {
    // Create timestamped backup before changes
  }

  validateChanges(originalContent, modifiedContent) {
    // Verify changes are correct
  }

  rollbackChanges(backupId) {
    // Restore files from backup
  }
}
```

**Safety Features**:

- Automatic backups before any changes
- Atomic file operations where possible
- Validation of changes before commit
- Rollback capability

**Deliverables**:

- [ ] Backup creation and management
- [ ] Change validation system
- [ ] Rollback functionality
- [ ] Error recovery mechanisms

---

## Phase 5: Integration and Polish (Week 9-10)

### 5.1 Integration with Existing Rename Command

#### Task 5.1.1: Command Integration

```javascript
// Integrate with existing rename command
class RenameWithLinkUpdate {
  async execute(oldPath, newPath, options = {}) {
    // 1. Validate rename operation
    // 2. Create backups if needed
    // 3. Scan for links
    // 4. Perform file rename (existing code)
    // 5. Update all links
    // 6. Report results
  }
}
```

**Integration Points**:

- Hook into existing command structure
- Preserve existing CLI interface
- Add new options without breaking changes
- Maintain backward compatibility

#### Task 5.1.2: Command Line Options

```bash
# New options to add to existing rename command
mdnotes rename old.md new.md [existing-options] [new-options]

# New options:
--update-links      # Enable link updating (default: true)
--no-link-update    # Disable link updating
--dry-run          # Preview changes without executing
--backup           # Force backup creation
--no-backup        # Skip backup creation
```

**Deliverables**:

- [ ] Seamless integration with existing command
- [ ] New CLI options
- [ ] Backward compatibility
- [ ] Help text updates

### 5.2 Error Handling and User Experience

#### Task 5.2.1: Comprehensive Error Handling

```javascript
class ErrorHandler {
  handleLinkUpdateErrors(errors) {
    // Process and report link update failures
  }

  generateUserReport(results) {
    // Create human-readable summary
  }

  suggestResolution(error) {
    // Provide actionable error resolution
  }
}
```

**Error Scenarios**:

- File permission issues
- Corrupted or binary files
- Ambiguous link references
- Circular reference detection

#### Task 5.2.2: User Feedback System

```
Example output:
Renaming: project-notes.md → project-overview.md

Scanning vault for links... ✓ (234 files scanned)
Found 15 links to update across 8 files

Updating links:
  ✓ research/analysis.md (3 links)
  ✓ daily/2024-06-20.md (1 link)
  ✓ projects/index.md (2 links)
  ⚠ templates/note.md (1 ambiguous link - skipped)

Summary:
  • Files renamed: 1
  • Links updated: 14
  • Links skipped: 1
  • Files modified: 7
  • Backup created: .mdnotes/backups/2024-06-22-14-30-15/

Time: 1.2s
```

**Deliverables**:

- [ ] Comprehensive error handling
- [ ] Clear progress reporting
- [ ] User-friendly output formatting
- [ ] Warning and error categorization

---

## Testing Strategy

### Unit Tests (Throughout Development)

- [ ] Regex pattern testing (1000+ test cases)
- [ ] Path resolution logic
- [ ] Link reconstruction accuracy
- [ ] Edge case handling

### Integration Tests

- [ ] End-to-end workflow testing
- [ ] Performance benchmarking
- [ ] Cross-platform compatibility
- [ ] Large vault testing (stress tests)

### Edge Case Testing

- [ ] Special character filenames
- [ ] Complex vault structures
- [ ] Circular references
- [ ] Malformed links

---

## Risk Mitigation

### Technical Risks

- **Complex regex causing performance issues**
  - Mitigation: Benchmark patterns, use compiled regex
- **False positive link detection**
  - Mitigation: Extensive test suite, conservative matching
- **Memory usage with large vaults**
  - Mitigation: Streaming processing, file chunking

### User Experience Risks

- **Breaking existing workflows**
  - Mitigation: Maintain backward compatibility, feature flags
- **Data loss during updates**
  - Mitigation: Automatic backups, validation checks
- **Performance degradation**
  - Mitigation: Performance testing, optimization passes

---

## Deliverable Timeline

| Week | Phase                | Key Deliverables                       |
| ---- | -------------------- | -------------------------------------- |
| 1-2  | Foundation           | Link detection engine, data structures |
| 3-4  | Core Processing      | Path resolution, basic link updating   |
| 5-6  | Advanced Features    | Complex links, encoding handling       |
| 7-8  | Performance & Safety | Optimization, backup system            |
| 9-10 | Integration          | Command integration, polish            |

## Definition of Done

Each phase is complete when:

- [ ] All code is written and tested
- [ ] Unit tests achieve 95%+ coverage
- [ ] Integration tests pass
- [ ] Performance targets are met
- [ ] Documentation is updated
- [ ] Code review is completed

This implementation plan provides a structured approach to building comprehensive link updating functionality while integrating smoothly with the existing mdnotes rename command.
