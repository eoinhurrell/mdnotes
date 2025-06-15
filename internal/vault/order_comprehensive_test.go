package vault

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractFieldOrder_ComplexKeys(t *testing.T) {
	// Test field order extraction with various key formats
	yamlContent := `id: 95356843-4ceb-46af-b307-6593755b91a0
title: Fast Food Nation Readthrough
date created: 2009-03-21T12:00:00Z
date modified: 2025-06-03T02:43:17Z
"quoted key": value
'single quoted': value  
end:
rating: 3
start: "2009-03-21"
tags: "[]"
note_type:
  - readthrough`

	order := extractFieldOrder(yamlContent)
	
	expected := []string{
		"id", 
		"title", 
		"date created", 
		"date modified", 
		"quoted key", 
		"single quoted", 
		"end", 
		"rating", 
		"start", 
		"tags", 
		"note_type",
	}
	
	assert.Equal(t, expected, order, "Field order should be preserved exactly")
}

func TestVaultFile_OrderPreservationWithModifications(t *testing.T) {
	// Test that order is preserved when modifying existing fields
	content := `---
id: 95356843-4ceb-46af-b307-6593755b91a0
title: Fast Food Nation Readthrough
date created: 2009-03-21T12:00:00Z
date modified: 2025-06-03T02:43:17Z
end:
rating: 3
start: "2009-03-21"
tags: "[]"
note_type:
  - readthrough
---

# Content
`

	vf := &VaultFile{}
	err := vf.Parse([]byte(content))
	require.NoError(t, err)

	// Modify multiple existing fields
	vf.SetField("rating", 5)
	vf.SetField("start", "2009-03-22")
	vf.SetField("title", "Updated Title")

	// Serialize and verify order is preserved
	serialized, err := vf.Serialize()
	require.NoError(t, err)

	serializedStr := string(serialized)
	
	// Check that fields appear in the correct order
	idIndex := strings.Index(serializedStr, "id:")
	titleIndex := strings.Index(serializedStr, "title:")
	dateCreatedIndex := strings.Index(serializedStr, "date created:")
	dateModifiedIndex := strings.Index(serializedStr, "date modified:")
	endIndex := strings.Index(serializedStr, "end:")
	ratingIndex := strings.Index(serializedStr, "rating:")
	startIndex := strings.Index(serializedStr, "start:")
	tagsIndex := strings.Index(serializedStr, "tags:")
	noteTypeIndex := strings.Index(serializedStr, "note_type:")

	assert.True(t, idIndex < titleIndex, "id should come before title")
	assert.True(t, titleIndex < dateCreatedIndex, "title should come before date created")
	assert.True(t, dateCreatedIndex < dateModifiedIndex, "date created should come before date modified")
	assert.True(t, dateModifiedIndex < endIndex, "date modified should come before end")
	assert.True(t, endIndex < ratingIndex, "end should come before rating")
	assert.True(t, ratingIndex < startIndex, "rating should come before start")
	assert.True(t, startIndex < tagsIndex, "start should come before tags")
	assert.True(t, tagsIndex < noteTypeIndex, "tags should come before note_type")

	// Verify the modified values are present
	assert.Contains(t, serializedStr, "rating: 5")
	assert.Contains(t, serializedStr, "2009-03-22") // The date value itself
	assert.Contains(t, serializedStr, "title: Updated Title")
}