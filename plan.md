# 🗃️ mdnotes vNext - Complete Implementation Plan

## 🎯 Overview

This plan implements the complete mdnotes CLI revamp based on the PRD, breaking down work into **5 phases** over **5 weeks** with **25 detailed tickets**. Each ticket is designed to be handed off to junior developers with comprehensive requirements and testing expectations.

---

## 📋 **Phase 1: Foundation & Architecture** (Week 1)

_Goal: Establish new architecture, unified config, and CLI structure_

### Task 1.1: Project Restructuring & Package Layout

**Effort:** 2-3 days | **Value:** High | **Risk:** Medium

**Description:** Reorganize codebase into new package structure with clear boundaries and modern Go practices.

**Acceptance Criteria:**

- [ ] New package structure implemented:
  ```
  cmd/mdnotes/
  pkg/config/
  pkg/vault/
  pkg/frontmatter/
  pkg/linkding/
  pkg/analyze/
  pkg/links/
  pkg/export/
  pkg/watch/
  pkg/plugins/
  internal/rgsearch/
  internal/workerpool/
  internal/templates/
  ```
- [ ] All existing functionality preserved during migration
- [ ] Package interfaces clearly defined
- [ ] Import cycles eliminated
- [ ] go.mod updated with proper module structure

**Implementation Notes:**

- Move existing code to appropriate packages
- Define clear interfaces between packages
- Use internal/ for implementation details
- Maintain backward compatibility during transition

**Test Requirements:**

- [ ] Unit tests: 90%+ coverage for each package
- [ ] Integration tests: All existing CLI commands work
- [ ] Performance tests: No regression from current implementation
- [ ] Build tests: Clean builds on Go 1.21+

**Test Cases:**

- Package boundary enforcement
- Interface compliance
- Import cycle detection
- Cross-platform builds

---

### Task 1.2: Unified Configuration System

**Effort:** 2-3 days | **Value:** High | **Risk:** Low

**Description:** Implement centralized YAML configuration with hierarchical loading and validation.

**Acceptance Criteria:**

- [ ] `mdnotes.yaml` config file support
- [ ] Hierarchical loading: CWD → user home → `/etc`
- [ ] Environment variable overrides
- [ ] Configuration validation with clear error messages
- [ ] Auto-migration from legacy `obsidian-admin.yaml`
- [ ] Schema includes all sections from PRD:
  ```yaml
  vault:
    path: "."
    ignore_patterns: [".obsidian/*", "*.tmp"]
  frontmatter:
    # upsert defaults, type rules
  linkding:
    api_url: ""
    api_token: ""
    sync_title: true
    sync_tags: true
  export:
    default_strategy: "remove"
    include_assets: true
  watch:
    debounce: "2s"
    rules: []
  performance:
    max_workers: 0
  plugins:
    enabled: true
    paths: ["~/.mdnotes/plugins"]
  ```

**Implementation Notes:**

- Use viper or similar for configuration management
- JSON Schema validation for config structure
- Secure handling of API tokens
- Path expansion for ~ and environment variables

**Test Requirements:**

- [ ] Unit tests: All config loading scenarios
- [ ] Integration tests: Config precedence rules
- [ ] Security tests: Token handling, path traversal prevention
- [ ] Migration tests: Legacy config compatibility

**Test Cases:**

- Config file discovery and precedence
- Invalid YAML handling
- Missing required fields
- Environment variable substitution
- Legacy config migration
- Security: path traversal attempts
- Unicode in config values

---

### Task 1.3: Modern CLI Structure & Aliases

**Effort:** 2 days | **Value:** High | **Risk:** Low

**Description:** Redesign CLI with domain-grouped commands, power aliases, and consistent flag patterns.

**Acceptance Criteria:**

- [ ] New command structure:
  - `fm` (frontmatter), `analyze`, `links`, `export`, `watch`, `rename`, `linkding`
- [ ] Power aliases implemented:
  - `mdnotes u` → `fm upsert`
  - `mdnotes q` → `fm query`
  - `mdnotes a c` → `analyze content`
  - `mdnotes a i` → `analyze inbox`
  - `mdnotes r` → `rename`
  - `mdnotes x` → `export`
  - `mdnotes ld sync` → `linkding sync`
- [ ] Global flags: `--dry-run`, `--verbose`, `--quiet`, `--config`
- [ ] Consistent help text and examples
- [ ] Deprecation warnings for legacy commands

