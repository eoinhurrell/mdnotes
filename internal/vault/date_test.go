package vault

import (
	"strings"
	"testing"
	"time"
	
	"gopkg.in/yaml.v3"
)

func TestDate_YAMLSerialization(t *testing.T) {
	tests := []struct {
		name     string
		date     Date
		expected string
	}{
		{
			name:     "date only (midnight)",
			date:     Date{Time: time.Date(2025, 2, 16, 0, 0, 0, 0, time.UTC)},
			expected: "start: 2025-02-16",
		},
		{
			name:     "datetime with time",
			date:     Date{Time: time.Date(2023, 1, 1, 10, 30, 0, 0, time.UTC)},
			expected: "start: 2023-01-01 10:30:00",
		},
		{
			name:     "datetime with seconds",
			date:     Date{Time: time.Date(2023, 6, 15, 14, 45, 30, 0, time.UTC)},
			expected: "start: 2023-06-15 14:45:30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := map[string]interface{}{
				"start": tt.date,
			}
			
			yamlBytes, err := yaml.Marshal(data)
			if err != nil {
				t.Fatalf("yaml.Marshal() error = %v", err)
			}
			
			yamlStr := string(yamlBytes)
			
			// Should contain the expected format without quotes
			if !strings.Contains(yamlStr, tt.expected) {
				t.Errorf("Expected '%s' (without quotes), got: %s", tt.expected, yamlStr)
			}
			
			// Should NOT contain quoted values
			if strings.Contains(yamlStr, `"`+tt.date.Time.Format("2006-01-02")+`"`) {
				t.Errorf("Date should not be quoted, got: %s", yamlStr)
			}
		})
	}
}

func TestVaultFile_AutomaticDateTypeConversion(t *testing.T) {
	content := `---
title: Test File
start: 2022-11-22
end: 2024-12-31
date created: 2023-01-01 10:30:00
date modified: 2023-06-15 14:45:30
created: 2023-12-01 09:00:00
modified: 2024-01-15 16:30:45
birthday: 1985-03-15
timestamp: 2023-06-15 12:00:00
datetime: 2024-03-20 18:30:00
some_date: 2020-01-01
---

# Test Content
`

	vf := &VaultFile{}
	err := vf.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	
	// Check that ALL date fields are converted to Date type
	allDateFields := []string{"start", "end", "birthday", "some_date", "date created", "date modified", "created", "modified", "timestamp", "datetime"}
	for _, field := range allDateFields {
		value, exists := vf.Frontmatter[field]
		if !exists {
			t.Errorf("Field %s should exist", field)
			continue
		}
		
		if _, ok := value.(Date); !ok {
			t.Errorf("Field %s should be Date type, got %T", field, value)
		}
	}
	
	// Test serialization
	serialized, err := vf.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}
	
	serializedStr := string(serialized)
	
	// Date-only fields should serialize as YYYY-MM-DD
	expectedDates := []string{
		"start: 2022-11-22",
		"end: 2024-12-31", 
		"birthday: 1985-03-15",
		"some_date: 2020-01-01",
	}
	
	for _, expected := range expectedDates {
		if !strings.Contains(serializedStr, expected) {
			t.Errorf("Expected '%s' in serialized output, got: %s", expected, serializedStr)
		}
	}
	
	// Datetime fields should serialize as YYYY-MM-DD HH:mm:ss
	expectedDatetimes := []string{
		"date created: 2023-01-01 10:30:00",
		"date modified: 2023-06-15 14:45:30",
		"created: 2023-12-01 09:00:00",
		"modified: 2024-01-15 16:30:45",
		"timestamp: 2023-06-15 12:00:00",
		"datetime: 2024-03-20 18:30:00",
	}
	
	for _, expected := range expectedDatetimes {
		if !strings.Contains(serializedStr, expected) {
			t.Errorf("Expected '%s' in serialized output, got: %s", expected, serializedStr)
		}
	}
}

func TestVaultFile_RoundTripPreservesDateTypes(t *testing.T) {
	content := `---
title: Round Trip Test
regular_date: 2023-05-15
datetime_field: 2023-05-15 14:30:00
---

# Content
`

	// First parse
	vf1 := &VaultFile{}
	err := vf1.Parse([]byte(content))
	if err != nil {
		t.Fatalf("First Parse() error = %v", err)
	}
	
	// Serialize
	serialized, err := vf1.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}
	
	// Second parse of serialized content
	vf2 := &VaultFile{}
	err = vf2.Parse(serialized)
	if err != nil {
		t.Fatalf("Second Parse() error = %v", err)
	}
	
	// Check both are Date types after round trip
	if _, ok := vf2.Frontmatter["regular_date"].(Date); !ok {
		t.Errorf("regular_date should be Date type after round trip, got %T", vf2.Frontmatter["regular_date"])
	}
	
	if _, ok := vf2.Frontmatter["datetime_field"].(Date); !ok {
		t.Errorf("datetime_field should be Date type after round trip, got %T", vf2.Frontmatter["datetime_field"])
	}
	
	// Check the serialized output formats
	serializedStr := string(serialized)
	if !strings.Contains(serializedStr, "regular_date: 2023-05-15") {
		t.Errorf("Expected 'regular_date: 2023-05-15' in output, got: %s", serializedStr)
	}
	
	if !strings.Contains(serializedStr, "datetime_field: 2023-05-15 14:30:00") {
		t.Errorf("Expected 'datetime_field: 2023-05-15 14:30:00' in output, got: %s", serializedStr)
	}
}