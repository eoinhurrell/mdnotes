# Obsidian Admin CLI - TDD Implementation Plan

**Role**: Principal Engineer  
**Methodology**: Test-Driven Development with Jujutsu VCS  
**Duration**: 12 weeks (6 two-week cycles)

---

## Engineering Principles

1. **Red-Green-Refactor**: Write failing tests → Implement minimal code → Refactor
2. **Incremental Delivery**: Each cycle produces working, tested features
3. **Clean Architecture**: Separate concerns, dependency injection, testable design
4. **Commit Hygiene**: Logical commits with jj, squashing related changes
5. **Continuous Integration**: Every commit passes all tests

---

## Project Setup and Tooling

### Initial Repository Structure

```bash
# Initialize repository with jujutsu
jj git init --colocate

# Create initial structure
mkdir -p {cmd,internal/{vault,processor,linkding,config},pkg/{markdown,template},test/fixtures}

# Initial go.mod
go mod init github.com/eoinhurrell/mdnotes
```

### Development Tools

```bash
# Install development dependencies
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install gotest.tools/gotestsum@latest
go install github.com/vektra/mockery/v2@latest

# .golangci.yml for code quality
# Makefile for common tasks
# .github/workflows/ci.yml for CI
```

### Jujutsu Workflow

```bash
# Create feature branch
jj new -m "feat: initial project structure"

# After each logical unit of work
jj squash  # Combine related changes
jj split   # Split unrelated changes
jj describe -m "test: add frontmatter parser test cases"
```

---

## Cycle 1: Foundation (Weeks 1-2)

### Goal

Establish core abstractions, file handling, and basic frontmatter parsing.

### Day 1-2: Core Interfaces and File Handling

#### Tests First

```go
// internal/vault/file_test.go
func TestVaultFile_Load(t *testing.T) {
    tests := []struct {
        name    string
        content string
        want    *VaultFile
        wantErr bool
    }{
        {
            name: "valid markdown with frontmatter",
            content: `---
title: Test Note
tags: [test, example]
---

# Test Note

Content here.`,
            want: &VaultFile{
                Frontmatter: map[string]interface{}{
                    "title": "Test Note",
                    "tags":  []interface{}{"test", "example"},
                },
                Body: "# Test Note\n\nContent here.",
            },
        },
        {
            name: "markdown without frontmatter",
            content: "# Just Content\n\nNo frontmatter here.",
            want: &VaultFile{
                Frontmatter: map[string]interface{}{},
                Body:        "# Just Content\n\nNo frontmatter here.",
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            vf := &VaultFile{}
            err := vf.Parse([]byte(tt.content))
            if (err != nil) != tt.wantErr {
                t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
            }
            if !reflect.DeepEqual(vf.Frontmatter, tt.want.Frontmatter) {
                t.Errorf("Frontmatter = %v, want %v", vf.Frontmatter, tt.want.Frontmatter)
            }
        })
    }
}
```

#### Implementation

```go
// internal/vault/file.go
package vault

import (
    "bytes"
    "gopkg.in/yaml.v3"
)

type VaultFile struct {
    Path         string
    RelativePath string
    Content      []byte
    Frontmatter  map[string]interface{}
    Body         string
    Modified     time.Time
}

func (vf *VaultFile) Parse(content []byte) error {
    vf.Content = content

    // Check for frontmatter
    if bytes.HasPrefix(content, []byte("---\n")) {
        parts := bytes.SplitN(content, []byte("---\n"), 3)
        if len(parts) >= 3 {
            // Parse YAML frontmatter
            if err := yaml.Unmarshal(parts[1], &vf.Frontmatter); err != nil {
                return fmt.Errorf("parsing frontmatter: %w", err)
            }
            vf.Body = string(parts[2])
        } else {
            vf.Body = string(content)
        }
    } else {
        vf.Frontmatter = make(map[string]interface{})
        vf.Body = string(content)
    }

    return nil
}
```

#### Jujutsu Commits

```bash
jj new -m "test: add VaultFile parsing tests"
# Add test file
jj squash

jj new -m "feat: implement VaultFile with frontmatter parsing"
# Add implementation
jj squash
```

### Day 3-4: Vault Scanner

#### Tests First

```go
// internal/vault/scanner_test.go
func TestScanner_Walk(t *testing.T) {
    // Create test fixture
    tmpDir := t.TempDir()
    createTestVault(t, tmpDir)

    scanner := NewScanner()
    files, err := scanner.Walk(tmpDir)

    require.NoError(t, err)
    assert.Len(t, files, 3)
    assert.Contains(t, files[0].RelativePath, ".md")
}

func TestScanner_WithIgnorePatterns(t *testing.T) {
    scanner := NewScanner(
        WithIgnorePatterns([]string{".obsidian/*", "*.tmp"}),
    )

    // Test that ignored files are skipped
}
```

#### Implementation

```go
// internal/vault/scanner.go
type Scanner struct {
    ignorePatterns []string
    fileSystem     FileSystem // Interface for testing
}

type ScannerOption func(*Scanner)

func WithIgnorePatterns(patterns []string) ScannerOption {
    return func(s *Scanner) {
        s.ignorePatterns = patterns
    }
}

func (s *Scanner) Walk(root string) ([]*VaultFile, error) {
    var files []*VaultFile

    err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        if s.shouldIgnore(path) {
            if info.IsDir() {
                return filepath.SkipDir
            }
            return nil
        }

        if strings.HasSuffix(path, ".md") {
            vf, err := s.loadFile(path, root)
            if err != nil {
                return fmt.Errorf("loading %s: %w", path, err)
            }
            files = append(files, vf)
        }

        return nil
    })

    return files, err
}
```

