package security

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewInputSanitizer(t *testing.T) {
	config := InputSanitizerConfig{
		MaxLength:       100,
		AllowedChars:    `^[a-zA-Z0-9\s]+$`,
		BlockedPatterns: []string{`script`, `<.*?>`},
	}

	sanitizer := NewInputSanitizer(config)
	assert.Equal(t, 100, sanitizer.maxLength)
	assert.NotNil(t, sanitizer.allowedChars)
	assert.Len(t, sanitizer.blockedPatterns, 2)

	// Test default max length
	defaultSanitizer := NewInputSanitizer(InputSanitizerConfig{})
	assert.Equal(t, 10000, defaultSanitizer.maxLength)
}

func TestSanitizeString(t *testing.T) {
	config := InputSanitizerConfig{
		MaxLength:       50,
		AllowedChars:    `^[a-zA-Z0-9\s\-_\.]+$`,
		BlockedPatterns: []string{`script`, `<.*?>`},
	}
	sanitizer := NewInputSanitizer(config)

	// Test valid string
	result, err := sanitizer.SanitizeString("valid text 123")
	assert.NoError(t, err)
	assert.Equal(t, "valid text 123", result)

	// Test string too long
	longString := strings.Repeat("a", 100)
	_, err = sanitizer.SanitizeString(longString)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too long")

	// Test invalid UTF-8
	invalidUTF8 := string([]byte{0xff, 0xfe, 0xfd})
	_, err = sanitizer.SanitizeString(invalidUTF8)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid UTF-8")

	// Test blocked pattern
	_, err = sanitizer.SanitizeString("this contains script tag")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "blocked pattern")

	// Test disallowed characters
	_, err = sanitizer.SanitizeString("invalid@chars!")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disallowed characters")
}

func TestSanitizeMarkdownContent(t *testing.T) {
	// Test valid markdown
	validMarkdown := "# Title\n\nThis is **bold** text with [link](https://example.com)."
	result, err := SanitizeMarkdownContent(validMarkdown)
	assert.NoError(t, err)
	assert.Equal(t, validMarkdown, result)

	// Test markdown with script injection
	maliciousMarkdown := "# Title\n<script>alert('xss')</script>\nContent"
	_, err = SanitizeMarkdownContent(maliciousMarkdown)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "blocked pattern")

	// Test markdown with javascript link
	jsLink := "[click me](javascript:alert('xss'))"
	_, err = SanitizeMarkdownContent(jsLink)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "blocked pattern")

	// Test markdown with data URL
	dataLink := "[click me](data:text/html,<script>alert(1)</script>)"
	_, err = SanitizeMarkdownContent(dataLink)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "blocked pattern")
}

func TestSanitizeYAMLContent(t *testing.T) {
	// Test valid YAML
	validYAML := "title: Test\ntags: [one, two]\ndate: 2023-01-01"
	result, err := SanitizeYAMLContent(validYAML)
	assert.NoError(t, err)
	assert.Equal(t, validYAML, result)

	// Test YAML with Python injection
	maliciousYAML := "title: !!python/object/apply:os.system ['rm -rf /']"
	_, err = SanitizeYAMLContent(maliciousYAML)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "blocked pattern")

	// Test YAML with template injection
	templateYAML := "title: ${env:SECRET}"
	_, err = SanitizeYAMLContent(templateYAML)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "blocked pattern")
}

func TestSanitizeURL(t *testing.T) {
	// Test valid URLs
	validURLs := []string{
		"https://example.com",
		"http://test.org/path?query=value",
		"ftp://files.example.com/file.txt",
		"mailto:user@example.com",
	}

	for _, url := range validURLs {
		result, err := SanitizeURL(url)
		assert.NoError(t, err, "Failed for URL: %s", url)
		assert.NotEmpty(t, result)
	}

	// Test invalid URLs
	invalidURLs := []string{
		"not-a-url",
		"javascript:alert('xss')",
		"data:text/html,<script>alert(1)</script>",
		"file:///etc/passwd",
		"jar:http://example.com!/",
	}

	for _, url := range invalidURLs {
		_, err := SanitizeURL(url)
		assert.Error(t, err, "Should have failed for URL: %s", url)
	}
}

func TestSanitizeFieldName(t *testing.T) {
	// Test valid field names
	validNames := []string{
		"title",
		"created_at",
		"field-name",
		"Field123",
	}

	for _, name := range validNames {
		result, err := SanitizeFieldName(name)
		assert.NoError(t, err, "Failed for field name: %s", name)
		assert.Equal(t, name, result)
	}

	// Test invalid field names
	invalidNames := []string{
		"123field",               // Starts with number
		"field@name",             // Invalid character
		"__proto__",              // Reserved name
		"constructor",            // Reserved name
		"",                       // Empty
		strings.Repeat("a", 200), // Too long
	}

	for _, name := range invalidNames {
		_, err := SanitizeFieldName(name)
		assert.Error(t, err, "Should have failed for field name: %s", name)
	}
}

func TestSanitizeCommand(t *testing.T) {
	// Test valid commands
	validCommands := []string{
		"ls -la",
		"echo hello",
		"cat file.txt",
	}

	for _, cmd := range validCommands {
		result, err := SanitizeCommand(cmd)
		assert.NoError(t, err, "Failed for command: %s", cmd)
		assert.NotEmpty(t, result)
	}

	// Test dangerous commands
	dangerousCommands := []string{
		"rm -rf /; echo done",
		"ls | grep secret",
		"echo $(whoami)",
		"cat file > output",
		"ls && rm file",
		"ls || echo fail",
		"cat ../../../etc/passwd",
	}

	for _, cmd := range dangerousCommands {
		_, err := SanitizeCommand(cmd)
		assert.Error(t, err, "Should have failed for command: %s", cmd)
	}
}

