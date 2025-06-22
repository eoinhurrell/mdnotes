package processor

import (
	"testing"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

func TestLinkUpdater_UpdateReferences(t *testing.T) {
	tests := []struct {
		name    string
		content string
		moves   []FileMove
		want    string
	}{
		{
			name:    "update wiki link",
			content: "See [[old/path/note]] for more info",
			moves: []FileMove{
				{From: "old/path/note.md", To: "new/location/note.md"},
			},
			want: "See [[new/location/note]] for more info",
		},
		{
			name:    "update markdown link",
			content: "See [text](old/path/note.md) for more info",
			moves: []FileMove{
				{From: "old/path/note.md", To: "new/location/note.md"},
			},
			want: "See [text](new/location/note.md) for more info",
		},
		{
			name:    "update embed",
			content: "![[old/path/image.png]]",
			moves: []FileMove{
				{From: "old/path/image.png", To: "assets/image.png"},
			},
			want: "![[assets/image.png]]",
		},
		{
			name:    "multiple updates",
			content: "See [[note1]] and [link](note2.md) and ![[image.png]]",
			moves: []FileMove{
				{From: "note1.md", To: "folder/note1.md"},
				{From: "note2.md", To: "folder/note2.md"},
				{From: "image.png", To: "assets/image.png"},
			},
			want: "See [[folder/note1]] and [link](folder/note2.md) and ![[assets/image.png]]",
		},
		{
			name:    "no matching links",
			content: "See [[other]] and [text](different.md)",
			moves: []FileMove{
				{From: "note.md", To: "folder/note.md"},
			},
			want: "See [[other]] and [text](different.md)",
		},
		{
			name:    "wiki link with alias",
			content: "See [[old/note|Custom Text]] for more",
			moves: []FileMove{
				{From: "old/note.md", To: "new/note.md"},
			},
			want: "See [[new/note|Custom Text]] for more",
		},
		{
			name:    "vault-relative path handling",
			content: "See [[folder/note]] and [link](resources/test.md)",
			moves: []FileMove{
				{From: "folder/note.md", To: "new-folder/note.md"},
				{From: "resources/test.md", To: "resources/renamed-test.md"},
			},
			want: "See [[new-folder/note]] and [link](resources/renamed-test.md)",
		},
		{
			name:    "subdirectory markdown links",
			content: "Reference [file](resources/docs/readme.md) and [other](utils/helper.md)",
			moves: []FileMove{
				{From: "resources/docs/readme.md", To: "documentation/readme.md"},
				{From: "utils/helper.md", To: "lib/utils.md"},
			},
			want: "Reference [file](documentation/readme.md) and [other](lib/utils.md)",
		},
		{
			name:    "URL encoded links",
			content: "- [Blood's Hiding](resources/books/20250527111132-Blood's%20Hiding.md), Ken Baumann",
			moves: []FileMove{
				{From: "resources/books/20250527111132-Blood's Hiding.md", To: "resources/books/20250527111132-renamed.md"},
			},
			want: "- [Blood's Hiding](resources/books/20250527111132-renamed.md), Ken Baumann",
		},
		{
			name:    "URL encoded links with spaces",
			content: "See [file with spaces](folder/file%20with%20spaces.md) here",
			moves: []FileMove{
				{From: "folder/file with spaces.md", To: "folder/renamed file.md"},
			},
			want: "See [file with spaces](folder/renamed%20file.md) here",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updater := NewLinkUpdater()
			got := updater.UpdateReferences(tt.content, tt.moves)
			if got != tt.want {
				t.Errorf("UpdateReferences() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLinkUpdater_UpdateFile(t *testing.T) {
	tests := []struct {
		name     string
		file     *vault.VaultFile
		moves    []FileMove
		wantBody string
		modified bool
	}{
		{
			name: "update file with links",
			file: &vault.VaultFile{
				Body: "# Note\n\nSee [[other]] and [link](file.md)",
			},
			moves: []FileMove{
				{From: "other.md", To: "folder/other.md"},
				{From: "file.md", To: "folder/file.md"},
			},
			wantBody: "# Note\n\nSee [[folder/other]] and [link](folder/file.md)",
			modified: true,
		},
		{
			name: "no updates needed",
			file: &vault.VaultFile{
				Body: "# Note\n\nNo links to update",
			},
			moves: []FileMove{
				{From: "other.md", To: "folder/other.md"},
			},
			wantBody: "# Note\n\nNo links to update",
			modified: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updater := NewLinkUpdater()
			modified := updater.UpdateFile(tt.file, tt.moves)

			if modified != tt.modified {
				t.Errorf("UpdateFile() modified = %v, want %v", modified, tt.modified)
			}

			if tt.file.Body != tt.wantBody {
				t.Errorf("UpdateFile() body = %q, want %q", tt.file.Body, tt.wantBody)
			}
		})
	}
}

func TestLinkUpdater_NormalizePaths(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		linkType LinkType
		want     string
	}{
		{"wiki link without extension", "note", WikiLink, "note.md"},
		{"wiki link with extension", "note.md", WikiLink, "note.md"},
		{"markdown link", "note.md", MarkdownLink, "note.md"},
		{"embed link", "image.png", EmbedLink, "image.png"},
		{"path with folder", "folder/note", WikiLink, "folder/note.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updater := NewLinkUpdater()
			got := updater.normalizeLinkTarget(tt.target, tt.linkType)
			if got != tt.want {
				t.Errorf("normalizeLinkTarget(%q, %v) = %q, want %q", tt.target, tt.linkType, got, tt.want)
			}
		})
	}
}

func TestLinkUpdater_CreateMoveMap(t *testing.T) {
	moves := []FileMove{
		{From: "note.md", To: "folder/note.md"},
		{From: "other.md", To: "different.md"},
		{From: "image.png", To: "assets/image.png"},
	}

	updater := NewLinkUpdater()
	moveMap := updater.createMoveMap(moves)

	tests := []struct {
		from string
		want string
		ok   bool
	}{
		{"note.md", "folder/note.md", true},
		{"other.md", "different.md", true},
		{"image.png", "assets/image.png", true},
		{"nonexistent.md", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.from, func(t *testing.T) {
			got, ok := moveMap[tt.from]
			if ok != tt.ok {
				t.Errorf("moveMap[%q] ok = %v, want %v", tt.from, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("moveMap[%q] = %q, want %q", tt.from, got, tt.want)
			}
		})
	}
}

func TestLinkUpdater_UpdateBatch(t *testing.T) {
	files := []*vault.VaultFile{
		{
			Path: "file1.md",
			Body: "See [[note]] and [link](other.md)",
		},
		{
			Path: "file2.md",
			Body: "Another [[note]] reference",
		},
		{
			Path: "file3.md",
			Body: "No relevant links here",
		},
	}

	moves := []FileMove{
		{From: "note.md", To: "folder/note.md"},
		{From: "other.md", To: "folder/other.md"},
	}

	updater := NewLinkUpdater()
	results := updater.UpdateBatch(files, moves)

	// Check results
	if len(results) != 2 {
		t.Errorf("UpdateBatch() returned %d modified files, want 2", len(results))
	}

	// Check first file was updated
	expected1 := "See [[folder/note]] and [link](folder/other.md)"
	if files[0].Body != expected1 {
		t.Errorf("File1 body = %q, want %q", files[0].Body, expected1)
	}

	// Check second file was updated
	expected2 := "Another [[folder/note]] reference"
	if files[1].Body != expected2 {
		t.Errorf("File2 body = %q, want %q", files[1].Body, expected2)
	}

	// Check third file was not changed
	expected3 := "No relevant links here"
	if files[2].Body != expected3 {
		t.Errorf("File3 body = %q, want %q", files[2].Body, expected3)
	}
}
