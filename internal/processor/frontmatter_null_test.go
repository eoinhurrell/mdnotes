package processor

import (
	"testing"
	
	"github.com/eoinhurrell/mdnotes/internal/vault"
)

func TestFrontmatterProcessor_EnsureWithNull(t *testing.T) {
	tests := []struct {
		name           string
		initialFrontmatter map[string]interface{}
		field          string
		defaultValue   interface{}
		expectedModified bool
		expectedValue    interface{}
	}{
		{
			name:           "add null field",
			initialFrontmatter: map[string]interface{}{
				"title": "Test",
			},
			field:          "optional_field",
			defaultValue:   nil,
			expectedModified: true,
			expectedValue:    nil,
		},
		{
			name:           "preserve existing field over null default",
			initialFrontmatter: map[string]interface{}{
				"title": "Test",
				"optional_field": "existing_value",
			},
			field:          "optional_field",
			defaultValue:   nil,
			expectedModified: false,
			expectedValue:    "existing_value",
		},
		{
			name:           "preserve null field as is",
			initialFrontmatter: map[string]interface{}{
				"title": "Test",
				"optional_field": nil,
			},
			field:          "optional_field",
			defaultValue:   nil,
			expectedModified: false,
			expectedValue:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vf := &vault.VaultFile{
				Frontmatter: make(map[string]interface{}),
			}
			
			// Set initial frontmatter
			for k, v := range tt.initialFrontmatter {
				vf.Frontmatter[k] = v
			}
			
			processor := NewFrontmatterProcessor()
			modified := processor.Ensure(vf, tt.field, tt.defaultValue)
			
			if modified != tt.expectedModified {
				t.Errorf("Expected modified = %v, got %v", tt.expectedModified, modified)
			}
			
			actualValue, exists := vf.Frontmatter[tt.field]
			if !exists {
				t.Errorf("Field %s should exist", tt.field)
				return
			}
			
			if actualValue != tt.expectedValue {
				t.Errorf("Expected value = %v, got %v", tt.expectedValue, actualValue)
			}
		})
	}
}