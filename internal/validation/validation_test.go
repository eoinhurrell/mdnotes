package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValidator(t *testing.T) {
	v := NewValidator()
	assert.NotNil(t, v)
	assert.False(t, v.HasErrors())
	assert.NotNil(t, v.GetErrors())
}

func TestValidateRequired(t *testing.T) {
	v := NewValidator()

	// Test required string - valid
	v.ValidateRequired("name", "test")
	assert.False(t, v.HasErrors())

	// Test required string - empty
	v = NewValidator()
	v.ValidateRequired("name", "")
	assert.True(t, v.HasErrors())

	// Test required string - whitespace only
	v = NewValidator()
	v.ValidateRequired("name", "   ")
	assert.True(t, v.HasErrors())

	// Test required value - nil
	v = NewValidator()
	v.ValidateRequired("name", nil)
	assert.True(t, v.HasErrors())

	// Test required array - empty
	v = NewValidator()
	v.ValidateRequired("items", []interface{}{})
	assert.True(t, v.HasErrors())

	// Test required map - empty
	v = NewValidator()
	v.ValidateRequired("config", map[string]interface{}{})
	assert.True(t, v.HasErrors())

	// Test custom message
	v = NewValidator()
	v.ValidateRequired("name", nil, "Custom error message")
	assert.True(t, v.HasErrors())
	errors := v.GetErrors()
	assert.Greater(t, len(errors), 0)
	assert.Contains(t, errors[0].Error(), "Custom error message")
}

func TestValidateString(t *testing.T) {
	// Test valid string
	v := NewValidator()
	constraints := StringConstraints{
		MinLength: 3,
		MaxLength: 10,
		Pattern:   "^[a-z]+$",
	}
	v.ValidateString("name", "hello", constraints)
	assert.False(t, v.HasErrors())

	// Test invalid type
	v = NewValidator()
	v.ValidateString("name", 123, constraints)
	assert.True(t, v.HasErrors())

	// Test min length violation
	v = NewValidator()
	v.ValidateString("name", "hi", constraints)
	assert.True(t, v.HasErrors())

	// Test max length violation
	v = NewValidator()
	v.ValidateString("name", "verylongstring", constraints)
	assert.True(t, v.HasErrors())

	// Test pattern violation
	v = NewValidator()
	v.ValidateString("name", "hello123", constraints)
	assert.True(t, v.HasErrors())

	// Test allowed values
	v = NewValidator()
	constraints = StringConstraints{
		AllowedValues: []string{"red", "green", "blue"},
	}
	v.ValidateString("color", "yellow", constraints)
	assert.True(t, v.HasErrors())

	v = NewValidator()
	v.ValidateString("color", "red", constraints)
	assert.False(t, v.HasErrors())

	// Test custom validator
	v = NewValidator()
	constraints = StringConstraints{
		Validator: func(s string) error {
			if s == "forbidden" {
				return fmt.Errorf("this value is forbidden")
			}
			return nil
		},
	}
	v.ValidateString("name", "forbidden", constraints)
	assert.True(t, v.HasErrors())
}

func TestValidateNumber(t *testing.T) {
	constraints := NumberConstraints{
		Min:          Float64Ptr(0),
		Max:          Float64Ptr(100),
		PositiveOnly: true,
	}

	// Test valid number
	v := NewValidator()
	v.ValidateNumber("count", 50, constraints)
	assert.False(t, v.HasErrors())

	// Test different number types
	v = NewValidator()
	v.ValidateNumber("count", int64(50), constraints)
	assert.False(t, v.HasErrors())

	v = NewValidator()
	v.ValidateNumber("count", 50.5, constraints)
	assert.False(t, v.HasErrors())

	v = NewValidator()
	v.ValidateNumber("count", "50", constraints)
	assert.False(t, v.HasErrors())

	// Test invalid type
	v = NewValidator()
	v.ValidateNumber("count", "not-a-number", constraints)
	assert.True(t, v.HasErrors())

	// Test min violation
	v = NewValidator()
	v.ValidateNumber("count", -5, constraints)
	assert.True(t, v.HasErrors())

	// Test max violation
	v = NewValidator()
	v.ValidateNumber("count", 150, constraints)
	assert.True(t, v.HasErrors())

	// Test positive only violation
	v = NewValidator()
	v.ValidateNumber("count", 0, constraints)
	assert.True(t, v.HasErrors())

	// Test integer only
	constraints.IntegerOnly = true
	constraints.PositiveOnly = false
	constraints.Min = nil
	constraints.Max = nil

	v = NewValidator()
	v.ValidateNumber("count", 50.5, constraints)
	assert.True(t, v.HasErrors())

	v = NewValidator()
	v.ValidateNumber("count", 50, constraints)
	assert.False(t, v.HasErrors())
}