**Implementation Notes:**

- Use cobra for CLI framework
- Implement alias mapping
- Consistent flag naming across commands
- Rich help text with examples

**Test Requirements:**

- [ ] Unit tests: Command registration and aliases
- [ ] Integration tests: All command combinations work
- [ ] UX tests: Help text clarity and completeness
- [ ] Backward compatibility tests: Legacy command warnings

**Test Cases:**

- All alias combinations
- Global flag inheritance
- Help text formatting
- Command auto-completion
- Invalid command handling
- Legacy command deprecation warnings

---

### Task 1.4: Template Engine Foundation

**Effort:** 1-2 days | **Value:** Medium | **Risk:** Low

**Description:** Centralized template engine for upsert, rename, export, and watch commands.

**Acceptance Criteria:**

- [ ] Template variables supported:
  - `{{current_date}}`, `{{current_datetime}}`
  - `{{filename}}`, `{{filename|slug}}`
  - `{{title}}`, `{{file_mtime}}`
  - `{{relative_path}}`, `{{parent_dir}}`
  - `{{uuid}}`
- [ ] Template filters:
  - `|upper`, `|lower`, `|slug`
  - `|date:format`
- [ ] Error handling for invalid templates
- [ ] Template validation
- [ ] Performance optimized for repeated use

**Implementation Notes:**

- Use text/template with custom functions
- Implement filter functions as template funcs
- Cache compiled templates
- Secure template execution (no arbitrary code)

**Test Requirements:**

- [ ] Unit tests: All template variables and filters
- [ ] Integration tests: Template usage in commands
- [ ] Performance tests: Template rendering speed
- [ ] Security tests: Template injection prevention

**Test Cases:**

- All template variables and combinations
- Date formatting with various formats
- Unicode in template values
- Invalid template syntax
- Security: template injection attempts
- Performance: repeated template rendering

---

## 📋 **Phase 2: Frontmatter & Linkding Revamp** (Week 2)

_Goal: New upsert command, enhanced Linkding integration, sync capabilities_

### Task 2.1: `fm upsert` Command Implementation

**Effort:** 2-3 days | **Value:** High | **Risk:** Medium

**Description:** Replace `fm ensure` and `fm set` with unified `upsert` command supporting templates and conditional overwrite.

**Acceptance Criteria:**

- [ ] `mdnotes fm upsert [path] --field name --default value` syntax
- [ ] `--overwrite` flag for conditional replacement
- [ ] Template support in default values
- [ ] Batch processing for multiple fields
- [ ] Atomic operations (all or nothing)
- [ ] Detailed operation reporting
- [ ] Backward compatibility warnings for `fm ensure`/`fm set`

**Implementation Notes:**

- Parse frontmatter safely (preserve formatting)
- Template evaluation per file context
- File modification only when changes needed
- Progress reporting for large operations

**Test Requirements:**

- [ ] Unit tests: Frontmatter parsing and modification
- [ ] Integration tests: End-to-end upsert operations
- [ ] Regression tests: Backward compatibility
- [ ] Performance tests: Large vault operations

**Test Cases:**

- Field creation vs update scenarios
- Template variable substitution
- Multiple field operations
- YAML formatting preservation
- File encoding preservation (UTF-8)
- Concurrent file access handling
- Invalid frontmatter handling
- Missing file scenarios

---

### Task 2.2: `fm download` Implementation

**Effort:** 2 days | **Value:** Medium | **Risk:** Medium

**Description:** Download web resources from frontmatter URLs and replace with local links.

**Acceptance Criteria:**

- [ ] Scan frontmatter for HTTP(S) URLs
- [ ] Download to configurable `attachments/` directory
- [ ] Date-based subfolder organization
- [ ] Add fields: `<attribute>-original`, `<attribute>-downloaded_at`
- [ ] Support common file types (images, PDFs, documents)
- [ ] Retry logic with exponential backoff
- [ ] Content-Type validation
- [ ] File size limits

**Implementation Notes:**

- HTTP client with proper timeouts
- MIME type detection and validation
- Atomic file operations
- Progress reporting for large downloads

**Test Requirements:**

- [ ] Unit tests: URL detection and file operations
- [ ] Integration tests: Full download workflow
- [ ] Network tests: HTTP error handling
- [ ] Security tests: Malicious URL handling

