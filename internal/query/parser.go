package query

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// Token types for lexical analysis
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdentifier
	TokenString
	TokenNumber
	TokenDate
	TokenBoolean
	TokenOperator
	TokenLogical
	TokenKeyword
	TokenParen
	TokenComma
	TokenFunction
)

// Token represents a lexical token
type Token struct {
	Type  TokenType
	Value string
	Pos   int
}

// Expression represents a parsed query expression
type Expression interface {
	Evaluate(file *vault.VaultFile) bool
}

// NotExpression represents NOT operations
type NotExpression struct {
	Expr Expression
}

// ComparisonExpression represents field comparisons with full operator support
type ComparisonExpression struct {
	Field    string
	Operator string // "=", "!=", ">", ">=", "<", "<=", "contains", "not contains", "in", "not in"
	Value    interface{}
}

// LogicalExpression represents AND/OR operations with proper precedence
type LogicalExpression struct {
	Left     Expression
	Operator string // "AND" or "OR"
	Right    Expression
}

// FunctionCallExpression represents built-in function calls
type FunctionCallExpression struct {
	Name string
	Args []Expression
}

// LiteralExpression represents literal values
type LiteralExpression struct {
	Value interface{}
}

// FieldExpression represents field references
type FieldExpression struct {
	Name string
}

// Legacy expressions for backward compatibility
type ContainsExpression struct {
	Field string
	Value string
}

type DateExpression struct {
	Field    string
	Operator string // "after", "before", "within"
	Value    interface{}
}

// Parser handles parsing query expressions with enhanced lexical analysis
type Parser struct {
	input  string
	tokens []Token
	pos    int
}

// NewParser creates a new expression parser
func NewParser(input string) *Parser {
	p := &Parser{
		input: strings.TrimSpace(input),
		pos:   0,
	}
	p.tokenize()
	return p
}

// tokenize performs lexical analysis
func (p *Parser) tokenize() {
	input := p.input
	pos := 0

	for pos < len(input) {
		// Skip whitespace
		if isWhitespace(input[pos]) {
			pos++
			continue
		}

		start := pos

		// String literals
		if input[pos] == '"' || input[pos] == '\'' {
			quote := input[pos]
			pos++
			for pos < len(input) && input[pos] != quote {
				if input[pos] == '\\' && pos+1 < len(input) {
					pos += 2 // Skip escaped character
				} else {
					pos++
				}
			}
			if pos < len(input) {
				pos++ // Skip closing quote
			}
			p.tokens = append(p.tokens, Token{
				Type:  TokenString,
				Value: input[start+1 : pos-1], // Remove quotes
				Pos:   start,
			})
			continue
		}

		// Numbers (integer or float)
		if isDigit(input[pos]) || (input[pos] == '.' && pos+1 < len(input) && isDigit(input[pos+1])) {
			for pos < len(input) && (isDigit(input[pos]) || input[pos] == '.') {
				pos++
			}
			p.tokens = append(p.tokens, Token{
				Type:  TokenNumber,
				Value: input[start:pos],
				Pos:   start,
			})
			continue
		}

		// Operators
		if pos+1 < len(input) {
			twoChar := input[pos : pos+2]
			if twoChar == ">=" || twoChar == "<=" || twoChar == "!=" {
				p.tokens = append(p.tokens, Token{
					Type:  TokenOperator,
					Value: twoChar,
					Pos:   start,
				})
				pos += 2
				continue
			}
		}

		if input[pos] == '=' || input[pos] == '>' || input[pos] == '<' {
			p.tokens = append(p.tokens, Token{
				Type:  TokenOperator,
				Value: string(input[pos]),
				Pos:   start,
			})
			pos++
			continue
		}

		// Parentheses
		if input[pos] == '(' || input[pos] == ')' {
			p.tokens = append(p.tokens, Token{
				Type:  TokenParen,
				Value: string(input[pos]),
				Pos:   start,
			})
			pos++
			continue
		}

		// Comma
		if input[pos] == ',' {
			p.tokens = append(p.tokens, Token{
				Type:  TokenComma,
				Value: ",",
				Pos:   start,
			})
			pos++
			continue
		}

		// Identifiers and keywords
		if isAlpha(input[pos]) || input[pos] == '_' {
			for pos < len(input) && (isAlphaNumeric(input[pos]) || input[pos] == '_') {
				pos++
			}
			value := input[start:pos]
			
			// Check for keywords and logical operators
			valueLower := strings.ToLower(value)
			switch valueLower {
			case "and", "or":
				p.tokens = append(p.tokens, Token{
					Type:  TokenLogical,
					Value: strings.ToUpper(value),
					Pos:   start,
				})
			case "not":
				p.tokens = append(p.tokens, Token{
					Type:  TokenKeyword,
					Value: "NOT",
					Pos:   start,
				})
			case "contains", "in", "after", "before", "within":
				p.tokens = append(p.tokens, Token{
					Type:  TokenKeyword,
					Value: valueLower,
					Pos:   start,
				})
			case "true", "false":
				p.tokens = append(p.tokens, Token{
					Type:  TokenBoolean,
					Value: valueLower,
					Pos:   start,
				})
			default:
				// Check if it's a function call (followed by '(')
				nextPos := pos
				for nextPos < len(input) && isWhitespace(input[nextPos]) {
					nextPos++
				}
				if nextPos < len(input) && input[nextPos] == '(' {
					p.tokens = append(p.tokens, Token{
						Type:  TokenFunction,
						Value: value,
						Pos:   start,
					})
				} else {
					p.tokens = append(p.tokens, Token{
						Type:  TokenIdentifier,
						Value: value,
						Pos:   start,
					})
				}
			}
			continue
		}

		// Unknown character - skip it
		pos++
	}

	// Add EOF token
	p.tokens = append(p.tokens, Token{
		Type:  TokenEOF,
		Value: "",
		Pos:   len(input),
	})
}