func TestValidatePath(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(tmpFile, []byte("test"), 0644)
	require.NoError(t, err)

	// Test existing file
	v := NewValidator()
	constraints := PathConstraints{
		MustExist:         true,
		MustBeFile:        true,
		AllowedExtensions: []string{".md", ".txt"},
	}
	v.ValidatePath("file", tmpFile, constraints)
	assert.False(t, v.HasErrors())

	// Test non-existent file
	v = NewValidator()
	v.ValidatePath("file", "/non/existent/file.md", constraints)
	assert.True(t, v.HasErrors())

	// Test directory when file expected
	v = NewValidator()
	constraints.MustBeFile = false
	constraints.MustBeDirectory = true
	v.ValidatePath("dir", tmpFile, constraints)
	assert.True(t, v.HasErrors())

	// Test valid directory
	v = NewValidator()
	constraints = PathConstraints{
		MustExist:       true,
		MustBeDirectory: true,
	}
	v.ValidatePath("dir", tmpDir, constraints)
	assert.False(t, v.HasErrors())

	// Test extension validation
	v = NewValidator()
	constraints = PathConstraints{
		AllowedExtensions: []string{".txt"},
	}
	v.ValidatePath("file", "test.md", constraints)
	assert.True(t, v.HasErrors())

	// Test absolute path requirement
	v = NewValidator()
	constraints = PathConstraints{
		MustBeAbsolute: true,
	}
	v.ValidatePath("file", "relative/path", constraints)
	assert.True(t, v.HasErrors())

	v = NewValidator()
	v.ValidatePath("file", "/absolute/path", constraints)
	assert.False(t, v.HasErrors())

	// Test path traversal prevention
	v = NewValidator()
	constraints = PathConstraints{
		PreventTraversal: true,
	}
	v.ValidatePath("file", "../../../etc/passwd", constraints)
	assert.True(t, v.HasErrors())

	// Test invalid type
	v = NewValidator()
	v.ValidatePath("file", 123, PathConstraints{})
	assert.True(t, v.HasErrors())
}

func TestValidateURL(t *testing.T) {
	constraints := URLConstraints{
		AllowedSchemes: []string{"http", "https"},
		RequireHost:    true,
	}

	// Test valid URL
	v := NewValidator()
	v.ValidateURL("url", "https://example.com", constraints)
	assert.False(t, v.HasErrors())

	// Test invalid URL
	v = NewValidator()
	v.ValidateURL("url", "not-a-url", constraints)
	assert.True(t, v.HasErrors())

	// Test invalid scheme
	v = NewValidator()
	v.ValidateURL("url", "ftp://example.com", constraints)
	assert.True(t, v.HasErrors())

	// Test missing host
	v = NewValidator()
	v.ValidateURL("url", "https://", constraints)
	assert.True(t, v.HasErrors())

	// Test invalid type
	v = NewValidator()
	v.ValidateURL("url", 123, constraints)
	assert.True(t, v.HasErrors())
}

func TestValidateDate(t *testing.T) {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	tomorrow := now.AddDate(0, 0, 1)

	constraints := DateConstraints{
		After:  &yesterday,
		Before: &tomorrow,
	}

	// Test valid date
	v := NewValidator()
	v.ValidateDate("date", now, constraints)
	assert.False(t, v.HasErrors())

	// Test valid date string
	v = NewValidator()
	v.ValidateDate("date", now.Format("2006-01-02"), constraints)
	assert.False(t, v.HasErrors())

	// Test date too early
	v = NewValidator()
	pastDate := yesterday.AddDate(0, 0, -1)
	v.ValidateDate("date", pastDate, constraints)
	assert.True(t, v.HasErrors())

	// Test date too late
	v = NewValidator()
	futureDate := tomorrow.AddDate(0, 0, 1)
	v.ValidateDate("date", futureDate, constraints)
	assert.True(t, v.HasErrors())

	// Test invalid date string
	v = NewValidator()
	v.ValidateDate("date", "not-a-date", constraints)
	assert.True(t, v.HasErrors())

	// Test custom format
	v = NewValidator()
	constraints = DateConstraints{
		Format: "02/01/2006",
	}
	v.ValidateDate("date", "15/12/2023", constraints)
	assert.False(t, v.HasErrors())

	// Test invalid type
	v = NewValidator()
	v.ValidateDate("date", 123, constraints)
	assert.True(t, v.HasErrors())
}

