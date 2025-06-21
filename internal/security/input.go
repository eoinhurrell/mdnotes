package security

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/eoinhurrell/mdnotes/internal/errors"
)

// InputSanitizer provides secure input sanitization
type InputSanitizer struct {
	maxLength      int
	allowedChars   *regexp.Regexp
	blockedPatterns []*regexp.Regexp
}

// InputSanitizerConfig configures input sanitization
type InputSanitizerConfig struct {
	MaxLength      int
	AllowedChars   string
	BlockedPatterns []string
}

// NewInputSanitizer creates a new input sanitizer
func NewInputSanitizer(config InputSanitizerConfig) *InputSanitizer {
	sanitizer := &InputSanitizer{
		maxLength: config.MaxLength,
	}
	
	// Set default max length if not specified
	if sanitizer.maxLength <= 0 {
		sanitizer.maxLength = 10000 // 10KB default
	}
	
	// Compile allowed characters regex
	if config.AllowedChars != "" {
		sanitizer.allowedChars = regexp.MustCompile(config.AllowedChars)
	}
	
	// Compile blocked patterns
	for _, pattern := range config.BlockedPatterns {
		if compiled, err := regexp.Compile(pattern); err == nil {
			sanitizer.blockedPatterns = append(sanitizer.blockedPatterns, compiled)
		}
	}
	
	return sanitizer
}

// SanitizeString sanitizes a string input
func (is *InputSanitizer) SanitizeString(input string) (string, error) {
	// Check length
	if len(input) > is.maxLength {
		return "", errors.NewErrorBuilder().
			WithOperation("input sanitization").
			WithError(fmt.Errorf("input too long: %d characters (max %d)", len(input), is.maxLength)).
			WithCode(errors.ErrCodeInputInvalid).
			WithSuggestion(fmt.Sprintf("Reduce input to %d characters or less", is.maxLength)).
			Build()
	}
	
	// Check for valid UTF-8
	if !utf8.ValidString(input) {
		return "", errors.NewErrorBuilder().
			WithOperation("input validation").
			WithError(fmt.Errorf("input contains invalid UTF-8 sequences")).
			WithCode(errors.ErrCodeEncodingUnsupported).
			WithSuggestion("Ensure input is valid UTF-8 encoded text").
			Build()
	}
	
	// Remove control characters (except common whitespace)
	sanitized := removeControlChars(input)
	
	// Check against allowed characters
	if is.allowedChars != nil && !is.allowedChars.MatchString(sanitized) {
		return "", errors.NewErrorBuilder().
			WithOperation("character validation").
			WithError(fmt.Errorf("input contains disallowed characters")).
			WithCode(errors.ErrCodeInputInvalid).
			WithSuggestion("Remove special characters that are not allowed").
			Build()
	}
	
	// Check blocked patterns
	for _, pattern := range is.blockedPatterns {
		if pattern.MatchString(sanitized) {
			return "", errors.NewErrorBuilder().
				WithOperation("pattern validation").
				WithError(fmt.Errorf("input matches blocked pattern")).
				WithCode(errors.ErrCodeInputInvalid).
				WithSuggestion("Remove or modify content that matches security patterns").
				Build()
		}
	}
	
	return sanitized, nil
}

// SanitizeMarkdownContent sanitizes markdown content
func SanitizeMarkdownContent(content string) (string, error) {
	// Create sanitizer for markdown
	config := InputSanitizerConfig{
		MaxLength: 1024 * 1024, // 1MB max for markdown files
		BlockedPatterns: []string{
			// Potential script injection in markdown
			`<script[^>]*>.*?</script>`,
			`javascript:`,
			`data:text/html`,
			// Potentially dangerous markdown links
			`\[.*?\]\s*\(\s*javascript:`,
			`\[.*?\]\s*\(\s*data:`,
		},
	}
	
	sanitizer := NewInputSanitizer(config)
	return sanitizer.SanitizeString(content)
}

// SanitizeYAMLContent sanitizes YAML frontmatter content
func SanitizeYAMLContent(content string) (string, error) {
	config := InputSanitizerConfig{
		MaxLength: 64 * 1024, // 64KB max for frontmatter
		BlockedPatterns: []string{
			// YAML injection patterns
			`!!python/`,
			`!!java/`,
			`!!ruby/`,
			// Script execution attempts
			`\$\{.*?\}`,
			`<%.*?%>`,
			`\{\{.*?\}\}`, // Template injection (be careful with legitimate templates)
		},
	}
	
	sanitizer := NewInputSanitizer(config)
	return sanitizer.SanitizeString(content)
}

// SanitizeURL sanitizes and validates URLs
func SanitizeURL(input string) (string, error) {
	// Parse URL
	parsed, err := url.Parse(input)
	if err != nil {
		return "", errors.NewErrorBuilder().
			WithOperation("URL validation").
			WithError(fmt.Errorf("invalid URL format: %w", err)).
			WithCode(errors.ErrCodeInputInvalid).
			WithSuggestion("Ensure URL follows proper format (e.g., https://example.com)").
			Build()
	}
	
	// Check scheme whitelist
	allowedSchemes := []string{"http", "https", "ftp", "ftps", "mailto"}
	if !contains(allowedSchemes, parsed.Scheme) {
		return "", errors.NewErrorBuilder().
			WithOperation("URL scheme validation").
			WithError(fmt.Errorf("disallowed URL scheme: %s", parsed.Scheme)).
			WithCode(errors.ErrCodeInputInvalid).
			WithSuggestion("Use http, https, ftp, ftps, or mailto schemes only").
			Build()
	}
	
	// Check for suspicious patterns in URL
	suspicious := []string{
		"javascript:",
		"data:",
		"vbscript:",
		"file:",
		"jar:",
	}
	
	lowerURL := strings.ToLower(input)
	for _, pattern := range suspicious {
		if strings.Contains(lowerURL, pattern) {
			return "", errors.NewErrorBuilder().
				WithOperation("URL security check").
				WithError(fmt.Errorf("URL contains suspicious pattern: %s", pattern)).
				WithCode(errors.ErrCodeInputInvalid).
				WithSuggestion("Remove suspicious protocols or patterns from URL").
				Build()
		}
	}
	
	// Return cleaned URL
	return parsed.String(), nil
}

