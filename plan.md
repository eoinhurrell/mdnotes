# 🗃️ mdnotes export - Implementation Plan

## 🎯 Phase-Based Implementation Strategy

Breaking down the export feature into **4 phases** with **12 focused tasks** that provide incremental value and can be developed by junior engineers.

---

## 📋 **Phase 1: Core Export Foundation** (Weeks 1-2)

_Goal: Basic file copying with query selection_

### Task 1.1: Basic Export Command Structure

**Effort:** 2-3 days | **Value:** High | **Risk:** Low

**Description:** Create the CLI command scaffold and basic file copying mechanism.

**Acceptance Criteria:**

- [ ] `mdnotes export <output-folder>` command exists
- [ ] Copies all markdown files from vault to output folder
- [ ] Preserves directory structure
- [ ] Handles basic error cases (output exists, permissions)
- [ ] Returns exit codes (0=success, 1=error)

**Implementation Notes:**

- Extend existing CLI parser with export subcommand
- Use filesystem utilities for recursive copying
- Create output directory if it doesn't exist
- Basic validation of input/output paths

**Test Cases:**

- Export empty vault
- Export vault with nested folders
- Export to existing/non-existing directories
- Permission errors

---

### Task 1.2: Query Integration

**Effort:** 2-3 days | **Value:** High | **Risk:** Low

**Description:** Integrate existing query engine to filter which files get exported.

**Acceptance Criteria:**

- [ ] `--query` flag accepts query strings
- [ ] Only files matching query are exported
- [ ] Uses existing `mdnotes frontmatter query` predicates
- [ ] Maintains directory structure for filtered results
- [ ] Logs count of files selected vs total

**Implementation Notes:**

- Reuse existing query parser and execution engine
- Filter file list before copying
- Preserve relative paths in output

**Test Cases:**

- `--query "folder=areas/"`
- `--query "tags=philosophy"`
- `--query "created>=2024-01-01"`
- Complex queries with AND/OR
- No files match query

---

### Task 1.3: Export Summary & Dry Run

**Effort:** 1-2 days | **Value:** Medium | **Risk:** Low

**Description:** Add reporting and preview capabilities.

**Acceptance Criteria:**

- [ ] `--dry-run` flag shows what would be exported without copying
- [ ] Summary shows: files included, total size, output path
- [ ] `--verbose` flag shows individual file paths
- [ ] Clean, readable output format

**Implementation Notes:**

- Separate planning phase from execution phase
- File size calculation
- Pretty-printed summary table

**Test Cases:**

- Dry run with various queries
- Verbose output formatting
- Large vault performance

---

## 📋 **Phase 2: Link Processing** (Weeks 3-4)

_Goal: Handle internal links in exported files_

### Task 2.1: Link Discovery & Analysis

**Effort:** 3-4 days | **Value:** High | **Risk:** Medium

**Description:** Scan markdown files to find and categorize all links.

**Acceptance Criteria:**

- [ ] Detects wikilinks: `[[Note Name]]`, `[[Note|Display]]`
- [ ] Detects markdown links: `[text](path.md)`
- [ ] Detects image embeds: `![[image.png]]`, `![alt](image.png)`
- [ ] Categorizes links as: internal (in export), external (not in export), assets
- [ ] Returns structured data about all links found

**Implementation Notes:**

- Use regex or markdown parser to find links
- Resolve relative paths correctly
- Handle edge cases: encoded characters, spaces in filenames
- Create link registry data structure

**Test Cases:**

- Various link formats and syntaxes
- Nested folders and relative paths
- Malformed links
- Unicode in filenames

---

### Task 2.2: Link Rewrite Engine

**Effort:** 2-3 days | **Value:** High | **Risk:** Medium

**Description:** Core engine to rewrite links based on strategy.

**Acceptance Criteria:**

- [ ] `--strategy remove` converts external links to plain text
- [ ] `--strategy url` uses frontmatter `url:` field when available
- [ ] Preserves internal links (updates paths if needed)
- [ ] Maintains link text/display names
- [ ] Handles both wikilinks and markdown links

**Implementation Notes:**

- String replacement with careful boundary detection
- Preserve surrounding markdown formatting
- Handle frontmatter parsing for URL strategy

**Test Cases:**

- External wikilinks → plain text
- External markdown links → plain text
- URL strategy with/without frontmatter URLs
- Mixed link types in same file
- Edge cases: nested brackets, special characters

---

### Task 2.3: Link Processing Integration

**Effort:** 2 days | **Value:** High | **Risk:** Low

**Description:** Integrate link processing into export workflow.

**Acceptance Criteria:**

- [ ] Links are rewritten during file copying
- [ ] Summary includes "External links rewritten: X"
- [ ] File content is modified in output, not source
- [ ] UTF-8 encoding preserved
- [ ] YAML frontmatter preserved

**Implementation Notes:**

