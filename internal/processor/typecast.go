package processor

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TypeValidator interface for type-specific validation and casting
type TypeValidator interface {
	Cast(value string) (interface{}, error)
	Matches(value string) bool
}

// TypeCaster handles type conversion and detection
type TypeCaster struct {
	validators map[string]TypeValidator
}

// NewTypeCaster creates a new type caster with built-in validators
func NewTypeCaster() *TypeCaster {
	return &TypeCaster{
		validators: map[string]TypeValidator{
			"date":    &DateValidator{},
			"number":  &NumberValidator{},
			"boolean": &BooleanValidator{},
			"array":   &ArrayValidator{},
			"null":    &NullValidator{},
		},
	}
}

// Cast converts a value to the specified type
func (tc *TypeCaster) Cast(value interface{}, toType string) (interface{}, error) {
	// Handle already correct type
	if tc.isType(value, toType) {
		return value, nil
	}

	// Convert from string
	strVal, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("cannot cast non-string value %T to %s", value, toType)
	}

	validator, exists := tc.validators[toType]
	if !exists {
		return nil, fmt.Errorf("unknown type: %s", toType)
	}

	return validator.Cast(strVal)
}

// AutoDetect automatically detects the most appropriate type for a value
func (tc *TypeCaster) AutoDetect(value interface{}) string {
	// If already typed, return its type
	if !tc.isStringType(value) {
		return tc.getType(value)
	}

	strVal := value.(string)

	// Try each validator in order of specificity
	order := []string{"date", "number", "boolean", "array"}
	for _, typeName := range order {
		if tc.validators[typeName].Matches(strVal) {
			return typeName
		}
	}

	return "string"
}

// isType checks if a value is already of the specified type
func (tc *TypeCaster) isType(value interface{}, typeName string) bool {
	switch typeName {
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
	case "date":
		_, ok := value.(time.Time)
		return ok
	case "null":
		return value == nil
	default:
		return false
	}
}

// isStringType checks if a value is a string
func (tc *TypeCaster) isStringType(value interface{}) bool {
	_, ok := value.(string)
	return ok
}

// getType returns the type name for a typed value
func (tc *TypeCaster) getType(value interface{}) string {
	switch value.(type) {
	case string:
		return "string"
	case int, int32, int64, float32, float64:
		return "number"
	case bool:
		return "boolean"
	case time.Time:
		return "date"
	case nil:
		return "null"
	default:
		rv := reflect.ValueOf(value)
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
			return "array"
		}
		return "unknown"
	}
}

// DateValidator handles date type validation and casting
type DateValidator struct{}

func (d *DateValidator) Cast(value string) (interface{}, error) {
	// Try common date formats
	formats := []string{
		"2006-01-02",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, value); err == nil {
			return t, nil
		}
	}

	return nil, fmt.Errorf("invalid date format: %s", value)
}

func (d *DateValidator) Matches(value string) bool {
	// Simple regex for date-like strings
	datePattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}(T\d{2}:\d{2}:\d{2}.*)?$`)
	return datePattern.MatchString(value)
}

// NumberValidator handles number type validation and casting
type NumberValidator struct{}

func (n *NumberValidator) Cast(value string) (interface{}, error) {
	// Try integer first
	if intVal, err := strconv.Atoi(value); err == nil {
		return intVal, nil
	}

	// Try float
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal, nil
	}

	return nil, fmt.Errorf("invalid number format: %s", value)
}

func (n *NumberValidator) Matches(value string) bool {
	_, err1 := strconv.Atoi(value)
	_, err2 := strconv.ParseFloat(value, 64)
	return err1 == nil || err2 == nil
}

// BooleanValidator handles boolean type validation and casting
type BooleanValidator struct{}

func (b *BooleanValidator) Cast(value string) (interface{}, error) {
	lower := strings.ToLower(strings.TrimSpace(value))
	switch lower {
	case "true", "yes", "1", "on":
		return true, nil
	case "false", "no", "0", "off":
		return false, nil
	default:
		return nil, fmt.Errorf("invalid boolean format: %s", value)
	}
}

func (b *BooleanValidator) Matches(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	validBools := []string{"true", "false", "yes", "no", "1", "0", "on", "off"}
	for _, valid := range validBools {
		if lower == valid {
			return true
		}
	}
	return false
}

// ArrayValidator handles array type validation and casting
type ArrayValidator struct{}

func (a *ArrayValidator) Cast(value string) (interface{}, error) {
	trimmed := strings.TrimSpace(value)
	
	// Handle bracket notation [item1, item2]
	if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
		inner := strings.Trim(trimmed, "[]")
		if inner == "" {
			return []string{}, nil
		}
		trimmed = inner
	}

	// Split by comma
	if strings.Contains(trimmed, ",") {
		parts := strings.Split(trimmed, ",")
		result := make([]string, len(parts))
		for i, part := range parts {
			result[i] = strings.TrimSpace(part)
		}
		return result, nil
	}

	// Single item
	if trimmed != "" {
		return []string{trimmed}, nil
	}

	return []string{}, nil
}

func (a *ArrayValidator) Matches(value string) bool {
	trimmed := strings.TrimSpace(value)
	// Check for bracket notation or comma-separated values
	return (strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) ||
		strings.Contains(trimmed, ",")
}

// NullValidator handles null type validation and casting
type NullValidator struct{}

func (n *NullValidator) Cast(value string) (interface{}, error) {
	return nil, nil
}

func (n *NullValidator) Matches(value string) bool {
	return strings.TrimSpace(value) == ""
}