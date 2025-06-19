# ✅ `mdnotes rename` – Full Implementation Plan

---

## 🚀 Summary

> Rename a Markdown file in an Obsidian vault, and update all **vault-relative references** (wiki links and markdown links) to it, safely and efficiently.

---

## 📁 Vault Model & Assumptions

- A vault is a directory (`--vault`) containing `.md` files.
- All link references are **relative to vault root**.
- We **only** update files **within the same vault**.
- Obsidian-style links:

  - `[[filename]]`
  - `[[filename#Header]]`
  - `[[filename|Alias]]`
  - Markdown: `[text](filename.md)` or `[text](../folder/filename.md)`

---

## ⚙️ Functional Flow

### Step 1: Parse CLI Inputs

- **Inputs**:

  - `source` (positional)
  - `target` (optional)
  - Flags: `--vault`, `--template`, `--ignore`
  - Global flags: `--dry-run`, `--verbose`, `--quiet`, `--config`

- **Validation**:

  - `source` must exist and be `.md` file
  - Vault must be directory (default: current directory)
  - `source` must be inside vault
  - Template must be valid (if provided)
  - Target must not exist (prevent overwrites)

---

### Step 2: Determine New Filename

If `target` is:

- **Given explicitly** → Use as-is
- **Missing** → Expand template

#### Template Data

```go
type TemplateData struct {
    Filename     string
    CreatedTime  time.Time
    Year         string
    Month        string
    Day          string
}
```

#### Template Example

```gotemplate
{{.CreatedTime|YYYYMMDDHHmmss}}-{{.Filename|slugify}}.md
```

**Default template**: `\"{{.CreatedTime|YYYYMMDDHHmmss}}-{{.Filename|slugify}}.md\"`

#### Template Engine Integration

Uses the existing mdnotes template engine from `pkg/template/`:

- Supports same variables as frontmatter commands
- Includes `slugify` filter for filename normalization
- Consistent with project's template system

#### Created Time Detection

1. Check `created` field in frontmatter
2. Fall back to file modification time
3. Use existing frontmatter parsing from `internal/vault/`

---

### Step 3: Vault Scanning

#### Walk Vault

Use existing `internal/vault.Scanner`:

- Reuses project's vault scanning logic
- Supports ignore patterns from config (`.obsidian/*`, `*.tmp`)
- Consistent with other commands' file discovery
- Returns `VaultFile` objects with parsed frontmatter

---

### Step 4: Parse and Match Links

#### Link Types to Match

- `[[note]]`
- `[[note#Header]]`
- `[[note|Alias]]`
- `[text](note.md)`
- `[text](../folder/note.md)`

#### Regex Matchers

```go
Wiki:      \[\[([^\]\|#]+)(#[^\|\]]*)?(?:\|([^\]]+))?\]\]
Markdown:  \[([^\]]+)\]\(([^)]+\.md)\)
```

#### Normalization

- Strip `.md`
- Compare basename of path
- Everything is vault-relative

---

### Step 5: Update Links

#### Match Logic

```go
func MatchesLink(linkTarget string, oldRelPath string) bool {
    return stripExt(filepath.Base(linkTarget)) == stripExt(filepath.Base(oldRelPath))
}
```

#### Rewrite Strategy

- **Wiki link**: replace `[[old-name]]` → `[[new-name]]`, preserving alias and header.
- **Markdown**: update `[text](old.md)` → `[text](new.md)` maintaining relative path.
- If source/target in different directories, recompute `relativePath := filepath.Rel(linkingFileDir, newPath)`

---

### Step 6: Write Changes

If file needs change:

- Use atomic file operations (consistent with project safety patterns)
- Leverage existing `VaultFile.Write()` method
- No backup flag needed (atomic operations provide safety)

---

### Step 7: Rename File

```go
func RenameFile(oldPath, newPath string) error {
    os.MkdirAll(filepath.Dir(newPath), 0755)
    return os.Rename(oldPath, newPath)
}
```

- Must happen _after_ all link edits are written
- Error if target exists (no `--force` flag in this project)
- Create target directories as needed

---

### Step 8: Dry-Run Support

If `--dry-run`:

- Perform all computation
- Use consistent "Would rename:" output format
- Show planned changes in same style as other mdnotes commands
- Skip rename and write operations

---

### Step 9: Summary Output

Print final report consistent with project style:

```
Renamed: old.md → new.md
Updated 17 links in 11 files
```

In verbose mode, show each file being examined:
```
Examining: file1.md - updated 2 links
Examining: file2.md - no changes needed
```

---

## 🧪 Testing Strategy

- ✅ Rename file with wiki links, aliases, markdown links
- ✅ Dry-run mode shows correct diff
- ✅ Ignores files via pattern
- ✅ Link updates for nested folders
- ✅ Invalid input (missing file, existing target)
- ✅ No-op if no links found
- ✅ Regression: Obsidian-style links don’t break

---

## ⚡ Performance Notes

- Parallel file scanning (e.g. goroutine pool w/ semaphore)
- Avoid reading files twice (scan + rewrite in one pass)
- Link cache? Optional, unnecessary unless vault > 10k notes

---

## 🔗 Integration with mdnotes Architecture

### Reuse Existing Components

- **VaultFile**: Use existing frontmatter parsing and file operations
- **Scanner**: Leverage vault scanning with ignore patterns
- **Template Engine**: Reuse `pkg/template/` for filename generation
- **Link Parser**: Extend existing link parsing from `internal/processor/`
- **Safety Patterns**: Follow atomic file operations used elsewhere

### Consistent Flag Patterns

- `--ignore`: Same as other commands (default: `[".obsidian/*", "*.tmp"]`)
- `--vault`: Specify vault root (default: `"."`)
- `--template`: Use project's template syntax with filters
- Global flags: `--dry-run`, `--verbose`, `--quiet`, `--config`

### Error Handling

- Follow project's error wrapping patterns with `fmt.Errorf` and `%w`
- Non-halting: Report link update errors but continue processing
- Clear, actionable error messages consistent with other commands

## 📌 Optional Enhancements (Post-v1)

- `--move`: rename + move across folders  
- Support `aliases` from frontmatter for additional link matching
- Integration with `linkding` sync when URLs are involved
- Bulk rename operations with pattern matching

---

Ready to code? Implementation should follow existing mdnotes patterns and reuse established components.