**Test Cases:**

- Various URL formats and protocols
- Different file types and sizes
- Network failures and retries
- Invalid/malicious URLs
- File system permission errors
- Concurrent downloads
- Unicode in URLs and filenames

---

### Task 2.3: Enhanced `fm sync` Command

**Effort:** 1-2 days | **Value:** Medium | **Risk:** Low

**Description:** Map file metadata sources to frontmatter fields with regex extraction support.

**Acceptance Criteria:**

- [ ] Source options:
  - `--source file-mtime`
  - `--source filename`
  - `--source path:dir`
  - `--source filename:pattern:REGEX`
- [ ] Regex capture group support
- [ ] Date format normalization
- [ ] Batch processing with progress
- [ ] Validation of regex patterns

**Implementation Notes:**

- File stat operations for metadata
- Regex compilation and validation
- Date parsing and formatting
- Path manipulation utilities

**Test Requirements:**

- [ ] Unit tests: All source types and regex patterns
- [ ] Integration tests: Multi-source sync operations
- [ ] Performance tests: Large vault sync
- [ ] Validation tests: Invalid regex handling

**Test Cases:**

- File metadata extraction
- Regex pattern matching and capture
- Date format conversions
- Unicode in filenames and paths
- Invalid regex patterns
- Missing file scenarios

---

### Task 2.4: Robust Linkding Integration

**Effort:** 2-3 days | **Value:** High | **Risk:** Medium

**Description:** Implement comprehensive Linkding sync with two-way integration, retry logic, and status tracking.

**Acceptance Criteria:**

- [ ] `linkding sync` command with idempotent operations
- [ ] `linkding list` command showing sync status
- [ ] Two-way sync: vault → Linkding and Linkding → vault
- [ ] Retry logic with exponential backoff for rate limits
- [ ] Status tracking: `synced #123`, `unsynced`, `error`
- [ ] Batch operations for performance
- [ ] Configuration validation
- [ ] Table and JSON output formats

**Implementation Notes:**

- HTTP client with rate limiting
- Linkding API v1 integration
- Local state tracking for sync status
- Concurrent request handling with limits

**Test Requirements:**

- [ ] Unit tests: API client and sync logic
- [ ] Integration tests: Full sync workflows
- [ ] Network tests: Rate limiting and error handling
- [ ] API tests: Mock Linkding server

**Test Cases:**

- Successful bookmark creation/update
- Rate limiting and retry scenarios
- Network failures and recovery
- API authentication errors
- Malformed bookmark data
- Large batch operations
- Sync conflict resolution

---

## 📋 **Phase 3: Analysis Engine & Content Scoring** (Week 3)

_Goal: Zettelkasten-aware analysis with content scoring and inbox triage_

### Task 3.1: Content Quality Scoring Engine

**Effort:** 3-4 days | **Value:** High | **Risk:** Medium

**Description:** Implement five-factor content scoring system based on Zettelkasten principles.

**Acceptance Criteria:**

- [ ] Five scoring factors (each weighted 0.2):
  1. **Readability** (Flesch-Kincaid Reading Ease, normalized 0-1)
  2. **Link Density** (outbound links ÷ word count, ideal 0.02-0.04)
  3. **Completeness** (H1 matches title +0.2, ≥100 words +0.2, summary +0.2)
  4. **Atomicity** (>1 H2 -0.2, >500 words -0.2)
  5. **Recency** (max(0, 1 - age_days/365))
- [ ] Score calculation: (sum of factors) × 100 → 0-100 scale
- [ ] Per-note breakdown with suggestions
- [ ] Summary showing worst N notes
- [ ] Performance optimized for large vaults

**Implementation Notes:**

- Markdown parsing for heading detection
- Word count algorithms
- Date parsing from frontmatter
- Link counting from existing link detection
- Flesch-Kincaid implementation

**Test Requirements:**

- [ ] Unit tests: Each scoring factor independently
- [ ] Integration tests: End-to-end scoring
- [ ] Performance tests: Large vault analysis
- [ ] Accuracy tests: Manual verification of scores

**Test Cases:**

- High-quality notes (score >80)
- Low-quality notes (score <40)
- Edge cases: empty files, no frontmatter
- Various markdown structures
- Unicode content handling
- Performance with 10k+ files

---

### Task 3.2: `analyze content` Command

