package errors

import (
	"fmt"
	"strings"
)

// UserError provides user-friendly error messages with suggestions
type UserError struct {
	Operation  string // The operation that failed (e.g., "frontmatter.ensure")
	File       string // File path where error occurred
	Err        error  // Original error
	Suggestion string // Helpful suggestion for the user
	Code       string // Error code for programmatic handling
}

// Error implements the error interface
func (e UserError) Error() string {
	var buf strings.Builder

	// Main error message
	fmt.Fprintf(&buf, "Error: %s", e.Err)

	// Add context information
	if e.Operation != "" {
		fmt.Fprintf(&buf, "\nOperation: %s", e.Operation)
	}

	if e.File != "" {
		fmt.Fprintf(&buf, "\nFile: %s", e.File)
	}

	// Add helpful suggestion
	if e.Suggestion != "" {
		fmt.Fprintf(&buf, "\n\nSuggestion: %s", e.Suggestion)
	}

	return buf.String()
}

// Unwrap returns the underlying error for error chain compatibility
func (e UserError) Unwrap() error {
	return e.Err
}

// ErrorCode returns the error code for programmatic handling
func (e UserError) ErrorCode() string {
	return e.Code
}

// Common error codes
const (
	ErrCodeInvalidFile       = "INVALID_FILE"
	ErrCodeMissingField      = "MISSING_FIELD"
	ErrCodeInvalidType       = "INVALID_TYPE"
	ErrCodeInvalidValue      = "INVALID_VALUE"
	ErrCodeFileNotFound      = "FILE_NOT_FOUND"
	ErrCodePermissionDenied  = "PERMISSION_DENIED"
	ErrCodeInvalidConfig     = "INVALID_CONFIG"
	ErrCodeNetworkError      = "NETWORK_ERROR"
	ErrCodeQuotaExceeded     = "QUOTA_EXCEEDED"
	ErrCodeOperationTimeout  = "OPERATION_TIMEOUT"
	ErrCodeInvalidSyntax     = "INVALID_SYNTAX"
	ErrCodeDuplicateResource = "DUPLICATE_RESOURCE"
	ErrCodeResourceNotFound  = "RESOURCE_NOT_FOUND"
)

// ErrorBuilder helps construct user-friendly errors with suggestions
type ErrorBuilder struct {
	operation  string
	file       string
	err        error
	suggestion string
	code       string
}

// NewErrorBuilder creates a new error builder
func NewErrorBuilder() *ErrorBuilder {
	return &ErrorBuilder{}
}

// WithOperation sets the operation context
func (b *ErrorBuilder) WithOperation(operation string) *ErrorBuilder {
	b.operation = operation
	return b
}

// WithFile sets the file context
func (b *ErrorBuilder) WithFile(file string) *ErrorBuilder {
	b.file = file
	return b
}

// WithError sets the underlying error
func (b *ErrorBuilder) WithError(err error) *ErrorBuilder {
	b.err = err
	return b
}

// WithSuggestion sets a helpful suggestion
func (b *ErrorBuilder) WithSuggestion(suggestion string) *ErrorBuilder {
	b.suggestion = suggestion
	return b
}

// WithCode sets the error code
func (b *ErrorBuilder) WithCode(code string) *ErrorBuilder {
	b.code = code
	return b
}

// Build creates the UserError
func (b *ErrorBuilder) Build() UserError {
	return UserError{
		Operation:  b.operation,
		File:       b.file,
		Err:        b.err,
		Suggestion: b.suggestion,
		Code:       b.code,
	}
}

// Common error constructors for frequently occurring scenarios

// NewFileNotFoundError creates an error for missing files
func NewFileNotFoundError(file string, suggestion string) UserError {
	return NewErrorBuilder().
		WithFile(file).
		WithError(fmt.Errorf("file not found: %s", file)).
		WithCode(ErrCodeFileNotFound).
		WithSuggestion(suggestion).
		Build()
}

// NewInvalidTypeError creates an error for type validation failures
func NewInvalidTypeError(field, expectedType, actualValue string, file string) UserError {
	suggestion := fmt.Sprintf("Field '%s' should be of type '%s'. ", field, expectedType)

	switch expectedType {
	case "date":
		suggestion += "Date must be in YYYY-MM-DD format (e.g., 2023-01-15) or ISO datetime format."
	case "number":
		suggestion += "Value must be a valid number (integer or decimal)."
	case "boolean":
		suggestion += "Value must be 'true' or 'false'."
	case "array":
		suggestion += "Value should be a YAML array like [item1, item2] or comma-separated like 'item1, item2'."
	default:
		suggestion += "Check the expected format in your configuration."
	}

	return NewErrorBuilder().
		WithOperation("type validation").
		WithFile(file).
		WithError(fmt.Errorf("invalid type for field '%s': expected %s, got '%s'", field, expectedType, actualValue)).
		WithCode(ErrCodeInvalidType).
		WithSuggestion(suggestion).
		Build()
}

// NewMissingFieldError creates an error for missing required fields
func NewMissingFieldError(field string, file string) UserError {
	suggestion := fmt.Sprintf("Add the field '%s' to the frontmatter of this file. You can use the 'frontmatter ensure' command to add it automatically.", field)

	return NewErrorBuilder().
		WithOperation("field validation").
		WithFile(file).
		WithError(fmt.Errorf("required field '%s' is missing", field)).
		WithCode(ErrCodeMissingField).
		WithSuggestion(suggestion).
		Build()
}