// SanitizeFieldName sanitizes field names for frontmatter
func SanitizeFieldName(fieldName string) (string, error) {
	config := InputSanitizerConfig{
		MaxLength:   100,
		AllowedChars: `^[a-zA-Z0-9_-]+$`,
	}
	
	sanitizer := NewInputSanitizer(config)
	
	// Additional validation for field names
	sanitized, err := sanitizer.SanitizeString(fieldName)
	if err != nil {
		return "", err
	}
	
	// Must start with letter
	if len(sanitized) > 0 && !unicode.IsLetter(rune(sanitized[0])) {
		return "", errors.NewErrorBuilder().
			WithOperation("field name validation").
			WithError(fmt.Errorf("field name must start with a letter")).
			WithCode(errors.ErrCodeInputInvalid).
			WithSuggestion("Start field names with a letter (a-z, A-Z)").
			Build()
	}
	
	// Check for reserved field names
	reserved := []string{
		"__proto__",
		"constructor",
		"prototype",
		"toString",
		"valueOf",
	}
	
	for _, res := range reserved {
		if strings.EqualFold(sanitized, res) {
			return "", errors.NewErrorBuilder().
				WithOperation("field name validation").
				WithError(fmt.Errorf("field name is reserved: %s", sanitized)).
				WithCode(errors.ErrCodeInputInvalid).
				WithSuggestion("Use a different field name that is not reserved").
				Build()
		}
	}
	
	return sanitized, nil
}

// SanitizeCommand sanitizes command line arguments
func SanitizeCommand(command string) (string, error) {
	config := InputSanitizerConfig{
		MaxLength: 1000,
		BlockedPatterns: []string{
			// Command injection patterns
			`;`,
			`\|`,
			`&`,
			`\$\(`,
			`\$\{`,
			"`",
			`\|\|`,
			`&&`,
			// Path traversal in commands
			`\.\.`,
			// Redirection
			`>`,
			`<`,
			`>>`,
		},
	}
	
	sanitizer := NewInputSanitizer(config)
	return sanitizer.SanitizeString(command)
}

// EscapeHTML escapes HTML special characters
func EscapeHTML(input string) string {
	return html.EscapeString(input)
}

// EscapeRegex escapes special regex characters
func EscapeRegex(input string) string {
	return regexp.QuoteMeta(input)
}

// ValidateEmail validates email addresses
func ValidateEmail(email string) error {
	// Basic email regex (not RFC 5322 compliant but good enough for basic validation)
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	
	if !emailRegex.MatchString(email) {
		return errors.NewErrorBuilder().
			WithOperation("email validation").
			WithError(fmt.Errorf("invalid email format: %s", email)).
			WithCode(errors.ErrCodeInputInvalid).
			WithSuggestion("Use proper email format (user@domain.com)").
			Build()
	}
	
	return nil
}

// Helper functions

// removeControlChars removes control characters except common whitespace
func removeControlChars(input string) string {
	var result strings.Builder
	
	for _, r := range input {
		// Keep common whitespace characters
		if r == '\t' || r == '\n' || r == '\r' || r == ' ' {
			result.WriteRune(r)
			continue
		}
		
		// Remove other control characters
		if unicode.IsControl(r) {
			continue
		}
		
		result.WriteRune(r)
	}
	
	return result.String()
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// truncateString safely truncates a string to max length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	
	// Find a safe truncation point (avoid breaking UTF-8)
	for i := maxLen; i >= 0; i-- {
		if utf8.ValidString(s[:i]) {
			return s[:i]
		}
	}
	
	return ""
}

// NormalizeWhitespace normalizes whitespace in strings
func NormalizeWhitespace(input string) string {
	// Replace multiple whitespace with single space
	wsRegex := regexp.MustCompile(`\s+`)
	normalized := wsRegex.ReplaceAllString(input, " ")
	
	// Trim leading/trailing whitespace
	return strings.TrimSpace(normalized)
}

// DetectBinaryContent checks if content appears to be binary
func DetectBinaryContent(content []byte) bool {
	// Check for null bytes (common in binary files)
	for _, b := range content {
		if b == 0 {
			return true
		}
	}
	
	// Check UTF-8 validity for text content
	return !utf8.Valid(content)
}

// SanitizeFilenameForURL creates URL-safe filenames
func SanitizeFilenameForURL(filename string) string {
	// Convert to lowercase
	result := strings.ToLower(filename)
	
	// Replace spaces and special chars with hyphens
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	result = reg.ReplaceAllString(result, "-")
	
	// Remove leading/trailing hyphens
	result = strings.Trim(result, "-")
	
	// Ensure not empty
	if result == "" {
		result = "untitled"
	}
	
	return result
}