// Parse parses the tokens into an expression
func (p *Parser) Parse() (Expression, error) {
	if len(p.tokens) == 0 {
		return nil, fmt.Errorf("empty expression")
	}
	
	expr, err := p.parseOrExpression()
	if err != nil {
		return nil, err
	}

	// Check that we consumed all tokens
	if p.current().Type != TokenEOF {
		return nil, fmt.Errorf("unexpected token '%s' at position %d", p.current().Value, p.current().Pos)
	}

	return expr, nil
}

// parseOrExpression handles OR operations (lowest precedence)
func (p *Parser) parseOrExpression() (Expression, error) {
	left, err := p.parseAndExpression()
	if err != nil {
		return nil, err
	}

	for p.current().Type == TokenLogical && p.current().Value == "OR" {
		p.advance() // consume OR
		right, err := p.parseAndExpression()
		if err != nil {
			return nil, err
		}
		left = &LogicalExpression{
			Left:     left,
			Operator: "OR",
			Right:    right,
		}
	}

	return left, nil
}

// parseAndExpression handles AND operations
func (p *Parser) parseAndExpression() (Expression, error) {
	left, err := p.parseNotExpression()
	if err != nil {
		return nil, err
	}

	for p.current().Type == TokenLogical && p.current().Value == "AND" {
		p.advance() // consume AND
		right, err := p.parseNotExpression()
		if err != nil {
			return nil, err
		}
		left = &LogicalExpression{
			Left:     left,
			Operator: "AND",
			Right:    right,
		}
	}

	return left, nil
}

// parseNotExpression handles NOT operations
func (p *Parser) parseNotExpression() (Expression, error) {
	if p.current().Type == TokenKeyword && p.current().Value == "NOT" {
		p.advance() // consume NOT
		expr, err := p.parseNotExpression() // Right associative
		if err != nil {
			return nil, err
		}
		return &NotExpression{Expr: expr}, nil
	}

	return p.parseComparisonExpression()
}

// parseComparisonExpression handles comparison operations
func (p *Parser) parseComparisonExpression() (Expression, error) {
	left, err := p.parseTerm()
	if err != nil {
		return nil, err
	}

	// Check for comparison operators
	if p.current().Type == TokenOperator {
		op := p.current().Value
		p.advance()
		right, err := p.parseTerm()
		if err != nil {
			return nil, err
		}

		// Convert right term to a literal value if it's a field expression
		var rightValue interface{}
		if fieldExpr, ok := right.(*FieldExpression); ok {
			rightValue = fieldExpr.Name
		} else if litExpr, ok := right.(*LiteralExpression); ok {
			rightValue = litExpr.Value
		} else if funcExpr, ok := right.(*FunctionCallExpression); ok {
			// For now, just use the function name as a placeholder
			// In a full implementation, you'd evaluate the function
			rightValue = funcExpr.Name + "()"
		} else {
			return nil, fmt.Errorf("comparison operator '%s' requires a literal value on the right side", op)
		}

		// Left side must be a field expression
		if fieldExpr, ok := left.(*FieldExpression); ok {
			return &ComparisonExpression{
				Field:    fieldExpr.Name,
				Operator: op,
				Value:    rightValue,
			}, nil
		} else {
			return nil, fmt.Errorf("comparison operator '%s' requires a field on the left side", op)
		}
	}

	// Check for keyword operators (contains, in, etc.)
	if p.current().Type == TokenKeyword {
		keyword := p.current().Value
		
		// Handle "not contains" and "not in"
		if keyword == "NOT" {
			p.advance()
			if p.current().Type != TokenKeyword {
				return nil, fmt.Errorf("expected 'contains' or 'in' after 'not'")
			}
			keyword = "not " + p.current().Value
		}

		switch keyword {
		case "contains", "not contains", "in", "not in", "after", "before", "within":
			p.advance()
			right, err := p.parseTerm()
			if err != nil {
				return nil, err
			}

			var rightValue interface{}
			if fieldExpr, ok := right.(*FieldExpression); ok {
				rightValue = fieldExpr.Name
			} else if litExpr, ok := right.(*LiteralExpression); ok {
				rightValue = litExpr.Value
			} else if funcExpr, ok := right.(*FunctionCallExpression); ok {
				// For now, just use the function name as a placeholder
				// In a full implementation, you'd evaluate the function
				rightValue = funcExpr.Name + "()"
			} else {
				return nil, fmt.Errorf("operator '%s' requires a literal value on the right side", keyword)
			}

			if fieldExpr, ok := left.(*FieldExpression); ok {
				// Use the enhanced comparison expression for all operators
				return &ComparisonExpression{
					Field:    fieldExpr.Name,
					Operator: keyword,
					Value:    rightValue,
				}, nil
			} else {
				return nil, fmt.Errorf("operator '%s' requires a field on the left side", keyword)
			}
		}
	}

	return left, nil
}

