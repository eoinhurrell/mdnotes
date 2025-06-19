package query

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// Expression represents a parsed query expression
type Expression interface {
	Evaluate(file *vault.VaultFile) bool
}

// ComparisonExpression represents field comparisons
type ComparisonExpression struct {
	Field    string
	Operator string
	Value    interface{}
}

// LogicalExpression represents AND/OR operations
type LogicalExpression struct {
	Left     Expression
	Operator string // "AND" or "OR"
	Right    Expression
}

// ContainsExpression represents contains operations
type ContainsExpression struct {
	Field string
	Value string
}

// DateExpression represents date comparisons
type DateExpression struct {
	Field    string
	Operator string // "after", "before", "within"
	Value    interface{}
}

// Parser handles parsing query expressions
type Parser struct {
	input string
	pos   int
}

// NewParser creates a new expression parser
func NewParser(input string) *Parser {
	return &Parser{
		input: strings.TrimSpace(input),
		pos:   0,
	}
}

// Parse parses the input string into an Expression
func (p *Parser) Parse() (Expression, error) {
	return p.parseLogicalExpression()
}

// parseLogicalExpression handles AND/OR operations
func (p *Parser) parseLogicalExpression() (Expression, error) {
	left, err := p.parseComparisonExpression()
	if err != nil {
		return nil, err
	}

	for p.peek() != "" {
		// Look for AND/OR operators
		if p.consumeKeyword("AND") {
			right, err := p.parseComparisonExpression()
			if err != nil {
				return nil, err
			}
			left = &LogicalExpression{
				Left:     left,
				Operator: "AND",
				Right:    right,
			}
		} else if p.consumeKeyword("OR") {
			right, err := p.parseComparisonExpression()
			if err != nil {
				return nil, err
			}
			left = &LogicalExpression{
				Left:     left,
				Operator: "OR",
				Right:    right,
			}
		} else {
			break
		}
	}

	return left, nil
}

// parseComparisonExpression handles field comparisons
func (p *Parser) parseComparisonExpression() (Expression, error) {
	p.skipWhitespace()

	// Parse field name
	field := p.consumeIdentifier()
	if field == "" {
		return nil, fmt.Errorf("expected field name at position %d", p.pos)
	}

	p.skipWhitespace()

	// Check for special operators first
	if p.consumeKeyword("contains") {
		p.skipWhitespace()
		value := p.consumeValue()
		if value == "" {
			return nil, fmt.Errorf("expected value after 'contains' at position %d", p.pos)
		}
		return &ContainsExpression{
			Field: field,
			Value: strings.Trim(value, "'\""),
		}, nil
	}

	if p.consumeKeyword("after") || p.consumeKeyword("before") {
		operator := "after"
		if strings.Contains(p.input[p.pos-10:p.pos], "before") {
			operator = "before"
		}
		p.skipWhitespace()
		value := p.consumeValue()
		if value == "" {
			return nil, fmt.Errorf("expected date value after '%s' at position %d", operator, p.pos)
		}
		return &DateExpression{
			Field:    field,
			Operator: operator,
			Value:    strings.Trim(value, "'\""),
		}, nil
	}

	if p.consumeKeyword("within") {
		p.skipWhitespace()
		value := p.consumeValue()
		if value == "" {
			return nil, fmt.Errorf("expected duration after 'within' at position %d", p.pos)
		}
		return &DateExpression{
			Field:    field,
			Operator: "within",
			Value:    strings.Trim(value, "'\""),
		}, nil
	}

	// Parse comparison operators
	p.skipWhitespace()
	var operator string
	if p.consume(">=") {
		operator = ">="
	} else if p.consume("<=") {
		operator = "<="
	} else if p.consume("!=") {
		operator = "!="
	} else if p.consume(">") {
		operator = ">"
	} else if p.consume("<") {
		operator = "<"
	} else if p.consume("=") {
		operator = "="
	} else {
		return nil, fmt.Errorf("expected comparison operator at position %d", p.pos)
	}

	p.skipWhitespace()

	// Parse value
	valueStr := p.consumeValue()
	if valueStr == "" {
		return nil, fmt.Errorf("expected value after '%s' at position %d", operator, p.pos)
	}

	// Convert value to appropriate type
	value := p.parseValue(valueStr)

	return &ComparisonExpression{
		Field:    field,
		Operator: operator,
		Value:    value,
	}, nil
}

