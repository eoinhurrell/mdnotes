package query

import (
	"testing"
	"time"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// Test data setup
func createTestFile(frontmatter map[string]interface{}) *vault.VaultFile {
	file := &vault.VaultFile{
		Path:         "/test/file.md",
		RelativePath: "file.md",
		Frontmatter:  frontmatter,
		Modified:     time.Now(),
	}
	return file
}

// Test tokenization
func TestTokenization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "simple comparison",
			input: `status = "draft"`,
			expected: []Token{
				{Type: TokenIdentifier, Value: "status", Pos: 0},
				{Type: TokenOperator, Value: "=", Pos: 7},
				{Type: TokenString, Value: "draft", Pos: 9},
				{Type: TokenEOF, Value: "", Pos: 16},
			},
		},
		{
			name:  "numeric comparison",
			input: "priority >= 3",
			expected: []Token{
				{Type: TokenIdentifier, Value: "priority", Pos: 0},
				{Type: TokenOperator, Value: ">=", Pos: 9},
				{Type: TokenNumber, Value: "3", Pos: 12},
				{Type: TokenEOF, Value: "", Pos: 13},
			},
		},
		{
			name:  "logical expression",
			input: `status = "draft" AND priority > 3`,
			expected: []Token{
				{Type: TokenIdentifier, Value: "status", Pos: 0},
				{Type: TokenOperator, Value: "=", Pos: 7},
				{Type: TokenString, Value: "draft", Pos: 9},
				{Type: TokenLogical, Value: "AND", Pos: 17},
				{Type: TokenIdentifier, Value: "priority", Pos: 21},
				{Type: TokenOperator, Value: ">", Pos: 30},
				{Type: TokenNumber, Value: "3", Pos: 32},
				{Type: TokenEOF, Value: "", Pos: 33},
			},
		},
		{
			name:  "contains operator",
			input: `tags contains "urgent"`,
			expected: []Token{
				{Type: TokenIdentifier, Value: "tags", Pos: 0},
				{Type: TokenKeyword, Value: "contains", Pos: 5},
				{Type: TokenString, Value: "urgent", Pos: 14},
				{Type: TokenEOF, Value: "", Pos: 22},
			},
		},
		{
			name:  "function call",
			input: `created after date("2024-01-01")`,
			expected: []Token{
				{Type: TokenIdentifier, Value: "created", Pos: 0},
				{Type: TokenKeyword, Value: "after", Pos: 8},
				{Type: TokenFunction, Value: "date", Pos: 14},
				{Type: TokenParen, Value: "(", Pos: 18},
				{Type: TokenString, Value: "2024-01-01", Pos: 19},
				{Type: TokenParen, Value: ")", Pos: 31},
				{Type: TokenEOF, Value: "", Pos: 32},
			},
		},
		{
			name:  "parentheses grouping",
			input: `(priority > 3 OR status = "urgent") AND tags contains "active"`,
			expected: []Token{
				{Type: TokenParen, Value: "(", Pos: 0},
				{Type: TokenIdentifier, Value: "priority", Pos: 1},
				{Type: TokenOperator, Value: ">", Pos: 10},
				{Type: TokenNumber, Value: "3", Pos: 12},
				{Type: TokenLogical, Value: "OR", Pos: 14},
				{Type: TokenIdentifier, Value: "status", Pos: 17},
				{Type: TokenOperator, Value: "=", Pos: 24},
				{Type: TokenString, Value: "urgent", Pos: 26},
				{Type: TokenParen, Value: ")", Pos: 34},
				{Type: TokenLogical, Value: "AND", Pos: 36},
				{Type: TokenIdentifier, Value: "tags", Pos: 40},
				{Type: TokenKeyword, Value: "contains", Pos: 45},
				{Type: TokenString, Value: "active", Pos: 54},
				{Type: TokenEOF, Value: "", Pos: 62},
			},
		},
		{
			name:  "has operator tokenization",
			input: `tags has "learning"`,
			expected: []Token{
				{Type: TokenIdentifier, Value: "tags", Pos: 0},
				{Type: TokenKeyword, Value: "has", Pos: 5},
				{Type: TokenString, Value: "learning", Pos: 9},
				{Type: TokenEOF, Value: "", Pos: 19},
			},
		},
		{
			name:  "starts_with operator tokenization",
			input: `title starts_with "Project"`,
			expected: []Token{
				{Type: TokenIdentifier, Value: "title", Pos: 0},
				{Type: TokenKeyword, Value: "starts_with", Pos: 6},
				{Type: TokenString, Value: "Project", Pos: 19},
				{Type: TokenEOF, Value: "", Pos: 28},
			},
		},
		{
			name:  "ends_with operator tokenization",
			input: `filename ends_with ".md"`,
			expected: []Token{
				{Type: TokenIdentifier, Value: "filename", Pos: 0},
				{Type: TokenKeyword, Value: "ends_with", Pos: 9},
				{Type: TokenString, Value: ".md", Pos: 19},
				{Type: TokenEOF, Value: "", Pos: 24},
			},
		},
		{
			name:  "matches operator tokenization",
			input: `title matches "^Project.*"`,
			expected: []Token{
				{Type: TokenIdentifier, Value: "title", Pos: 0},
				{Type: TokenKeyword, Value: "matches", Pos: 6},
				{Type: TokenString, Value: "^Project.*", Pos: 14},
				{Type: TokenEOF, Value: "", Pos: 26},
			},
		},
		{
			name:  "between operator tokenization",
			input: `priority between "1,10"`,
			expected: []Token{
				{Type: TokenIdentifier, Value: "priority", Pos: 0},
				{Type: TokenKeyword, Value: "between", Pos: 9},
				{Type: TokenString, Value: "1,10", Pos: 17},
				{Type: TokenEOF, Value: "", Pos: 23},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.input)
			
			if len(parser.tokens) != len(tt.expected) {
				t.Errorf("Expected %d tokens, got %d", len(tt.expected), len(parser.tokens))
				return
			}

			for i, expected := range tt.expected {
				actual := parser.tokens[i]
				if actual.Type != expected.Type || actual.Value != expected.Value {
					t.Errorf("Token %d: expected {Type: %v, Value: %q}, got {Type: %v, Value: %q}", 
						i, expected.Type, expected.Value, actual.Type, actual.Value)
				}
			}
		})
	}
}