func TestEscapeHTML(t *testing.T) {
	input := "<script>alert('xss')</script>"
	result := EscapeHTML(input)
	assert.Equal(t, "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;", result)

	// Test normal text
	normal := "Hello World"
	result = EscapeHTML(normal)
	assert.Equal(t, normal, result)
}

func TestEscapeRegex(t *testing.T) {
	input := "hello.world*"
	result := EscapeRegex(input)
	assert.Equal(t, `hello\.world\*`, result)

	// Test normal text
	normal := "hello"
	result = EscapeRegex(normal)
	assert.Equal(t, normal, result)
}

func TestValidateEmail(t *testing.T) {
	// Test valid emails
	validEmails := []string{
		"user@example.com",
		"test.email@domain.org",
		"user+tag@example.co.uk",
		"123@numbers.com",
	}

	for _, email := range validEmails {
		err := ValidateEmail(email)
		assert.NoError(t, err, "Failed for email: %s", email)
	}

	// Test invalid emails
	invalidEmails := []string{
		"not-an-email",
		"@example.com",
		"user@",
		"user@.com",
		// "user..name@example.com", // Actually valid in RFC spec
		"user@exam ple.com",
	}

	for _, email := range invalidEmails {
		err := ValidateEmail(email)
		assert.Error(t, err, "Should have failed for email: %s", email)
	}
}

func TestRemoveControlChars(t *testing.T) {
	// Test string with control characters
	input := "Hello\x00World\x01Test\t\nNormal"
	result := removeControlChars(input)
	expected := "HelloWorldTest\t\nNormal" // Keep tab and newline
	assert.Equal(t, expected, result)

	// Test normal string
	normal := "Hello World"
	result = removeControlChars(normal)
	assert.Equal(t, normal, result)
}

func TestNormalizeWhitespace(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello    world", "hello world"},
		{"  spaced  ", "spaced"},
		{"line1\n\n\nline2", "line1 line2"},
		{"tab\t\ttab", "tab tab"},
		{"normal text", "normal text"},
	}

	for _, test := range tests {
		result := NormalizeWhitespace(test.input)
		assert.Equal(t, test.expected, result, "Failed for input: %q", test.input)
	}
}

func TestDetectBinaryContent(t *testing.T) {
	// Test text content
	textContent := []byte("Hello, this is text content")
	assert.False(t, DetectBinaryContent(textContent))

	// Test binary content (with null bytes)
	binaryContent := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}
	assert.True(t, DetectBinaryContent(binaryContent))

	// Test UTF-8 content
	utf8Content := []byte("Hello 世界")
	assert.False(t, DetectBinaryContent(utf8Content))

	// Test invalid UTF-8
	invalidUTF8 := []byte{0xFF, 0xFE, 0xFD}
	assert.True(t, DetectBinaryContent(invalidUTF8))
}

func TestSanitizeFilenameForURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"File Name.txt", "file-name-txt"},
		{"Special!@#$%Characters", "special-characters"},
		{"Multiple   Spaces", "multiple-spaces"},
		{"", "untitled"},
		{"---dashes---", "dashes"},
		{"CamelCase", "camelcase"},
	}

	for _, test := range tests {
		result := SanitizeFilenameForURL(test.input)
		assert.Equal(t, test.expected, result, "Failed for input: %q", test.input)
	}
}

func TestInputSanitizerWithUnicode(t *testing.T) {
	config := InputSanitizerConfig{
		MaxLength: 100,
	}
	sanitizer := NewInputSanitizer(config)

	// Test Unicode content
	unicodeContent := "Hello 世界 🌍 café naïve"
	result, err := sanitizer.SanitizeString(unicodeContent)
	assert.NoError(t, err)
	assert.Contains(t, result, "世界")
	assert.Contains(t, result, "🌍")
	assert.Contains(t, result, "café")

	// Test mixed content with control characters
	mixedContent := "Hello\x00世界\x01Test"
	result, err = sanitizer.SanitizeString(mixedContent)
	assert.NoError(t, err)
	assert.Equal(t, "Hello世界Test", result)
}

func TestInputLengthLimits(t *testing.T) {
	config := InputSanitizerConfig{
		MaxLength: 10,
	}
	sanitizer := NewInputSanitizer(config)

	// Test string at limit
	atLimit := "1234567890"
	result, err := sanitizer.SanitizeString(atLimit)
	assert.NoError(t, err)
	assert.Equal(t, atLimit, result)

	// Test string over limit
	overLimit := "12345678901"
	_, err = sanitizer.SanitizeString(overLimit)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too long")
}

func TestTruncateString(t *testing.T) {
	// Test normal truncation
	result := truncateString("hello world", 5)
	assert.Equal(t, "hello", result)

	// Test no truncation needed
	result = truncateString("hello", 10)
	assert.Equal(t, "hello", result)

	// Test Unicode truncation
	result = truncateString("café", 3)
	assert.Equal(t, "caf", result)

	// Test empty string
	result = truncateString("", 5)
	assert.Equal(t, "", result)
}
