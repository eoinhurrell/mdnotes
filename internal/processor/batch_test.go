package processor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/eoinhurrell/mdnotes/internal/vault"
)

func TestBatchProcessor_Execute(t *testing.T) {
	// Create test vault with files
	testVault := createTestVault(t)

	// Create test batch config
	config := BatchConfig{
		StopOnError: false,
		Operations: []Operation{
			{
				Name:    "Ensure tags",
				Command: "frontmatter.ensure",
				Parameters: map[string]interface{}{
					"field":   "tags",
					"default": []string{},
				},
			},
			{
				Name:    "Cast types",
				Command: "frontmatter.cast",
				Parameters: map[string]interface{}{
					"field": "priority",
					"type":  "number",
				},
			},
		},
	}

	processor := NewBatchProcessor()
	results, err := processor.Execute(context.Background(), testVault, config)

	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.True(t, results[0].Success)
	assert.True(t, results[1].Success)
	assert.NotZero(t, results[0].Duration)
}

func TestBatchProcessor_ExecuteWithError(t *testing.T) {
	testVault := createTestVault(t)

	config := BatchConfig{
		StopOnError: true,
		Operations: []Operation{
			{
				Name:    "Valid operation",
				Command: "frontmatter.ensure",
				Parameters: map[string]interface{}{
					"field":   "tags",
					"default": []string{},
				},
			},
			{
				Name:    "Invalid operation",
				Command: "invalid.command",
				Parameters: map[string]interface{}{},
			},
		},
	}

	processor := NewBatchProcessor()
	results, err := processor.Execute(context.Background(), testVault, config)

	assert.Error(t, err)
	assert.Len(t, results, 2)
	assert.True(t, results[0].Success)
	assert.False(t, results[1].Success)
	assert.NotNil(t, results[1].Error)
}

func TestBatchProcessor_ContinueOnError(t *testing.T) {
	testVault := createTestVault(t)

	config := BatchConfig{
		StopOnError: false,
		Operations: []Operation{
			{
				Name:    "Invalid operation",
				Command: "invalid.command",
				Parameters: map[string]interface{}{},
			},
			{
				Name:    "Valid operation",
				Command: "frontmatter.ensure",
				Parameters: map[string]interface{}{
					"field":   "tags",
					"default": []string{},
				},
			},
		},
	}

	processor := NewBatchProcessor()
	results, err := processor.Execute(context.Background(), testVault, config)

	assert.NoError(t, err) // Should not error when StopOnError is false
	assert.Len(t, results, 2)
	assert.False(t, results[0].Success)
	assert.True(t, results[1].Success)
}

func TestBatchProcessor_ContextCancellation(t *testing.T) {
	testVault := createTestVault(t)

	config := BatchConfig{
		Operations: []Operation{
			{
				Name:    "Slow operation",
				Command: "test.slow",
				Parameters: map[string]interface{}{
					"duration": "100ms",
				},
			},
			{
				Name:    "Another operation",
				Command: "frontmatter.ensure",
				Parameters: map[string]interface{}{
					"field":   "tags",
					"default": []string{},
				},
			},
		},
	}

	processor := NewBatchProcessor()
	
	// Register a slow test processor
	processor.RegisterProcessor("test.slow", &SlowTestProcessor{})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := processor.Execute(ctx, testVault, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestBatchProcessor_Backup(t *testing.T) {
	testVault := createTestVault(t)

	config := BatchConfig{
		CreateBackup: true,
		Operations: []Operation{
			{
				Name:    "Modify files",
				Command: "frontmatter.ensure",
				Parameters: map[string]interface{}{
					"field":   "modified",
					"default": true,
				},
			},
		},
	}

	processor := NewBatchProcessor()
	results, err := processor.Execute(context.Background(), testVault, config)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.True(t, results[0].Success)
	
	// Check that backup was created
	assert.NotNil(t, processor.lastBackup)
	assert.NotEmpty(t, processor.lastBackup.ID)
}

func TestBatchProcessor_DryRun(t *testing.T) {
	testVault := createTestVault(t)

	config := BatchConfig{
		DryRun: true,
		Operations: []Operation{
			{
				Name:    "Ensure tags",
				Command: "frontmatter.ensure",
				Parameters: map[string]interface{}{
					"field":   "tags",
					"default": []string{},
				},
			},
		},
	}

	processor := NewBatchProcessor()
	results, err := processor.Execute(context.Background(), testVault, config)

	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.True(t, results[0].Success)
	assert.Contains(t, results[0].Message, "would")
}

func TestBatchProcessor_RegisterProcessor(t *testing.T) {
	processor := NewBatchProcessor()
	testProc := &TestProcessor{}

	processor.RegisterProcessor("test.custom", testProc)

	// Verify it was registered
	registered, exists := processor.processors["test.custom"]
	assert.True(t, exists)
	assert.Equal(t, testProc, registered)
}

// Helper function to create a test vault
func createTestVault(t *testing.T) *Vault {
	files := []*vault.VaultFile{
		{
			Path: "test1.md",
			Frontmatter: map[string]interface{}{
				"title":    "Test Note 1",
				"priority": "5",
			},
			Body: "# Test Note 1\n\nContent here.",
		},
		{
			Path: "test2.md",
			Frontmatter: map[string]interface{}{
				"title": "Test Note 2",
			},
			Body: "# Test Note 2\n\nMore content.",
		},
	}

	return &Vault{
		Files: files,
		Path:  "/test/vault",
	}
}

// TestProcessor for testing
type TestProcessor struct {
	executed bool
}

func (tp *TestProcessor) Process(ctx context.Context, vault *Vault, params map[string]interface{}) error {
	tp.executed = true
	return nil
}

func (tp *TestProcessor) Name() string {
	return "test"
}

// SlowTestProcessor for testing cancellation
type SlowTestProcessor struct{}

func (stp *SlowTestProcessor) Process(ctx context.Context, vault *Vault, params map[string]interface{}) error {
	durationStr, ok := params["duration"].(string)
	if !ok {
		durationStr = "100ms"
	}
	
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return err
	}

	select {
	case <-time.After(duration):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (stp *SlowTestProcessor) Name() string {
	return "slow"
}

func TestOperationResult_String(t *testing.T) {
	tests := []struct {
		name   string
		result OperationResult
		want   string
	}{
		{
			name: "successful operation",
			result: OperationResult{
				Operation: "test.operation",
				Success:   true,
				Duration:  100 * time.Millisecond,
				Message:   "Operation completed",
			},
			want: "✓ test.operation (100ms): Operation completed",
		},
		{
			name: "failed operation",
			result: OperationResult{
				Operation: "test.operation",
				Success:   false,
				Duration:  50 * time.Millisecond,
				Error:     assert.AnError,
				Message:   "Operation failed",
			},
			want: "✗ test.operation (50ms): Operation failed - assert.AnError general error for testing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.String()
			assert.Equal(t, tt.want, got)
		})
	}
}