// NewInvalidSyntaxError creates an error for syntax issues
func NewInvalidSyntaxError(file string, line int, details string) UserError {
	suggestion := "Check the YAML syntax in your frontmatter. Common issues include incorrect indentation, missing quotes around special characters, or malformed lists."

	err := fmt.Errorf("syntax error in file %s", file)
	if line > 0 {
		err = fmt.Errorf("syntax error in file %s at line %d: %s", file, line, details)
	}

	return NewErrorBuilder().
		WithOperation("file parsing").
		WithFile(file).
		WithError(err).
		WithCode(ErrCodeInvalidSyntax).
		WithSuggestion(suggestion).
		Build()
}

// NewConfigError creates an error for configuration issues
func NewConfigError(configPath string, details string) UserError {
	suggestion := "Check your configuration file for syntax errors and ensure all required fields are present. You can use 'mdnotes batch validate' to verify the configuration."

	return NewErrorBuilder().
		WithOperation("configuration loading").
		WithFile(configPath).
		WithError(fmt.Errorf("configuration error: %s", details)).
		WithCode(ErrCodeInvalidConfig).
		WithSuggestion(suggestion).
		Build()
}

// NewNetworkError creates an error for network-related issues
func NewNetworkError(operation string, url string, err error) UserError {
	suggestion := "Check your internet connection and verify that the service URL is correct. If using API tokens, ensure they are valid and have sufficient permissions."

	return NewErrorBuilder().
		WithOperation(operation).
		WithError(fmt.Errorf("network error accessing %s: %w", url, err)).
		WithCode(ErrCodeNetworkError).
		WithSuggestion(suggestion).
		Build()
}

// NewPermissionError creates an error for permission issues
func NewPermissionError(file string, operation string) UserError {
	suggestion := "Check that you have read/write permissions for this file and its parent directory. You may need to run the command with different permissions or change file ownership."

	return NewErrorBuilder().
		WithOperation(operation).
		WithFile(file).
		WithError(fmt.Errorf("permission denied accessing file: %s", file)).
		WithCode(ErrCodePermissionDenied).
		WithSuggestion(suggestion).
		Build()
}

// ErrorHandler provides consistent error formatting and logging
type ErrorHandler struct {
	verbose bool
	quiet   bool
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(verbose, quiet bool) *ErrorHandler {
	return &ErrorHandler{
		verbose: verbose,
		quiet:   quiet,
	}
}

// Handle processes an error and returns a formatted message
func (h *ErrorHandler) Handle(err error) string {
	if err == nil {
		return ""
	}

	// Check if it's already a UserError
	if userErr, ok := err.(UserError); ok {
		return h.formatUserError(userErr)
	}

	// Convert regular errors to user-friendly format
	return h.formatRegularError(err)
}

// formatUserError formats a UserError based on verbosity settings
func (h *ErrorHandler) formatUserError(err UserError) string {
	if h.quiet {
		return err.Err.Error()
	}

	var buf strings.Builder

	// Use colors if terminal supports it (simplified for now)
	errorColor := "\033[31m"      // Red
	contextColor := "\033[33m"    // Yellow
	suggestionColor := "\033[36m" // Cyan
	resetColor := "\033[0m"

	// Main error (red)
	fmt.Fprintf(&buf, "%sError:%s %s\n", errorColor, resetColor, err.Err.Error())

	// Context information (yellow)
	if err.Operation != "" {
		fmt.Fprintf(&buf, "%sOperation:%s %s\n", contextColor, resetColor, err.Operation)
	}
	if err.File != "" {
		fmt.Fprintf(&buf, "%sFile:%s %s\n", contextColor, resetColor, err.File)
	}

	// Suggestion (cyan)
	if err.Suggestion != "" {
		fmt.Fprintf(&buf, "\n%sSuggestion:%s %s\n", suggestionColor, resetColor, err.Suggestion)
	}

	// Verbose mode: add error code and stack trace if available
	if h.verbose {
		if err.Code != "" {
			fmt.Fprintf(&buf, "\nError Code: %s\n", err.Code)
		}
	}

	return buf.String()
}

// formatRegularError formats a regular error with basic enhancement
func (h *ErrorHandler) formatRegularError(err error) string {
	if h.quiet {
		return err.Error()
	}

	// Add basic context enhancement for common error patterns
	errMsg := err.Error()

	var suggestion string
	switch {
	case strings.Contains(errMsg, "no such file or directory"):
		suggestion = "Check that the file path is correct and the file exists."
	case strings.Contains(errMsg, "permission denied"):
		suggestion = "Check that you have the necessary permissions to access this file."
	case strings.Contains(errMsg, "connection refused"):
		suggestion = "Check that the service is running and accessible."
	case strings.Contains(errMsg, "invalid character"):
		suggestion = "Check for syntax errors in your YAML or JSON."
	case strings.Contains(errMsg, "timeout"):
		suggestion = "The operation took too long. Try reducing the scope or checking your network connection."
	}

	var buf strings.Builder
	fmt.Fprintf(&buf, "Error: %s", errMsg)

	if suggestion != "" {
		fmt.Fprintf(&buf, "\n\nSuggestion: %s", suggestion)
	}

	return buf.String()
}

// WrapError wraps a regular error into a UserError with context
func WrapError(err error, operation, file string) UserError {
	return NewErrorBuilder().
		WithOperation(operation).
		WithFile(file).
		WithError(err).
		Build()
}

// ExitCode returns an appropriate exit code for an error
func ExitCode(err error) int {
	if err == nil {
		return 0
	}

	if userErr, ok := err.(UserError); ok {
		switch userErr.Code {
		case ErrCodeFileNotFound, ErrCodeResourceNotFound:
			return 2
		case ErrCodePermissionDenied:
			return 3
		case ErrCodeInvalidConfig, ErrCodeInvalidSyntax:
			return 4
		case ErrCodeNetworkError:
			return 5
		case ErrCodeOperationTimeout:
			return 6
		default:
			return 1
		}
	}

	return 1
}
