package frontmatter

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockTemplateEngine for testing
type MockTemplateEngine struct {
	processFunc func(template string, ctx interface{}) (string, error)
}

func (m *MockTemplateEngine) Process(template string, ctx interface{}) (string, error) {
	if m.processFunc != nil {
		return m.processFunc(template, ctx)
	}
	return template, nil // Default: return template as-is
}

func TestProcessor_ParseBytes(t *testing.T) {
	mockEngine := &MockTemplateEngine{}
	processor := NewProcessor(mockEngine)

	tests := []struct {
		name     string
		input    []byte
		expected *Document
		wantErr  bool
	}{
		{
			name:  "valid frontmatter with body",
			input: []byte("---\ntitle: Test Note\ntags: [test, example]\n---\n\n# Content\n\nThis is the body."),
			expected: &Document{
				Frontmatter: map[string]interface{}{
					"title": "Test Note",
					"tags":  []interface{}{"test", "example"},
				},
				Body: "\n# Content\n\nThis is the body.",
			},
			wantErr: false,
		},
		{
			name:  "no frontmatter",
			input: []byte("# Just Content\n\nNo frontmatter here."),
			expected: &Document{
				Frontmatter: map[string]interface{}{},
				Body:        "# Just Content\n\nNo frontmatter here.",
			},
			wantErr: false,
		},
		{
			name:  "empty frontmatter",
			input: []byte("---\n---\n\n# Content"),
			expected: &Document{
				Frontmatter: map[string]interface{}{},
				Body:        "\n# Content",
			},
			wantErr: false,
		},
		{
			name:    "invalid yaml",
			input:   []byte("---\ntitle: Test\ninvalid: [unclosed\n---\n\n# Content"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := processor.ParseBytes(tt.input)
			
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			assert.Equal(t, tt.expected.Body, doc.Body)
			assert.Equal(t, tt.expected.Frontmatter, doc.Frontmatter)
		})
	}
}

func TestProcessor_Upsert(t *testing.T) {
	tests := []struct {
		name           string
		doc            *Document
		options        UpsertOptions
		templateFunc   func(template string, ctx interface{}) (string, error)
		expectedFM     map[string]interface{}
		wantErr        bool
	}{
		{
			name: "add new field",
			doc: &Document{
				Frontmatter: map[string]interface{}{
					"title": "Existing",
				},
			},
			options: UpsertOptions{
				Fields:   []string{"tags"},
				Defaults: []string{"[]"},
			},
			expectedFM: map[string]interface{}{
				"title": "Existing",
				"tags":  []string{},
			},
		},
		{
			name: "don't overwrite existing without flag",
			doc: &Document{
				Frontmatter: map[string]interface{}{
					"title": "Existing",
				},
			},
			options: UpsertOptions{
				Fields:    []string{"title"},
				Defaults:  []string{"New Title"},
				Overwrite: false,
			},
			expectedFM: map[string]interface{}{
				"title": "Existing",
			},
		},
		{
			name: "overwrite existing with flag",
			doc: &Document{
				Frontmatter: map[string]interface{}{
					"title": "Existing",
				},
			},
			options: UpsertOptions{
				Fields:    []string{"title"},
				Defaults:  []string{"New Title"},
				Overwrite: true,
			},
			expectedFM: map[string]interface{}{
				"title": "New Title",
			},
		},
		{
			name: "template processing",
			doc: &Document{
				Frontmatter: map[string]interface{}{},
			},
			options: UpsertOptions{
				Fields:   []string{"created"},
				Defaults: []string{"{{current_date}}"},
			},
			templateFunc: func(template string, ctx interface{}) (string, error) {
				if template == "{{current_date}}" {
					return "2024-01-15", nil
				}
				return template, nil
			},
			expectedFM: map[string]interface{}{
				"created": "2024-01-15",
			},
		},
		{
			name: "mismatched fields and defaults",
			doc:  &Document{Frontmatter: map[string]interface{}{}},
			options: UpsertOptions{
				Fields:   []string{"field1", "field2"},
				Defaults: []string{"value1"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEngine := &MockTemplateEngine{processFunc: tt.templateFunc}
			processor := NewProcessor(mockEngine)

			err := processor.Upsert(tt.doc, tt.options, nil)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedFM, tt.doc.Frontmatter)
		})
	}
}

func TestProcessor_Serialize(t *testing.T) {
	mockEngine := &MockTemplateEngine{}
	processor := NewProcessor(mockEngine)

	tests := []struct {
		name     string
		doc      *Document
		expected string
	}{
		{
			name: "with frontmatter and body",
			doc: &Document{
				Frontmatter: map[string]interface{}{
					"title": "Test Note",
					"tags":  []string{"test"},
				},
				Body: "\n# Content\n\nBody text.",
			},
			expected: "---\ntags:\n    - test\ntitle: Test Note\n---\n\n# Content\n\nBody text.",
		},
		{
			name: "body only",
			doc: &Document{
				Frontmatter: map[string]interface{}{},
				Body:        "# Just Content",
			},
			expected: "# Just Content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.Serialize(tt.doc)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

func TestProcessor_ParseAndWrite(t *testing.T) {
	mockEngine := &MockTemplateEngine{}
	processor := NewProcessor(mockEngine)

	// Create temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	
	content := []byte("---\ntitle: Original Title\ntags: [original]\n---\n\n# Original Content")
	err := os.WriteFile(testFile, content, 0644)
	require.NoError(t, err)

	// Parse file
	doc, err := processor.Parse(testFile)
	require.NoError(t, err)

	// Verify parsing
	assert.Equal(t, "Original Title", doc.Frontmatter["title"])
	assert.Equal(t, "\n# Original Content", doc.Body)

	// Modify frontmatter
	doc.Frontmatter["modified"] = "2024-01-15"

	// Write back
	err = processor.Write(doc, testFile)
	require.NoError(t, err)

	// Verify file was written correctly
	newContent, err := os.ReadFile(testFile)
	require.NoError(t, err)

	// Parse again to verify
	newDoc, err := processor.ParseBytes(newContent)
	require.NoError(t, err)
	
	assert.Equal(t, "Original Title", newDoc.Frontmatter["title"])
	assert.Equal(t, "2024-01-15", newDoc.Frontmatter["modified"])
	assert.Equal(t, "\n# Original Content", newDoc.Body)
}

func TestConvertValue(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{"[]", []string{}},
		{"{}", map[string]interface{}{}},
		{"true", true},
		{"false", false},
		{"2024-01-15", "2024-01-15"},
		{"regular string", "regular string"},
		{"123", "123"}, // Numbers stay as strings unless specifically converted
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertValue(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("convertValue(%q) = %v (%T), want %v (%T)", 
					tt.input, result, result, tt.expected, tt.expected)
			}
		})
	}
}

func TestNewFileContext(t *testing.T) {
	// Create temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-note.md")
	
	content := []byte("# Test Note")
	err := os.WriteFile(testFile, content, 0644)
	require.NoError(t, err)

	ctx, err := NewFileContext(testFile)
	require.NoError(t, err)

	assert.Equal(t, "test-note", ctx.Filename)
	assert.True(t, time.Since(ctx.FileModTime) < time.Minute) // Recently created
}

func TestNewFileContext_NonexistentFile(t *testing.T) {
	_, err := NewFileContext("/nonexistent/file.md")
	assert.Error(t, err)
}