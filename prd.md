# Product Requirements Document: Smart Link Updating for mdnotes Rename Command

## Document Information

- **Product**: mdnotes CLI
- **Feature**: Smart Link Updating for Rename Command
- **Version**: 1.0
- **Date**: June 22, 2025
- **Author**: Product Team

---

## Executive Summary

The mdnotes rename command currently only renames files without updating links that reference those files, leading to broken links and vault inconsistency. This PRD defines a comprehensive solution to automatically update all link references when renaming files, ensuring vault integrity and user productivity.

---

## Problem Statement

### Current State

- Users rename files using `mdnotes rename old-name.md new-name.md`
- All existing links to the renamed file become broken
- Users must manually find and update references across potentially hundreds of files
- This creates friction and discourages file organization

### Pain Points

1. **Broken link ecosystem**: Renaming destroys the knowledge graph connections
2. **Manual link hunting**: Time-consuming and error-prone process to find all references
3. **Incomplete updates**: Users often miss obscure link formats or embedded references
4. **Vault inconsistency**: Broken links degrade the vault's reliability over time
5. **Workflow disruption**: Fear of breaking links prevents necessary reorganization

---

## Goals and Objectives

### Primary Goals

- **Zero broken links**: Ensure no links are broken after renaming operations
- **Comprehensive coverage**: Update all link types that Obsidian supports
- **Performance**: Complete updates efficiently even in large vaults (10,000+ files)
- **Reliability**: Provide bulletproof link detection and updating

### Success Metrics

- 100% of referenceable links updated correctly
- Sub-5-second execution time for vaults with 1,000+ files
- Zero false positives (incorrect updates)
- Zero false negatives (missed updates)

---

## User Stories

### Primary User Stories

**As a knowledge worker**, I want to rename files without worrying about breaking links, so that I can organize my vault freely.

**As a content creator**, I want to refactor my note structure confidently, so that I can improve my knowledge organization over time.

**As a team member**, I want to standardize file naming in shared vaults, so that our collective knowledge remains accessible.

### Edge Case User Stories

**As a power user**, I want complex link formats (blocks, headings, embeds) to be updated correctly, so that my advanced workflows continue functioning.

**As a vault maintainer**, I want to rename files with special characters or complex paths, so that I can clean up legacy naming issues.

---

## Functional Requirements

### Core Functionality

#### FR-1: Comprehensive Link Detection

The system MUST detect and update all of the following link types:

**Wikilinks:**

- Basic: `[[old-name]]` → `[[new-name]]`
- With extension: `[[old-name.md]]` → `[[new-name.md]]`
- With path: `[[folder/old-name]]` → `[[folder/new-name]]`
- With alias: `[[old-name|Display Text]]` → `[[new-name|Display Text]]`
- With heading: `[[old-name#heading]]` → `[[new-name#heading]]`
- With block: `[[old-name#^blockid]]` → `[[new-name#^blockid]]`

**Embedded Wikilinks:**

- File embed: `![[old-name]]` → `![[new-name]]`
- Section embed: `![[old-name#heading]]` → `![[new-name#heading]]`
- Block embed: `![[old-name#^blockid]]` → `![[new-name#^blockid]]`

**Markdown Links:**

- Basic: `[text](old-name.md)` → `[text](new-name.md)`
- With path: `[text](folder/old-name.md)` → `[text](folder/new-name.md)`
- With fragment: `[text](old-name.md#heading)` → `[text](new-name.md#heading)`
- URL encoded: `[text](old%20name.md)` → `[text](new%20name.md)`

**Markdown Images:**

- `![alt](old-image.png)` → `![alt](new-image.png)`

#### FR-2: Path Format Handling

The system MUST handle all path reference formats:

- **Filename only**: `[[old-name]]` (when file is uniquely named)
- **Relative to vault root**: `[[folder/subfolder/old-name]]`
- **With/without extensions**: Both `.md` explicit and implicit
- **URL encoding**: Handle `%20` for spaces and other encoded characters
- **Angle bracket escaping**: `[text](<old name.md>)`

#### FR-3: Special Character Support

The system MUST correctly handle filenames containing:

- Spaces: `My Note.md`
- Unicode characters: `測試.md`
- Special characters: `Note (v2).md`, `Note-2024.md`
- Parentheses, brackets, hyphens, underscores
- Reserved characters in different encoding contexts

#### FR-4: Vault Scanning

The system MUST:

- Scan all `.md` files in the vault recursively
- Process files in all subdirectories
- Handle large vaults (10,000+ files) efficiently
- Provide progress indication for long operations
- Skip non-text files and respect `.gitignore` patterns

### Advanced Functionality

#### FR-5: Disambiguation Handling

When multiple files have the same name:

- Update only exact path matches for fully-qualified links
- Prompt user for ambiguous filename-only links
- Provide clear resolution options
- Allow batch decisions for multiple ambiguous cases

#### FR-6: Backup and Safety

The system MUST:

- Create automatic backups before making changes
- Provide dry-run mode to preview changes
- Allow rollback of rename operations
- Validate changes before committing

#### FR-7: Reporting

The system MUST provide:

- Summary of files scanned
- Count of links updated by type
- List of files modified
- Any warnings or errors encountered
- Performance timing information

---

## Technical Requirements

### TR-1: Link Detection Algorithm

- Use comprehensive regex patterns for each link type
- Handle nested markdown structures correctly
- Avoid false positives in code blocks and comments
- Process frontmatter and metadata sections

### TR-2: File Processing

