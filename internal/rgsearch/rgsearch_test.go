package rgsearch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSearcher(t *testing.T) {
	searcher := NewSearcher()
	assert.NotNil(t, searcher)

	// Check if ripgrep is available on the system
	if searcher.IsAvailable() {
		version, err := searcher.GetVersion()
		assert.NoError(t, err)
		assert.NotEmpty(t, version)
	}
}

func TestDefaultSearchOptions(t *testing.T) {
	options := DefaultSearchOptions()

	assert.False(t, options.CaseSensitive)
	assert.False(t, options.WordBoundary)
	assert.False(t, options.FixedStrings)
	assert.True(t, options.Regex)
	assert.Equal(t, -1, options.MaxDepth)
	assert.False(t, options.FollowSymlinks)
	assert.False(t, options.SearchZip)
	assert.False(t, options.Multiline)
	assert.Equal(t, 0, options.ContextBefore)
	assert.Equal(t, 0, options.ContextAfter)
	assert.Equal(t, 1000, options.MaxMatches)
	assert.Equal(t, 100, options.MaxFileMatches)
	assert.Equal(t, 30*time.Second, options.Timeout)
}

func TestSearcherNotAvailable(t *testing.T) {
	// Create searcher with no ripgrep available
	searcher := &Searcher{
		rgPath:    "",
		available: false,
	}

	assert.False(t, searcher.IsAvailable())

	_, err := searcher.GetVersion()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ripgrep not available")

	options := DefaultSearchOptions()
	options.Pattern = "test"

	_, err = searcher.Search(context.Background(), options)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ripgrep not available")

	_, err = searcher.SearchFiles(context.Background(), options)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ripgrep not available")

	_, err = searcher.CountMatches(context.Background(), options)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ripgrep not available")
}

func TestSearchWithInvalidPattern(t *testing.T) {
	searcher := NewSearcher()
	if !searcher.IsAvailable() {
		t.Skip("ripgrep not available")
	}

	options := DefaultSearchOptions()
	options.Pattern = "" // Empty pattern

	_, err := searcher.Search(context.Background(), options)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "search pattern is required")
}

func TestSearchInTestFiles(t *testing.T) {
	searcher := NewSearcher()
	if !searcher.IsAvailable() {
		t.Skip("ripgrep not available")
	}

	// Create temporary test files
	tmpDir := t.TempDir()

	testFiles := map[string]string{
		"file1.txt": "This is a test file\nwith multiple lines\ncontaining test data",
		"file2.txt": "Another file\nwith different content\nno matches here",
		"file3.md":  "# Test Document\n\nThis is a **test** markdown file\nwith some test content",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tmpDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Test basic search
	options := DefaultSearchOptions()
	options.Pattern = "test"
	options.Path = tmpDir

	results, err := searcher.Search(context.Background(), options)
	require.NoError(t, err)

	// Should find matches in file1.txt and file3.md
	assert.Greater(t, len(results), 0)

	// Check that we got actual match results
	for _, result := range results {
		assert.Equal(t, "match", result.Type)
		assert.NotEmpty(t, result.Data.Path.Text)
		assert.NotEmpty(t, result.Data.Lines.Text)
		assert.Contains(t, result.Data.Lines.Text, "test")
	}
}

func TestSearchFiles(t *testing.T) {
	searcher := NewSearcher()
	if !searcher.IsAvailable() {
		t.Skip("ripgrep not available")
	}

	tmpDir := t.TempDir()

	testFiles := map[string]string{
		"match1.txt":  "This file contains the keyword",
		"match2.md":   "# Header\nAnother file with keyword",
		"nomatch.txt": "This file does not contain the search term",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tmpDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)
	}

	options := DefaultSearchOptions()
	options.Pattern = "keyword"
	options.Path = tmpDir

	files, err := searcher.SearchFiles(context.Background(), options)
	require.NoError(t, err)

	assert.Len(t, files, 2)

	// Convert to basenames for easier checking
	basenames := make([]string, len(files))
	for i, file := range files {
		basenames[i] = filepath.Base(file)
	}

	assert.Contains(t, basenames, "match1.txt")
	assert.Contains(t, basenames, "match2.md")
	assert.NotContains(t, basenames, "nomatch.txt")
}

