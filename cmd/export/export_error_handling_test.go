package export

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/eoinhurrell/mdnotes/internal/processor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ExportError
		expected string
	}{
		{
			name: "Error without cause",
			err: &ExportError{
				Type:    ErrInvalidInput,
				Message: "Invalid input provided",
			},
			expected: "Invalid input provided",
		},
		{
			name: "Error with cause",
			err: &ExportError{
				Type:    ErrFileSystem,
				Message: "Cannot access file",
				Cause:   os.ErrNotExist,
			},
			expected: "Cannot access file: file does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestExportError_Unwrap(t *testing.T) {
	cause := os.ErrPermission
	err := &ExportError{
		Type:    ErrPermission,
		Message: "Permission denied",
		Cause:   cause,
	}

	assert.Equal(t, cause, err.Unwrap())
}

func TestNewExportError(t *testing.T) {
	err := NewExportError(ErrInvalidInput, "Test message")

	assert.Equal(t, ErrInvalidInput, err.Type)
	assert.Equal(t, "Test message", err.Message)
	assert.Nil(t, err.Cause)
}

func TestNewExportErrorWithCause(t *testing.T) {
	cause := os.ErrNotExist
	err := NewExportErrorWithCause(ErrFileSystem, "Test message", cause)

	assert.Equal(t, ErrFileSystem, err.Type)
	assert.Equal(t, "Test message", err.Message)
	assert.Equal(t, cause, err.Cause)
}

func TestValidateExportInputs(t *testing.T) {
	tests := []struct {
		name         string
		outputPath   string
		vaultPath    string
		query        string
		linkStrategy string
		processLinks bool
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "Valid inputs",
			outputPath:   "./test-output",
			vaultPath:    "./test/test-vault",
			query:        "",
			linkStrategy: "remove",
			processLinks: true,
			expectError:  false,
		},
		{
			name:         "Empty output path",
			outputPath:   "",
			vaultPath:    "/tmp/test/test-vault",
			query:        "",
			linkStrategy: "remove",
			processLinks: true,
			expectError:  true,
			errorMsg:     "output path cannot be empty",
		},
		{
			name:         "Empty vault path",
			outputPath:   "./test-output",
			vaultPath:    "",
			query:        "",
			linkStrategy: "remove",
			processLinks: true,
			expectError:  true,
			errorMsg:     "vault path cannot be empty",
		},
		{
			name:         "Invalid link strategy",
			outputPath:   "./test-output",
			vaultPath:    "./test/test-vault",
			query:        "",
			linkStrategy: "invalid",
			processLinks: true,
			expectError:  true,
			errorMsg:     "invalid link strategy 'invalid'",
		},
		{
			name:         "Invalid query with unmatched quotes",
			outputPath:   "./test-output",
			vaultPath:    "./test/test-vault",
			query:        "title = \"unclosed quote",
			linkStrategy: "remove",
			processLinks: true,
			expectError:  true,
			errorMsg:     "unmatched quotes",
		},
		{
			name:         "Invalid query with backslashes",
			outputPath:   "./test-output",
			vaultPath:    "./test/test-vault",
			query:        "title = \"test\\backslash\"",
			linkStrategy: "remove",
			processLinks: true,
			expectError:  true,
			errorMsg:     "backslashes are not supported",
		},
		{
			name:         "Unsafe output path - system directory",
			outputPath:   "/usr/bin/test",
			vaultPath:    "./test/test-vault",
			query:        "",
			linkStrategy: "remove",
			processLinks: true,
			expectError:  true,
			errorMsg:     "unsafe output path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExportInputs(tt.outputPath, tt.vaultPath, tt.query, tt.linkStrategy, tt.processLinks)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateQuerySyntax(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid query",
			query:       "title = \"test\"",
			expectError: false,
		},
		{
			name:        "Valid complex query",
			query:       "tags contains \"work\" AND created >= \"2024-01-01\"",
			expectError: false,
		},
		{
			name:        "Empty query",
			query:       "",
			expectError: true,
			errorMsg:    "query cannot be empty",
		},
		{
			name:        "Whitespace only query",
			query:       "   ",
			expectError: true,
			errorMsg:    "query cannot be empty",
		},
		{
			name:        "Query with backslashes",
			query:       "title = \"test\\value\"",
			expectError: true,
			errorMsg:    "backslashes are not supported",
		},
		{
			name:        "Query with unmatched quotes",
			query:       "title = \"unclosed",
			expectError: true,
			errorMsg:    "unmatched quotes",
		},
		{
			name:        "Query with odd number of quotes",
			query:       "title = \"test\" AND description = \"another",
			expectError: true,
			errorMsg:    "unmatched quotes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateQuerySyntax(tt.query)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateOutputPathSafety(t *testing.T) {
	tests := []struct {
		name        string
		outputPath  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Safe path in temp directory",
			outputPath:  "/tmp/export",
			expectError: false,
		},
		{
			name:        "Safe path in current directory",
			outputPath:  "./export",
			expectError: false,
		},
		{
			name:        "Unsafe path - usr bin",
			outputPath:  "/usr/bin",
			expectError: true,
			errorMsg:    "cannot write to system directory",
		},
		{
			name:        "Unsafe path - usr bin",
			outputPath:  "/usr/bin",
			expectError: true,
			errorMsg:    "cannot write to system directory",
		},
		{
			name:        "Unsafe path - system library",
			outputPath:  "/System/Library",
			expectError: true,
			errorMsg:    "cannot write to system directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOutputPathSafety(tt.outputPath)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAndResolvePaths(t *testing.T) {
	// Create temporary directories for testing
	tempDir := t.TempDir()
	vaultDir := filepath.Join(tempDir, "vault")
	outputDir := filepath.Join(tempDir, "output")
	nonEmptyDir := filepath.Join(tempDir, "nonempty")

	require.NoError(t, os.MkdirAll(vaultDir, 0755))
	require.NoError(t, os.MkdirAll(nonEmptyDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(nonEmptyDir, "test.txt"), []byte("test"), 0644))

	tests := []struct {
		name        string
		vaultPath   string
		outputPath  string
		dryRun      bool
		expectError bool
		errorType   ExportErrorType
		errorMsg    string
	}{
		{
			name:        "Valid paths",
			vaultPath:   vaultDir,
			outputPath:  outputDir,
			dryRun:      false,
			expectError: false,
		},
		{
			name:        "Valid paths with dry run on non-empty output",
			vaultPath:   vaultDir,
			outputPath:  nonEmptyDir,
			dryRun:      true,
			expectError: false,
		},
		{
			name:        "Non-existent vault path",
			vaultPath:   filepath.Join(tempDir, "nonexistent"),
			outputPath:  outputDir,
			dryRun:      false,
			expectError: true,
			errorType:   ErrFileSystem,
			errorMsg:    "does not exist",
		},
		{
			name:        "Vault path is not directory",
			vaultPath:   filepath.Join(nonEmptyDir, "test.txt"),
			outputPath:  outputDir,
			dryRun:      false,
			expectError: true,
			errorType:   ErrInvalidInput,
			errorMsg:    "not a directory",
		},
		{
			name:        "Output directory not empty (non-dry run)",
			vaultPath:   vaultDir,
			outputPath:  nonEmptyDir,
			dryRun:      false,
			expectError: true,
			errorType:   ErrInvalidInput,
			errorMsg:    "not empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vaultAbs, outputAbs, err := validateAndResolvePaths(tt.vaultPath, tt.outputPath, tt.dryRun)

			if tt.expectError {
				assert.Error(t, err)
				var exportErr *ExportError
				assert.ErrorAs(t, err, &exportErr)
				assert.Equal(t, tt.errorType, exportErr.Type)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.True(t, filepath.IsAbs(vaultAbs))
				assert.True(t, filepath.IsAbs(outputAbs))
			}
		})
	}
}

func TestHandleExportError(t *testing.T) {
	options := processor.ExportOptions{
		Query: "test query",
	}

	tests := []struct {
		name         string
		inputError   error
		expectedType ExportErrorType
		expectedMsg  string
	}{
		{
			name:         "Context cancelled",
			inputError:   context.Canceled,
			expectedType: ErrCancellation,
			expectedMsg:  "cancelled",
		},
		{
			name:         "Context deadline exceeded",
			inputError:   context.DeadlineExceeded,
			expectedType: ErrCancellation,
			expectedMsg:  "timed out",
		},
		{
			name:         "Query parsing error",
			inputError:   mockQueryError{msg: "syntax error"},
			expectedType: ErrQuery,
			expectedMsg:  "Query error",
		},
		{
			name:         "Permission error",
			inputError:   os.ErrPermission,
			expectedType: ErrPermission,
			expectedMsg:  "Permission denied",
		},
		{
			name:         "File not found error",
			inputError:   os.ErrNotExist,
			expectedType: ErrFileSystem,
			expectedMsg:  "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handleExportError(tt.inputError, options)

			var exportErr *ExportError
			assert.ErrorAs(t, err, &exportErr)
			assert.Equal(t, tt.expectedType, exportErr.Type)
			assert.Contains(t, err.Error(), tt.expectedMsg)
		})
	}
}

// Test helper functions for mock errors
type mockQueryError struct {
	msg string
}

func (e mockQueryError) Error() string {
	return "parsing query: " + e.msg
}

func TestHandleExportError_QueryError(t *testing.T) {
	options := processor.ExportOptions{
		Query: "invalid syntax",
	}

	inputError := mockQueryError{msg: "syntax error"}
	err := handleExportError(inputError, options)

	var exportErr *ExportError
	assert.ErrorAs(t, err, &exportErr)
	assert.Equal(t, ErrQuery, exportErr.Type)
	assert.Contains(t, err.Error(), "Query error")
	assert.Contains(t, err.Error(), "invalid syntax")
}

type mockProcessingError struct {
	msg string
}

func (e mockProcessingError) Error() string {
	return "link processing failed: " + e.msg
}

func TestHandleExportError_ProcessingError(t *testing.T) {
	options := processor.ExportOptions{}

	inputError := mockProcessingError{msg: "invalid link format"}
	err := handleExportError(inputError, options)

	var exportErr *ExportError
	assert.ErrorAs(t, err, &exportErr)
	assert.Equal(t, ErrProcessing, exportErr.Type)
	assert.Contains(t, err.Error(), "content processing")
}
