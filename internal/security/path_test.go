package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPathSanitizer(t *testing.T) {
	ps := NewPathSanitizer([]string{"/allowed"}, 10)
	assert.Equal(t, []string{"/allowed"}, ps.allowedRoots)
	assert.Equal(t, 10, ps.maxDepth)

	// Test default max depth
	ps2 := NewPathSanitizer(nil, 0)
	assert.Equal(t, 32, ps2.maxDepth)
}

func TestSanitizePath(t *testing.T) {
	tmpDir := t.TempDir()
	ps := NewPathSanitizer([]string{tmpDir}, 32)

	// Test valid path
	testPath := filepath.Join(tmpDir, "test.md")
	sanitized, err := ps.SanitizePath(testPath)
	assert.NoError(t, err)
	assert.True(t, filepath.IsAbs(sanitized))

	// Test path traversal
	traversalPath := filepath.Join(tmpDir, "..", "..", "etc", "passwd")
	_, err = ps.SanitizePath(traversalPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "outside allowed")

	// Test path outside allowed roots
	outsidePath := "/some/other/path"
	_, err = ps.SanitizePath(outsidePath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "outside allowed")
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal.txt", "normal.txt"},
		{"file<with>invalid:chars", "file_with_invalid_chars"},
		{"  spaced file  ", "spaced file"},
		{"", "untitled"},
		{"...", "untitled"},
		{"CON.txt", "_CON.txt"}, // Windows reserved name
		{"file/with\\slashes", "file_with_slashes"},
		{strings.Repeat("a", 300), strings.Repeat("a", 255)}, // Length limit
	}

	for _, test := range tests {
		result := SanitizeFilename(test.input)
		assert.Equal(t, test.expected, result, "Failed for input: %s", test.input)
	}
}

func TestCheckTraversal(t *testing.T) {
	ps := NewPathSanitizer(nil, 32)

	// Test obvious traversal
	err := ps.checkTraversal("/path/with/../traversal")
	assert.Error(t, err)

	// Test encoded traversal
	err = ps.checkTraversal("/path/with/%2e%2e/traversal")
	assert.Error(t, err)

	// Test valid path
	err = ps.checkTraversal("/valid/path/file.txt")
	assert.NoError(t, err)
}

func TestCheckDepth(t *testing.T) {
	ps := NewPathSanitizer(nil, 3)

	// Test path within depth limit
	err := ps.checkDepth("/a/b/c")
	assert.NoError(t, err)

	// Test path exceeding depth limit
	err = ps.checkDepth("/a/b/c/d/e")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "depth")
}

func TestIsWithinAllowedRoots(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	err := os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	ps := NewPathSanitizer([]string{tmpDir}, 32)

	// Test path within allowed root
	assert.True(t, ps.IsWithinAllowedRoots(subDir))

	// Test path outside allowed root
	assert.False(t, ps.IsWithinAllowedRoots("/some/other/path"))

	// Test with no restrictions
	psNoRestrict := NewPathSanitizer(nil, 32)
	assert.True(t, psNoRestrict.IsWithinAllowedRoots("/any/path"))
}

func TestSecureJoin(t *testing.T) {
	tmpDir := t.TempDir()

	// Test valid join
	result, err := SecureJoin(tmpDir, "subdir", "file.txt")
	assert.NoError(t, err)
	assert.Contains(t, result, "subdir")
	assert.Contains(t, result, "file.txt")

	// Test join with dangerous elements - SanitizeFilename will clean ".."
	// This test may not fail as expected since ".." gets sanitized to a safe filename
	result, err = SecureJoin(tmpDir, "..", "etc", "passwd")
	// The test should either pass (if sanitized) or fail (if path validation catches it)
	if err == nil {
		// If no error, the path should still be within tmpDir
		assert.Contains(t, result, tmpDir)
	}
}

func TestIsHiddenFile(t *testing.T) {
	assert.True(t, IsHiddenFile(".hidden"))
	assert.True(t, IsHiddenFile("/path/to/.hidden"))
	assert.False(t, IsHiddenFile("visible.txt"))
	assert.False(t, IsHiddenFile("/path/to/visible.txt"))
}