// Test expression parsing
func TestExpressionParsing(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{
			name:      "simple comparison",
			input:     `status = "draft"`,
			shouldErr: false,
		},
		{
			name:      "numeric comparison",
			input:     "priority >= 3",
			shouldErr: false,
		},
		{
			name:      "logical AND",
			input:     `status = "draft" AND priority > 3`,
			shouldErr: false,
		},
		{
			name:      "logical OR",
			input:     `priority > 5 OR status = "urgent"`,
			shouldErr: false,
		},
		{
			name:      "contains operator",
			input:     `tags contains "urgent"`,
			shouldErr: false,
		},
		{
			name:      "not contains operator",
			input:     `NOT tags contains "archived"`,
			shouldErr: false,
		},
		{
			name:      "date comparison",
			input:     `created after "2024-01-01"`,
			shouldErr: false,
		},
		{
			name:      "within duration",
			input:     `modified within "7 days"`,
			shouldErr: false,
		},
		{
			name:      "complex grouping",
			input:     `(priority > 3 OR status = "urgent") AND tags contains "active"`,
			shouldErr: false,
		},
		{
			name:      "NOT expression",
			input:     `NOT status = "archived"`,
			shouldErr: false,
		},
		{
			name:      "function call",
			input:     `created after now()`,
			shouldErr: false,
		},
		{
			name:      "has operator parsing",
			input:     `tags has "learning"`,
			shouldErr: false,
		},
		{
			name:      "starts_with operator parsing",
			input:     `title starts_with "Project"`,
			shouldErr: false,
		},
		{
			name:      "ends_with operator parsing",
			input:     `filename ends_with ".md"`,
			shouldErr: false,
		},
		{
			name:      "matches operator parsing",
			input:     `title matches "^Project.*"`,
			shouldErr: false,
		},
		{
			name:      "between operator parsing",
			input:     `priority between "1,10"`,
			shouldErr: false,
		},
		{
			name:      "not has operator parsing",
			input:     `NOT tags has "archived"`,
			shouldErr: false,
		},
		{
			name:      "invalid syntax - missing value",
			input:     "status =",
			shouldErr: true,
		},
		{
			name:      "invalid syntax - unknown operator",
			input:     "status ~~ draft",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.input)
			expr, err := parser.Parse()

			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected error for input %q, but got none", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input %q: %v", tt.input, err)
				}
				if expr == nil {
					t.Errorf("Expected expression for input %q, but got nil", tt.input)
				}
			}
		})
	}
}

