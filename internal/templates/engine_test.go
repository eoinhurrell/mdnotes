package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEngine(t *testing.T) {
	engine := NewEngine()

	assert.NotNil(t, engine)
	assert.NotNil(t, engine.cache)
	assert.NotNil(t, engine.funcs)

	// Check that default functions are registered
	assert.Contains(t, engine.funcs, "current_date")
	assert.Contains(t, engine.funcs, "current_datetime")
	assert.Contains(t, engine.funcs, "uuid")
	assert.Contains(t, engine.funcs, "upper")
	assert.Contains(t, engine.funcs, "lower")
	assert.Contains(t, engine.funcs, "slug")
	assert.Contains(t, engine.funcs, "date")
}

func TestEngine_Process_SimpleVariable(t *testing.T) {
	engine := NewEngine()

	ctx := &Context{
		Filename: "test-file",
		Title:    "Test Title",
	}

	result, err := engine.Process("{{.filename}}", ctx)

	require.NoError(t, err)
	assert.Equal(t, "test-file", result)
}

func TestEngine_Process_CurrentDate(t *testing.T) {
	engine := NewEngine()

	result, err := engine.Process("{{.current_date}}", nil)

	require.NoError(t, err)
	assert.Equal(t, time.Now().Format("2006-01-02"), result)
}

func TestEngine_Process_CurrentDateTime(t *testing.T) {
	engine := NewEngine()

	result, err := engine.Process("{{.current_datetime}}", nil)

	require.NoError(t, err)

	// Check that it's a valid datetime format
	_, err = time.Parse("2006-01-02T15:04:05Z", result)
	assert.NoError(t, err)
}

func TestEngine_Process_UUID(t *testing.T) {
	engine := NewEngine()

	result, err := engine.Process("{{.uuid}}", nil)

	require.NoError(t, err)
	assert.Len(t, result, 36) // UUID format: 8-4-4-4-12
	assert.Contains(t, result, "-")
}

func TestEngine_Process_StringFilters(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		template string
		ctx      *Context
		expected string
	}{
		{
			name:     "upper filter",
			template: "{{.filename | upper}}",
			ctx:      &Context{Filename: "hello"},
			expected: "HELLO",
		},
		{
			name:     "lower filter",
			template: "{{.filename | lower}}",
			ctx:      &Context{Filename: "HELLO"},
			expected: "hello",
		},
		{
			name:     "slug filter",
			template: "{{.filename | slug}}",
			ctx:      &Context{Filename: "Hello World! Test"},
			expected: "hello-world-test",
		},
		{
			name:     "chained filters",
			template: "{{.title | lower | slug}}",
			ctx:      &Context{Title: "My Great Title"},
			expected: "my-great-title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Process(tt.template, tt.ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEngine_Process_DateFilter(t *testing.T) {
	engine := NewEngine()

	testTime := time.Date(2023, 12, 25, 14, 30, 0, 0, time.UTC)

	ctx := &Context{
		FileModTime: testTime,
	}

	result, err := engine.Process("{{.file_mtime | date \"2006-01-02\"}}", ctx)

	require.NoError(t, err)
	assert.Equal(t, "2023-12-25", result)
}

func TestEngine_Process_ComplexTemplate(t *testing.T) {
	engine := NewEngine()

	ctx := &Context{
		Filename: "test-note",
		Title:    "My Test Note",
		Variables: map[string]interface{}{
			"tags": []string{"test", "example"},
		},
	}

	template := "{{.current_date}}-{{.filename | slug}}.md"
	result, err := engine.Process(template, ctx)

	require.NoError(t, err)

	today := time.Now().Format("2006-01-02")
	expected := today + "-test-note.md"
	assert.Equal(t, expected, result)
}

func TestEngine_Process_TemplateCache(t *testing.T) {
	engine := NewEngine()

	ctx := &Context{Filename: "test"}
	template := "{{.filename}}"

	// First call - should compile and cache
	result1, err1 := engine.Process(template, ctx)
	require.NoError(t, err1)
	assert.Equal(t, "test", result1)

	// Verify template is cached
	assert.Len(t, engine.cache, 1)

	// Second call - should use cache
	result2, err2 := engine.Process(template, ctx)
	require.NoError(t, err2)
	assert.Equal(t, "test", result2)

	// Cache should still have one entry
	assert.Len(t, engine.cache, 1)
}

func TestEngine_Process_InvalidTemplate(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		template string
		errorMsg string
	}{
		{
			name:     "unmatched braces",
			template: "{{.filename",
			errorMsg: "unmatched braces",
		},
		{
			name:     "dangerous call directive",
			template: "{{call .SomeFunction}}",
			errorMsg: "potentially unsafe directive",
		},
		{
			name:     "dangerous range directive",
			template: "{{range .Items}}{{end}}",
			errorMsg: "potentially unsafe directive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := engine.Process(tt.template, nil)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorMsg)
		})
	}
}

