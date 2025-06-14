package processor

import (
	"testing"
	"time"
)

func TestTypeCaster_Cast(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		toType    string
		want      interface{}
		wantErr   bool
	}{
		{
			name:   "string to date",
			value:  "2023-01-01",
			toType: "date",
			want:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:   "string to number int",
			value:  "42",
			toType: "number",
			want:   42,
		},
		{
			name:   "string to number float",
			value:  "3.14",
			toType: "number",
			want:   3.14,
		},
		{
			name:   "string to boolean true",
			value:  "true",
			toType: "boolean",
			want:   true,
		},
		{
			name:   "string to boolean false",
			value:  "false",
			toType: "boolean",
			want:   false,
		},
		{
			name:   "comma string to array",
			value:  "tag1, tag2, tag3",
			toType: "array",
			want:   []string{"tag1", "tag2", "tag3"},
		},
		{
			name:   "single item to array",
			value:  "single-tag",
			toType: "array",
			want:   []string{"single-tag"},
		},
		{
			name:   "empty string to null",
			value:  "",
			toType: "null",
			want:   nil,
		},
		{
			name:    "invalid date format",
			value:   "not-a-date",
			toType:  "date",
			wantErr: true,
		},
		{
			name:    "invalid number format",
			value:   "not-a-number",
			toType:  "number",
			wantErr: true,
		},
		{
			name:    "invalid boolean format",
			value:   "maybe",
			toType:  "boolean",
			wantErr: true,
		},
		{
			name:   "already correct type - number",
			value:  42,
			toType: "number",
			want:   42,
		},
		{
			name:   "already correct type - boolean",
			value:  true,
			toType: "boolean",
			want:   true,
		},
		{
			name:   "already correct type - array",
			value:  []string{"tag1", "tag2"},
			toType: "array",
			want:   []string{"tag1", "tag2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := NewTypeCaster()
			got, err := tc.Cast(tt.value, tt.toType)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Cast() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.wantErr {
				return
			}

			// Special handling for time comparison
			if wantTime, ok := tt.want.(time.Time); ok {
				if gotTime, ok := got.(time.Time); ok {
					if !gotTime.Equal(wantTime) {
						t.Errorf("Cast() = %v, want %v", got, tt.want)
					}
					return
				}
			}

			// Special handling for slice comparison
			if wantSlice, ok := tt.want.([]string); ok {
				if gotSlice, ok := got.([]string); ok {
					if len(gotSlice) != len(wantSlice) {
						t.Errorf("Cast() = %v, want %v", got, tt.want)
						return
					}
					for i, v := range gotSlice {
						if i >= len(wantSlice) || v != wantSlice[i] {
							t.Errorf("Cast() = %v, want %v", got, tt.want)
							return
						}
					}
					return
				}
			}

			if got != tt.want {
				t.Errorf("Cast() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTypeCaster_AutoDetect(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		wantType string
	}{
		{"detect date", "2023-01-01", "date"},
		{"detect date with time", "2023-01-01T10:00:00Z", "date"},
		{"detect integer", "42", "number"},
		{"detect float", "3.14", "number"},
		{"detect boolean true", "true", "boolean"},
		{"detect boolean false", "false", "boolean"},
		{"detect array comma separated", "tag1, tag2", "array"},
		{"detect array bracket notation", "[tag1, tag2]", "array"},
		{"detect string", "just text", "string"},
		{"detect empty string", "", "string"},
		{"detect already number", 42, "number"},
		{"detect already boolean", true, "boolean"},
		{"detect already array", []string{"tag1"}, "array"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := NewTypeCaster()
			got := tc.AutoDetect(tt.value)
			if got != tt.wantType {
				t.Errorf("AutoDetect(%v) = %v, want %v", tt.value, got, tt.wantType)
			}
		})
	}
}

func TestTypeCaster_IsType(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		typeName string
		want     bool
	}{
		{"string is string", "hello", "string", true},
		{"int is number", 42, "number", true},
		{"float is number", 3.14, "number", true},
		{"bool is boolean", true, "boolean", true},
		{"slice is array", []string{"a"}, "array", true},
		{"interface slice is array", []interface{}{"a"}, "array", true},
		{"time is date", time.Now(), "date", true},
		{"string not number", "hello", "number", false},
		{"number not string", 42, "string", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := NewTypeCaster()
			got := tc.isType(tt.value, tt.typeName)
			if got != tt.want {
				t.Errorf("isType(%v, %s) = %v, want %v", tt.value, tt.typeName, got, tt.want)
			}
		})
	}
}