# 📄 Product Requirements Document

## Revamp of **mdnotes** CLI (vNext)

---

## 1. Vision & Objectives

**Vision**
Create a rock‑solid, high‑performance CLI for managing, analyzing, and publishing Obsidian‑style Zettelkasten vaults—empowering researchers, writers, and power users to automate metadata, evaluate content quality, and extract or share focused subgraphs with ease.

**Objectives**

1. **Streamline UX**: Intuitive commands, consistent aliases, and sensible defaults
2. **Unified Configuration**: Single YAML config that covers vault, ignore rules, templates, integrations
3. **Elevated Performance**: Ripgrep‑powered indexing, parallel worker pools, in‑memory caching
4. **Deep Zettelkasten Awareness**: Atomicity & content‑quality scoring, subgraph exports, backlink‑driven queries
5. **Seamless Linkding Sync**: Robust two‑way bookmark integration—list, sync, status tracking
6. **Modular & Extensible**: Clean package boundaries, plugin hooks, clear extension APIs

---

## 2. Target Users & Needs

| Persona              | Needs & Goals                                                                          |
| -------------------- | -------------------------------------------------------------------------------------- |
| **Researcher**       | Extract thematic clusters, triage raw notes, monitor note quality over time            |
| **Writer / Blogger** | Export clean subgraphs for publication, rewrite external links to URLs, include assets |
| **Power User**       | Craft complex queries, automate pipelines with watch mode, customize via plugins       |
| **Developer**        | Well‑structured codebase, easy testing & profiling, clear extension points             |

---

## 3. Key Changes & Detailed Descriptions

### 3.1 CLI Usability & Structure

- **Domain‑Grouped Commands**

  - `fm` (frontmatter), `analyze`, `links`, `export`, `watch`, `rename`, `linkding`

- **Power Aliases**

  - `mdnotes u` → `fm upsert`
  - `mdnotes q` → `fm query`
  - `mdnotes a c` → `analyze content`
  - `mdnotes a i` → `analyze inbox`
  - `mdnotes r` → `rename`
  - `mdnotes x` → `export`
  - `mdnotes ld sync` → `linkding sync`

- **Flag Hygiene & Smart Defaults**

  - Global flags: `--dry-run`, `--verbose`, `--quiet`, `--config`
  - Default `--recursive=true` on file‑targeting commands
  - Built‑in ignore patterns: `.obsidian/*`, `*.tmp`, `.git/*`
  - Implicit vault root detection (walk upward until vault marker)

---

### 3.2 Unified Configuration

- **Single YAML config** (`mdnotes.yaml`), loaded in order: CWD → user home → `/etc`
- Sections:

  ```yaml
  vault:
    path: "."
    ignore_patterns: [".obsidian/*", "*.tmp"]
  frontmatter:
    # upsert defaults, type rules, download settings
  linkding:
    api_url: "https://…"
    api_token: "…"
    sync_title: true
    sync_tags: true
    auto_download_favicons: true
  export:
    default_strategy: "remove"
    include_assets: true
  watch:
    debounce: "2s"
    rules: […]
  performance:
    max_workers: 0 # 0 = auto-detect
  plugins:
    enabled: true
    paths: ["~/.mdnotes/plugins"]
  ```

---

### 3.3 Frontmatter & Linkding Integration

1. **`fm upsert`** (new; power‑alias `u`)

   - **Replaces**: `fm ensure` + `fm set`
   - **Usage**:

     ```bash
     mdnotes u [path] \
       --field name --default <value> \
       [--overwrite]  # only overwrite existing if this flag present
     ```

   - **Behavior**:

     - If `<field>` **absent**, set to `<value>`.
     - If `<field>` **present** and `--overwrite` **not** given, leave as is.
     - If `--overwrite` given, replace existing with `<value>`.

   - **Templates**: same engine as `rename` & `export` (supporting `{{current_date}}`, `{{filename|slug}}`, etc.)

2. **`fm download`**

   - Scan frontmatter for any HTTP(S) URLs
   - Download to `attachments/` (configurable) with subfolders for date
   - Add fields:

     - `url_original` (string)
     - `url_local` (relative path)
     - `downloaded_at` (ISO datetime)

3. **`fm sync`**

   - Map file metadata sources to frontmatter:

     - `--source file-mtime`
     - `--source filename`
     - `--source path:dir`

   - Can extract via regex from filename (e.g. dates)

4. **`linkding sync`** & **`linkding list`**

   - **Sync**:

     1. Query vault for notes with `url:` frontmatter
     2. POST new bookmarks, PATCH existing (idempotent)
     3. Write back `linkding_id`, update tags/title if configured
     4. Retry on 429 with exponential backoff

   - **List**:

     - Show notes with `url`, status (`synced #123`, `unsynced`, `error`)
     - Output in table or JSON

---

### 3.4 Zettelkasten‑Aware Analysis

1. **Content Scoring** (`analyze content`)

   - **Factors** (each weighted 0.2):

     1. **Readability** (Flesch–Kincaid Reading Ease, normalized to 0–1)
     2. **Link Density** = (outbound links ÷ word count), ideal 0.02–0.04
     3. **Completeness**:

        - +0.2 if H1 matches `title`
        - +0.2 if ≥100 words
        - +0.2 if summary paragraph detected

     4. **Atomicity Heuristic**:

        - −0.2 if >1 `<h2>` heading
        - −0.2 if >500 words

     5. **Recency Decay** = max(0, 1 − age_days/365)

   - **Score** = (sum of factors) × 100 → 0–100 scale
   - **Output**:

     - Summary: worst N notes
     - Per-note breakdown: factor scores + suggestions (“Split note”, “Add links”, “Shorten”, etc.)