- Process content after reading, before writing
- Ensure no corruption of file encoding
- Maintain file metadata (timestamps, etc.)

**Test Cases:**

- Files with no links (unchanged)
- Files with only internal links
- Files with only external links
- Mixed scenarios

---

## 📋 **Phase 3: Asset & Advanced Features** (Week 5)

_Goal: Asset copying and enhanced functionality_

### Task 3.1: Asset Discovery & Copying

**Effort:** 2-3 days | **Value:** Medium | **Risk:** Low

**Description:** Find and copy linked assets (images, PDFs, etc.).

**Acceptance Criteria:**

- [ ] `--include-assets` flag copies referenced files
- [ ] Supports: `.png`, `.jpg`, `.pdf`, `.csv`, `.xlsx`
- [ ] Updates asset links in markdown to match new paths
- [ ] Handles missing assets gracefully (log warning, continue)
- [ ] Summary includes "Assets copied: X"

**Implementation Notes:**

- Extend link discovery to track asset references
- Copy assets to preserve relative path structure
- Update asset links after copying

**Test Cases:**

- Images in various formats
- Relative vs absolute asset paths
- Missing asset files
- Assets in subdirectories

---

### Task 3.2: Backlinks Support

**Effort:** 2-3 days | **Value:** Medium | **Risk:** Medium

**Description:** Include notes that link TO exported files.

**Acceptance Criteria:**

- [ ] `--with-backlinks` flag includes additional files
- [ ] Finds files that link to any file in the export set
- [ ] Recursive: if backlink file has backlinks, include those too
- [ ] Prevents infinite loops
- [ ] Summary shows backlinks added

**Implementation Notes:**

- Build reverse link index from existing link discovery
- Iterative expansion of file set
- Cycle detection for safety

**Test Cases:**

- Simple backlink inclusion
- Multi-level backlink chains
- Circular reference handling
- Performance with large vaults

---

### Task 3.3: Filename Normalization

**Effort:** 1-2 days | **Value:** Low | **Risk:** Low

**Description:** Optional filename transformations for compatibility.

**Acceptance Criteria:**

- [ ] `--slugify` converts filenames to URL-safe slugs
- [ ] `--flatten` puts all files in single directory
- [ ] Updates internal links to match new filenames
- [ ] Handles name collisions (add numbers)

**Implementation Notes:**

- Slug generation: lowercase, replace spaces/special chars
- Collision detection and resolution
- Link path updates

**Test Cases:**

- Various filename formats
- Unicode in filenames
- Name collision scenarios
- Link consistency after renaming

---

## 📋 **Phase 4: Polish & Documentation** (Week 6)

_Goal: Production readiness and user experience_

### Task 4.1: Error Handling & Validation

**Effort:** 2 days | **Value:** High | **Risk:** Low

**Description:** Robust error handling and input validation.

**Acceptance Criteria:**

- [ ] Clear error messages for invalid queries
- [ ] Graceful handling of filesystem errors
- [ ] Validation of output path safety
- [ ] Progress indicators for large exports
- [ ] Consistent exit codes

**Implementation Notes:**

- Input sanitization and validation
- User-friendly error messages
- Progress bars for long operations

---

### Task 4.2: Performance Optimization

**Effort:** 1-2 days | **Value:** Medium | **Risk:** Low

**Description:** Ensure good performance with large vaults.

**Acceptance Criteria:**

- [ ] <1s export time for <100 files
- [ ] <10s export time for <1000 files
- [ ] Memory usage scales reasonably
- [ ] Parallel file operations where safe

**Implementation Notes:**

- Profile current implementation
- Optimize file I/O operations
- Consider parallel processing

---

### Task 4.3: Documentation & Examples

**Effort:** 1-2 days | **Value:** High | **Risk:** Low

**Description:** Complete user documentation and examples.

**Acceptance Criteria:**

- [ ] CLI help text with all options
- [ ] README examples for common use cases
- [ ] Error scenarios and troubleshooting
- [ ] Performance guidelines

---

## 🧩 **Task Dependencies**

```
1.1 (CLI) → 1.2 (Query) → 1.3 (Summary)
                ↓
2.1 (Link Discovery) → 2.2 (Rewrite) → 2.3 (Integration)
                ↓
3.1 (Assets) ← 3.2 (Backlinks) ← 3.3 (Normalization)
                ↓
4.1 (Errors) → 4.2 (Performance) → 4.3 (Docs)
```

## 🎯 **Minimal Viable Product (MVP)**

Tasks 1.1, 1.2, 2.1, 2.2, 2.3 provide a working export with link rewriting - the core value proposition.

## 🧪 **Integration Points**

- Reuse existing query engine (no changes needed)
- Extend CLI parser (minor addition)
- Use existing file scanning utilities
- Add new export module alongside existing commands

This plan provides **incremental value** at each phase while keeping tasks **focused and achievable** for junior developers.
