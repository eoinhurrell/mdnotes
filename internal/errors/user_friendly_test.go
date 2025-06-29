package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserError_Error(t *testing.T) {
	err := UserError{
		Operation:  "frontmatter.ensure",
		File:       "/path/to/file.md",
		Err:        errors.New("field validation failed"),
		Suggestion: "Check your field configuration",
		Code:       ErrCodeInvalidType,
	}

	result := err.Error()
	assert.Contains(t, result, "Error: field validation failed")
	assert.Contains(t, result, "Operation: frontmatter.ensure")
	assert.Contains(t, result, "File: /path/to/file.md")
	assert.Contains(t, result, "Suggestion: Check your field configuration")
}

func TestUserError_ErrorMinimal(t *testing.T) {
	err := UserError{
		Err: errors.New("simple error"),
	}

	result := err.Error()
	assert.Equal(t, "Error: simple error", result)
}

func TestUserError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	userErr := UserError{Err: originalErr}

	assert.Equal(t, originalErr, userErr.Unwrap())
}

func TestUserError_ErrorCode(t *testing.T) {
	userErr := UserError{Code: ErrCodeFileNotFound}
	assert.Equal(t, ErrCodeFileNotFound, userErr.ErrorCode())
}

func TestErrorBuilder(t *testing.T) {
	originalErr := errors.New("test error")
	
	userErr := NewErrorBuilder().
		WithOperation("test operation").
		WithFile("/test/file.md").
		WithError(originalErr).
		WithSuggestion("test suggestion").
		WithCode(ErrCodeInvalidConfig).
		Build()

	assert.Equal(t, "test operation", userErr.Operation)
	assert.Equal(t, "/test/file.md", userErr.File)
	assert.Equal(t, originalErr, userErr.Err)
	assert.Equal(t, "test suggestion", userErr.Suggestion)
	assert.Equal(t, ErrCodeInvalidConfig, userErr.Code)
}

func TestNewFileNotFoundError(t *testing.T) {
	err := NewFileNotFoundError("/missing/file.md", "Create the file first")

	assert.Equal(t, "/missing/file.md", err.File)
	assert.Contains(t, err.Error(), "file not found")
	assert.Contains(t, err.Suggestion, "Create the file first")
	assert.Equal(t, ErrCodeFileNotFound, err.Code)
}

func TestNewInvalidTypeError(t *testing.T) {
	err := NewInvalidTypeError("tags", "array", "string", "/test/file.md")

	assert.Equal(t, "/test/file.md", err.File)
	assert.Contains(t, err.Error(), "invalid type for field 'tags'")
	assert.Contains(t, err.Suggestion, "Field 'tags' should be of type 'array'")
	assert.Contains(t, err.Suggestion, "YAML array")
	assert.Equal(t, ErrCodeInvalidType, err.Code)
}

func TestNewInvalidTypeError_DateSuggestion(t *testing.T) {
	err := NewInvalidTypeError("created", "date", "invalid", "/test/file.md")
	assert.Contains(t, err.Suggestion, "YYYY-MM-DD format")
}

func TestNewInvalidTypeError_NumberSuggestion(t *testing.T) {
	err := NewInvalidTypeError("priority", "number", "not-a-number", "/test/file.md")
	assert.Contains(t, err.Suggestion, "valid number")
}

func TestNewInvalidTypeError_BooleanSuggestion(t *testing.T) {
	err := NewInvalidTypeError("published", "boolean", "yes", "/test/file.md")
	assert.Contains(t, err.Suggestion, "'true' or 'false'")
}

func TestNewMissingFieldError(t *testing.T) {
	err := NewMissingFieldError("title", "/test/file.md")

	assert.Equal(t, "/test/file.md", err.File)
	assert.Contains(t, err.Error(), "required field 'title' is missing")
	assert.Contains(t, err.Suggestion, "frontmatter ensure")
	assert.Equal(t, ErrCodeMissingField, err.Code)
}

func TestNewInvalidSyntaxError(t *testing.T) {
	err := NewInvalidSyntaxError("/test/file.md", 10, "unexpected character")

	assert.Equal(t, "/test/file.md", err.File)
	assert.Contains(t, err.Error(), "syntax error")
	assert.Contains(t, err.Error(), "line 10")
	assert.Contains(t, err.Error(), "unexpected character")
	assert.Contains(t, err.Suggestion, "YAML syntax")
	assert.Equal(t, ErrCodeInvalidSyntax, err.Code)
}

func TestNewInvalidSyntaxError_NoLine(t *testing.T) {
	err := NewInvalidSyntaxError("/test/file.md", 0, "")

	assert.Contains(t, err.Error(), "syntax error in file /test/file.md")
	assert.NotContains(t, err.Error(), "line")
}

func TestNewConfigError(t *testing.T) {
	err := NewConfigError("/config/file.yaml", "missing required field")

	assert.Equal(t, "/config/file.yaml", err.File)
	assert.Contains(t, err.Error(), "configuration error")
	assert.Contains(t, err.Suggestion, "configuration file")
	assert.Equal(t, ErrCodeInvalidConfig, err.Code)
}

func TestNewNetworkError(t *testing.T) {
	originalErr := errors.New("connection timeout")
	err := NewNetworkError("API call", "https://api.example.com", originalErr)

	assert.Contains(t, err.Error(), "network error")
	assert.Contains(t, err.Error(), "https://api.example.com")
	assert.Contains(t, err.Suggestion, "internet connection")
	assert.Equal(t, ErrCodeNetworkError, err.Code)
}