func TestValidateArray(t *testing.T) {
	constraints := ArrayConstraints{
		MinLength:    2,
		MaxLength:    5,
		UniqueValues: true,
	}

	// Test valid array
	v := NewValidator()
	v.ValidateArray("items", []interface{}{"a", "b", "c"}, constraints)
	assert.False(t, v.HasErrors())

	// Test string array
	v = NewValidator()
	v.ValidateArray("items", []string{"a", "b", "c"}, constraints)
	assert.False(t, v.HasErrors())

	// Test too short
	v = NewValidator()
	v.ValidateArray("items", []interface{}{"a"}, constraints)
	assert.True(t, v.HasErrors())

	// Test too long
	v = NewValidator()
	v.ValidateArray("items", []interface{}{"a", "b", "c", "d", "e", "f"}, constraints)
	assert.True(t, v.HasErrors())

	// Test duplicate values
	v = NewValidator()
	v.ValidateArray("items", []interface{}{"a", "b", "a"}, constraints)
	assert.True(t, v.HasErrors())

	// Test element validator
	v = NewValidator()
	constraints = ArrayConstraints{
		ElementValidator: func(item interface{}) error {
			if str, ok := item.(string); ok && str == "invalid" {
				return fmt.Errorf("invalid value")
			}
			return nil
		},
	}
	v.ValidateArray("items", []interface{}{"valid", "invalid", "valid"}, constraints)
	assert.True(t, v.HasErrors())

	// Test invalid type
	v = NewValidator()
	v.ValidateArray("items", "not-an-array", constraints)
	assert.True(t, v.HasErrors())
}

func TestCommonValidators(t *testing.T) {
	// Test ValidateEmail
	assert.NoError(t, ValidateEmail("test@example.com"))
	assert.Error(t, ValidateEmail("invalid-email"))
	assert.Error(t, ValidateEmail("test@"))
	assert.Error(t, ValidateEmail("@example.com"))

	// Test ValidateMarkdownExtension
	assert.NoError(t, ValidateMarkdownExtension("test.md"))
	assert.NoError(t, ValidateMarkdownExtension("test.markdown"))
	assert.NoError(t, ValidateMarkdownExtension("test.mdown"))
	assert.NoError(t, ValidateMarkdownExtension("test.mkd"))
	assert.Error(t, ValidateMarkdownExtension("test.txt"))

	// Test ValidateYAMLExtension
	assert.NoError(t, ValidateYAMLExtension("config.yaml"))
	assert.NoError(t, ValidateYAMLExtension("config.yml"))
	assert.Error(t, ValidateYAMLExtension("config.json"))

	// Test ValidateSlug
	assert.NoError(t, ValidateSlug("hello-world"))
	assert.NoError(t, ValidateSlug("test123"))
	assert.Error(t, ValidateSlug("Hello-World"))  // uppercase
	assert.Error(t, ValidateSlug("hello_world"))  // underscore
	assert.Error(t, ValidateSlug("hello--world")) // double hyphen

	// Test ValidateVersion
	assert.NoError(t, ValidateVersion("1.2.3"))
	assert.NoError(t, ValidateVersion("v1.2.3"))
	assert.NoError(t, ValidateVersion("1.2.3-alpha"))
	assert.NoError(t, ValidateVersion("1.2.3+build"))
	assert.Error(t, ValidateVersion("1.2"))     // incomplete
	assert.Error(t, ValidateVersion("1.2.3.4")) // too many parts
}