### Day 5-6: Frontmatter Operations Interface

#### Tests First

```go
// internal/processor/frontmatter_test.go
func TestFrontmatterProcessor_Ensure(t *testing.T) {
    tests := []struct {
        name     string
        file     *vault.VaultFile
        field    string
        defValue interface{}
        want     interface{}
        modified bool
    }{
        {
            name: "add missing field",
            file: &vault.VaultFile{
                Frontmatter: map[string]interface{}{
                    "title": "Test",
                },
            },
            field:    "tags",
            defValue: []string{},
            want:     []string{},
            modified: true,
        },
        {
            name: "preserve existing field",
            file: &vault.VaultFile{
                Frontmatter: map[string]interface{}{
                    "tags": []string{"existing"},
                },
            },
            field:    "tags",
            defValue: []string{},
            want:     []string{"existing"},
            modified: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            p := NewFrontmatterProcessor()
            modified := p.Ensure(tt.file, tt.field, tt.defValue)

            assert.Equal(t, tt.modified, modified)
            assert.Equal(t, tt.want, tt.file.Frontmatter[tt.field])
        })
    }
}
```

#### Implementation

```go
// internal/processor/frontmatter.go
type FrontmatterProcessor struct {
    preserveOrder bool
    templateEngine *template.Engine
}

func (p *FrontmatterProcessor) Ensure(file *vault.VaultFile, field string, defaultValue interface{}) bool {
    if _, exists := file.Frontmatter[field]; !exists {
        // Process template if string
        if strVal, ok := defaultValue.(string); ok {
            defaultValue = p.processTemplate(strVal, file)
        }

        file.Frontmatter[field] = defaultValue
        return true
    }
    return false
}
```

### Day 7-8: CLI Structure and First Command

#### Tests First

```go
// cmd/frontmatter_test.go
func TestFrontmatterEnsureCommand(t *testing.T) {
    // Create test vault
    tmpDir := t.TempDir()
    testFile := filepath.Join(tmpDir, "test.md")
    os.WriteFile(testFile, []byte("# Test\n\nContent"), 0644)

    // Execute command
    cmd := NewRootCommand()
    cmd.SetArgs([]string{
        "frontmatter", "ensure",
        "--field", "tags",
        "--default", "[]",
        testFile,
    })

    err := cmd.Execute()
    assert.NoError(t, err)

    // Verify file was updated
    content, _ := os.ReadFile(testFile)
    assert.Contains(t, string(content), "tags: []")
}
```

#### Implementation

```go
// cmd/frontmatter.go
var frontmatterCmd = &cobra.Command{
    Use:   "frontmatter",
    Short: "Manage frontmatter in markdown files",
}

var ensureCmd = &cobra.Command{
    Use:   "ensure [path]",
    Short: "Ensure frontmatter fields exist",
    RunE:  runEnsure,
}

func init() {
    ensureCmd.Flags().StringSlice("field", nil, "Field to ensure")
    ensureCmd.Flags().StringSlice("default", nil, "Default value")
    ensureCmd.MarkFlagRequired("field")

    frontmatterCmd.AddCommand(ensureCmd)
}

func runEnsure(cmd *cobra.Command, args []string) error {
    // Implementation using scanner and processor
}
```

### Day 9-10: Integration and Refactoring

#### Integration Tests

```go
// test/integration/frontmatter_test.go
func TestFrontmatterEnsureIntegration(t *testing.T) {
    // Test with real file system
    // Test with multiple files
    // Test error scenarios
}
```

### Jujutsu Workflow for Cycle 1

```bash
# Daily commits
jj new -m "test: add vault scanner test suite"
jj new -m "feat: implement vault scanner with ignore patterns"
jj new -m "test: add frontmatter processor tests"
jj new -m "feat: implement frontmatter ensure operation"
jj new -m "test: add CLI integration tests"
jj new -m "feat: wire up frontmatter ensure command"
jj new -m "refactor: extract common test fixtures"
jj new -m "docs: add initial README and usage examples"

# Before merging
jj log  # Review commit history
jj squash --from <first-commit> --to <last-commit> -m "feat: implement core vault operations and frontmatter ensure"
```

---

## Cycle 2: Frontmatter Features (Weeks 3-4)

### Goal

Complete frontmatter operations: validate, sync, and type casting.

### Day 1-2: Frontmatter Validation

#### Tests First

```go
// internal/processor/validator_test.go
func TestFrontmatterValidator_Validate(t *testing.T) {
    tests := []struct {
        name     string
        rules    ValidationRules
        file     *vault.VaultFile
        wantErrs []ValidationError
    }{
        {
            name: "missing required field",
            rules: ValidationRules{
                Required: []string{"title", "tags"},
            },
            file: &vault.VaultFile{
                Path: "test.md",
                Frontmatter: map[string]interface{}{
                    "title": "Test",
                },
            },
            wantErrs: []ValidationError{
                {Field: "tags", Type: "missing_required"},
            },
        },
        {
            name: "invalid type",
            rules: ValidationRules{
                Types: map[string]string{
                    "tags": "array",
                },
            },
            file: &vault.VaultFile{
                Frontmatter: map[string]interface{}{
                    "tags": "not-an-array",
                },
            },
            wantErrs: []ValidationError{
                {Field: "tags", Type: "invalid_type", Expected: "array"},
            },
        },
    }
}
```