// Test expression evaluation
func TestExpressionEvaluation(t *testing.T) {
	tests := []struct {
		name        string
		expression  string
		frontmatter map[string]interface{}
		expected    bool
	}{
		{
			name:       "simple string equality",
			expression: `status = "draft"`,
			frontmatter: map[string]interface{}{
				"status": "draft",
			},
			expected: true,
		},
		{
			name:       "string inequality",
			expression: `status != "published"`,
			frontmatter: map[string]interface{}{
				"status": "draft",
			},
			expected: true,
		},
		{
			name:       "numeric comparison",
			expression: "priority > 3",
			frontmatter: map[string]interface{}{
				"priority": 5,
			},
			expected: true,
		},
		{
			name:       "numeric comparison false",
			expression: "priority > 10",
			frontmatter: map[string]interface{}{
				"priority": 5,
			},
			expected: false,
		},
		{
			name:       "contains string",
			expression: `title contains "project"`,
			frontmatter: map[string]interface{}{
				"title": "My Project Notes",
			},
			expected: true,
		},
		{
			name:       "contains array",
			expression: `tags contains "urgent"`,
			frontmatter: map[string]interface{}{
				"tags": []string{"urgent", "work", "deadline"},
			},
			expected: true,
		},
		{
			name:       "not contains",
			expression: `NOT tags contains "archived"`,
			frontmatter: map[string]interface{}{
				"tags": []string{"urgent", "work", "deadline"},
			},
			expected: true,
		},
		{
			name:       "logical AND true",
			expression: `status = "draft" AND priority > 3`,
			frontmatter: map[string]interface{}{
				"status":   "draft",
				"priority": 5,
			},
			expected: true,
		},
		{
			name:       "logical AND false",
			expression: `status = "draft" AND priority > 10`,
			frontmatter: map[string]interface{}{
				"status":   "draft",
				"priority": 5,
			},
			expected: false,
		},
		{
			name:       "logical OR true",
			expression: `status = "published" OR priority > 3`,
			frontmatter: map[string]interface{}{
				"status":   "draft",
				"priority": 5,
			},
			expected: true,
		},
		{
			name:       "logical OR false",
			expression: `status = "published" OR priority > 10`,
			frontmatter: map[string]interface{}{
				"status":   "draft",
				"priority": 5,
			},
			expected: false,
		},
		{
			name:       "NOT expression",
			expression: `NOT status = "archived"`,
			frontmatter: map[string]interface{}{
				"status": "draft",
			},
			expected: true,
		},
		{
			name:       "complex grouping",
			expression: `(priority > 3 OR status = "urgent") AND tags contains "active"`,
			frontmatter: map[string]interface{}{
				"priority": 5,
				"status":   "draft",
				"tags":     []string{"active", "work"},
			},
			expected: true,
		},
		{
			name:       "missing field",
			expression: `missing_field = "value"`,
			frontmatter: map[string]interface{}{
				"status": "draft",
			},
			expected: false,
		},
		{
			name:       "date comparison",
			expression: `created after "2024-01-01"`,
			frontmatter: map[string]interface{}{
				"created": "2024-06-15",
			},
			expected: true,
		},
		{
			name:       "date comparison false",
			expression: `created after "2024-12-01"`,
			frontmatter: map[string]interface{}{
				"created": "2024-06-15",
			},
			expected: false,
		},
		{
			name:       "within duration - recent date should match",
			expression: `created within "30 days"`,
			frontmatter: map[string]interface{}{
				"created": time.Now().AddDate(0, 0, -5).Format("2006-01-02"), // 5 days ago
			},
			expected: true,
		},
		{
			name:       "within duration - old date should not match",
			expression: `created within "7 days"`,
			frontmatter: map[string]interface{}{
				"created": time.Now().AddDate(0, 0, -15).Format("2006-01-02"), // 15 days ago
			},
			expected: false,
		},
		{
			name:       "within duration - future date should match",
			expression: `due_date within "1 week"`,
			frontmatter: map[string]interface{}{
				"due_date": time.Now().AddDate(0, 0, 3).Format("2006-01-02"), // 3 days from now
			},
			expected: true,
		},
		{
			name:       "within duration - far future date should not match",
			expression: `due_date within "1 week"`,
			frontmatter: map[string]interface{}{
				"due_date": time.Now().AddDate(0, 0, 15).Format("2006-01-02"), // 15 days from now
			},
			expected: false,
		},
		{
			name:       "has operator - exact array element match",
			expression: `tags has "learning"`,
			frontmatter: map[string]interface{}{
				"tags": []string{"learning", "machine_learning", "ai"},
			},
			expected: true,
		},
		{
			name:       "has operator - should not match partial string",
			expression: `tags has "learning"`,
			frontmatter: map[string]interface{}{
				"tags": []string{"machine_learning", "deep_learning", "ai"},
			},
			expected: false,
		},
		{
			name:       "not has operator",
			expression: `NOT tags has "archived"`,
			frontmatter: map[string]interface{}{
				"tags": []string{"learning", "active"},
			},
			expected: true,
		},
		{
			name:       "starts_with operator - string field",
			expression: `title starts_with "Project"`,
			frontmatter: map[string]interface{}{
				"title": "Project Alpha Notes",
			},
			expected: true,
		},
		{
			name:       "starts_with operator - array field",
			expression: `tags starts_with "machine"`,
			frontmatter: map[string]interface{}{
				"tags": []string{"machine_learning", "deep_learning"},
			},
			expected: true,
		},
		{
			name:       "ends_with operator - string field",
			expression: `filename ends_with ".md"`,
			frontmatter: map[string]interface{}{
				"filename": "notes.md",
			},
			expected: true,
		},
		{
			name:       "ends_with operator - array field",
			expression: `tags ends_with "ing"`,
			frontmatter: map[string]interface{}{
				"tags": []string{"learning", "coding", "testing"},
			},
			expected: true,
		},
		{
			name:       "matches operator - regex pattern",
			expression: `title matches "^Project [A-Z]+"`,
			frontmatter: map[string]interface{}{
				"title": "Project ALPHA",
			},
			expected: true,
		},
		{
			name:       "matches operator - regex pattern false",
			expression: `title matches "^Project [A-Z]+"`,
			frontmatter: map[string]interface{}{
				"title": "project alpha", // lowercase doesn't match
			},
			expected: false,
		},
		{
			name:       "matches operator - array field",
			expression: `tags matches "^[a-z]+_learning$"`,
			frontmatter: map[string]interface{}{
				"tags": []string{"machine_learning", "deep_learning", "ai"},
			},
			expected: true,
		},
		{
			name:       "between operator - numeric range",
			expression: `priority between "1,5"`,
			frontmatter: map[string]interface{}{
				"priority": 3,
			},
			expected: true,
		},
		{
			name:       "between operator - numeric range false",
			expression: `priority between "1,5"`,
			frontmatter: map[string]interface{}{
				"priority": 7,
			},
			expected: false,
		},
		{
			name:       "between operator - date range",
			expression: `created between "2024-01-01,2024-12-31"`,
			frontmatter: map[string]interface{}{
				"created": "2024-06-15",
			},
			expected: true,
		},
		{
			name:       "not matches operator",
			expression: `NOT title matches "archived"`,
			frontmatter: map[string]interface{}{
				"title": "Active Project Notes",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.expression)
			expr, err := parser.Parse()
			if err != nil {
				t.Fatalf("Failed to parse expression %q: %v", tt.expression, err)
			}

			file := createTestFile(tt.frontmatter)
			result := expr.Evaluate(file)

			if result != tt.expected {
				t.Errorf("Expression %q evaluated to %v, expected %v", tt.expression, result, tt.expected)
			}
		})
	}
}

