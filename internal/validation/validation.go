package validation

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/eoinhurrell/mdnotes/internal/errors"
)

// ValidationResult holds validation results
type ValidationResult struct {
	errors []errors.UserError
}

// Validator provides input validation functionality
type Validator struct {
	result *ValidationResult
}

// NewValidator creates a new validator instance
func NewValidator() *Validator {
	return &Validator{
		result: &ValidationResult{
			errors: make([]errors.UserError, 0),
		},
	}
}

// HasErrors returns true if validation errors were found
func (v *Validator) HasErrors() bool {
	return len(v.result.errors) > 0
}

// GetErrors returns all validation errors
func (v *Validator) GetErrors() []errors.UserError {
	return v.result.errors
}

// addError adds a validation error
func (v *Validator) addError(code, field, message string) {
	userErr := errors.NewErrorBuilder().
		WithOperation("validation").
		WithError(fmt.Errorf("%s", message)).
		WithCode(code).
		Build()
	v.result.errors = append(v.result.errors, userErr)
}

// ValidateRequired checks if a value is present
func (v *Validator) ValidateRequired(field string, value interface{}, message ...string) *Validator {
	msg := fmt.Sprintf("Field '%s' is required", field)
	if len(message) > 0 {
		msg = message[0]
	}

	if value == nil {
		v.addError(errors.ErrCodeMissingField, field, msg)
		return v
	}

	switch val := value.(type) {
	case string:
		if strings.TrimSpace(val) == "" {
			v.addError(errors.ErrCodeMissingField, field, msg)
		}
	case []interface{}:
		if len(val) == 0 {
			v.addError(errors.ErrCodeMissingField, field, msg)
		}
	case map[string]interface{}:
		if len(val) == 0 {
			v.addError(errors.ErrCodeMissingField, field, msg)
		}
	}

	return v
}

// ValidateString checks string constraints
func (v *Validator) ValidateString(field string, value interface{}, constraints StringConstraints) *Validator {
	str, ok := value.(string)
	if !ok {
		if value != nil {
			v.addError(errors.ErrCodeInvalidType, field, fmt.Sprintf("Field '%s' must be a string", field))
		}
		return v
	}

	// Length constraints
	if constraints.MinLength > 0 && len(str) < constraints.MinLength {
		v.addError(errors.ErrCodeInvalidValue, field,
			fmt.Sprintf("Field '%s' must be at least %d characters long", field, constraints.MinLength))
	}

	if constraints.MaxLength > 0 && len(str) > constraints.MaxLength {
		v.addError(errors.ErrCodeInvalidValue, field,
			fmt.Sprintf("Field '%s' must be at most %d characters long", field, constraints.MaxLength))
	}

	// Pattern validation
	if constraints.Pattern != "" {
		if matched, err := regexp.MatchString(constraints.Pattern, str); err != nil {
			v.addError(errors.ErrCodeValidationFailed, field,
				fmt.Sprintf("Invalid pattern for field '%s': %v", field, err))
		} else if !matched {
			v.addError(errors.ErrCodeInvalidValue, field,
				fmt.Sprintf("Field '%s' does not match required pattern", field))
		}
	}

	// Allowed values
	if len(constraints.AllowedValues) > 0 {
		allowed := false
		for _, allowedValue := range constraints.AllowedValues {
			if str == allowedValue {
				allowed = true
				break
			}
		}
		if !allowed {
			v.addError(errors.ErrCodeInvalidValue, field,
				fmt.Sprintf("Field '%s' must be one of: %s", field, strings.Join(constraints.AllowedValues, ", ")))
		}
	}

	// Custom validator
	if constraints.Validator != nil {
		if err := constraints.Validator(str); err != nil {
			v.addError(errors.ErrCodeValidationFailed, field,
				fmt.Sprintf("Validation failed for field '%s': %v", field, err))
		}
	}

	return v
}