**Effort:** 2 days | **Value:** High | **Risk:** Low

**Description:** CLI command for content quality analysis with actionable suggestions.

**Acceptance Criteria:**

- [ ] `analyze content` command with quality scoring
- [ ] `--scores` flag for individual file scores
- [ ] `--min-score` filter for quality threshold
- [ ] Verbose mode with metric breakdown
- [ ] Suggestions for improvement:
  - "Split note" for large files
  - "Add links" for low link density
  - "Add title" for missing H1
  - "Expand content" for short notes
- [ ] Table and JSON output formats
- [ ] Performance indicators

**Implementation Notes:**

- Rich terminal output with colors
- Suggestion generation based on scoring factors
- Sorting and filtering capabilities
- Progress reporting for analysis

**Test Requirements:**

- [ ] Unit tests: Command parsing and output
- [ ] Integration tests: Full analysis workflow
- [ ] Output tests: Format validation
- [ ] Performance tests: Large vault analysis

**Test Cases:**

- Various score ranges and filtering
- Output format consistency
- Suggestion accuracy
- Progress reporting
- Error handling for corrupted files

---

### Task 3.3: `analyze inbox` Command

**Effort:** 2 days | **Value:** Medium | **Risk:** Low

**Description:** Specialized analysis for inbox files needing attention.

**Acceptance Criteria:**

- [ ] Detect files with issues:
  - Top-level `# INBOX` heading
  - Missing required frontmatter (`created`, `tags`)
  - Word count anomalies (<20 or >200 words)
- [ ] Tabular output with issues and suggestions
- [ ] File snippet preview (first 50 chars)
- [ ] Suggested actions:
  - "Add tags; expand" for short files
  - "Add frontmatter" for missing metadata
  - "Process inbox item" for inbox files
- [ ] Configurable word count thresholds

**Implementation Notes:**

- Heading detection algorithms
- Frontmatter field checking
- Text snippet extraction
- Configurable thresholds

**Test Requirements:**

- [ ] Unit tests: Issue detection algorithms
- [ ] Integration tests: Full inbox analysis
- [ ] Configuration tests: Custom thresholds
- [ ] Output tests: Table formatting

**Test Cases:**

- Various inbox file scenarios
- Missing frontmatter combinations
- Word count edge cases
- Snippet extraction with unicode
- Configuration override testing

---

### Task 3.4: Enhanced `analyze links` & `analyze health`

**Effort:** 2 days | **Value:** Medium | **Risk:** Low

**Description:** Comprehensive link analysis and overall vault health assessment.

**Acceptance Criteria:**

- [ ] `analyze links` features:
  - Graph statistics (nodes, edges, density)
  - Orphan detection (files with no links)
  - Hub scores (most-linked files)
  - Broken link detection
- [ ] `analyze health` features:
  - Composite health index (0-100)
  - Frontmatter completeness score
  - Link integrity percentage
  - Content score average
  - Recommendations for improvement
- [ ] JSON output for automation
- [ ] Historical tracking support

**Implementation Notes:**

- Graph algorithms for link analysis
- Weighted scoring for health index
- Efficient link traversal
- Caching for performance

**Test Requirements:**

- [ ] Unit tests: Graph algorithms and health calculations
- [ ] Integration tests: Full analysis workflows
- [ ] Performance tests: Large vault analysis
- [ ] Accuracy tests: Manual verification

**Test Cases:**

- Various vault sizes and structures
- Orphaned file detection
- Broken link scenarios
- Health score calculations
- JSON output validation

---

## 📋 **Phase 4: Export Enhancement & Watch Mode** (Week 4)

_Goal: Advanced export features and automation framework_

### Task 4.1: Enhanced Export with Subgraph Support

**Effort:** 3 days | **Value:** High | **Risk:** Medium

**Description:** Upgrade export with backlinks, advanced filtering, and link strategies.

**Acceptance Criteria:**

- [ ] `--with-backlinks` for inbound neighbor inclusion
- [ ] Enhanced link strategies:
  - `remove`: strip unexported links to plain text
  - `url`: replace with frontmatter `url:` field
  - `stub`: generate minimal stub notes (future)
- [ ] `--include-assets` with asset copying
- [ ] `--slugify` and `--flatten` filename options
- [ ] Parallel export processing
- [ ] Export metadata file (`export-metadata.yaml`)
- [ ] In-memory link adjustment