- Read files with proper encoding detection (UTF-8, etc.)
- Preserve file formatting and whitespace
- Handle large files efficiently (streaming when needed)
- Maintain file permissions and timestamps

### TR-3: Path Resolution

- Implement vault-relative path resolution
- Handle case sensitivity per filesystem
- Normalize path separators across platforms
- Resolve symbolic links appropriately

### TR-4: Performance Requirements

- Process 1,000 files in under 5 seconds
- Use memory efficiently for large vaults
- Implement parallel processing where safe
- Cache file system operations

### TR-5: Error Handling

- Graceful failure with detailed error messages
- Continue processing after non-critical errors
- Validate rename target doesn't conflict
- Handle permission errors appropriately

---

## Detailed Behavior Specifications

### Link Update Logic

#### Exact Match Priority

1. **Full path matches**: Update `[[folder/old-name]]` first
2. **Filename matches**: Update `[[old-name]]` if unambiguous
3. **Extension variants**: Handle both `[[old-name]]` and `[[old-name.md]]`

#### Fragment Preservation

- Maintain heading references: `[[old#heading]]` → `[[new#heading]]`
- Preserve block references: `[[old#^block]]` → `[[new#^block]]`
- Keep custom display text: `[[old|Custom]]` → `[[new|Custom]]`

#### Encoding Consistency

- Preserve original encoding style in markdown links
- Convert between encoding styles only when necessary
- Maintain URL encoding for special characters

### Edge Cases

#### EC-1: Circular References

- Detect when renaming would create circular embeds
- Warn user but allow operation to proceed
- Document behavior in help text

#### EC-2: Missing Files

- Handle links to non-existent files (future references)
- Update these "stub" links appropriately
- Don't break intentional dead links

#### EC-3: Binary Files

- Update links to images, PDFs, and other assets
- Handle different file extensions correctly
- Preserve MIME type associations

#### EC-4: Case Sensitivity

- Respect filesystem case sensitivity rules
- Handle case-only renames correctly
- Warn about potential case conflicts

### Command Interface

#### Basic Usage

```bash
mdnotes rename old-file.md  # renames using default template
```

```bash
mdnotes rename old-file.md new-file.md
```

#### Advanced Options

```bash
mdnotes rename old-file.md new-file.md [options]

Options:
  --dry-run, -n          Show what would be changed without making changes
  --verbose, -v          Show detailed progress and file list
  --no-backup           Skip creating backup files
  --force               Skip confirmation prompts
  --include=PATTERN     Only process files matching pattern
  --exclude=PATTERN     Skip files matching pattern
  --parallel=N          Use N parallel workers (default: auto)
```

#### Interactive Mode

For ambiguous cases:

```
Found multiple files named "note.md":
1. projects/note.md
2. archive/note.md
3. templates/note.md

Update links for:
[a] All files
[1] projects/note.md only
[2] archive/note.md only
[3] templates/note.md only
[s] Skip this ambiguity
Choice:
```

---

## Success Criteria

### Functional Success

- [ ] All supported link types are detected and updated
- [ ] No broken links result from rename operations
- [ ] Complex paths and special characters handled correctly
- [ ] Ambiguous cases are resolved appropriately

### Performance Success

- [ ] 1,000 file vault processed in <5 seconds
- [ ] 10,000 file vault processed in <30 seconds
- [ ] Memory usage scales reasonably with vault size
- [ ] Progress indication for operations >2 seconds

### Reliability Success

- [ ] Zero data loss in 1,000 test operations
- [ ] Backup and rollback functions work correctly
- [ ] Error handling prevents vault corruption
- [ ] Edge cases handled gracefully

### User Experience Success

- [ ] Clear, actionable error messages
- [ ] Intuitive command interface
- [ ] Helpful dry-run and preview modes
- [ ] Comprehensive documentation

---

## Implementation Considerations

### Phase 1: Core Implementation

1. Basic wikilink detection and updating
2. Simple markdown link handling
3. File system operations and safety
4. Basic command interface

### Phase 2: Advanced Features

1. Complex link formats (blocks, headings)
2. Embedded content handling
3. Special character support
4. Performance optimization

### Phase 3: Polish and Edge Cases

1. Disambiguation interface
2. Advanced command options
3. Comprehensive testing
4. Documentation and examples

### Dependencies

- File system watching capabilities
- Regex engine with Unicode support
- Progress indication libraries
- Backup/restore functionality

### Risks and Mitigations

- **Risk**: Performance on very large vaults
  - **Mitigation**: Implement streaming and parallel processing
- **Risk**: Complex regex causing false positives
  - **Mitigation**: Extensive test suite with edge cases
- **Risk**: Data loss during rename operations
  - **Mitigation**: Automatic backups and dry-run mode

---

## Testing Strategy

### Unit Tests

- Link detection regex patterns
- Path resolution logic
- File encoding handling
- Error condition handling

### Integration Tests

- End-to-end rename workflows
- Large vault performance tests
- Cross-platform compatibility
- Backup and restore functionality

### Edge Case Tests

- Special character filenames
- Complex vault structures
- Ambiguous link scenarios
- Corrupted or unusual files

---

## Documentation Requirements

### User Documentation

- Command reference with examples
- Common use cases and workflows
- Troubleshooting guide
- Best practices for vault organization

### Technical Documentation

- Algorithm explanation
- Performance characteristics
- Extension points for future features
- Contribution guidelines

---

This PRD ensures that the mdnotes rename command becomes a robust, reliable tool that maintains vault integrity while enabling fearless file organization.