func TestEngine_Slugify(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"Test_File.Name", "test-file-name"},
		{"My Great Title!", "my-great-title"},
		{"Multiple   Spaces", "multiple-spaces"},
		{"Special@#$%Characters", "specialcharacters"},
		{"--Leading-And-Trailing--", "leading-and-trailing"},
		{"", ""},
		{"123-numbers-456", "123-numbers-456"},
		{"CamelCaseText", "camelcasetext"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := engine.slugify(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEngine_ProcessFile(t *testing.T) {
	// Create a temporary file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test-note.md")

	err := os.WriteFile(testFile, []byte("# Test Content"), 0644)
	require.NoError(t, err)

	engine := NewEngine()

	result, err := engine.ProcessFile("{{.current_date}}-{{.filename | slug}}", testFile)

	require.NoError(t, err)

	today := time.Now().Format("2006-01-02")
	expected := today + "-test-note"
	assert.Equal(t, expected, result)
}

func TestEngine_ProcessFile_NonexistentFile(t *testing.T) {
	engine := NewEngine()

	// Should still work, just without file modification time
	result, err := engine.ProcessFile("{{.filename}}", "/nonexistent/file.md")

	require.NoError(t, err)
	assert.Equal(t, "file", result)
}

func TestEngine_RegisterFunction(t *testing.T) {
	engine := NewEngine()

	// Register custom function
	engine.RegisterFunction("reverse", func(s string) string {
		runes := []rune(s)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return string(runes)
	})

	ctx := &Context{Filename: "hello"}
	result, err := engine.Process("{{.filename | reverse}}", ctx)

	require.NoError(t, err)
	assert.Equal(t, "olleh", result)
}

func TestEngine_ClearCache(t *testing.T) {
	engine := NewEngine()

	// Process a template to populate cache
	_, err := engine.Process("{{.filename}}", &Context{Filename: "test"})
	require.NoError(t, err)
	assert.Len(t, engine.cache, 1)

	// Clear cache
	engine.ClearCache()
	assert.Len(t, engine.cache, 0)
}

func TestIsTemplate(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"{{.filename}}", true},
		{"plain text", false},
		{"{{.current_date}}-note.md", true},
		{"note-{{.title | slug}}", true},
		{"{single brace}", false},
		{"", false},
		{"{{incomplete", false},
		{"incomplete}}", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := IsTemplate(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractVariables(t *testing.T) {
	tests := []struct {
		name     string
		template string
		expected []string
	}{
		{
			name:     "single variable",
			template: "{{.filename}}",
			expected: []string{"filename"},
		},
		{
			name:     "multiple variables",
			template: "{{.current_date}}-{{.filename | slug}}",
			expected: []string{"current_date", "filename"},
		},
		{
			name:     "duplicate variables",
			template: "{{.filename}}-{{.filename | upper}}",
			expected: []string{"filename"}, // Should deduplicate
		},
		{
			name:     "no variables",
			template: "plain text",
			expected: []string{},
		},
		{
			name:     "variables with filters",
			template: "{{.title | lower | slug}}",
			expected: []string{"title"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractVariables(tt.template)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestEngine_ValidateVariables(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name        string
		template    string
		ctx         *Context
		expectError bool
	}{
		{
			name:        "all variables available",
			template:    "{{.filename}}-{{.title}}",
			ctx:         &Context{Filename: "test", Title: "Test"},
			expectError: false,
		},
		{
			name:        "missing variable",
			template:    "{{.filename}}-{{.missing_var}}",
			ctx:         &Context{Filename: "test"},
			expectError: true,
		},
		{
			name:        "built-in variables always available",
			template:    "{{.current_date}}-{{.uuid}}",
			ctx:         nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.ValidateVariables(tt.template, tt.ctx)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "undefined variables")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEngine_Process_SecurityValidation(t *testing.T) {
	engine := NewEngine()

	// Test various potentially dangerous template constructs
	dangerousTemplates := []string{
		"{{call .os.Exit}}",
		"{{range .Files}}{{.Delete}}{{end}}",
		"{{with .Config}}{{.Destroy}}{{end}}",
		"{{block \"evil\" .}}danger{{end}}",
		"{{define \"hack\"}}evil{{end}}",
		"{{template \"external\" .}}",
	}

	for _, dangerous := range dangerousTemplates {
		t.Run(dangerous, func(t *testing.T) {
			_, err := engine.Process(dangerous, nil)
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "unsafe")
		})
	}
}

// Benchmark tests
func BenchmarkEngine_Process(b *testing.B) {
	engine := NewEngine()
	ctx := &Context{
		Filename: "test-file",
		Title:    "Test Title",
	}
	template := "{{.current_date}}-{{.filename | slug}}"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Process(template, ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEngine_ProcessCached(b *testing.B) {
	engine := NewEngine()
	ctx := &Context{
		Filename: "test-file",
		Title:    "Test Title",
	}
	template := "{{.current_date}}-{{.filename | slug}}"

	// Pre-compile template
	_, err := engine.Process(template, ctx)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Process(template, ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSlugify(b *testing.B) {
	engine := NewEngine()
	input := "My Very Long Title With Many Words And Special Characters!"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.slugify(input)
	}
}