2. **Inbox Triage** (`analyze inbox`)

   - Detect files:

     - Containing top-level `# INBOX`
     - Missing required frontmatter (`created`, `tags`)
     - Word count anomalies (<20 or >200)

   - **Output**:

     | File           | Issues                          | Snippet    | Suggested Action |
     | -------------- | ------------------------------- | ---------- | ---------------- |
     | `inbox/foo.md` | missing tags, short (<20 words) | “Today I…” | Add tags; expand |

3. **Link Graph & Health**

   - `analyze links`: graph statistics, orphan detection, hub scores
   - `analyze health`: composite index from frontmatter coverage, link integrity, content score

---

### 3.5 Subgraph Exports & Publishing

- **Command**: `mdnotes x [output] --query "<expr>" [flags]`
- **Features**:

  - **Filtering** via full `fm query` syntax
  - `--with-backlinks`: include inbound neighbors
  - **Link‑rewrite strategies**:

    - `remove`: strip unexported links, leave plain text
    - `url`: replace with frontmatter `url:` if present, else remove
    - `stub` (future): generate minimal stub note

  - `--include-assets`: copy images/media under `assets/`
  - `--slugify`, `--flatten` filename options
  - Parallel export, in‑memory link adjustments

- **Output**: mirror directory structure (or flat), plus `export-metadata.yaml` with query and stats

---

### 3.6 Automation & Watch Mode

- **Command**: `mdnotes watch [--config mdnotes.yaml] [--daemon]`
- **YAML Rules**:

  ```yaml
  watch:
    debounce: "2s"
    ignore: [".obsidian/*", "*.tmp"]
    rules:
      - name: "Inbox processing"
        paths: ["inbox/**/*.md"]
        events: ["create", "write"]
        actions:
          - "mdnotes u {{file}} --field created --default '{{current_date}}'"
          - "mdnotes ld sync {{file}}"
      - name: "Daily cleanup"
        cron: "0 2 * * *"
        actions:
          - "mdnotes analyze health"
          - 'mdnotes export daily-archive --query "created = ''{{current_date}}''"'
  ```

- **Features**: file events, optional cron rules, templated commands, dry‑run preview, integrated logging

---

### 3.7 Plugin System

- **Hook Points**:

  - Pre‑command, per‑file, post‑command, export‑complete

- **Discovery**: `~/.mdnotes/plugins/*.so` or Go modules declared in config
- **API**: register new commands, flags, query predicates, template funcs

---

## 4. Architecture & Package Layout

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

---

## 5. Non‑Functional Requirements

| Aspect            | Target                                  |
| ----------------- | --------------------------------------- |
| **Performance**   | <500 ms for 10 k‑note rename/export     |
| **Memory**        | <200 MB peak on 10 k vault              |
| **Reliability**   | 99.9% success; clear, actionable errors |
| **Test Coverage** | ≥ 90% unit + integration                |
| **Security**      | Sanitize paths; sandbox plugins         |

---

## 6. Migration & Backward Compatibility

- **Flag Deprecation**: warn on use of removed commands (`fm ensure`, `fm set`); suggest `fm upsert`.
- **Config Auto‑Upgrade**: import legacy `obsidian-admin.yaml` into new schema.
- **Alias Shims**: legacy aliases map to new commands until v2.0.

---

## 7. Roadmap & Milestones

| Week | Deliverables                                          |
| ---- | ----------------------------------------------------- |
| 1    | Code restructuring, config loader, CLI skeleton       |
| 2    | `fm upsert` + template engine + Linkding sync/list    |
| 3    | Analysis engine & content‑scoring implementation      |
| 4    | Export subgraph with link strategies & asset handling |
| 5    | Watch mode, plugin hooks, comprehensive docs & tests  |

---

## 8. Success Metrics

- **Continuous Integration**: ≥ 95% green CI on tests & benchmarks
- **Performance**: 90th‑percentile CLI ops < 1 s
- **Adoption**: ↑ GitHub stars, forks, community plugins
- **UX Feedback**: Positive user survey results on CLI ergonomics

---

## 📝 Change Summary

1. **Combine** `fm ensure` & `fm set` → **`fm upsert`** (alias `u`) with optional `--overwrite`
2. **Unified config** (`mdnotes.yaml`) covering vault, frontmatter, linkding, export, watch, performance, plugins
3. **Template engine** centralized for upsert, rename, export, watch
4. **Deep Linkding integration**: sync & list commands, two‑way idempotent sync, retry/backoff
5. **Zettelkasten scoring**: five weighted factors → 0–100 content score + actionable suggestions
6. **Subgraph export**: query & backlinks, three link‑rewrite strategies, asset copying, slug/flatten
7. **Watch automation**: YAML‑driven rules, debounce, cron support, templated actions
8. **Plugin framework**: hook points, discovery, extension API
9. **Performance improvements**: Ripgrep back‑end, worker pool, in‑memory index

---

_Ready for development hand‑off._