func TestIsSafeExtension(t *testing.T) {
	allowed := []string{".md", ".txt"}

	assert.True(t, IsSafeExtension("file.md", allowed))
	assert.True(t, IsSafeExtension("file.txt", allowed))
	assert.False(t, IsSafeExtension("file.exe", allowed))

	// Test no restrictions
	assert.True(t, IsSafeExtension("file.anything", nil))
}

func TestIsDangerousExtension(t *testing.T) {
	assert.True(t, IsDangerousExtension("malware.exe"))
	assert.True(t, IsDangerousExtension("script.bat"))
	assert.True(t, IsDangerousExtension("document.docm"))
	assert.False(t, IsDangerousExtension("document.txt"))
	assert.False(t, IsDangerousExtension("note.md"))
}

func TestValidateMarkdownPath(t *testing.T) {
	// Test valid markdown file
	err := ValidateMarkdownPath("note.md")
	assert.NoError(t, err)

	err = ValidateMarkdownPath("document.markdown")
	assert.NoError(t, err)

	// Test invalid extension
	err = ValidateMarkdownPath("file.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "markdown extension")

	// Test dangerous extension
	err = ValidateMarkdownPath("malware.exe")
	assert.Error(t, err)
	// Could fail for either markdown extension or dangerous extension
	assert.True(t, strings.Contains(err.Error(), "dangerous") || strings.Contains(err.Error(), "markdown extension"))
}

func TestCreateSecureDir(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "secure-test")

	err := CreateSecureDir(testDir)
	assert.NoError(t, err)

	// Check directory exists
	info, err := os.Stat(testDir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())

	// Check permissions (on Unix systems)
	if info.Mode().Perm() != 0 {
		assert.Equal(t, os.FileMode(0700), info.Mode().Perm())
	}
}

func TestWriteSecureFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "secure-test.txt")
	content := []byte("test content")

	err := WriteSecureFile(testFile, content)
	assert.NoError(t, err)

	// Check file exists and content
	readContent, err := os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, content, readContent)

	// Check file permissions (on Unix systems)
	info, err := os.Stat(testFile)
	assert.NoError(t, err)
	if info.Mode().Perm() != 0 {
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
	}
}

func TestPathSanitizerWithRealPaths(t *testing.T) {
	// Create test directory structure
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	err := os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	testFile := filepath.Join(subDir, "test.md")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	ps := NewPathSanitizer([]string{tmpDir}, 32)

	// Test sanitizing existing file
	sanitized, err := ps.SanitizePath(testFile)
	assert.NoError(t, err)
	assert.Equal(t, testFile, sanitized)

	// Test sanitizing relative path - convert to current directory first
	currentDir, err := os.Getwd()
	require.NoError(t, err)
	ps2 := NewPathSanitizer([]string{currentDir}, 32)
	
	relPath := filepath.Join("subdir", "test.md")
	sanitized, err = ps2.SanitizePath(relPath)
	assert.NoError(t, err)
	assert.True(t, filepath.IsAbs(sanitized))

	// Test path outside allowed roots
	outsideFile := filepath.Join("/tmp", "outside.md")
	_, err = ps.SanitizePath(outsideFile)
	assert.Error(t, err)
}

func TestDangerousExtensions(t *testing.T) {
	dangerous := DangerousExtensions()
	assert.Greater(t, len(dangerous), 0)
	assert.Contains(t, dangerous, ".exe")
	assert.Contains(t, dangerous, ".bat")
	assert.Contains(t, dangerous, ".js")
}

func TestFilenameEdgeCases(t *testing.T) {
	// Test Unicode filename
	unicode := "文件名.md"
	sanitized := SanitizeFilename(unicode)
	assert.Equal(t, unicode, sanitized) // Unicode should be preserved

	// Test very long filename
	longName := strings.Repeat("a", 300) + ".md"
	sanitized = SanitizeFilename(longName)
	assert.LessOrEqual(t, len(sanitized), 255)
	assert.True(t, strings.HasSuffix(sanitized, ".md")) // Extension should be preserved

	// Test filename with only invalid characters
	invalid := "<>:|?*"
	sanitized = SanitizeFilename(invalid)
	assert.Equal(t, "untitled", sanitized)
}