// ValidateNumber checks numeric constraints
func (v *Validator) ValidateNumber(field string, value interface{}, constraints NumberConstraints) *Validator {
	var num float64
	var ok bool

	switch val := value.(type) {
	case int:
		num = float64(val)
		ok = true
	case int64:
		num = float64(val)
		ok = true
	case float64:
		num = val
		ok = true
	case string:
		if parsed, err := strconv.ParseFloat(val, 64); err == nil {
			num = parsed
			ok = true
		}
	}

	if !ok {
		if value != nil {
			v.addError(errors.ErrCodeInvalidType, field, fmt.Sprintf("Field '%s' must be a number", field))
		}
		return v
	}

	// Range constraints
	if constraints.Min != nil && num < *constraints.Min {
		v.addError(errors.ErrCodeInvalidValue, field,
			fmt.Sprintf("Field '%s' must be at least %g", field, *constraints.Min))
	}

	if constraints.Max != nil && num > *constraints.Max {
		v.addError(errors.ErrCodeInvalidValue, field,
			fmt.Sprintf("Field '%s' must be at most %g", field, *constraints.Max))
	}

	// Integer check
	if constraints.IntegerOnly && num != float64(int64(num)) {
		v.addError(errors.ErrCodeInvalidValue, field,
			fmt.Sprintf("Field '%s' must be an integer", field))
	}

	// Positive check
	if constraints.PositiveOnly && num <= 0 {
		v.addError(errors.ErrCodeInvalidValue, field,
			fmt.Sprintf("Field '%s' must be positive", field))
	}

	return v
}

// ValidatePath checks file path constraints
func (v *Validator) ValidatePath(field string, value interface{}, constraints PathConstraints) *Validator {
	path, ok := value.(string)
	if !ok {
		if value != nil {
			v.addError(errors.ErrCodeInvalidType, field, fmt.Sprintf("Field '%s' must be a path string", field))
		}
		return v
	}

	// Clean the path
	path = filepath.Clean(path)

	// Check if path is absolute when required
	if constraints.MustBeAbsolute && !filepath.IsAbs(path) {
		v.addError(errors.ErrCodePathInvalid, field,
			fmt.Sprintf("Field '%s' must be an absolute path", field))
	}

	// Check if path exists when required
	if constraints.MustExist {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			v.addError(errors.ErrCodeFileNotFound, field,
				fmt.Sprintf("Path '%s' does not exist", path))
		}
	}

	// Check if path is a directory when required
	if constraints.MustBeDirectory {
		if info, err := os.Stat(path); err == nil {
			if !info.IsDir() {
				v.addError(errors.ErrCodePathInvalid, field,
					fmt.Sprintf("Field '%s' must be a directory", field))
			}
		}
	}

	// Check if path is a file when required
	if constraints.MustBeFile {
		if info, err := os.Stat(path); err == nil {
			if info.IsDir() {
				v.addError(errors.ErrCodePathInvalid, field,
					fmt.Sprintf("Field '%s' must be a file", field))
			}
		}
	}

	// Check allowed extensions
	if len(constraints.AllowedExtensions) > 0 {
		ext := strings.ToLower(filepath.Ext(path))
		allowed := false
		for _, allowedExt := range constraints.AllowedExtensions {
			if ext == strings.ToLower(allowedExt) {
				allowed = true
				break
			}
		}
		if !allowed {
			v.addError(errors.ErrCodePathInvalid, field,
				fmt.Sprintf("Field '%s' must have one of these extensions: %s",
					field, strings.Join(constraints.AllowedExtensions, ", ")))
		}
	}

	// Prevent path traversal
	if constraints.PreventTraversal {
		if strings.Contains(path, "..") {
			v.addError(errors.ErrCodePathInvalid, field,
				fmt.Sprintf("Field '%s' contains invalid path traversal", field))
		}
	}

	return v
}