// parseTerm handles terms (identifiers, literals, function calls, parentheses)
func (p *Parser) parseTerm() (Expression, error) {
	token := p.current()

	switch token.Type {
	case TokenIdentifier:
		p.advance()
		return &FieldExpression{Name: token.Value}, nil

	case TokenString:
		p.advance()
		return &LiteralExpression{Value: token.Value}, nil

	case TokenNumber:
		p.advance()
		// Try to parse as int first, then float
		if val, err := strconv.Atoi(token.Value); err == nil {
			return &LiteralExpression{Value: val}, nil
		}
		if val, err := strconv.ParseFloat(token.Value, 64); err == nil {
			return &LiteralExpression{Value: val}, nil
		}
		return &LiteralExpression{Value: token.Value}, nil

	case TokenBoolean:
		p.advance()
		val := token.Value == "true"
		return &LiteralExpression{Value: val}, nil

	case TokenFunction:
		return p.parseFunctionCall()

	case TokenParen:
		if token.Value == "(" {
			p.advance() // consume '('
			expr, err := p.parseOrExpression()
			if err != nil {
				return nil, err
			}
			if p.current().Value != ")" {
				return nil, fmt.Errorf("expected ')' at position %d", p.current().Pos)
			}
			p.advance() // consume ')'
			return expr, nil
		}
		return nil, fmt.Errorf("unexpected token '%s' at position %d", token.Value, token.Pos)

	default:
		return nil, fmt.Errorf("unexpected token '%s' at position %d", token.Value, token.Pos)
	}
}

// parseFunctionCall handles function calls
func (p *Parser) parseFunctionCall() (Expression, error) {
	name := p.current().Value
	p.advance() // consume function name

	if p.current().Value != "(" {
		return nil, fmt.Errorf("expected '(' after function name at position %d", p.current().Pos)
	}
	p.advance() // consume '('

	var args []Expression

	// Parse arguments
	if p.current().Value != ")" {
		for {
			arg, err := p.parseOrExpression()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)

			if p.current().Type == TokenComma {
				p.advance() // consume ','
			} else {
				break
			}
		}
	}

	if p.current().Value != ")" {
		return nil, fmt.Errorf("expected ')' at position %d", p.current().Pos)
	}
	p.advance() // consume ')'

	return &FunctionCallExpression{
		Name: name,
		Args: args,
	}, nil
}

// Helper methods
func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

// Character classification helpers
func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// Evaluation methods for new expression types

func (e *NotExpression) Evaluate(file *vault.VaultFile) bool {
	return !e.Expr.Evaluate(file)
}

func (e *FunctionCallExpression) Evaluate(file *vault.VaultFile) bool {
	// Functions typically return values used in comparisons
	// For now, just return true (this would be enhanced for actual function evaluation)
	return true
}

func (e *LiteralExpression) Evaluate(file *vault.VaultFile) bool {
	// Literals are typically used in comparisons, not standalone
	return true
}