// Test operator precedence
func TestOperatorPrecedence(t *testing.T) {
	tests := []struct {
		name        string
		expression  string
		frontmatter map[string]interface{}
		expected    bool
	}{
		{
			name:       "AND has higher precedence than OR",
			expression: `status = "draft" OR priority > 5 AND tags contains "urgent"`,
			frontmatter: map[string]interface{}{
				"status":   "published",
				"priority": 3,
				"tags":     []string{"urgent"},
			},
			expected: false, // Should be evaluated as: status = "draft" OR (priority > 5 AND tags contains "urgent")
		},
		{
			name:       "parentheses override precedence",
			expression: `(status = "draft" OR priority > 5) AND tags contains "urgent"`,
			frontmatter: map[string]interface{}{
				"status":   "published",
				"priority": 3,
				"tags":     []string{"urgent"},
			},
			expected: false, // (false OR false) AND true = false
		},
		{
			name:       "NOT has highest precedence",
			expression: `NOT status = "archived" AND priority > 3`,
			frontmatter: map[string]interface{}{
				"status":   "draft",
				"priority": 5,
			},
			expected: true, // (NOT false) AND true = true AND true = true
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.expression)
			expr, err := parser.Parse()
			if err != nil {
				t.Fatalf("Failed to parse expression %q: %v", tt.expression, err)
			}

			file := createTestFile(tt.frontmatter)
			result := expr.Evaluate(file)

			if result != tt.expected {
				t.Errorf("Expression %q evaluated to %v, expected %v", tt.expression, result, tt.expected)
			}
		})
	}
}

