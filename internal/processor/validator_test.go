package processor

import (
	"testing"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

func TestFrontmatterValidator_Validate(t *testing.T) {
	tests := []struct {
		name     string
		rules    ValidationRules
		file     *vault.VaultFile
		wantErrs []ValidationError
	}{
		{
			name: "missing required field",
			rules: ValidationRules{
				Required: []string{"title", "tags"},
			},
			file: &vault.VaultFile{
				Path: "test.md",
				Frontmatter: map[string]interface{}{
					"title": "Test",
				},
			},
			wantErrs: []ValidationError{
				{Field: "tags", Type: "missing_required", File: "test.md"},
			},
		},
		{
			name: "all required fields present",
			rules: ValidationRules{
				Required: []string{"title", "tags"},
			},
			file: &vault.VaultFile{
				Path: "test.md",
				Frontmatter: map[string]interface{}{
					"title": "Test",
					"tags":  []string{"test"},
				},
			},
			wantErrs: []ValidationError{},
		},
		{
			name: "invalid type - string instead of array",
			rules: ValidationRules{
				Types: map[string]string{
					"tags": "array",
				},
			},
			file: &vault.VaultFile{
				Path: "test.md",
				Frontmatter: map[string]interface{}{
					"tags": "not-an-array",
				},
			},
			wantErrs: []ValidationError{
				{Field: "tags", Type: "invalid_type", Expected: "array", File: "test.md"},
			},
		},
		{
			name: "valid types",
			rules: ValidationRules{
				Types: map[string]string{
					"tags":      "array",
					"published": "boolean",
					"priority":  "number",
					"created":   "string",
				},
			},
			file: &vault.VaultFile{
				Path: "test.md",
				Frontmatter: map[string]interface{}{
					"tags":      []string{"test"},
					"published": true,
					"priority":  5,
					"created":   "2023-01-01",
				},
			},
			wantErrs: []ValidationError{},
		},
		{
			name: "multiple validation errors",
			rules: ValidationRules{
				Required: []string{"title"},
				Types: map[string]string{
					"tags": "array",
				},
			},
			file: &vault.VaultFile{
				Path: "test.md",
				Frontmatter: map[string]interface{}{
					"tags": "not-an-array",
				},
			},
			wantErrs: []ValidationError{
				{Field: "title", Type: "missing_required", File: "test.md"},
				{Field: "tags", Type: "invalid_type", Expected: "array", File: "test.md"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewValidator(tt.rules)
			errors := validator.Validate(tt.file)

			if len(errors) != len(tt.wantErrs) {
				t.Errorf("Validate() errors count = %d, want %d", len(errors), len(tt.wantErrs))
				t.Errorf("Got errors: %+v", errors)
				t.Errorf("Want errors: %+v", tt.wantErrs)
				return
			}

			for i, err := range errors {
				if i >= len(tt.wantErrs) {
					t.Errorf("Unexpected error: %+v", err)
					continue
				}
				want := tt.wantErrs[i]
				if err.Field != want.Field || err.Type != want.Type || err.Expected != want.Expected || err.File != want.File {
					t.Errorf("Error %d = %+v, want %+v", i, err, want)
				}
			}
		})
	}
}

func TestValidator_ValidateType(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		typeName string
		want     bool
	}{
		{"string type valid", "hello", "string", true},
		{"string type invalid", 123, "string", false},
		{"number type valid int", 42, "number", true},
		{"number type valid float", 3.14, "number", true},
		{"number type invalid", "not-a-number", "number", false},
		{"boolean type valid", true, "boolean", true},
		{"boolean type invalid", "true", "boolean", false},
		{"array type valid slice", []string{"a", "b"}, "array", true},
		{"array type valid interface slice", []interface{}{"a", "b"}, "array", true},
		{"array type invalid", "not-an-array", "array", false},
		{"unknown type", "value", "unknown", false},
	}

	validator := NewValidator(ValidationRules{})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validator.validateType(tt.value, tt.typeName)
			if got != tt.want {
				t.Errorf("validateType(%v, %s) = %v, want %v", tt.value, tt.typeName, got, tt.want)
			}
		})
	}
}