func TestNewPermissionError(t *testing.T) {
	err := NewPermissionError("/protected/file.md", "file.write")

	assert.Equal(t, "/protected/file.md", err.File)
	assert.Contains(t, err.Error(), "permission denied")
	assert.Contains(t, err.Suggestion, "read/write permissions")
	assert.Equal(t, ErrCodePermissionDenied, err.Code)
}

func TestErrorHandler_Handle_UserError(t *testing.T) {
	handler := NewErrorHandler(false, false)
	userErr := UserError{
		Err:        errors.New("test error"),
		Operation:  "test",
		File:       "/test.md",
		Suggestion: "test suggestion",
	}

	result := handler.Handle(userErr)
	assert.Contains(t, result, "Error:")
	assert.Contains(t, result, "test error")
	assert.Contains(t, result, "Operation:")
	assert.Contains(t, result, "Suggestion:")
}

func TestErrorHandler_Handle_RegularError(t *testing.T) {
	handler := NewErrorHandler(false, false)
	err := errors.New("no such file or directory")

	result := handler.Handle(err)
	assert.Contains(t, result, "Error: no such file or directory")
	assert.Contains(t, result, "Check that the file path is correct")
}

func TestErrorHandler_Handle_Quiet(t *testing.T) {
	handler := NewErrorHandler(false, true)
	userErr := UserError{
		Err:        errors.New("test error"),
		Suggestion: "test suggestion",
	}

	result := handler.Handle(userErr)
	assert.Equal(t, "test error", result)
	assert.NotContains(t, result, "Suggestion:")
}

func TestErrorHandler_Handle_Verbose(t *testing.T) {
	handler := NewErrorHandler(true, false)
	userErr := UserError{
		Err:  errors.New("test error"),
		Code: ErrCodeInvalidType,
	}

	result := handler.Handle(userErr)
	assert.Contains(t, result, "Error Code:")
	assert.Contains(t, result, ErrCodeInvalidType)
}

func TestErrorHandler_Handle_Nil(t *testing.T) {
	handler := NewErrorHandler(false, false)
	result := handler.Handle(nil)
	assert.Empty(t, result)
}

func TestErrorHandler_FormatRegularError_Patterns(t *testing.T) {
	handler := NewErrorHandler(false, false)

	tests := []struct {
		name          string
		errorMsg      string
		expectedSuggestion string
	}{
		{
			name:               "permission denied",
			errorMsg:           "permission denied accessing file",
			expectedSuggestion: "necessary permissions",
		},
		{
			name:               "connection refused",
			errorMsg:           "connection refused by server",
			expectedSuggestion: "service is running",
		},
		{
			name:               "invalid character",
			errorMsg:           "invalid character in JSON",
			expectedSuggestion: "syntax errors",
		},
		{
			name:               "timeout",
			errorMsg:           "operation timeout exceeded",
			expectedSuggestion: "took too long",
		},
		{
			name:               "unknown error",
			errorMsg:           "some unknown error",
			expectedSuggestion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errorMsg)
			result := handler.Handle(err)
			
			if tt.expectedSuggestion != "" {
				assert.Contains(t, result, "Suggestion:")
				assert.Contains(t, result, tt.expectedSuggestion)
			} else {
				assert.NotContains(t, result, "Suggestion:")
			}
		})
	}
}

func TestWrapError(t *testing.T) {
	originalErr := errors.New("original error")
	userErr := WrapError(originalErr, "test operation", "/test/file.md")

	assert.Equal(t, originalErr, userErr.Err)
	assert.Equal(t, "test operation", userErr.Operation)
	assert.Equal(t, "/test/file.md", userErr.File)
}

func TestExitCode(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode int
	}{
		{
			name:         "nil error",
			err:          nil,
			expectedCode: 0,
		},
		{
			name:         "file not found",
			err:          UserError{Code: ErrCodeFileNotFound},
			expectedCode: 2,
		},
		{
			name:         "permission denied",
			err:          UserError{Code: ErrCodePermissionDenied},
			expectedCode: 3,
		},
		{
			name:         "invalid config",
			err:          UserError{Code: ErrCodeInvalidConfig},
			expectedCode: 4,
		},
		{
			name:         "network error",
			err:          UserError{Code: ErrCodeNetworkError},
			expectedCode: 5,
		},
		{
			name:         "operation timeout",
			err:          UserError{Code: ErrCodeOperationTimeout},
			expectedCode: 6,
		},
		{
			name:         "unknown user error",
			err:          UserError{Code: "UNKNOWN"},
			expectedCode: 1,
		},
		{
			name:         "regular error",
			err:          errors.New("regular error"),
			expectedCode: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedCode, ExitCode(tt.err))
		})
	}
}

func TestErrorConstants(t *testing.T) {
	// Just verify that all constants are defined and unique
	codes := []string{
		ErrCodeInvalidFile, ErrCodeFileNotFound, ErrCodeFilePermission,
		ErrCodeFrontmatterInvalid, ErrCodeMissingField, ErrCodeInvalidType,
		ErrCodeContentEmpty, ErrCodeLinkBroken, ErrCodeNetworkError,
		ErrCodeInvalidConfig, ErrCodePluginNotFound, ErrCodeValidationFailed,
		ErrCodeOperationCancelled, ErrCodeResourceNotFound,
	}

	// Check that codes are not empty
	for _, code := range codes {
		assert.NotEmpty(t, code)
	}

	// Check for basic uniqueness (simplified test)
	codeMap := make(map[string]bool)
	for _, code := range codes {
		assert.False(t, codeMap[code], "Duplicate error code: %s", code)
		codeMap[code] = true
	}
}