// Test built-in functions
func TestBuiltInFunctions(t *testing.T) {
	tests := []struct {
		name     string
		function string
		args     []interface{}
		wantErr  bool
	}{
		{
			name:     "now() with no args",
			function: "now",
			args:     []interface{}{},
			wantErr:  false,
		},
		{
			name:     "now() with args should error",
			function: "now",
			args:     []interface{}{"invalid"},
			wantErr:  true,
		},
		{
			name:     "date() with valid string",
			function: "date",
			args:     []interface{}{"2024-01-01"},
			wantErr:  false,
		},
		{
			name:     "date() with invalid string",
			function: "date",
			args:     []interface{}{"invalid-date"},
			wantErr:  true,
		},
		{
			name:     "len() with string",
			function: "len",
			args:     []interface{}{"hello"},
			wantErr:  false,
		},
		{
			name:     "len() with array",
			function: "len",
			args:     []interface{}{[]string{"a", "b", "c"}},
			wantErr:  false,
		},
		{
			name:     "lower() with string",
			function: "lower",
			args:     []interface{}{"HELLO"},
			wantErr:  false,
		},
		{
			name:     "upper() with string",
			function: "upper",
			args:     []interface{}{"hello"},
			wantErr:  false,
		},
		{
			name:     "unknown function",
			function: "unknown",
			args:     []interface{}{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateFunction(tt.function, tt.args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for function %s, but got none", tt.function)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for function %s: %v", tt.function, err)
				}
				if result == nil {
					t.Errorf("Expected result for function %s, but got nil", tt.function)
				}
			}
		})
	}
}