#### Implementation

```go
// internal/processor/validator.go
type ValidationRules struct {
    Required []string
    Types    map[string]string
    Schema   *Schema
}

type Validator struct {
    rules ValidationRules
}

func (v *Validator) Validate(file *vault.VaultFile) []ValidationError {
    var errors []ValidationError

    // Check required fields
    for _, field := range v.rules.Required {
        if _, exists := file.Frontmatter[field]; !exists {
            errors = append(errors, ValidationError{
                Field: field,
                Type:  "missing_required",
                File:  file.Path,
            })
        }
    }

    // Validate types
    for field, expectedType := range v.rules.Types {
        if !v.validateType(file.Frontmatter[field], expectedType) {
            errors = append(errors, ValidationError{
                Field:    field,
                Type:     "invalid_type",
                Expected: expectedType,
                File:     file.Path,
            })
        }
    }

    return errors
}
```

### Day 3-4: Type Casting System

#### Tests First

```go
// internal/processor/typecast_test.go
func TestTypeCaster_Cast(t *testing.T) {
    tests := []struct {
        name      string
        value     interface{}
        toType    string
        want      interface{}
        wantErr   bool
    }{
        {
            name:   "string to date",
            value:  "2023-01-01",
            toType: "date",
            want:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
        },
        {
            name:   "string to number",
            value:  "42",
            toType: "number",
            want:   42,
        },
        {
            name:   "string to boolean",
            value:  "true",
            toType: "boolean",
            want:   true,
        },
        {
            name:   "comma string to array",
            value:  "tag1, tag2, tag3",
            toType: "array",
            want:   []string{"tag1", "tag2", "tag3"},
        },
        {
            name:    "invalid date format",
            value:   "not-a-date",
            toType:  "date",
            wantErr: true,
        },
    }
}

func TestTypeCaster_AutoDetect(t *testing.T) {
    tests := []struct {
        value    interface{}
        wantType string
    }{
        {"2023-01-01", "date"},
        {"42", "number"},
        {"true", "boolean"},
        {"tag1, tag2", "array"},
        {"just text", "string"},
    }
}
```

#### Implementation

```go
// internal/processor/typecast.go
type TypeCaster struct {
    validators map[string]TypeValidator
}

func NewTypeCaster() *TypeCaster {
    return &TypeCaster{
        validators: map[string]TypeValidator{
            "date":    &DateValidator{},
            "number":  &NumberValidator{},
            "boolean": &BooleanValidator{},
            "array":   &ArrayValidator{},
        },
    }
}

func (tc *TypeCaster) Cast(value interface{}, toType string) (interface{}, error) {
    // Handle already correct type
    if tc.isType(value, toType) {
        return value, nil
    }

    // Convert from string
    strVal, ok := value.(string)
    if !ok {
        return nil, fmt.Errorf("cannot cast non-string value")
    }

    validator, exists := tc.validators[toType]
    if !exists {
        return nil, fmt.Errorf("unknown type: %s", toType)
    }

    return validator.Cast(strVal)
}

func (tc *TypeCaster) AutoDetect(value interface{}) string {
    strVal, ok := value.(string)
    if !ok {
        return tc.getType(value)
    }

    // Try each validator in order of specificity
    order := []string{"date", "number", "boolean", "array"}
    for _, typeName := range order {
        if tc.validators[typeName].Matches(strVal) {
            return typeName
        }
    }

    return "string"
}
```

### Day 5-6: Frontmatter Sync Implementation

#### Tests First

```go
// internal/processor/sync_test.go
func TestFrontmatterSync_SyncField(t *testing.T) {
    now := time.Now()

    tests := []struct {
        name     string
        field    string
        source   string
        file     *vault.VaultFile
        want     interface{}
    }{
        {
            name:   "sync from file modification time",
            field:  "modified",
            source: "file-mtime",
            file: &vault.VaultFile{
                Modified: now,
            },
            want: now.Format("2006-01-02"),
        },
        {
            name:   "sync from filename",
            field:  "id",
            source: "filename",
            file: &vault.VaultFile{
                Path: "/vault/20230101-test-note.md",
            },
            want: "20230101-test-note",
        },
    }
}
```

### Day 7-8: Template Engine

#### Tests First

```go
// pkg/template/engine_test.go
func TestTemplateEngine_Process(t *testing.T) {
    engine := NewEngine()
    ctx := Context{
        "filename": "test-note",
        "current_date": "2023-01-01",
    }

    tests := []struct {
        template string
        want     string
    }{
        {"{{current_date}}", "2023-01-01"},
        {"{{filename|upper}}", "TEST-NOTE"},
        {"{{uuid}}", "<valid-uuid>"},
    }
}
```

### Day 9-10: CLI Commands Integration

```go
// cmd/frontmatter_validate.go
var validateCmd = &cobra.Command{
    Use:   "validate [path]",
    Short: "Validate frontmatter against rules",
    RunE:  runValidate,
}

// cmd/frontmatter_cast.go
var castCmd = &cobra.Command{
    Use:   "cast [path]",
    Short: "Cast frontmatter fields to proper types",
    RunE:  runCast,
}
```