**Implementation Notes:**

- Extend existing export functionality
- Backlink graph traversal
- Asset dependency tracking
- Parallel processing with worker pools
- Link rewriting engine

**Test Requirements:**

- [ ] Unit tests: Each export feature independently
- [ ] Integration tests: Complex export scenarios
- [ ] Performance tests: Large vault exports
- [ ] Data integrity tests: Link consistency

**Test Cases:**

- Backlink discovery and inclusion
- Asset copying with various file types
- Link strategy application
- Filename normalization
- Parallel processing correctness
- Export metadata accuracy

---

### Task 4.2: Watch Mode Implementation

**Effort:** 3 days | **Value:** High | **Risk:** High

**Description:** File system monitoring with configurable rules and action execution.

**Acceptance Criteria:**

- [ ] `mdnotes watch` command with daemon mode
- [ ] YAML rule configuration:
  ```yaml
  watch:
    debounce: "2s"
    rules:
      - name: "Inbox processing"
        paths: ["inbox/**/*.md"]
        events: ["create", "write"]
        actions:
          ["mdnotes u {{file}} --field created --default '{{current_date}}'"]
  ```
- [ ] File event types: create, write, remove, rename, chmod
- [ ] Template placeholders: `{{file}}`, `{{dir}}`, `{{basename}}`
- [ ] Debouncing to prevent duplicate processing
- [ ] Action execution with error handling
- [ ] Logging and monitoring

**Implementation Notes:**

- File system watcher (fsnotify)
- Event debouncing algorithms
- Template execution in action context
- Process management for daemon mode
- Signal handling for graceful shutdown

**Test Requirements:**

- [ ] Unit tests: Event processing and rule matching
- [ ] Integration tests: Full watch workflows
- [ ] Stress tests: High-frequency file changes
- [ ] Reliability tests: Long-running daemon stability

**Test Cases:**

- Various file event scenarios
- Rule matching and action execution
- Debouncing behavior
- Template substitution in actions
- Error handling and recovery
- Daemon mode operation
- Signal handling

---

### Task 4.3: Plugin System Foundation

**Effort:** 2 days | **Value:** Low | **Risk:** High

**Description:** Basic plugin architecture with hook points and discovery.

**Acceptance Criteria:**

- [ ] Plugin hook points:
  - Pre-command, per-file, post-command, export-complete
- [ ] Plugin discovery from `~/.mdnotes/plugins/*.so`
- [ ] Plugin API for:
  - Registering new commands
  - Adding custom flags
  - Custom query predicates
  - Template functions
- [ ] Plugin configuration in main config
- [ ] Security sandboxing
- [ ] Plugin lifecycle management

**Implementation Notes:**

- Go plugin system or RPC-based plugins
- Interface definitions for plugin API
- Security considerations for plugin execution
- Plugin state management

**Test Requirements:**

- [ ] Unit tests: Plugin loading and API
- [ ] Integration tests: Plugin hook execution
- [ ] Security tests: Plugin sandboxing
- [ ] Example plugins for testing

**Test Cases:**

- Plugin discovery and loading
- Hook point execution
- API contract compliance
- Security boundary enforcement
- Plugin failure handling
- Configuration validation

---

## 📋 **Phase 5: Performance & Production Readiness** (Week 5)

_Goal: Optimization, comprehensive testing, and production deployment_

### Task 5.1: Performance Optimization & Worker Pools

**Effort:** 2-3 days | **Value:** High | **Risk:** Medium

**Description:** Implement parallel processing and performance optimizations throughout the system.

**Acceptance Criteria:**

- [ ] Performance targets met:
  - <500ms for 10k-note rename/export
  - <200MB peak memory on 10k vault
- [ ] Worker pool implementation for CPU-bound tasks
- [ ] Ripgrep integration for fast file searching
- [ ] In-memory caching for repeated operations
- [ ] Parallel file processing where safe
- [ ] Memory usage optimization
- [ ] Progress reporting for long operations

**Implementation Notes:**

- Worker pool with configurable size
- Ripgrep subprocess integration
- LRU caching for frequently accessed data
- Memory profiling and optimization
- Goroutine management

**Test Requirements:**

- [ ] Performance tests: All target scenarios
- [ ] Benchmark tests: Before/after comparisons
- [ ] Memory tests: Leak detection
- [ ] Stress tests: Large vault operations