func (e *FieldExpression) Evaluate(file *vault.VaultFile) bool {
	_, exists := file.GetField(e.Name)
	return exists
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
	case "contains":
		return evaluateContains(value, e.Value)
	case "not contains":
		return !evaluateContains(value, e.Value)
	case "in":
		return evaluateIn(e.Value, value)
	case "not in":
		return !evaluateIn(e.Value, value)
	case "after":
		return evaluateDateComparison(value, e.Value, "after")
	case "before":
		return evaluateDateComparison(value, e.Value, "before")
	case "within":
		return evaluateDateComparison(value, e.Value, "within")
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

// Enhanced helper evaluation functions

func evaluateContains(haystack, needle interface{}) bool {
	switch h := haystack.(type) {
	case string:
		needleStr := fmt.Sprintf("%v", needle)
		return strings.Contains(strings.ToLower(h), strings.ToLower(needleStr))
	case []interface{}:
		needleStr := strings.ToLower(fmt.Sprintf("%v", needle))
		for _, item := range h {
			if strings.Contains(strings.ToLower(fmt.Sprintf("%v", item)), needleStr) {
				return true
			}
		}
		return false
	case []string:
		needleStr := strings.ToLower(fmt.Sprintf("%v", needle))
		for _, item := range h {
			if strings.Contains(strings.ToLower(item), needleStr) {
				return true
			}
		}
		return false
	default:
		// Convert to string and check
		haystackStr := strings.ToLower(fmt.Sprintf("%v", h))
		needleStr := strings.ToLower(fmt.Sprintf("%v", needle))
		return strings.Contains(haystackStr, needleStr)
	}
}

func evaluateIn(needle, haystack interface{}) bool {
	switch h := haystack.(type) {
	case []interface{}:
		needleStr := fmt.Sprintf("%v", needle)
		for _, item := range h {
			if fmt.Sprintf("%v", item) == needleStr {
				return true
			}
		}
		return false
	case []string:
		needleStr := fmt.Sprintf("%v", needle)
		for _, item := range h {
			if item == needleStr {
				return true
			}
		}
		return false
	default:
		// For non-arrays, treat as contains
		return evaluateContains(haystack, needle)
	}
}

func evaluateDateComparison(fieldValue, compareValue interface{}, operator string) bool {
	// Parse the field value as a date
	fieldDate, err := parseDate(fieldValue)
	if err != nil {
		return false
	}

	switch operator {
	case "after":
		compareDate, err := parseDate(compareValue)
		if err != nil {
			return false
		}
		return fieldDate.After(compareDate)
	case "before":
		compareDate, err := parseDate(compareValue)
		if err != nil {
			return false
		}
		return fieldDate.Before(compareDate)
	case "within":
		duration, err := parseDuration(fmt.Sprintf("%v", compareValue))
		if err != nil {
			return false
		}
		now := time.Now()
		// Check if the field date is within the duration (plus or minus) from now
		return fieldDate.After(now.Add(-duration)) && fieldDate.Before(now.Add(duration))
	default:
		return false
	}
}

// Built-in function evaluation
func EvaluateFunction(name string, args []interface{}) (interface{}, error) {
	switch name {
	case "now":
		if len(args) != 0 {
			return nil, fmt.Errorf("now() takes no arguments")
		}
		return time.Now(), nil
	case "date":
		if len(args) != 1 {
			return nil, fmt.Errorf("date() takes exactly one argument")
		}
		dateStr, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("date() argument must be a string")
		}
		return parseDate(dateStr)
	case "len":
		if len(args) != 1 {
			return nil, fmt.Errorf("len() takes exactly one argument")
		}
		return evaluateLen(args[0]), nil
	case "lower":
		if len(args) != 1 {
			return nil, fmt.Errorf("lower() takes exactly one argument")
		}
		return strings.ToLower(fmt.Sprintf("%v", args[0])), nil
	case "upper":
		if len(args) != 1 {
			return nil, fmt.Errorf("upper() takes exactly one argument")
		}
		return strings.ToUpper(fmt.Sprintf("%v", args[0])), nil
	default:
		return nil, fmt.Errorf("unknown function: %s", name)
	}
}

func evaluateLen(value interface{}) int {
	switch v := value.(type) {
	case string:
		return len(v)
	case []interface{}:
		return len(v)
	case []string:
		return len(v)
	default:
		return len(fmt.Sprintf("%v", v))
	}
}

// Legacy helper functions for comparisons

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
	// Handle common duration formats including minutes and hours
	re := regexp.MustCompile(`(\d+)\s*(minutes?|mins?|hours?|hrs?|days?|weeks?|months?|years?)`)
	matches := re.FindStringSubmatch(strings.ToLower(s))

	if len(matches) == 3 {
		num, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, err
		}

		unit := matches[2]
		switch {
		case strings.HasPrefix(unit, "min"):
			return time.Duration(num) * time.Minute, nil
		case strings.HasPrefix(unit, "hour") || strings.HasPrefix(unit, "hr"):
			return time.Duration(num) * time.Hour, nil
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

	// Try standard Go duration parsing (supports ns, us, ms, s, m, h)
	return time.ParseDuration(s)
}