// ValidateURL checks URL constraints
func (v *Validator) ValidateURL(field string, value interface{}, constraints URLConstraints) *Validator {
	urlStr, ok := value.(string)
	if !ok {
		if value != nil {
			v.addError(errors.ErrCodeInvalidType, field, fmt.Sprintf("Field '%s' must be a URL string", field))
		}
		return v
	}

	// Parse URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		v.addError(errors.ErrCodeInvalidValue, field,
			fmt.Sprintf("Field '%s' is not a valid URL: %v", field, err))
		return v
	}

	// Check allowed schemes
	if len(constraints.AllowedSchemes) > 0 {
		allowed := false
		for _, scheme := range constraints.AllowedSchemes {
			if parsedURL.Scheme == scheme {
				allowed = true
				break
			}
		}
		if !allowed {
			v.addError(errors.ErrCodeInvalidValue, field,
				fmt.Sprintf("Field '%s' must use one of these schemes: %s",
					field, strings.Join(constraints.AllowedSchemes, ", ")))
		}
	}

	// Require host
	if constraints.RequireHost && parsedURL.Host == "" {
		v.addError(errors.ErrCodeInvalidValue, field,
			fmt.Sprintf("Field '%s' must include a host", field))
	}

	return v
}

// ValidateDate checks date constraints
func (v *Validator) ValidateDate(field string, value interface{}, constraints DateConstraints) *Validator {
	var date time.Time
	var err error

	switch val := value.(type) {
	case time.Time:
		date = val
	case string:
		// Try common date formats
		formats := []string{
			time.RFC3339,
			"2006-01-02",
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
		}

		if constraints.Format != "" {
			formats = []string{constraints.Format}
		}

		for _, format := range formats {
			parsed, parseErr := time.Parse(format, val)
			if parseErr == nil {
				date = parsed
				break
			}
			err = parseErr
		}

		if date.IsZero() {
			v.addError(errors.ErrCodeInvalidValue, field,
				fmt.Sprintf("Field '%s' is not a valid date: %v", field, err))
			return v
		}
	default:
		if value != nil {
			v.addError(errors.ErrCodeInvalidType, field, fmt.Sprintf("Field '%s' must be a date", field))
		}
		return v
	}

	// Check date range
	if constraints.After != nil && date.Before(*constraints.After) {
		v.addError(errors.ErrCodeInvalidValue, field,
			fmt.Sprintf("Field '%s' must be after %s", field, constraints.After.Format("2006-01-02")))
	}

	if constraints.Before != nil && date.After(*constraints.Before) {
		v.addError(errors.ErrCodeInvalidValue, field,
			fmt.Sprintf("Field '%s' must be before %s", field, constraints.Before.Format("2006-01-02")))
	}

	return v
}

// ValidateArray checks array constraints
func (v *Validator) ValidateArray(field string, value interface{}, constraints ArrayConstraints) *Validator {
	var arr []interface{}
	var ok bool

	switch val := value.(type) {
	case []interface{}:
		arr = val
		ok = true
	case []string:
		arr = make([]interface{}, len(val))
		for i, s := range val {
			arr[i] = s
		}
		ok = true
	}

	if !ok {
		if value != nil {
			v.addError(errors.ErrCodeInvalidType, field, fmt.Sprintf("Field '%s' must be an array", field))
		}
		return v
	}

	// Length constraints
	if constraints.MinLength > 0 && len(arr) < constraints.MinLength {
		v.addError(errors.ErrCodeInvalidValue, field,
			fmt.Sprintf("Field '%s' must have at least %d items", field, constraints.MinLength))
	}

	if constraints.MaxLength > 0 && len(arr) > constraints.MaxLength {
		v.addError(errors.ErrCodeInvalidValue, field,
			fmt.Sprintf("Field '%s' must have at most %d items", field, constraints.MaxLength))
	}

	// Unique values
	if constraints.UniqueValues {
		seen := make(map[interface{}]bool)
		for _, item := range arr {
			if seen[item] {
				v.addError(errors.ErrCodeInvalidValue, field,
					fmt.Sprintf("Field '%s' must have unique values", field))
				break
			}
			seen[item] = true
		}
	}

	// Element validator
	if constraints.ElementValidator != nil {
		for i, item := range arr {
			if err := constraints.ElementValidator(item); err != nil {
				v.addError(errors.ErrCodeValidationFailed, field,
					fmt.Sprintf("Validation failed for %s[%d]: %v", field, i, err))
			}
		}
	}

	return v
}

// Constraint types

