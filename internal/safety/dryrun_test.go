package safety

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDryRun_Operations(t *testing.T) {
	dryRun := NewDryRunRecorder()

	// Record operations without executing
	dryRun.Record(Operation{
		Type: "frontmatter.ensure",
		File: "test.md",
		Changes: []Change{
			{Field: "tags", NewValue: []string{}, Action: "add"},
		},
		Description: "Add tags field to test.md",
	})

	dryRun.Record(Operation{
		Type: "headings.fix",
		File: "another.md",
		Changes: []Change{
			{Field: "heading", OldValue: "## Wrong Level", NewValue: "# Correct Level", Action: "modify"},
		},
		Description: "Fix heading level in another.md",
	})

	// Generate report
	report := dryRun.GenerateReport()
	assert.Contains(t, report, "Would add field 'tags'")
	assert.Contains(t, report, "test.md")
	assert.Contains(t, report, "another.md")
	assert.Contains(t, report, "2 operations would be performed")

	// Check operation count
	assert.Equal(t, 2, dryRun.OperationCount())

	// Check specific operations
	operations := dryRun.GetOperations()
	assert.Len(t, operations, 2)
	assert.Equal(t, "frontmatter.ensure", operations[0].Type)
	assert.Equal(t, "headings.fix", operations[1].Type)
}

func TestDryRun_MultipleChanges(t *testing.T) {
	dryRun := NewDryRunRecorder()

	// Record operation with multiple changes
	dryRun.Record(Operation{
		Type: "frontmatter.cast",
		File: "complex.md",
		Changes: []Change{
			{Field: "priority", OldValue: "5", NewValue: 5, Action: "cast"},
			{Field: "created", OldValue: "2023-01-01", NewValue: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Action: "cast"},
			{Field: "tags", OldValue: "tag1, tag2", NewValue: []string{"tag1", "tag2"}, Action: "cast"},
		},
		Description: "Cast field types in complex.md",
	})

	report := dryRun.GenerateReport()
	assert.Contains(t, report, "Would cast field 'priority'")
	assert.Contains(t, report, "Would cast field 'created'")
	assert.Contains(t, report, "Would cast field 'tags'")
	assert.Contains(t, report, "Total changes: 3")
}

func TestDryRun_EmptyRecorder(t *testing.T) {
	dryRun := NewDryRunRecorder()

	report := dryRun.GenerateReport()
	assert.Contains(t, report, "No operations would be performed")
	assert.Equal(t, 0, dryRun.OperationCount())
	assert.Empty(t, dryRun.GetOperations())
}

func TestDryRun_Clear(t *testing.T) {
	dryRun := NewDryRunRecorder()

	// Add some operations
	dryRun.Record(Operation{
		Type: "test",
		File: "test.md",
		Changes: []Change{
			{Field: "test", Action: "add"},
		},
	})

	assert.Equal(t, 1, dryRun.OperationCount())

	// Clear
	dryRun.Clear()
	assert.Equal(t, 0, dryRun.OperationCount())
	assert.Empty(t, dryRun.GetOperations())
}

func TestDryRun_JSONReport(t *testing.T) {
	dryRun := NewDryRunRecorder()

	dryRun.Record(Operation{
		Type: "frontmatter.ensure",
		File: "test.md",
		Changes: []Change{
			{Field: "tags", NewValue: []string{}, Action: "add"},
		},
		Description:       "Add tags field",
		EstimatedDuration: 100 * time.Millisecond,
	})

	jsonReport := dryRun.GenerateJSONReport()
	assert.Contains(t, jsonReport, `"type": "frontmatter.ensure"`)
	assert.Contains(t, jsonReport, `"file": "test.md"`)
	assert.Contains(t, jsonReport, `"action": "add"`)
	assert.Contains(t, jsonReport, `"field": "tags"`)
}

func TestDryRun_SummaryStats(t *testing.T) {
	dryRun := NewDryRunRecorder()

	// Add various operations
	dryRun.Record(Operation{
		Type:    "frontmatter.ensure",
		File:    "file1.md",
		Changes: []Change{{Field: "tags", Action: "add"}},
	})

	dryRun.Record(Operation{
		Type:    "frontmatter.ensure",
		File:    "file2.md",
		Changes: []Change{{Field: "title", Action: "add"}},
	})

	dryRun.Record(Operation{
		Type:    "headings.fix",
		File:    "file3.md",
		Changes: []Change{{Field: "heading", Action: "modify"}},
	})

	stats := dryRun.GetSummaryStats()
	assert.Equal(t, 3, stats.TotalOperations)
	assert.Equal(t, 3, stats.FilesAffected)
	assert.Equal(t, 3, stats.TotalChanges)
	assert.Equal(t, map[string]int{
		"frontmatter.ensure": 2,
		"headings.fix":       1,
	}, stats.OperationTypes)
	assert.Equal(t, map[string]int{
		"add":    2,
		"modify": 1,
	}, stats.ChangeTypes)
}

func TestDryRun_FilterOperations(t *testing.T) {
	dryRun := NewDryRunRecorder()

	// Add operations for different files
	dryRun.Record(Operation{
		Type:    "frontmatter.ensure",
		File:    "important.md",
		Changes: []Change{{Field: "tags", Action: "add"}},
	})

	dryRun.Record(Operation{
		Type:    "headings.fix",
		File:    "other.md",
		Changes: []Change{{Field: "heading", Action: "modify"}},
	})

	dryRun.Record(Operation{
		Type:    "frontmatter.cast",
		File:    "important.md",
		Changes: []Change{{Field: "priority", Action: "cast"}},
	})

	// Filter by file
	importantOps := dryRun.GetOperationsForFile("important.md")
	assert.Len(t, importantOps, 2)
	assert.Equal(t, "frontmatter.ensure", importantOps[0].Type)
	assert.Equal(t, "frontmatter.cast", importantOps[1].Type)

	// Filter by type
	frontmatterOps := dryRun.GetOperationsByType("frontmatter.ensure")
	assert.Len(t, frontmatterOps, 1)
	assert.Equal(t, "important.md", frontmatterOps[0].File)
}

func TestChange_String(t *testing.T) {
	tests := []struct {
		name   string
		change Change
		want   string
	}{
		{
			name: "add action",
			change: Change{
				Field:    "tags",
				NewValue: []string{"test"},
				Action:   "add",
			},
			want: "add field 'tags'",
		},
		{
			name: "modify action",
			change: Change{
				Field:    "title",
				OldValue: "Old Title",
				NewValue: "New Title",
				Action:   "modify",
			},
			want: "modify field 'title' from 'Old Title' to 'New Title'",
		},
		{
			name: "remove action",
			change: Change{
				Field:    "deprecated",
				OldValue: "old value",
				Action:   "remove",
			},
			want: "remove field 'deprecated'",
		},
		{
			name: "cast action",
			change: Change{
				Field:    "priority",
				OldValue: "5",
				NewValue: 5,
				Action:   "cast",
			},
			want: "cast field 'priority' from string to number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.change.String()
			assert.Contains(t, got, tt.want)
		})
	}
}