func TestHelperFunctions(t *testing.T) {
	// Test Float64Ptr
	f := Float64Ptr(42.5)
	assert.Equal(t, 42.5, *f)

	// Test TimePtr
	now := time.Now()
	timePtr := TimePtr(now)
	assert.Equal(t, now, *timePtr)

	// Test SanitizeFilename
	tests := []struct {
		input    string
		expected string
	}{
		{"normal.txt", "normal.txt"},
		{"file<with>invalid:chars", "file_with_invalid_chars"},
		{"  spaced file  ", "spaced file"},
		{"", "untitled"},
		{"...", "untitled"},
		{"file/with\\slashes", "file_with_slashes"},
	}

	for _, test := range tests {
		result := SanitizeFilename(test.input)
		assert.Equal(t, test.expected, result, "Failed for input: %s", test.input)
	}

	// Test SanitizePath
	cleanPath, err := SanitizePath("path/to/file.txt")
	assert.NoError(t, err)
	assert.Equal(t, "path/to/file.txt", cleanPath)

	_, err = SanitizePath("path/../../../etc/passwd")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "traversal")

	cleanPath, err = SanitizePath("./path/to/./file.txt")
	assert.NoError(t, err)
	assert.Equal(t, "path/to/file.txt", cleanPath)
}

func TestValidatorChaining(t *testing.T) {
	v := NewValidator()

	// Chain multiple validations
	v.ValidateRequired("name", "test").
		ValidateString("name", "test", StringConstraints{MinLength: 3}).
		ValidateRequired("count", 5).
		ValidateNumber("count", 5, NumberConstraints{Min: Float64Ptr(0)})

	assert.False(t, v.HasErrors())

	// Chain with errors
	v = NewValidator()
	v.ValidateRequired("name", "").
		ValidateNumber("count", "invalid", NumberConstraints{})

	assert.True(t, v.HasErrors())
	assert.Equal(t, 2, len(v.GetErrors()))
}

func TestComplexValidationScenario(t *testing.T) {
	v := NewValidator()

	// Simulate validating a complex configuration
	config := map[string]interface{}{
		"name":        "test-project",
		"version":     "1.2.3",
		"description": "A test project",
		"url":         "https://example.com",
		"tags":        []string{"test", "example"},
		"count":       42,
		"enabled":     true,
	}

	// Validate each field
	v.ValidateRequired("name", config["name"]).
		ValidateString("name", config["name"], StringConstraints{
			MinLength: 1,
			MaxLength: 50,
			Pattern:   "^[a-z0-9-]+$",
		})

	v.ValidateRequired("version", config["version"]).
		ValidateString("version", config["version"], StringConstraints{
			Validator: ValidateVersion,
		})

	v.ValidateRequired("url", config["url"]).
		ValidateURL("url", config["url"], URLConstraints{
			AllowedSchemes: []string{"http", "https"},
			RequireHost:    true,
		})

	v.ValidateRequired("tags", config["tags"]).
		ValidateArray("tags", config["tags"], ArrayConstraints{
			MinLength:    1,
			MaxLength:    10,
			UniqueValues: true,
			ElementValidator: func(item interface{}) error {
				if str, ok := item.(string); ok {
					return ValidateSlug(str)
				}
				return fmt.Errorf("tag must be a string")
			},
		})

	v.ValidateRequired("count", config["count"]).
		ValidateNumber("count", config["count"], NumberConstraints{
			Min:          Float64Ptr(0),
			Max:          Float64Ptr(100),
			IntegerOnly:  true,
			PositiveOnly: true,
		})

	assert.False(t, v.HasErrors(), "Validation should pass for valid config")

	// Test with invalid config
	v = NewValidator()
	invalidConfig := map[string]interface{}{
		"name":    "Invalid Name!",
		"version": "invalid-version",
		"url":     "not-a-url",
		"tags":    []string{"Valid-Tag!", "duplicate", "duplicate"},
		"count":   -5,
	}

	v.ValidateString("name", invalidConfig["name"], StringConstraints{
		Pattern: "^[a-z0-9-]+$",
	}).
		ValidateString("version", invalidConfig["version"], StringConstraints{
			Validator: ValidateVersion,
		}).
		ValidateURL("url", invalidConfig["url"], URLConstraints{
			AllowedSchemes: []string{"http", "https"},
		}).
		ValidateArray("tags", invalidConfig["tags"], ArrayConstraints{
			UniqueValues: true,
			ElementValidator: func(item interface{}) error {
				if str, ok := item.(string); ok {
					return ValidateSlug(str)
				}
				return fmt.Errorf("tag must be a string")
			},
		}).
		ValidateNumber("count", invalidConfig["count"], NumberConstraints{
			PositiveOnly: true,
		})

	assert.True(t, v.HasErrors())
	assert.Greater(t, len(v.GetErrors()), 3) // Should have multiple errors
}