### Jujutsu Workflow for Cycle 2

```bash
jj new -m "test: add frontmatter validation test cases"
jj new -m "feat: implement frontmatter validator with rules"
jj new -m "test: add comprehensive type casting tests"
jj new -m "feat: implement type casting with auto-detection"
jj new -m "test: add frontmatter sync tests"
jj new -m "feat: implement field synchronization"
jj new -m "test: add template engine tests"
jj new -m "feat: implement template processing"
jj new -m "feat: wire up validate, cast, and sync commands"
jj new -m "docs: add frontmatter command documentation"
```

---

## Cycle 3: Content Operations (Weeks 5-6)

### Goal

Implement heading management, link parsing, and file organization.

### Day 1-3: Heading Analysis and Fixing

#### Tests First

```go
// internal/processor/heading_test.go
func TestHeadingProcessor_Analyze(t *testing.T) {
    tests := []struct {
        name    string
        content string
        want    HeadingAnalysis
    }{
        {
            name: "multiple H1s",
            content: `# First Title
Some content
# Second Title`,
            want: HeadingAnalysis{
                Issues: []HeadingIssue{
                    {Type: "multiple_h1", Line: 3},
                },
            },
        },
        {
            name: "H1 doesn't match title",
            content: `---
title: Expected Title
---
# Different Title`,
            want: HeadingAnalysis{
                Issues: []HeadingIssue{
                    {Type: "h1_title_mismatch", Expected: "Expected Title"},
                },
            },
        },
    }
}

func TestHeadingProcessor_Fix(t *testing.T) {
    tests := []struct {
        name    string
        file    *vault.VaultFile
        rules   HeadingRules
        want    string
    }{
        {
            name: "ensure H1 from title",
            file: &vault.VaultFile{
                Frontmatter: map[string]interface{}{
                    "title": "My Note",
                },
                Body: "Some content without heading",
            },
            rules: HeadingRules{
                EnsureH1Title: true,
            },
            want: "# My Note\n\nSome content without heading",
        },
    }
}
```

#### Implementation

```go
// internal/processor/heading.go
type HeadingProcessor struct {
    parser markdown.Parser
}

func (p *HeadingProcessor) Analyze(file *vault.VaultFile) HeadingAnalysis {
    headings := p.parser.ExtractHeadings(file.Body)
    analysis := HeadingAnalysis{}

    // Check for multiple H1s
    h1Count := 0
    for _, h := range headings {
        if h.Level == 1 {
            h1Count++
            if h1Count > 1 {
                analysis.Issues = append(analysis.Issues, HeadingIssue{
                    Type: "multiple_h1",
                    Line: h.Line,
                })
            }
        }
    }

    // Check H1 matches title
    if title, ok := file.Frontmatter["title"].(string); ok && h1Count > 0 {
        if headings[0].Text != title {
            analysis.Issues = append(analysis.Issues, HeadingIssue{
                Type:     "h1_title_mismatch",
                Expected: title,
                Actual:   headings[0].Text,
            })
        }
    }

    return analysis
}

func (p *HeadingProcessor) Fix(file *vault.VaultFile, rules HeadingRules) error {
    if rules.EnsureH1Title {
        if title, ok := file.Frontmatter["title"].(string); ok {
            file.Body = p.ensureH1(file.Body, title)
        }
    }

    if rules.SingleH1 {
        file.Body = p.convertExtraH1s(file.Body)
    }

    return nil
}
```

### Day 4-6: Link Parsing and Management

#### Tests First

```go
// internal/processor/links_test.go
func TestLinkParser_Extract(t *testing.T) {
    tests := []struct {
        name    string
        content string
        want    []Link
    }{
        {
            name:    "wiki links",
            content: "See [[other note]] and [[folder/note|custom text]]",
            want: []Link{
                {Type: WikiLink, Target: "other note", Text: "other note"},
                {Type: WikiLink, Target: "folder/note", Text: "custom text"},
            },
        },
        {
            name:    "markdown links",
            content: "See [text](note.md) and [](empty.md)",
            want: []Link{
                {Type: MarkdownLink, Target: "note.md", Text: "text"},
                {Type: MarkdownLink, Target: "empty.md", Text: ""},
            },
        },
        {
            name:    "embedded links",
            content: "![[image.png]] and ![[note.md]]",
            want: []Link{
                {Type: EmbedLink, Target: "image.png"},
                {Type: EmbedLink, Target: "note.md"},
            },
        },
    }
}

func TestLinkConverter_Convert(t *testing.T) {
    tests := []struct {
        name     string
        link     Link
        toFormat LinkFormat
        want     string
    }{
        {
            name:     "wiki to markdown",
            link:     Link{Type: WikiLink, Target: "note", Text: "note"},
            toFormat: MarkdownFormat,
            want:     "[note](note.md)",
        },
        {
            name:     "wiki with alias to markdown",
            link:     Link{Type: WikiLink, Target: "note", Text: "custom"},
            toFormat: MarkdownFormat,
            want:     "[custom](note.md)",
        },
    }
}
```

#### Implementation

```go
// internal/processor/links.go
type LinkParser struct {
    patterns map[LinkType]*regexp.Regexp
}

