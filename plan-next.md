# ✅ `mdnotes` CLI – Full Implementation Plan (vNext)

### 🎯 Goal

Deliver a reliable and ergonomic command-line tool for managing Zettelkasten-style markdown notes in an Obsidian vault. Emphasis:

- **Graph & content integrity**
- **Frontmatter automation**
- **Note quality analysis**
- **Developer velocity**
- **CLI usability & safety**

---

## 📌 Prioritized Execution Cycle

This implementation plan is structured for one clear dev cycle. Tasks are **prioritized for immediate impact**, with intelligent defaults and UX-driven design.

---

## 🥇 PRIORITY 1 – CORE WORKFLOWS

---

### ✅ `rename` Command with Smart Link Updates

#### Ticket: `rename/001`

**Title**: Implement `rename` command with vault-wide reference rewrites

**Description**:
Add a `rename` command that renames a note and rewrites **all inbound references**:

- Supports Obsidian-style wikilinks, markdown links, and embeds
- Handles relative and vault-rooted paths
- Uses `ripgrep` or embedded fast scanner for performance
- Warns on fuzzy or ambiguous matches

**Flags**:
`--dry-run`, `--verbose`, `--fuzzy`, `--simulate`

**Acceptance Criteria**:

- Accurate bidirectional link update
- Runs in <500ms on vaults with 10k notes
- Preserves surrounding whitespace and syntax style

---

### ✅ `watch` Command (Automation System)

#### Ticket: `watch/001`

**Title**: Create `watch` system to trigger mdnotes actions on change

**Description**:
Monitor specified folders for markdown file changes and automatically run `mdnotes` commands on edit, create, or rename.

**Features**:

- YAML config: define path triggers and actions
- Debounced event dispatch (default 2s)
- Ignores `.git/`, `.obsidian/`, `node_modules/`, temp files
- Triggers pipelines like: `frontmatter ensure + linkding sync`

**CLI**:

```bash
mdnotes watch --config .obsidian-admin.yaml --daemon
```

---

### ✅ `analyze content` (Content Quality Scoring)

#### Ticket: `analyze/001`

**Title**: Add `analyze content` for note complexity and clarity analysis

**Description**:
Add a scoring system to evaluate content health per Zettelkasten principles.

**Scoring Criteria** (0.0–1.0 scale):

1. **Readability** – Flesch–Kincaid Reading Ease
2. **Link Density** – outbound links per 100 words
3. **Completeness** – title, summary, word count
4. **Atomicity Heuristic** – flags if >1 heading or >500 words
5. **Recency Decay** – old untouched notes penalized

**Output**:
Table/JSON/CSV with:

- Total score
- Per-factor breakdown
- Suggested fixes

**CLI**:

```bash
mdnotes analyze content notes/ --format table --min-score 0.65
```

---

### ✅ `analyze inbox` (Zettelkasten Triage)

#### Ticket: `analyze/002`

**Title**: Add inbox triage system to identify rough/unfinished notes

**Description**:
Analyze files tagged or titled `inbox` or in an `inbox/` folder to identify and triage raw, unprocessed notes.

**Heuristics**:

- File has heading or title “Inbox”
- No tags / created date / summary
- Word count < 20 or > 200 (both suspicious)

**Output**:

- Table sorted by severity
- Snippet previews
- Action suggestions: move, tag, summarize, delete

**CLI**:

```bash
mdnotes analyze inbox --show-content --length 10
```

---

## 🥈 PRIORITY 2 – FRONTMATTER ENHANCEMENTS

---

### ✅ Frontmatter `set` Command

#### Ticket: `fm/005`

**Title**: Add command to set frontmatter key(s) on one or more files

**Description**:

- Assign one or more frontmatter fields
- Support `--value "{{today}}"` templates

**CLI**:

```bash
mdnotes fm set notes/*.md --field created --value "{{today}}"
```

**Supports**: YAML and JSON

---

### ✅ Frontmatter `download` Attribute

#### Ticket: `fm/004`

**Title**: Enable auto-downloading of content when `download: true`

**Description**:
If `download: true` is in frontmatter and `url:` exists, auto-fetch external page metadata.

**Extracted Fields**:

- `title`, `source`, `downloaded_at`, `content_length`

**Triggers**:

- On `frontmatter ensure`
- Or via `--download` flag

---

## 🥉 PRIORITY 3 – USABILITY, QUERYING & POWER ALIASES

---

### ✅ Query Aliases and Fixes

#### Ticket: `query/002`

**Title**: Add ergonomic aliases and query filters to frontmatter query

**Features**:

- `q` alias for `frontmatter query`
- `--missing`, `--duplicates`, `--fix-with "{{today}}"`

**Examples**:

```bash
mdnotes q --missing created --fix-with "{{today}}"
mdnotes q --duplicates title
```

---

### ✅ CLI Usability + Aliases

#### Ticket: `cli/001`

**Title**: Add smart aliases and help text polish for CLI

**Examples**:

```bash
mdnotes e [file]        # frontmatter ensure
mdnotes s [file]        # frontmatter set
mdnotes q               # query
mdnotes a c             # analyze content
mdnotes a i             # analyze inbox
mdnotes r note.md new.md
```

**Also**:

- Show updated file count and summaries
- Consistent `--format` support

---

## 🔧 FINAL PHASE – CODEBASE REFACTOR & QA

---

### ✅ Refactor & Polish

#### Ticket: `core/099`

**Title**: Final code cleanup and readiness for release

**Checklist**:

- Reorganize into packages: `cmd/`, `core/`, `analyze/`, `watch/`
- Use `golangci-lint` with default + custom rules
- Ensure >90% test coverage
- Add CONTRIBUTING.md and examples
- Profile performance on simulated vaults

---

## 🧭 Developer Handoff Checklist

| Area                | ✅ Status |
| ------------------- | --------- |
| Rename & Links      | ✅        |
| Analysis Engine     | ✅        |
| Inbox Triage        | ✅        |
| Watcher             | ✅        |
| Frontmatter Control | ✅        |
| CLI Polish & Docs   | ✅        |
| Tests & Cleanup     | ✅        |

---

## 🚀 Final CLI Overview

```bash
mdnotes e file.md                 # frontmatter ensure
mdnotes s file.md                # frontmatter set
mdnotes q                        # query frontmatter
mdnotes a c                      # analyze content
mdnotes a i                      # analyze inbox
mdnotes r old.md new.md          # rename and update links
mdnotes watch                    # file watcher for automation
mdnotes ld sync                  # sync from Linkding
```

---

## ✅ Success Metrics

| Metric                           | Target               | Status |
| -------------------------------- | -------------------- | ------ |
| Rename w/ full link update       | <500ms on 10k vaults | ✅     |
| Content analysis usable at scale | ✅                   | ✅     |
| Watcher runs without leak/crash  | ✅                   | ✅     |
| CLI help complete & intuitive    | ✅                   | ✅     |
| Full test + lint pass            | >90% & green CI      | ✅     |

---

Let me know if you want this turned into GitHub Issues (labels + milestones), a Notion board, or Linear backlog.