// StringConstraints defines validation rules for string fields
type StringConstraints struct {
	MinLength     int
	MaxLength     int
	Pattern       string
	AllowedValues []string
	Validator     func(string) error
}

// NumberConstraints defines validation rules for numeric fields
type NumberConstraints struct {
	Min          *float64
	Max          *float64
	IntegerOnly  bool
	PositiveOnly bool
}

// PathConstraints defines validation rules for file paths
type PathConstraints struct {
	MustBeAbsolute    bool
	MustExist         bool
	MustBeDirectory   bool
	MustBeFile        bool
	AllowedExtensions []string
	PreventTraversal  bool
}

// URLConstraints defines validation rules for URLs
type URLConstraints struct {
	AllowedSchemes []string
	RequireHost    bool
}

// DateConstraints defines validation rules for dates
type DateConstraints struct {
	Format string
	After  *time.Time
	Before *time.Time
}

// ArrayConstraints defines validation rules for arrays
type ArrayConstraints struct {
	MinLength        int
	MaxLength        int
	UniqueValues     bool
	ElementValidator func(interface{}) error
}

// Common validators

// ValidateEmail validates email addresses
func ValidateEmail(email string) error {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	if matched, err := regexp.MatchString(pattern, email); err != nil {
		return err
	} else if !matched {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

// ValidateMarkdownExtension validates markdown file extensions
func ValidateMarkdownExtension(path string) error {
	ext := strings.ToLower(filepath.Ext(path))
	allowed := []string{".md", ".markdown", ".mdown", ".mkd"}

	for _, allowedExt := range allowed {
		if ext == allowedExt {
			return nil
		}
	}

	return fmt.Errorf("file must have a markdown extension (.md, .markdown, .mdown, .mkd)")
}

// ValidateYAMLExtension validates YAML file extensions
func ValidateYAMLExtension(path string) error {
	ext := strings.ToLower(filepath.Ext(path))
	allowed := []string{".yaml", ".yml"}

	for _, allowedExt := range allowed {
		if ext == allowedExt {
			return nil
		}
	}

	return fmt.Errorf("file must have a YAML extension (.yaml, .yml)")
}

// ValidateSlug validates URL-friendly slugs
func ValidateSlug(slug string) error {
	pattern := `^[a-z0-9]+(?:-[a-z0-9]+)*$`
	if matched, err := regexp.MatchString(pattern, slug); err != nil {
		return err
	} else if !matched {
		return fmt.Errorf("slug must contain only lowercase letters, numbers, and hyphens")
	}
	return nil
}

// ValidateVersion validates semantic version strings
func ValidateVersion(version string) error {
	pattern := `^v?(\d+)\.(\d+)\.(\d+)(?:-([a-zA-Z0-9-]+(?:\.[a-zA-Z0-9-]+)*))?(?:\+([a-zA-Z0-9-]+(?:\.[a-zA-Z0-9-]+)*))?$`
	if matched, err := regexp.MatchString(pattern, version); err != nil {
		return err
	} else if !matched {
		return fmt.Errorf("version must follow semantic versioning format (e.g., 1.2.3)")
	}
	return nil
}

// Helper functions

// Float64Ptr returns a pointer to a float64 value
func Float64Ptr(f float64) *float64 {
	return &f
}

// TimePtr returns a pointer to a time.Time value
func TimePtr(t time.Time) *time.Time {
	return &t
}

// SanitizeFilename removes invalid characters from filenames
func SanitizeFilename(filename string) string {
	// Replace invalid characters with underscores
	invalid := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	sanitized := invalid.ReplaceAllString(filename, "_")

	// Trim spaces and dots from ends
	sanitized = strings.Trim(sanitized, " .")

	// Ensure filename is not empty
	if sanitized == "" {
		sanitized = "untitled"
	}

	return sanitized
}

// SanitizePath safely cleans and validates a file path
func SanitizePath(path string) (string, error) {
	// Clean the path
	cleaned := filepath.Clean(path)

	// Check for path traversal attempts
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("path contains invalid traversal")
	}

	// Convert to forward slashes for consistency
	cleaned = filepath.ToSlash(cleaned)

	return cleaned, nil
}