func NewLinkParser() *LinkParser {
    return &LinkParser{
        patterns: map[LinkType]*regexp.Regexp{
            WikiLink:     regexp.MustCompile(`\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`),
            MarkdownLink: regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`),
            EmbedLink:    regexp.MustCompile(`!\[\[([^\]]+)\]\]`),
        },
    }
}

func (p *LinkParser) Extract(content string) []Link {
    var links []Link

    for linkType, pattern := range p.patterns {
        matches := pattern.FindAllStringSubmatch(content, -1)
        for _, match := range matches {
            link := Link{Type: linkType}

            switch linkType {
            case WikiLink:
                link.Target = match[1]
                link.Text = match[2]
                if link.Text == "" {
                    link.Text = link.Target
                }
            case MarkdownLink:
                link.Text = match[1]
                link.Target = match[2]
            case EmbedLink:
                link.Target = match[1]
            }

            links = append(links, link)
        }
    }

    return links
}

type LinkConverter struct {
    resolver PathResolver
}

func (c *LinkConverter) Convert(content string, from, to LinkFormat) string {
    parser := NewLinkParser()
    links := parser.Extract(content)

    // Sort by position (reverse) to replace from end
    sort.Slice(links, func(i, j int) bool {
        return links[i].Position.Start > links[j].Position.Start
    })

    result := content
    for _, link := range links {
        if link.Type.Format() == from {
            newLink := c.formatLink(link, to)
            result = replaceAtPosition(result, link.Position, newLink)
        }
    }

    return result
}
```

### Day 7-8: File Organization

#### Tests First

```go
// internal/processor/organizer_test.go
func TestOrganizer_GenerateFilename(t *testing.T) {
    tests := []struct {
        name     string
        pattern  string
        file     *vault.VaultFile
        want     string
    }{
        {
            name:    "simple field replacement",
            pattern: "{{id}}.md",
            file: &vault.VaultFile{
                Frontmatter: map[string]interface{}{
                    "id": "12345",
                },
            },
            want: "12345.md",
        },
        {
            name:    "date formatting",
            pattern: "{{created|date:2006-01-02}}-{{title|slug}}.md",
            file: &vault.VaultFile{
                Frontmatter: map[string]interface{}{
                    "created": "2023-01-15",
                    "title":   "My Test Note!",
                },
            },
            want: "2023-01-15-my-test-note.md",
        },
    }
}
```

### Day 9-10: Link Update Tracking

#### Tests First

```go
// internal/processor/link_updater_test.go
func TestLinkUpdater_UpdateReferences(t *testing.T) {
    updater := NewLinkUpdater()

    // Simulate file move
    move := FileMove{
        From: "old/path/note.md",
        To:   "new/location/note.md",
    }

    tests := []struct {
        name    string
        content string
        want    string
    }{
        {
            name:    "update wiki link",
            content: "See [[old/path/note]]",
            want:    "See [[new/location/note]]",
        },
        {
            name:    "update markdown link",
            content: "See [text](old/path/note.md)",
            want:    "See [text](new/location/note.md)",
        },
        {
            name:    "update embed",
            content: "![[old/path/note]]",
            want:    "![[new/location/note]]",
        },
    }
}
```

### Jujutsu Workflow for Cycle 3

```bash
jj new -m "test: add heading analysis and fixing tests"
jj new -m "feat: implement heading processor with rules"
jj new -m "test: add comprehensive link parsing tests"
jj new -m "feat: implement link parser for all formats"
jj new -m "test: add link conversion tests"
jj new -m "feat: implement bidirectional link converter"
jj new -m "test: add file organization tests"
jj new -m "feat: implement filename generation from patterns"
jj new -m "test: add link update tracking tests"
jj new -m "feat: implement reference updater for moved files"
```

---

## Cycle 4: External Integration (Weeks 7-8)

### Goal

Implement Linkding integration and batch operations.

### Day 1-3: Linkding API Client

#### Tests First

```go
// internal/linkding/client_test.go
func TestClient_CreateBookmark(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "POST", r.Method)
        assert.Equal(t, "/api/bookmarks/", r.URL.Path)
        assert.Equal(t, "Token test-token", r.Header.Get("Authorization"))

        var req CreateBookmarkRequest
        json.NewDecoder(r.Body).Decode(&req)

        resp := BookmarkResponse{
            ID:    123,
            URL:   req.URL,
            Title: req.Title,
        }
        json.NewEncoder(w).Encode(resp)
    }))
    defer server.Close()

    client := NewClient(server.URL, "test-token")
    bookmark, err := client.CreateBookmark(context.Background(), CreateBookmarkRequest{
        URL:   "https://example.com",
        Title: "Example",
    })

    assert.NoError(t, err)
    assert.Equal(t, 123, bookmark.ID)
}

func TestClient_RateLimiting(t *testing.T) {
    // Test that client respects rate limits
    client := NewClient("", "", WithRateLimit(2)) // 2 req/sec

    start := time.Now()
    for i := 0; i < 4; i++ {
        client.rateLimiter.Wait(context.Background())
    }
    elapsed := time.Since(start)

    assert.Greater(t, elapsed, 1500*time.Millisecond)
}
```

#### Implementation

```go
// internal/linkding/client.go
type Client struct {
    baseURL     string
    apiToken    string
    httpClient  *http.Client
    rateLimiter *rate.Limiter
}

type ClientOption func(*Client)

func WithRateLimit(reqPerSec int) ClientOption {
    return func(c *Client) {
        c.rateLimiter = rate.NewLimiter(rate.Limit(reqPerSec), 1)
    }
}

func (c *Client) CreateBookmark(ctx context.Context, req CreateBookmarkRequest) (*BookmarkResponse, error) {
    // Wait for rate limit
    if err := c.rateLimiter.Wait(ctx); err != nil {
        return nil, err
    }

    body, err := json.Marshal(req)
    if err != nil {
        return nil, err
    }

    httpReq, err := http.NewRequestWithContext(ctx, "POST",
        c.baseURL+"/api/bookmarks/", bytes.NewReader(body))
    if err != nil {
        return nil, err
    }

    httpReq.Header.Set("Authorization", "Token "+c.apiToken)
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusTooManyRequests {
        // Handle rate limiting with exponential backoff
        return nil, ErrRateLimited
    }

    if resp.StatusCode != http.StatusCreated {
        return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }

    var bookmark BookmarkResponse
    if err := json.NewDecoder(resp.Body).Decode(&bookmark); err != nil {
        return nil, err
    }

    return &bookmark, nil
}
```

### Day 4-6: Linkding Sync Processor

#### Tests First

```go
// internal/processor/linkding_sync_test.go
func TestLinkdingSync_FindUnsyncedFiles(t *testing.T) {
    files := []*vault.VaultFile{
        {
            Path: "synced.md",
            Frontmatter: map[string]interface{}{
                "url":         "https://example.com",
                "linkding_id": 123,
            },
        },
        {
            Path: "unsynced.md",
            Frontmatter: map[string]interface{}{
                "url": "https://example2.com",
            },
        },
        {
            Path: "no-url.md",
            Frontmatter: map[string]interface{}{
                "title": "No URL here",
            },
        },
    }

    sync := NewLinkdingSync(LinkdingSyncConfig{
        URLField: "url",
        IDField:  "linkding_id",
    })

    unsynced := sync.FindUnsyncedFiles(files)
    assert.Len(t, unsynced, 1)
    assert.Equal(t, "unsynced.md", unsynced[0].Path)
}

func TestLinkdingSync_SyncFile(t *testing.T) {
    mockClient := &MockLinkdingClient{}
    mockClient.On("CreateBookmark", mock.Anything, mock.Anything).
        Return(&BookmarkResponse{ID: 456}, nil)

    sync := NewLinkdingSync(LinkdingSyncConfig{
        URLField:    "url",
        IDField:     "linkding_id",
        SyncTitle:   true,
        SyncTags:    true,
    })
    sync.client = mockClient

    file := &vault.VaultFile{
        Frontmatter: map[string]interface{}{
            "url":   "https://example.com",
            "title": "Example Article",
            "tags":  []string{"tech", "go"},
        },
    }

    err := sync.SyncFile(context.Background(), file)
    assert.NoError(t, err)
    assert.Equal(t, 456, file.Frontmatter["linkding_id"])
}
```

### Day 7-8: Batch Operations Framework

#### Tests First

```go
// internal/processor/batch_test.go
func TestBatchProcessor_Execute(t *testing.T) {
    // Create test batch config
    config := BatchConfig{
        Operations: []Operation{
            {
                Name:    "Ensure tags",
                Command: "frontmatter.ensure",
                Parameters: map[string]interface{}{
                    "field":   "tags",
                    "default": []string{},
                },
            },
            {
                Name:    "Fix headings",
                Command: "headings.fix",
                Parameters: map[string]interface{}{
                    "ensure_h1_title": true,
                },
            },
        },
    }

    processor := NewBatchProcessor()
    results, err := processor.Execute(context.Background(), testVault, config)

    assert.NoError(t, err)
    assert.Len(t, results, 2)
    assert.True(t, results[0].Success)
}
```

#### Implementation

```go
// internal/processor/batch.go
type BatchProcessor struct {
    registry map[string]Processor
    logger   Logger
}

func (b *BatchProcessor) Execute(ctx context.Context, vault *Vault, config BatchConfig) ([]OperationResult, error) {
    var results []OperationResult

    // Begin transaction (backup current state)
    backup, err := b.createBackup(vault)
    if err != nil {
        return nil, fmt.Errorf("creating backup: %w", err)
    }

    for i, op := range config.Operations {
        select {
        case <-ctx.Done():
            // Rollback on cancellation
            b.rollback(vault, backup)
            return results, ctx.Err()
        default:
        }

        result := b.executeOperation(vault, op)
        results = append(results, result)

        if !result.Success && config.StopOnError {
            b.rollback(vault, backup)
            return results, fmt.Errorf("operation %d failed: %w", i, result.Error)
        }
    }

    return results, nil
}
```

### Day 9-10: Progress Reporting

#### Implementation

```go
// internal/processor/progress.go
type ProgressReporter interface {
    Start(total int)
    Update(current int, message string)
    Finish()
}

type TerminalProgress struct {
    bar *progressbar.ProgressBar
}

func (t *TerminalProgress) Start(total int) {
    t.bar = progressbar.NewOptions(total,
        progressbar.OptionSetDescription("Processing files..."),
        progressbar.OptionShowCount(),
        progressbar.OptionShowIts(),
        progressbar.OptionSetPredictTime(true),
    )
}
```

### Jujutsu Workflow for Cycle 4

```bash
jj new -m "test: add Linkding API client tests"
jj new -m "feat: implement Linkding client with rate limiting"
jj new -m "test: add Linkding sync processor tests"
jj new -m "feat: implement file synchronization with Linkding"
jj new -m "test: add batch operations framework tests"
jj new -m "feat: implement transactional batch processor"
jj new -m "feat: add progress reporting for long operations"
jj new -m "docs: add Linkding integration guide"
```

---

## Cycle 5: Analysis and Safety (Weeks 9-10)

### Goal

Implement analysis commands, safety features, and configuration system.

### Day 1-3: Vault Analysis

#### Tests First

```go
// internal/analyzer/stats_test.go
func TestAnalyzer_GenerateStats(t *testing.T) {
    vault := createTestVault(t)
    analyzer := NewAnalyzer()

    stats := analyzer.GenerateStats(vault)

    assert.Equal(t, 10, stats.TotalFiles)
    assert.Equal(t, 5, stats.FilesWithFrontmatter)
    assert.Contains(t, stats.TagDistribution, "project")
    assert.Greater(t, stats.TotalLinks, 0)
}

func TestAnalyzer_FindDuplicates(t *testing.T) {
    analyzer := NewAnalyzer()

    files := []*vault.VaultFile{
        {Path: "a.md", Frontmatter: map[string]interface{}{"title": "Same Title"}},
        {Path: "b.md", Frontmatter: map[string]interface{}{"title": "Same Title"}},
        {Path: "c.md", Frontmatter: map[string]interface{}{"title": "Different"}},
    }

    duplicates := analyzer.FindDuplicates(files, "title")
    assert.Len(t, duplicates, 1)
    assert.Len(t, duplicates[0].Files, 2)
}
```

### Day 4-6: Safety Features

#### Tests First

```go
// internal/safety/backup_test.go
func TestBackupManager_Create(t *testing.T) {
    tmpDir := t.TempDir()
    manager := NewBackupManager(tmpDir)

    // Create test file
    testFile := filepath.Join(tmpDir, "test.md")
    content := []byte("# Original Content")
    os.WriteFile(testFile, content, 0644)

    // Create backup
    backupID, err := manager.CreateBackup(testFile)
    assert.NoError(t, err)
    assert.NotEmpty(t, backupID)

    // Modify original
    os.WriteFile(testFile, []byte("# Modified"), 0644)

    // Restore
    err = manager.Restore(backupID, testFile)
    assert.NoError(t, err)

    // Verify restored content
    restored, _ := os.ReadFile(testFile)
    assert.Equal(t, content, restored)
}

func TestDryRun_Operations(t *testing.T) {
    dryRun := NewDryRunRecorder()

    // Record operations without executing
    dryRun.Record(Operation{
        Type: "frontmatter.ensure",
        File: "test.md",
        Changes: []Change{
            {Field: "tags", NewValue: []string{}},
        },
    })

    // Generate report
    report := dryRun.GenerateReport()
    assert.Contains(t, report, "Would add field 'tags'")
}
```

### Day 7-8: Configuration System

#### Tests First

```go
// internal/config/config_test.go
func TestConfig_Load(t *testing.T) {
    configYAML := `
version: "1.0"
vault:
  ignore_patterns:
    - "*.tmp"
    - ".obsidian/*"
frontmatter:
  required_fields: ["id", "title"]
  type_rules:
    fields:
      created: date
      tags: array
linkding:
  api_url: "${LINKDING_URL}"
  api_token: "${LINKDING_TOKEN}"
`

    // Set env vars
    os.Setenv("LINKDING_URL", "https://linkding.example.com")
    os.Setenv("LINKDING_TOKEN", "secret-token")

    cfg, err := LoadConfig(strings.NewReader(configYAML))
    assert.NoError(t, err)
    assert.Equal(t, "https://linkding.example.com", cfg.Linkding.APIURL)
    assert.Contains(t, cfg.Vault.IgnorePatterns, "*.tmp")
}

func TestConfig_Validate(t *testing.T) {
    cfg := &Config{
        Frontmatter: FrontmatterConfig{
            TypeRules: TypeRules{
                Fields: map[string]string{
                    "date": "invalid-type",
                },
            },
        },
    }

    err := cfg.Validate()
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "invalid type")
}
```

### Day 9-10: Final Integration

#### Integration Tests

```go
// test/integration/full_workflow_test.go
func TestFullWorkflow(t *testing.T) {
    // Create realistic test vault
    vault := createRealisticVault(t)

    // Run full workflow
    app := NewApp()

    // 1. Analyze initial state
    stats := app.Analyze(vault)
    t.Logf("Initial stats: %+v", stats)

    // 2. Fix issues
    err := app.RunBatch(vault, "fix-all-issues.yaml")
    assert.NoError(t, err)

    // 3. Verify fixes
    validation := app.Validate(vault)
    assert.Empty(t, validation.Errors)
}
```

### Jujutsu Workflow for Cycle 5

```bash
jj new -m "test: add vault analyzer tests"
jj new -m "feat: implement statistics and duplicate detection"
jj new -m "test: add backup manager tests"
jj new -m "feat: implement backup and restore functionality"
jj new -m "test: add dry-run recorder tests"
jj new -m "feat: implement dry-run mode with detailed reporting"
jj new -m "test: add configuration loading tests"
jj new -m "feat: implement config system with env var support"
jj new -m "test: add full integration test suite"
jj new -m "docs: add safety features documentation"
```

---

## Cycle 6: Polish and Release (Weeks 11-12)

### Goal

Performance optimization, documentation, and release preparation.

### Day 1-3: Performance Optimization

#### Benchmarks

```go
// internal/processor/benchmark_test.go
func BenchmarkFrontmatterEnsure(b *testing.B) {
    vault := generateLargeVault(b, 10000) // 10k files
    processor := NewFrontmatterProcessor()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        for _, file := range vault.Files {
            processor.Ensure(file, "tags", []string{})
        }
    }
}

func BenchmarkParallelProcessing(b *testing.B) {
    vault := generateLargeVault(b, 10000)

    b.Run("Sequential", func(b *testing.B) {
        processSequential(vault)
    })

    b.Run("Parallel", func(b *testing.B) {
        processParallel(vault, runtime.NumCPU())
    })
}
```

#### Memory Profiling

```go
// cmd/profile.go
var profileCmd = &cobra.Command{
    Use:    "profile",
    Hidden: true,
    RunE: func(cmd *cobra.Command, args []string) error {
        f, _ := os.Create("mem.prof")
        defer f.Close()

        runtime.GC()
        pprof.WriteHeapProfile(f)

        return nil
    },
}
```

### Day 4-5: Error Messages and UX

#### Implementation

```go
// internal/errors/user_friendly.go
type UserError struct {
    Op         string
    Path       string
    Err        error
    Suggestion string
}

func (e UserError) Error() string {
    var buf strings.Builder

    fmt.Fprintf(&buf, "Error: %s\n", e.Err)

    if e.Path != "" {
        fmt.Fprintf(&buf, "File: %s\n", e.Path)
    }

    if e.Suggestion != "" {
        fmt.Fprintf(&buf, "\nSuggestion: %s\n", e.Suggestion)
    }

    return buf.String()
}

// Example usage
return UserError{
    Op:   "frontmatter.cast",
    Path: file.Path,
    Err:  fmt.Errorf("cannot cast 'not-a-date' to date"),
    Suggestion: "Date must be in YYYY-MM-DD format. " +
                "Use --on-error skip to ignore invalid values.",
}
```

### Day 6-7: Documentation

#### User Guide

````markdown
# Obsidian Admin User Guide

## Quick Start

1. Install the tool:
   ```bash
   go install github.com/eoinhurrell/mdnotes@latest
   ```
````

2. Navigate to your vault:

   ```bash
   cd /path/to/vault
   ```

3. Ensure all notes have tags:

   ```bash
   mdnotes frontmatter ensure --field tags --default "[]" .
   ```

## Common Workflows

### Daily Maintenance

...

### Bulk Import

...

````

### Day 8-9: Release Preparation

#### Goreleaser Configuration
```yaml
# .goreleaser.yaml
before:
  hooks:
    - go mod tidy
    - go test ./...

builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64

archives:
  - format: tar.gz
    format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: 'checksums.txt'

snapshot:
  name_template: "{{ incpatch .Version }}-next"

changelog:
  sort: asc
  filters:
    exclude:
      - '^docs:'
      - '^test:'
````

### Day 10: Final Testing and Release

#### Release Checklist

```bash
# Run all tests
make test

# Run benchmarks
make bench

# Build all platforms
goreleaser build --snapshot --rm-dist

# Test on each platform
./test-all-platforms.sh

# Create release
git tag v1.0.0
goreleaser release
```

### Jujutsu Workflow for Cycle 6

```bash
jj new -m "perf: add benchmarks for core operations"
jj new -m "perf: implement parallel processing for large vaults"
jj new -m "feat: improve error messages with suggestions"
jj new -m "docs: add comprehensive user guide"
jj new -m "docs: add API documentation"
jj new -m "build: add goreleaser configuration"
jj new -m "test: add cross-platform integration tests"
jj new -m "chore: prepare v1.0.0 release"

# Final squash for release
jj squash --from <start> --to @ -m "Release v1.0.0"
```

---

## Testing Strategy Summary

### Unit Test Coverage Goals

- Core packages: 90%+ coverage
- Command handlers: 80%+ coverage
- Utilities: 85%+ coverage

### Test Organization

```
test/
├── unit/           # Package-level tests
├── integration/    # Cross-package tests
├── e2e/           # End-to-end command tests
├── fixtures/      # Test data
└── benchmarks/    # Performance tests
```

### Continuous Integration

```yaml
# .github/workflows/ci.yml
name: CI

on: [push, pull_request]

jobs:
  test:
    strategy:
      matrix:
        go: [1.21, 1.22]
        os: [ubuntu-latest, macos-latest, windows-latest]

    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: ${{ matrix.go }}

      - name: Test
        run: |
          go test -race -coverprofile=coverage.out ./...
          go tool cover -html=coverage.out -o coverage.html

      - name: Lint
        run: golangci-lint run
```

---

## Key Engineering Decisions

1. **Interface-Driven Design**: Every major component has an interface for testing
2. **Functional Options**: Use options pattern for configurable components
3. **Context Propagation**: Pass context.Context for cancellation support
4. **Error Wrapping**: Use fmt.Errorf with %w for error chains
5. **Structured Logging**: Use structured logging for debugging
6. **Feature Flags**: Use build tags for optional features

## Success Metrics

- All tests passing on every commit
- No regression in performance benchmarks
- Clean golangci-lint output
- Documentation for every public API
- Examples for common use cases
