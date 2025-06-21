package vault

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVaultFile_PreserveFrontmatterOrder(t *testing.T) {
	// Test that frontmatter field order is preserved when modifying files
	content := `---
title: My Note
created: 2023-01-15
tags: [personal, work]
priority: 5
published: false
---

# My Note

This is the content of my note.
`

	vf := &VaultFile{}
	err := vf.Parse([]byte(content))
	require.NoError(t, err)

	// Verify original fields are parsed correctly
	assert.Equal(t, "My Note", vf.Frontmatter["title"])
	// Note: YAML parser converts date-like strings to time.Time objects
	_, ok := vf.Frontmatter["created"]
	assert.True(t, ok, "created field should exist")
	assert.Equal(t, []interface{}{"personal", "work"}, vf.Frontmatter["tags"])
	assert.Equal(t, 5, vf.Frontmatter["priority"])
	assert.Equal(t, false, vf.Frontmatter["published"])

	// Verify field order is preserved
	expectedOrder := []string{"title", "created", "tags", "priority", "published"}
	assert.Equal(t, expectedOrder, vf.frontmatterOrder)

	// Modify an existing field
	vf.SetField("priority", 10)

	// Add a new field
	vf.SetField("modified", "2023-01-16")

	// Serialize and check that order is preserved
	serialized, err := vf.Serialize()
	require.NoError(t, err)

	// Check that fields appear in the expected order by looking at the full serialized content
	serializedStr := string(serialized)

	// Check that the fields appear in order in the serialized content
	titleIndex := strings.Index(serializedStr, "title:")
	createdIndex := strings.Index(serializedStr, "created:")
	tagsIndex := strings.Index(serializedStr, "tags:")
	priorityIndex := strings.Index(serializedStr, "priority:")
	publishedIndex := strings.Index(serializedStr, "published:")
	modifiedIndex := strings.Index(serializedStr, "modified:")

	// Verify order is preserved (original fields first, then new field)
	assert.True(t, titleIndex < createdIndex, "title should come before created")
	assert.True(t, createdIndex < tagsIndex, "created should come before tags")
	assert.True(t, tagsIndex < priorityIndex, "tags should come before priority")
	assert.True(t, priorityIndex < publishedIndex, "priority should come before published")
	assert.True(t, publishedIndex < modifiedIndex, "published should come before modified (new field)")

	// Check that modified value was updated
	assert.Contains(t, serializedStr, "priority: 10")
	assert.Contains(t, serializedStr, "modified:")
	assert.Contains(t, serializedStr, "2023-01-16")
}

func TestVaultFile_NewFieldsAtEnd(t *testing.T) {
	// Test that new fields are added at the end in sorted order
	content := `---
title: Test
id: test-123
---

Content here.
`

	vf := &VaultFile{}
	err := vf.Parse([]byte(content))
	require.NoError(t, err)

	// Add multiple new fields
	vf.SetField("zebra", "last")
	vf.SetField("alpha", "first")
	vf.SetField("beta", "second")

	serialized, err := vf.Serialize()
	require.NoError(t, err)

	serializedStr := string(serialized)

	// Check order: original fields first, then new fields in alphabetical order
	titleIndex := strings.Index(serializedStr, "title:")
	idIndex := strings.Index(serializedStr, "id:")
	alphaIndex := strings.Index(serializedStr, "alpha:")
	betaIndex := strings.Index(serializedStr, "beta:")
	zebraIndex := strings.Index(serializedStr, "zebra:")

	// Verify original fields come first
	assert.True(t, titleIndex < idIndex, "title should come before id")
	assert.True(t, idIndex < alphaIndex, "id should come before alpha (first new field)")

	// Verify new fields are in alphabetical order
	assert.True(t, alphaIndex < betaIndex, "alpha should come before beta")
	assert.True(t, betaIndex < zebraIndex, "beta should come before zebra")
}

func TestVaultFile_ComplexYAMLValues(t *testing.T) {
	// Test that complex YAML values (arrays, objects) preserve order
	content := `---
title: Complex Note
authors:
  - Alice
  - Bob
metadata:
  status: draft
  version: 1.0
tags: [yaml, test]
---

Content.
`

	vf := &VaultFile{}
	err := vf.Parse([]byte(content))
	require.NoError(t, err)

	// Modify complex field
	vf.Frontmatter["authors"] = []string{"Alice", "Bob", "Charlie"}

	// Add new complex field
	vf.SetField("config", map[string]interface{}{
		"enabled": true,
		"count":   42,
	})

	serialized, err := vf.Serialize()
	require.NoError(t, err)

	// Verify the structure is maintained and order is preserved
	assert.Contains(t, string(serialized), "title: Complex Note")
	assert.Contains(t, string(serialized), "authors:")
	assert.Contains(t, string(serialized), "- Alice")
	assert.Contains(t, string(serialized), "- Charlie") // Modified content
	assert.Contains(t, string(serialized), "metadata:")
	assert.Contains(t, string(serialized), "tags:")
	assert.Contains(t, string(serialized), "config:") // New field

	// Check that title comes before authors, authors before metadata, etc.
	titleIndex := strings.Index(string(serialized), "title:")
	authorsIndex := strings.Index(string(serialized), "authors:")
	metadataIndex := strings.Index(string(serialized), "metadata:")
	tagsIndex := strings.Index(string(serialized), "tags:")
	configIndex := strings.Index(string(serialized), "config:")

	assert.True(t, titleIndex < authorsIndex, "title should come before authors")
	assert.True(t, authorsIndex < metadataIndex, "authors should come before metadata")
	assert.True(t, metadataIndex < tagsIndex, "metadata should come before tags")
	assert.True(t, tagsIndex < configIndex, "tags should come before config (new field)")
}

func TestVaultFile_EmptyFrontmatter(t *testing.T) {
	// Test behavior with empty frontmatter
	content := `---
---

Just content.
`

	vf := &VaultFile{}
	err := vf.Parse([]byte(content))
	require.NoError(t, err)

	// Add fields to empty frontmatter
	vf.SetField("title", "New Title")
	vf.SetField("created", "2023-01-15")

	serialized, err := vf.Serialize()
	require.NoError(t, err)

	// Should create proper frontmatter
	assert.Contains(t, string(serialized), "---")
	assert.Contains(t, string(serialized), "title: New Title")
	assert.Contains(t, string(serialized), "created:")
	assert.Contains(t, string(serialized), "2023-01-15")
	assert.Contains(t, string(serialized), "Just content.")
}

func TestVaultFile_NoFrontmatter(t *testing.T) {
	// Test behavior with no frontmatter
	content := `# Just Content

No frontmatter here.
`

	vf := &VaultFile{}
	err := vf.Parse([]byte(content))
	require.NoError(t, err)

	// Add fields
	vf.SetField("title", "Added Title")
	vf.SetField("created", "2023-01-15")

	serialized, err := vf.Serialize()
	require.NoError(t, err)

	// Should create frontmatter section
	assert.Contains(t, string(serialized), "---")
	assert.Contains(t, string(serialized), "title: Added Title")
	assert.Contains(t, string(serialized), "created:")
	assert.Contains(t, string(serialized), "2023-01-15")
	assert.Contains(t, string(serialized), "# Just Content")
}