**Test Cases:**

- Worker pool scaling behavior
- Memory usage under load
- Ripgrep integration reliability
- Cache hit/miss ratios
- Parallel processing correctness
- Performance regression detection

---

### Task 5.2: Comprehensive Error Handling & Validation

**Effort:** 2 days | **Value:** High | **Risk:** Low

**Description:** Production-grade error handling with clear, actionable error messages.

**Acceptance Criteria:**

- [ ] 99.9% operation success rate target
- [ ] Clear, actionable error messages
- [ ] Input validation and sanitization
- [ ] Graceful degradation for non-critical failures
- [ ] Error recovery mechanisms
- [ ] Detailed error logging
- [ ] User-friendly error suggestions
- [ ] Consistent exit codes

**Implementation Notes:**

- Error wrapping with context
- Input validation functions
- Error categorization and handling
- User-friendly error formatting

**Test Requirements:**

- [ ] Unit tests: All error scenarios
- [ ] Integration tests: Error recovery
- [ ] Chaos testing: Random failure injection
- [ ] User experience tests: Error message clarity

**Test Cases:**

- All possible error conditions
- Error message formatting
- Recovery mechanisms
- Input validation edge cases
- File system error scenarios
- Network error handling

---

### Task 5.3: Security Implementation

**Effort:** 1-2 days | **Value:** High | **Risk:** Medium

**Description:** Security hardening with path sanitization and plugin sandboxing.

**Acceptance Criteria:**

- [ ] Path traversal prevention
- [ ] Input sanitization for all user inputs
- [ ] Plugin execution sandboxing
- [ ] Safe template execution
- [ ] API token security
- [ ] File permission validation
- [ ] No arbitrary code execution vulnerabilities

**Implementation Notes:**

- Path cleaning and validation
- Input sanitization libraries
- Template execution limits
- Secure plugin interfaces

**Test Requirements:**

- [ ] Security tests: All attack vectors
- [ ] Penetration tests: Path traversal, injection
- [ ] Code review: Security best practices
- [ ] Static analysis: Security scanners

**Test Cases:**

- Path traversal attempts
- Template injection attacks
- Plugin security boundaries
- Input validation bypass attempts
- File permission escalation
- API token exposure

---

### Task 5.4: Migration & Backward Compatibility

**Effort:** 1 day | **Value:** Medium | **Risk:** Low

**Description:** Smooth migration from current version with backward compatibility.

**Acceptance Criteria:**

- [ ] Legacy command deprecation warnings
- [ ] Config auto-upgrade from `obsidian-admin.yaml`
- [ ] Alias shims for removed commands
- [ ] Migration guide documentation
- [ ] Data format compatibility
- [ ] Rollback capability

**Implementation Notes:**

- Command mapping for deprecated commands
- Config migration scripts
- Version detection and upgrade
- Backward compatibility testing

**Test Requirements:**

- [ ] Migration tests: All upgrade scenarios
- [ ] Compatibility tests: Legacy command support
- [ ] Rollback tests: Downgrade scenarios
- [ ] Data integrity tests: Migration correctness

**Test Cases:**

- Config file migration
- Legacy command warnings
- Data format upgrades
- Rollback procedures
- Version detection

---

### Task 5.5: Documentation & Testing Finalization

**Effort:** 2 days | **Value:** High | **Risk:** Low

**Description:** Complete documentation, testing, and release preparation.

**Acceptance Criteria:**

- [ ] ≥90% unit test coverage
- [ ] ≥95% integration test coverage
- [ ] Comprehensive CLI help text
- [ ] User guide with examples
- [ ] API documentation
- [ ] Performance benchmarks
- [ ] Troubleshooting guide
- [ ] Release notes

**Implementation Notes:**

- Test coverage analysis
- Documentation generation
- Example collection
- Performance documentation

**Test Requirements:**

- [ ] Coverage tests: Meet target thresholds
- [ ] Documentation tests: Link validation
- [ ] Example tests: All examples work
- [ ] Performance tests: Benchmark stability

**Test Cases:**

- Test coverage verification
- Documentation completeness
- Example accuracy
- Performance baseline establishment
- Release readiness checklist

---

Below is **Phase 6** added to the PRD, containing a **Linkding “get”** feature to retrieve and print HTML snapshots (or fallback to live URL) for a given note’s `linkding_id`. Each task is scoped for a junior developer, with an emphasis on test‑driven development (unit and integration).

