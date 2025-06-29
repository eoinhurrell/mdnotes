package links

import (
	"testing"

	"github.com/eoinhurrell/mdnotes/internal/vault"
	"github.com/stretchr/testify/assert"
)

func TestResolveTargetPath(t *testing.T) {
	file := &vault.VaultFile{
		RelativePath: "docs/guide.md",
	}
	vaultRoot := "/vault"

	tests := []struct {
		name         string
		link         vault.Link
		fileRelative bool
		expected     string
	}{
		// Wiki links - always vault-relative
		{
			name:         "wiki link vault-relative",
			link:         vault.Link{Type: vault.WikiLink, Target: "resources/test"},
			fileRelative: false,
			expected:     "resources/test",
		},
		{
			name:         "wiki link file-relative mode still vault-relative",
			link:         vault.Link{Type: vault.WikiLink, Target: "resources/test"},
			fileRelative: true,
			expected:     "resources/test",
		},

		// Markdown links - vault-relative by default
		{
			name:         "markdown link vault-relative",
			link:         vault.Link{Type: vault.MarkdownLink, Target: "resources/test.md"},
			fileRelative: false,
			expected:     "resources/test.md",
		},
		{
			name:         "markdown link with subdirectory",
			link:         vault.Link{Type: vault.MarkdownLink, Target: "utils/helper.md"},
			fileRelative: false,
			expected:     "utils/helper.md",
		},

		// Markdown links - file-relative mode
		{
			name:         "markdown link file-relative from subdirectory",
			link:         vault.Link{Type: vault.MarkdownLink, Target: "../resources/test.md"},
			fileRelative: true,
			expected:     "resources/test.md",
		},
		{
			name:         "markdown link file-relative same directory",
			link:         vault.Link{Type: vault.MarkdownLink, Target: "other.md"},
			fileRelative: true,
			expected:     "docs/other.md",
		},

		// Fragment handling
		{
			name:         "link with fragment",
			link:         vault.Link{Type: vault.MarkdownLink, Target: "resources/test.md#section"},
			fileRelative: false,
			expected:     "resources/test.md",
		},

		// Embed links
		{
			name:         "embed link vault-relative",
			link:         vault.Link{Type: vault.EmbedLink, Target: "images/photo.png"},
			fileRelative: false,
			expected:     "images/photo.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveTargetPath(tt.link, file, vaultRoot, tt.fileRelative)
			if result != tt.expected {
				t.Errorf("resolveTargetPath() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestCheckLinkExists(t *testing.T) {
	existingFiles := map[string]bool{
		"resources/test.md": true,
		"resources/test":    true,
		"utils/helper.md":   true,
		"utils/helper":      true,
		"docs/readme.md":    true,
		"docs/readme":       true,
		"images/photo.png":  true,
	}

	baseNameFiles := map[string][]string{
		"test":   {"resources/test.md"},
		"helper": {"utils/helper.md"},
		"readme": {"docs/readme.md"},
	}

	tests := []struct {
		name     string
		target   string
		linkType vault.LinkType
		expected bool
	}{
		// Existing files
		{
			name:     "markdown link to existing file",
			target:   "resources/test.md",
			linkType: vault.MarkdownLink,
			expected: true,
		},
		{
			name:     "markdown link to subdirectory file",
			target:   "utils/helper.md",
			linkType: vault.MarkdownLink,
			expected: true,
		},
		{
			name:     "wiki link by basename",
			target:   "test",
			linkType: vault.WikiLink,
			expected: true,
		},
		{
			name:     "wiki link by full path",
			target:   "resources/test",
			linkType: vault.WikiLink,
			expected: true,
		},
		{
			name:     "embed link to asset",
			target:   "images/photo.png",
			linkType: vault.EmbedLink,
			expected: true,
		},

		// Non-existing files
		{
			name:     "markdown link to non-existing file",
			target:   "nonexistent/file.md",
			linkType: vault.MarkdownLink,
			expected: false,
		},
		{
			name:     "wiki link to non-existing file",
			target:   "nonexistent",
			linkType: vault.WikiLink,
			expected: false,
		},

		// Fragment handling
		{
			name:     "link with fragment to existing file",
			target:   "resources/test.md#section",
			linkType: vault.MarkdownLink,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkLinkExists(tt.target, existingFiles, baseNameFiles, tt.linkType)
			if result != tt.expected {
				t.Errorf("checkLinkExists(%q, %v) = %v, expected %v",
					tt.target, tt.linkType, result, tt.expected)
			}
		})
	}
}

func TestNewLinksCommand(t *testing.T) {
	cmd := NewLinksCommand()
	
	assert.Equal(t, "links", cmd.Use)
	assert.Equal(t, "Manage links in markdown files", cmd.Short)
	assert.Contains(t, cmd.Long, "Commands for checking")
	
	// Should have subcommands
	subcommands := cmd.Commands()
	assert.Len(t, subcommands, 2)
}

func TestNewCheckCommand(t *testing.T) {
	cmd := NewCheckCommand()
	
	assert.Equal(t, "check [path]", cmd.Use)
	assert.Contains(t, cmd.Aliases, "c")
	assert.Equal(t, "Check for broken internal links", cmd.Short)
	assert.Contains(t, cmd.Long, "Check for broken internal links")
	
	// Should have flags
	assert.NotNil(t, cmd.Flags().Lookup("ignore"))
	assert.NotNil(t, cmd.Flags().Lookup("file-relative"))
}