func TestSearchInFiles(t *testing.T) {
	searcher := NewSearcher()
	if !searcher.IsAvailable() {
		t.Skip("ripgrep not available")
	}

	tmpDir := t.TempDir()

	// Create test files
	file1 := filepath.Join(tmpDir, "test1.txt")
	file2 := filepath.Join(tmpDir, "test2.txt")

	err := os.WriteFile(file1, []byte("This file has matches\nand more matches"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(file2, []byte("This file has no hits"), 0644)
	require.NoError(t, err)

	options := DefaultSearchOptions()
	files := []string{file1, file2}

	results, err := searcher.SearchInFiles(context.Background(), "matches", files, options)
	require.NoError(t, err)

	// Should find matches only in file1
	assert.Greater(t, len(results), 0)
	for _, result := range results {
		assert.Equal(t, file1, result.Data.Path.Text)
		assert.Contains(t, result.Data.Lines.Text, "matches")
	}
}

func TestSearchWithIncludeExcludePatterns(t *testing.T) {
	searcher := NewSearcher()
	if !searcher.IsAvailable() {
		t.Skip("ripgrep not available")
	}

	tmpDir := t.TempDir()

	// Create files with different extensions
	testFiles := map[string]string{
		"test.md":   "markdown content",
		"test.txt":  "text content",
		"test.log":  "log content",
		"README.md": "readme content",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tmpDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Search only markdown files
	options := DefaultSearchOptions()
	options.Pattern = "content"
	options.Path = tmpDir
	options.IncludePatterns = []string{"*.md"}

	files, err := searcher.SearchFiles(context.Background(), options)
	require.NoError(t, err)

	// Should only find .md files
	for _, file := range files {
		assert.True(t, filepath.Ext(file) == ".md")
	}
	assert.Len(t, files, 2) // test.md and README.md
}

func TestSearchWithFixedStrings(t *testing.T) {
	searcher := NewSearcher()
	if !searcher.IsAvailable() {
		t.Skip("ripgrep not available")
	}

	tmpDir := t.TempDir()

	// Create file with regex special characters
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "This file contains $pecial characters like . and *"
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	// Search with fixed strings (no regex interpretation)
	options := DefaultSearchOptions()
	options.Pattern = "$pecial"
	options.Path = tmpDir
	options.FixedStrings = true
	options.Regex = false

	results, err := searcher.Search(context.Background(), options)
	require.NoError(t, err)

	assert.Len(t, results, 1)
	assert.Contains(t, results[0].Data.Lines.Text, "$pecial")
}

func TestSearchWithCaseSensitivity(t *testing.T) {
	searcher := NewSearcher()
	if !searcher.IsAvailable() {
		t.Skip("ripgrep not available")
	}

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	content := "This file contains TEST and test"
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	// Case insensitive search (default)
	options := DefaultSearchOptions()
	options.Pattern = "test"
	options.Path = tmpDir
	options.CaseSensitive = false

	results, err := searcher.Search(context.Background(), options)
	require.NoError(t, err)
	assert.Len(t, results, 2) // Should match both "TEST" and "test"

	// Case sensitive search
	options.CaseSensitive = true
	results, err = searcher.Search(context.Background(), options)
	require.NoError(t, err)
	assert.Len(t, results, 1) // Should match only "test"
}

func TestCountMatches(t *testing.T) {
	searcher := NewSearcher()
	if !searcher.IsAvailable() {
		t.Skip("ripgrep not available")
	}

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	content := "test line 1\ntest line 2\nanother test line\nno match line"
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	options := DefaultSearchOptions()
	options.Pattern = "test"
	options.Path = tmpDir

	count, err := searcher.CountMatches(context.Background(), options)
	require.NoError(t, err)
	assert.Equal(t, 1, count) // One file with matches
}

func TestValidatePattern(t *testing.T) {
	// Valid patterns
	assert.NoError(t, ValidatePattern("test"))
	assert.NoError(t, ValidatePattern("[a-z]+"))
	assert.NoError(t, ValidatePattern("\\d+"))
	assert.NoError(t, ValidatePattern("(test|example)"))

	// Invalid patterns
	assert.Error(t, ValidatePattern("["))
	assert.Error(t, ValidatePattern("("))
	assert.Error(t, ValidatePattern("*"))
}

func TestEscapePattern(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"test.file", "test\\.file"},
		{"$pecial", "\\$pecial"},
		{"[bracket]", "\\[bracket\\]"},
		{"(paren)", "\\(paren\\)"},
		{"*star*", "\\*star\\*"},
		{"^start", "\\^start"},
		{"end$", "end\\$"},
	}

	for _, test := range tests {
		result := EscapePattern(test.input)
		assert.Equal(t, test.expected, result, "Failed for input: %s", test.input)
	}
}

func TestSearchWithContext(t *testing.T) {
	searcher := NewSearcher()
	if !searcher.IsAvailable() {
		t.Skip("ripgrep not available")
	}

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	content := "line 1\nline 2\nmatch line\nline 4\nline 5"
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	options := DefaultSearchOptions()
	options.Pattern = "match"
	options.Path = tmpDir
	options.ContextBefore = 1
	options.ContextAfter = 1

	results, err := searcher.Search(context.Background(), options)
	require.NoError(t, err)

	// With context, we should get multiple results (the match and context lines)
	assert.Greater(t, len(results), 1)
}

func TestSearchTimeout(t *testing.T) {
	searcher := NewSearcher()
	if !searcher.IsAvailable() {
		t.Skip("ripgrep not available")
	}

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	options := DefaultSearchOptions()
	options.Pattern = "test"
	options.Path = "/" // Search entire filesystem (should timeout)

	_, err := searcher.Search(ctx, options)
	assert.Error(t, err)
}