---

## 🏷️ Phase 6 – Linkding “get” Command

**Objective**
Add a new CLI command that, given a note with a `linkding_id` frontmatter attribute, will:

1. Query the Linkding API for any “snapshot” assets.
2. If found, download the latest snapshot (an HTML file), extract its text, print it to stdout, and clean up.
3. If no snapshots exist, fetch the live `url:` from frontmatter, retrieve the HTML (if under size threshold), strip tags, and print.

**Command**

```bash
mdnotes linkding get [path/to/note.md] [flags]
```

**Power Alias**
`mdnotes ld get`

**Flags**

- `--max-size <bytes>`: Maximum bytes to fetch from live URL (default: 1_000_000).
- `--timeout <duration>`: Request timeout (default: 10s).
- `--tmp-dir <path>`: Where to store downloaded asset (default: OS temp).
- Global flags: `--dry-run`, `--verbose`, `--quiet`.

---

### Task 6.1 – CLI Scaffolding & Argument Parsing

**Ticket**: `linkding/006-cli`
**Description**

- Create a new Cobra subcommand `linkding get` under `cmd/mdnotes/linkding/`.
- Accept a single positional argument: path to markdown file.
- Parse flags: `--max-size`, `--timeout`, `--tmp-dir`.
- Read frontmatter from file to extract `linkding_id` and fallback `url`.
- On missing `linkding_id` and `url`, show a clear error.

**Acceptance**

- Running `mdnotes ld get note.md --dry-run` prints parsed `linkding_id` and `url` without doing HTTP.
- Unit tests for flag parsing and frontmatter extraction (mock file).

---

### Task 6.2 – Linkding API Client Extensions

**Ticket**: `linkding/006-client`
**Description**

- In `pkg/linkding/client.go`, add methods:

  1. `ListAssets(bookmarkID int) ([]Asset, error)` → calls `GET /api/bookmarks/{id}/assets/`
  2. `DownloadAsset(bookmarkID, assetID int, destPath string) error` → calls `GET /…/download/` and writes file

- Use `context.WithTimeout` from the `--timeout` flag.
- Parse JSON into Go structs matching the API docs.

**Acceptance**

- Unit tests with an HTTP mock server returning the sample JSON (two assets).
- Verify that `ListAssets` returns only the `snapshot` items when filtered.
- Verify that `DownloadAsset` writes a file to disk with correct content stubbed by mock.

---

### Task 6.3 – Snapshot Selection Logic

**Ticket**: `linkding/006-select`
**Description**

- Given a slice of `Asset` from Task 6.2, implement `PickLatestSnapshot(assets []Asset) (*Asset, error)` that:

  - Filters on `asset_type == "snapshot"` and `status == "complete"`.
  - Chooses the one with the most recent `date_created`.
  - Returns `nil` / error if none found.

**Acceptance**

- Unit tests covering:

  - No snapshots → error
  - Multiple snapshots → picks latest
  - Snapshot with non‑complete status ignored

---

### Task 6.4 – HTML Text Extraction

**Ticket**: `linkding/006-extract`
**Description**

- Add utility `func ExtractTextFromHTML(path string) (string, error)` that:

  - Reads the HTML file.
  - Strips tags (e.g. via `golang.org/x/net/html` tokenizer).
  - Returns clean text.

- Ensure whitespace is collapsed sensibly.

**Acceptance**

- Unit tests reading small HTML fixtures (`<p>Hello <b>World</b></p>`) → `"Hello World"`.
- Handles large file without excessive memory (streaming).

---

### Task 6.5 – Live URL Fallback

**Ticket**: `linkding/006-fallback`
**Description**

- If no snapshot found:

  1. Read `url` from frontmatter.
  2. `http.Get` (with timeout) to fetch HTML.
  3. If the `Content-Length` header exceeds `--max-size`, abort with a message.
  4. Otherwise download into memory (or temp file) and `ExtractTextFromHTML`.

- Honor `--timeout` and `--max-size`.

**Acceptance**

- Integration test using an HTTP test server returning a small HTML page.
- Test that a large `Content-Length` triggers the size‑limit error.

---

### Task 6.6 – Orchestrator & Cleanup

**Ticket**: `linkding/006-orchestrator`
**Description**