// Helper methods for parsing

func (p *Parser) peek() string {
	if p.pos >= len(p.input) {
		return ""
	}
	return string(p.input[p.pos])
}

func (p *Parser) consume(s string) bool {
	if p.pos+len(s) <= len(p.input) && p.input[p.pos:p.pos+len(s)] == s {
		p.pos += len(s)
		return true
	}
	return false
}

func (p *Parser) consumeKeyword(keyword string) bool {
	p.skipWhitespace()
	start := p.pos
	if p.pos+len(keyword) <= len(p.input) &&
		strings.ToLower(p.input[p.pos:p.pos+len(keyword)]) == strings.ToLower(keyword) {
		// Check that it's a word boundary
		if p.pos+len(keyword) < len(p.input) {
			nextChar := p.input[p.pos+len(keyword)]
			if isAlphaNumeric(nextChar) {
				return false
			}
		}
		p.pos += len(keyword)
		return true
	}
	p.pos = start
	return false
}

func (p *Parser) consumeIdentifier() string {
	p.skipWhitespace()
	start := p.pos
	for p.pos < len(p.input) && (isAlphaNumeric(p.input[p.pos]) || p.input[p.pos] == '_') {
		p.pos++
	}
	return p.input[start:p.pos]
}

func (p *Parser) consumeValue() string {
	p.skipWhitespace()

	// Handle quoted strings
	if p.pos < len(p.input) && (p.input[p.pos] == '"' || p.input[p.pos] == '\'') {
		quote := p.input[p.pos]
		p.pos++
		start := p.pos
		for p.pos < len(p.input) && p.input[p.pos] != quote {
			p.pos++
		}
		if p.pos >= len(p.input) {
			return ""
		}
		value := p.input[start:p.pos]
		p.pos++ // Skip closing quote
		return value
	}

	// Handle unquoted values (until whitespace or operator)
	start := p.pos
	for p.pos < len(p.input) {
		char := p.input[p.pos]
		if char == ' ' || char == '\t' || char == '\n' ||
			strings.ContainsAny(string(char), "><!=") {
			break
		}
		// Stop at logical operators
		if p.pos+3 <= len(p.input) && strings.ToLower(p.input[p.pos:p.pos+3]) == "and" {
			break
		}
		if p.pos+2 <= len(p.input) && strings.ToLower(p.input[p.pos:p.pos+2]) == "or" {
			break
		}
		p.pos++
	}

	return p.input[start:p.pos]
}

func (p *Parser) parseValue(valueStr string) interface{} {
	// Remove quotes if present
	valueStr = strings.Trim(valueStr, "'\"")

	// Try to parse as number
	if intVal, err := strconv.Atoi(valueStr); err == nil {
		return intVal
	}
	if floatVal, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return floatVal
	}

	// Try to parse as boolean
	if boolVal, err := strconv.ParseBool(valueStr); err == nil {
		return boolVal
	}

	// Return as string
	return valueStr
}

func (p *Parser) skipWhitespace() {
	for p.pos < len(p.input) && (p.input[p.pos] == ' ' || p.input[p.pos] == '\t' || p.input[p.pos] == '\n') {
		p.pos++
	}
}

func isAlphaNumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// Evaluation methods

func (e *ComparisonExpression) Evaluate(file *vault.VaultFile) bool {
	value, exists := file.GetField(e.Field)
	if !exists {
		return false
	}

	switch e.Operator {
	case "=":
		return compareEqual(value, e.Value)
	case "!=":
		return !compareEqual(value, e.Value)
	case ">":
		return compareGreater(value, e.Value)
	case "<":
		return compareLess(value, e.Value)
	case ">=":
		return compareGreater(value, e.Value) || compareEqual(value, e.Value)
	case "<=":
		return compareLess(value, e.Value) || compareEqual(value, e.Value)
	default:
		return false
	}
}

