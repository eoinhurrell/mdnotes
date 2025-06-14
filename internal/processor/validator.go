package processor

import (
	"reflect"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// ValidationRules defines rules for validating frontmatter
type ValidationRules struct {
	Required []string            // Required field names
	Types    map[string]string   // Field name -> expected type
}

// ValidationError represents a validation error
type ValidationError struct {
	Field    string // Field name with error
	Type     string // Error type (missing_required, invalid_type)
	Expected string // Expected value/type
	File     string // File path
}

// Validator validates frontmatter against rules
type Validator struct {
	rules ValidationRules
}

// NewValidator creates a new frontmatter validator
func NewValidator(rules ValidationRules) *Validator {
	return &Validator{
		rules: rules,
	}
}

// Validate checks a file against validation rules
func (v *Validator) Validate(file *vault.VaultFile) []ValidationError {
	var errors []ValidationError

	// Check required fields
	for _, field := range v.rules.Required {
		if _, exists := file.Frontmatter[field]; !exists {
			errors = append(errors, ValidationError{
				Field: field,
				Type:  "missing_required",
				File:  file.Path,
			})
		}
	}

	// Validate types
	for field, expectedType := range v.rules.Types {
		if value, exists := file.Frontmatter[field]; exists {
			if !v.validateType(value, expectedType) {
				errors = append(errors, ValidationError{
					Field:    field,
					Type:     "invalid_type",
					Expected: expectedType,
					File:     file.Path,
				})
			}
		}
	}

	return errors
}

// validateType checks if a value matches the expected type
func (v *Validator) validateType(value interface{}, expectedType string) bool {
	if value == nil {
		return true // nil values are considered valid for any type
	}

	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		switch value.(type) {
		case int, int32, int64, float32, float64:
			return true
		default:
			return false
		}
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "array":
		rv := reflect.ValueOf(value)
		return rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array
	default:
		return false
	}
}

// Error implements the error interface for ValidationError
func (e ValidationError) Error() string {
	switch e.Type {
	case "missing_required":
		return "field '" + e.Field + "' is required"
	case "invalid_type":
		return "field '" + e.Field + "' must be of type " + e.Expected
	default:
		return "validation error in field '" + e.Field + "'"
	}
}