// Test duration parsing
func TestDurationParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{
			name:     "minutes",
			input:    "30 minutes",
			expected: 30 * time.Minute,
			wantErr:  false,
		},
		{
			name:     "minutes short form",
			input:    "45 mins",
			expected: 45 * time.Minute,
			wantErr:  false,
		},
		{
			name:     "hours",
			input:    "2 hours",
			expected: 2 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "hours short form",
			input:    "6 hrs",
			expected: 6 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "days",
			input:    "7 days",
			expected: 7 * 24 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "weeks",
			input:    "2 weeks",
			expected: 14 * 24 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "months",
			input:    "1 month",
			expected: 30 * 24 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "years",
			input:    "1 year",
			expected: 365 * 24 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "Go standard format",
			input:    "2h30m",
			expected: 2*time.Hour + 30*time.Minute,
			wantErr:  false,
		},
		{
			name:     "invalid format",
			input:    "invalid",
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDuration(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for input %q, but got none", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input %q: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("parseDuration(%q) = %v, expected %v", tt.input, result, tt.expected)
				}
			}
		})
	}
}

// Test helper evaluation functions
func TestHelperEvaluationFunctions(t *testing.T) {
	t.Run("evaluateContains", func(t *testing.T) {
		tests := []struct {
			haystack interface{}
			needle   interface{}
			expected bool
		}{
			{"hello world", "world", true},
			{"hello world", "foo", false},
			{[]string{"apple", "banana", "cherry"}, "banana", true},
			{[]string{"apple", "banana", "cherry"}, "grape", false},
			{[]interface{}{"apple", "banana", "cherry"}, "banana", true},
			{123, "23", true}, // Convert to string
		}

		for _, tt := range tests {
			result := evaluateContains(tt.haystack, tt.needle)
			if result != tt.expected {
				t.Errorf("evaluateContains(%v, %v) = %v, expected %v", 
					tt.haystack, tt.needle, result, tt.expected)
			}
		}
	})

	t.Run("evaluateIn", func(t *testing.T) {
		tests := []struct {
			needle   interface{}
			haystack interface{}
			expected bool
		}{
			{"banana", []string{"apple", "banana", "cherry"}, true},
			{"grape", []string{"apple", "banana", "cherry"}, false},
			{"banana", []interface{}{"apple", "banana", "cherry"}, true},
			{"world", "hello world", true}, // Falls back to contains
		}

		for _, tt := range tests {
			result := evaluateIn(tt.needle, tt.haystack)
			if result != tt.expected {
				t.Errorf("evaluateIn(%v, %v) = %v, expected %v", 
					tt.needle, tt.haystack, result, tt.expected)
			}
		}
	})

	t.Run("evaluateLen", func(t *testing.T) {
		tests := []struct {
			value    interface{}
			expected int
		}{
			{"hello", 5},
			{[]string{"a", "b", "c"}, 3},
			{[]interface{}{"a", "b", "c", "d"}, 4},
			{123, 3}, // Converted to string "123"
		}

		for _, tt := range tests {
			result := evaluateLen(tt.value)
			if result != tt.expected {
				t.Errorf("evaluateLen(%v) = %v, expected %v", 
					tt.value, result, tt.expected)
			}
		}
	})

	t.Run("evaluateHas", func(t *testing.T) {
		tests := []struct {
			haystack interface{}
			needle   interface{}
			expected bool
		}{
			{[]string{"learning", "machine_learning", "ai"}, "learning", true},
			{[]string{"machine_learning", "deep_learning"}, "learning", false},
			{[]interface{}{"apple", "banana", "cherry"}, "banana", true},
			{[]interface{}{"apple", "banana", "cherry"}, "grape", false},
			{"exact_match", "exact_match", true},
			{"partial_match", "partial", false},
		}

		for _, tt := range tests {
			result := evaluateHas(tt.haystack, tt.needle)
			if result != tt.expected {
				t.Errorf("evaluateHas(%v, %v) = %v, expected %v", 
					tt.haystack, tt.needle, result, tt.expected)
			}
		}
	})

	t.Run("evaluateStartsWith", func(t *testing.T) {
		tests := []struct {
			fieldValue interface{}
			prefix     interface{}
			expected   bool
		}{
			{"Project Alpha", "Project", true},
			{"project alpha", "Project", true}, // case insensitive
			{"Alpha Project", "Project", false},
			{[]string{"machine_learning", "deep_learning"}, "machine", true},
			{[]string{"deep_learning", "ai"}, "machine", false},
		}

		for _, tt := range tests {
			result := evaluateStartsWith(tt.fieldValue, tt.prefix)
			if result != tt.expected {
				t.Errorf("evaluateStartsWith(%v, %v) = %v, expected %v", 
					tt.fieldValue, tt.prefix, result, tt.expected)
			}
		}
	})

	t.Run("evaluateEndsWith", func(t *testing.T) {
		tests := []struct {
			fieldValue interface{}
			suffix     interface{}
			expected   bool
		}{
			{"notes.md", ".md", true},
			{"notes.txt", ".md", false},
			{[]string{"learning", "coding", "testing"}, "ing", true},
			{[]string{"apple", "banana"}, "ing", false},
		}

		for _, tt := range tests {
			result := evaluateEndsWith(tt.fieldValue, tt.suffix)
			if result != tt.expected {
				t.Errorf("evaluateEndsWith(%v, %v) = %v, expected %v", 
					tt.fieldValue, tt.suffix, result, tt.expected)
			}
		}
	})

	t.Run("evaluateMatches", func(t *testing.T) {
		tests := []struct {
			fieldValue interface{}
			pattern    interface{}
			expected   bool
		}{
			{"Project ALPHA", "^Project [A-Z]+$", true},
			{"project alpha", "^Project [A-Z]+$", false},
			{[]string{"machine_learning", "deep_learning"}, "^[a-z]+_learning$", true},
			{[]string{"ai", "nlp"}, "^[a-z]+_learning$", false},
			{"invalid", "[invalid regex", false}, // Invalid regex should return false
		}

		for _, tt := range tests {
			result := evaluateMatches(tt.fieldValue, tt.pattern)
			if result != tt.expected {
				t.Errorf("evaluateMatches(%v, %v) = %v, expected %v", 
					tt.fieldValue, tt.pattern, result, tt.expected)
			}
		}
	})

	t.Run("evaluateBetween", func(t *testing.T) {
		tests := []struct {
			fieldValue interface{}
			rangeValue interface{}
			expected   bool
		}{
			{3, "1,5", true},
			{7, "1,5", false},
			{3.5, "1.0,5.0", true},
			{"2024-06-15", "2024-01-01,2024-12-31", true},
			{"2023-06-15", "2024-01-01,2024-12-31", false},
			{"banana", "apple,cherry", true}, // String comparison
			{"invalid", "no_comma", false},   // Invalid range format
		}

		for _, tt := range tests {
			result := evaluateBetween(tt.fieldValue, tt.rangeValue)
			if result != tt.expected {
				t.Errorf("evaluateBetween(%v, %v) = %v, expected %v", 
					tt.fieldValue, tt.rangeValue, result, tt.expected)
			}
		}
	})
}

// Test error cases
func TestErrorCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "unmatched parentheses",
			input: "(status = 'draft'",
		},
		{
			name:  "invalid operator",
			input: "status ~~ 'draft'",
		},
		{
			name:  "missing value",
			input: "status =",
		},
		{
			name:  "invalid not syntax",
			input: "NOT",
		},
		{
			name:  "missing field name",
			input: "= 'draft'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.input)
			_, err := parser.Parse()
			if err == nil {
				t.Errorf("Expected error for input %q, but got none", tt.input)
			}
		})
	}
}

// Benchmark tests
func BenchmarkSimpleExpression(b *testing.B) {
	expression := `status = "draft"`
	file := createTestFile(map[string]interface{}{
		"status": "draft",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := NewParser(expression)
		expr, _ := parser.Parse()
		expr.Evaluate(file)
	}
}

func BenchmarkComplexExpression(b *testing.B) {
	expression := `(priority > 3 OR status = "urgent") AND tags contains "active" AND NOT archived = true`
	file := createTestFile(map[string]interface{}{
		"priority": 5,
		"status":   "draft",
		"tags":     []string{"active", "work"},
		"archived": false,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := NewParser(expression)
		expr, _ := parser.Parse()
		expr.Evaluate(file)
	}
}