func (e *LogicalExpression) Evaluate(file *vault.VaultFile) bool {
	switch e.Operator {
	case "AND":
		return e.Left.Evaluate(file) && e.Right.Evaluate(file)
	case "OR":
		return e.Left.Evaluate(file) || e.Right.Evaluate(file)
	default:
		return false
	}
}

func (e *ContainsExpression) Evaluate(file *vault.VaultFile) bool {
	value, exists := file.GetField(e.Field)
	if !exists {
		return false
	}

	switch v := value.(type) {
	case string:
		return strings.Contains(strings.ToLower(v), strings.ToLower(e.Value))
	case []interface{}:
		for _, item := range v {
			if strings.Contains(strings.ToLower(fmt.Sprintf("%v", item)), strings.ToLower(e.Value)) {
				return true
			}
		}
		return false
	case []string:
		for _, item := range v {
			if strings.Contains(strings.ToLower(item), strings.ToLower(e.Value)) {
				return true
			}
		}
		return false
	default:
		// Convert to string and check
		return strings.Contains(strings.ToLower(fmt.Sprintf("%v", v)), strings.ToLower(e.Value))
	}
}

func (e *DateExpression) Evaluate(file *vault.VaultFile) bool {
	value, exists := file.GetField(e.Field)
	if !exists {
		return false
	}

	// Parse the field value as a date
	fieldDate, err := parseDate(value)
	if err != nil {
		return false
	}

	switch e.Operator {
	case "after":
		compareDate, err := parseDate(e.Value)
		if err != nil {
			return false
		}
		return fieldDate.After(compareDate)
	case "before":
		compareDate, err := parseDate(e.Value)
		if err != nil {
			return false
		}
		return fieldDate.Before(compareDate)
	case "within":
		duration, err := parseDuration(fmt.Sprintf("%v", e.Value))
		if err != nil {
			return false
		}
		now := time.Now()
		return fieldDate.After(now.Add(-duration))
	default:
		return false
	}
}

// Helper functions for comparisons

func compareEqual(a, b interface{}) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func compareGreater(a, b interface{}) bool {
	// Try numeric comparison first
	aFloat, aErr := convertToFloat(a)
	bFloat, bErr := convertToFloat(b)
	if aErr == nil && bErr == nil {
		return aFloat > bFloat
	}

	// Fall back to string comparison
	return fmt.Sprintf("%v", a) > fmt.Sprintf("%v", b)
}

func compareLess(a, b interface{}) bool {
	// Try numeric comparison first
	aFloat, aErr := convertToFloat(a)
	bFloat, bErr := convertToFloat(b)
	if aErr == nil && bErr == nil {
		return aFloat < bFloat
	}

	// Fall back to string comparison
	return fmt.Sprintf("%v", a) < fmt.Sprintf("%v", b)
}

func convertToFloat(v interface{}) (float64, error) {
	switch val := v.(type) {
	case int:
		return float64(val), nil
	case float64:
		return val, nil
	case string:
		return strconv.ParseFloat(val, 64)
	default:
		return strconv.ParseFloat(fmt.Sprintf("%v", v), 64)
	}
}

func parseDate(v interface{}) (time.Time, error) {
	dateStr := fmt.Sprintf("%v", v)

	// Try common date formats
	formats := []string{
		"2006-01-02",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"Jan 2, 2006",
		"January 2, 2006",
		"2006/01/02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

func parseDuration(s string) (time.Duration, error) {
	// Handle common duration formats
	re := regexp.MustCompile(`(\d+)\s*(days?|weeks?|months?|years?)`)
	matches := re.FindStringSubmatch(strings.ToLower(s))

	if len(matches) == 3 {
		num, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, err
		}

		unit := matches[2]
		switch {
		case strings.HasPrefix(unit, "day"):
			return time.Duration(num) * 24 * time.Hour, nil
		case strings.HasPrefix(unit, "week"):
			return time.Duration(num) * 7 * 24 * time.Hour, nil
		case strings.HasPrefix(unit, "month"):
			return time.Duration(num) * 30 * 24 * time.Hour, nil // Approximate
		case strings.HasPrefix(unit, "year"):
			return time.Duration(num) * 365 * 24 * time.Hour, nil // Approximate
		}
	}

	// Try standard Go duration parsing
	return time.ParseDuration(s)
}