- In the `linkding get` command handler, orchestrate:

  1. Call `ListAssets` → `PickLatestSnapshot`.
  2. If found: download to `--tmp-dir`, run `ExtractTextFromHTML`, delete file.
  3. Else: fall back to live URL per Task 6.5.
  4. Print result text to stdout.
  5. Clean up any temp files even on error.

- Respect `--verbose` for logging steps; respect `--quiet` to suppress non‑errors.

**Acceptance**

- Integration tests mocking both Linkding API and live server.
- End‑to‑end: stub asset HTML returns expected printed text.
- Verify temp file is removed.

---

### Task 6.7 – Unit & Integration Test Coverage

**Ticket**: `linkding/006-tests`
**Description**

- **Unit tests** for each module: client, selector, extractor, fallback.
- **Integration tests** under `test/integration/linkding_get/`:

  - Mock Linkding API server (e.g. `httptest.Server`) and a live URL server.
  - Test snapshot path and fallback path.

- Aim for **100% coverage** on new code.

**Acceptance**

- CI runs tests and shows coverage for `pkg/linkding/*` and the new command.
- No flaky tests; deterministic scenarios.

---

### Task 6.8 – Documentation & Examples

**Ticket**: `linkding/006-docs`
**Description**

- Add CLI reference entry under “Linkding Integration” in `docs/USER_GUIDE.md`.
- Show examples:

  ```bash
  # Print stored snapshot text
  mdnotes ld get note.md

  # Override timeout and temp dir
  mdnotes linkding get note.md --timeout 5s --tmp-dir ./tmp

  # Fallback to live page if no snapshot
  mdnotes ld get note.md --max-size 500000
  ```

- Document environment variables (API URL & token).

**Acceptance**

- Documentation builds without errors.
- Examples validated manually.

---

All tasks follow **TDD**: write failing tests first, implement code to satisfy tests, refactor, commit. Continuous integration must pass on each merge.

## 🧩 **Dependencies & Critical Path**

```
Week 1: Foundation
1.1 (Restructure) → 1.2 (Config) → 1.3 (CLI) → 1.4 (Templates)

Week 2: Frontmatter & Linkding
1.4 → 2.1 (Upsert) → 2.2 (Download)
1.2 → 2.3 (Sync) → 2.4 (Linkding)

Week 3: Analysis
2.1 → 3.1 (Scoring) → 3.2 (Content Analysis)
3.1 → 3.3 (Inbox) → 3.4 (Links/Health)

Week 4: Export & Watch
3.4 → 4.1 (Export) → 4.2 (Watch)
1.4 → 4.3 (Plugins)

Week 5: Production
4.1,4.2 → 5.1 (Performance) → 5.2 (Errors) → 5.3 (Security) → 5.4 (Migration) → 5.5 (Docs)
```

## 🎯 **Testing Strategy**

### Unit Testing Requirements (≥90% Coverage)

- All public functions tested
- Error scenarios covered
- Edge cases identified and tested
- Mock dependencies for external services
- Performance regression tests

### Integration Testing Requirements (≥95% Coverage)

- End-to-end command workflows
- Configuration integration
- File system operations
- Network operations (with mocks)
- Cross-platform compatibility

### Performance Testing Requirements

- Benchmark tests for all performance targets
- Memory usage validation
- Large vault stress testing
- Concurrent operation testing
- Performance regression detection

### Security Testing Requirements

- Input validation testing
- Path traversal prevention
- Template injection prevention
- Plugin security boundary testing
- API security testing

## 🚀 **Success Metrics**

- **Performance**: 90th-percentile CLI operations <1s
- **Reliability**: 99.9% operation success rate
- **Test Coverage**: ≥90% unit, ≥95% integration
- **Memory**: <200MB peak on 10k vault
- **CI/CD**: ≥95% green builds
- **Security**: Zero critical vulnerabilities

## 📝 **Risk Mitigation**

- **High-Risk Tasks**: Extra testing, code review, incremental implementation
- **Performance Risks**: Early benchmarking, profiling, optimization iteration
- **Integration Risks**: Mock testing, staged rollout, rollback plans
- **Security Risks**: Security review, penetration testing, static analysis

This plan provides **comprehensive coverage** of the PRD requirements while ensuring **high-quality implementation** suitable for **junior developer execution** with proper **testing and validation**.
