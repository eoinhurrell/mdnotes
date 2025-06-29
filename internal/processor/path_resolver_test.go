package processor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eoinhurrell/mdnotes/internal/vault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathResolver_ResolveTarget(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "path_resolver_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	resolver := NewPathResolver(tmpDir)

	tests := []struct {
		name        string
		linkTarget  string
		contextFile string
		expected    string
	}{
		{
			name:        "simple filename",
			linkTarget:  "note.md",
			contextFile: "",
			expected:    filepath.Join(tmpDir, "note.md"),
		},
		{
			name:        "relative path",
			linkTarget:  "folder/note.md",
			contextFile: "",
			expected:    filepath.Join(tmpDir, "folder/note.md"),
		},
		{
			name:        "URL encoded target",
			linkTarget:  "folder/note%20with%20spaces.md",
			contextFile: "",
			expected:    filepath.Join(tmpDir, "folder/note with spaces.md"),
		},
		{
			name:        "target with fragment",
			linkTarget:  "note.md#heading",
			contextFile: "",
			expected:    filepath.Join(tmpDir, "note.md"),
		},
		{
			name:        "deep path",
			linkTarget:  "project/docs/readme.md",
			contextFile: "",
			expected:    filepath.Join(tmpDir, "project/docs/readme.md"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.ResolveTarget(tt.linkTarget, tt.contextFile)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPathResolver_AnalyzeLinkMatch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "path_resolver_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	resolver := NewPathResolver(tmpDir)

	tests := []struct {
		name     string
		link     vault.Link
		filePath string
		expected MatchPriority
	}{
		{
			name: "exact path match - wiki link",
			link: vault.Link{
				Type:   vault.WikiLink,
				Target: "folder/note",
			},
			filePath: filepath.Join(tmpDir, "folder/note.md"),
			expected: FullPathMatch,
		},
		{
			name: "exact path match - markdown link",
			link: vault.Link{
				Type:   vault.MarkdownLink,
				Target: "folder/note.md",
			},
			filePath: filepath.Join(tmpDir, "folder/note.md"),
			expected: FullPathMatch,
		},
		{
			name: "basename match - wiki link",
			link: vault.Link{
				Type:   vault.WikiLink,
				Target: "note",
			},
			filePath: filepath.Join(tmpDir, "different-folder/note.md"),
			expected: BaseNameMatch,
		},
		{
			name: "no basename match - markdown link",
			link: vault.Link{
				Type:   vault.MarkdownLink,
				Target: "note.md",
			},
			filePath: filepath.Join(tmpDir, "different-folder/note.md"),
			expected: NoMatch,
		},
		{
			name: "no match - different file",
			link: vault.Link{
				Type:   vault.WikiLink,
				Target: "other",
			},
			filePath: filepath.Join(tmpDir, "note.md"),
			expected: NoMatch,
		},
		{
			name: "URL encoded match",
			link: vault.Link{
				Type:   vault.MarkdownLink,
				Target: "folder/note%20with%20spaces.md",
			},
			filePath: filepath.Join(tmpDir, "folder/note with spaces.md"),
			expected: FullPathMatch,
		},
		{
			name: "fragment ignored in matching",
			link: vault.Link{
				Type:   vault.WikiLink,
				Target: "note#heading",
			},
			filePath: filepath.Join(tmpDir, "note.md"),
			expected: FullPathMatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.AnalyzeLinkMatch(tt.link, tt.filePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPathResolver_FindAllMatches(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "path_resolver_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	resolver := NewPathResolver(tmpDir)

	// Create test files
	vaultFiles := []*vault.VaultFile{
		{Path: filepath.Join(tmpDir, "note.md")},
		{Path: filepath.Join(tmpDir, "folder/note.md")},
		{Path: filepath.Join(tmpDir, "other-folder/note.md")},
		{Path: filepath.Join(tmpDir, "different.md")},
	}

	tests := []struct {
		name            string
		link            vault.Link
		expectedMatches int
		hasAmbiguity    bool
	}{
		{
			name: "wiki link with multiple basename matches",
			link: vault.Link{
				Type:   vault.WikiLink,
				Target: "note",
			},
			expectedMatches: 3, // All three note.md files match by basename
			hasAmbiguity:    true,
		},
		{
			name: "wiki link with exact path match",
			link: vault.Link{
				Type:   vault.WikiLink,
				Target: "folder/note",
			},
			expectedMatches: 1, // Only folder/note.md matches exactly
			hasAmbiguity:    false,
		},
		{
			name: "markdown link - exact path only",
			link: vault.Link{
				Type:   vault.MarkdownLink,
				Target: "note.md",
			},
			expectedMatches: 1, // Only root note.md matches exactly
			hasAmbiguity:    false,
		},
		{
			name: "no matches",
			link: vault.Link{
				Type:   vault.WikiLink,
				Target: "nonexistent",
			},
			expectedMatches: 0,
			hasAmbiguity:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.FindAllMatches(tt.link, vaultFiles)
			assert.Equal(t, tt.expectedMatches, len(result.Matches))
			assert.Equal(t, tt.hasAmbiguity, result.HasAmbiguity)
		})
	}
}

func TestPathResolver_ResolveBestMatch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "path_resolver_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	resolver := NewPathResolver(tmpDir)

	// Create test files
	vaultFiles := []*vault.VaultFile{
		{Path: filepath.Join(tmpDir, "note.md")},
		{Path: filepath.Join(tmpDir, "folder/note.md")},
		{Path: filepath.Join(tmpDir, "other-folder/note.md")},
		{Path: filepath.Join(tmpDir, "project/readme.md")},
	}

	tests := []struct {
		name        string
		link        vault.Link
		expectError bool
		expected    string
	}{
		{
			name: "unique exact match",
			link: vault.Link{
				Type:   vault.WikiLink,
				Target: "project/readme",
			},
			expectError: false,
			expected:    filepath.Join(tmpDir, "project/readme.md"),
		},
		{
			name: "unique basename match",
			link: vault.Link{
				Type:   vault.WikiLink,
				Target: "readme",
			},
			expectError: false,
			expected:    filepath.Join(tmpDir, "project/readme.md"),
		},
		{
			name: "ambiguous basename matches",
			link: vault.Link{
				Type:   vault.WikiLink,
				Target: "note",
			},
			expectError: true, // Multiple basename matches
		},
		{
			name: "no matches",
			link: vault.Link{
				Type:   vault.WikiLink,
				Target: "nonexistent",
			},
			expectError: true,
		},
		{
			name: "markdown link exact match",
			link: vault.Link{
				Type:   vault.MarkdownLink,
				Target: "folder/note.md",
			},
			expectError: false,
			expected:    filepath.Join(tmpDir, "folder/note.md"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.ResolveBestMatch(tt.link, vaultFiles)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestPathResolver_NormalizePath(t *testing.T) {
	tmpDir := "/vault/root"
	resolver := NewPathResolver(tmpDir)

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "relative path",
			path:     "folder/note.md",
			expected: "folder/note.md",
		},
		{
			name:     "absolute vault path",
			path:     "/vault/root/folder/note.md",
			expected: "folder/note.md",
		},
		{
			name:     "backslash separators",
			path:     "folder\\note.md",
			expected: "folder/note.md",
		},
		{
			name:     "mixed separators",
			path:     "folder\\subfolder/note.md",
			expected: "folder/subfolder/note.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.NormalizePath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPathResolver_IsVaultRelative(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "path_resolver_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	resolver := NewPathResolver(tmpDir)

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "relative path",
			path:     "folder/note.md",
			expected: true,
		},
		{
			name:     "absolute path within vault",
			path:     filepath.Join(tmpDir, "folder/note.md"),
			expected: true,
		},
		{
			name:     "absolute path outside vault",
			path:     "/different/path/note.md",
			expected: false,
		},
		{
			name:     "vault root",
			path:     tmpDir,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.IsVaultRelative(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPathResolver_GetVaultRelativePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "path_resolver_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	resolver := NewPathResolver(tmpDir)

	tests := []struct {
		name         string
		absolutePath string
		expected     string
		expectError  bool
	}{
		{
			name:         "file in vault root",
			absolutePath: filepath.Join(tmpDir, "note.md"),
			expected:     "note.md",
			expectError:  false,
		},
		{
			name:         "file in subdirectory",
			absolutePath: filepath.Join(tmpDir, "folder/subfolder/note.md"),
			expected:     "folder/subfolder/note.md",
			expectError:  false,
		},
		{
			name:         "file outside vault",
			absolutePath: "/different/path/note.md",
			expected:     "",
			expectError:  true,
		},
		{
			name:         "vault root itself",
			absolutePath: tmpDir,
			expected:     ".",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.GetVaultRelativePath(tt.absolutePath)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				// Normalize for cross-platform comparison
				expected := filepath.ToSlash(tt.expected)
				assert.Equal(t, expected, result)
			}
		})
	}
}

func TestPathResolver_ResolveAbsolutePath(t *testing.T) {
	tmpDir := "/vault/root"
	resolver := NewPathResolver(tmpDir)

	tests := []struct {
		name              string
		vaultRelativePath string
		expected          string
	}{
		{
			name:              "simple filename",
			vaultRelativePath: "note.md",
			expected:          "/vault/root/note.md",
		},
		{
			name:              "subdirectory path",
			vaultRelativePath: "folder/subfolder/note.md",
			expected:          "/vault/root/folder/subfolder/note.md",
		},
		{
			name:              "current directory",
			vaultRelativePath: ".",
			expected:          "/vault/root",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.ResolveAbsolutePath(tt.vaultRelativePath)
			// Use filepath.Clean to normalize the result for comparison
			assert.Equal(t, filepath.Clean(tt.expected), filepath.Clean(result))
		})
	}
}
