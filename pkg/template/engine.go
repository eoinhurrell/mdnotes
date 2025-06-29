package template

import (
	"crypto/rand"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// Engine processes template strings with variable substitution
type Engine struct {
	currentTime time.Time
}

// NewEngine creates a new template engine
func NewEngine() *Engine {
	return &Engine{
		currentTime: time.Now(),
	}
}

// SetCurrentTime sets a fixed time for testing
func (e *Engine) SetCurrentTime(t time.Time) {
	e.currentTime = t
}

// Process replaces template variables in a string with actual values
func (e *Engine) Process(template string, file *vault.VaultFile) string {
	result := template

	// First handle if-else constructs
	result = e.processConditionals(result, file)

	// Then replace all template variables
	variablePattern := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	result = variablePattern.ReplaceAllStringFunc(result, func(match string) string {
		// Extract variable name (without {{ }})
		variable := strings.Trim(match, "{}")
		return e.processVariable(variable, file)
	})

	return result
}

// processVariable handles a single template variable with optional filters
func (e *Engine) processVariable(variable string, file *vault.VaultFile) string {
	// Split variable and filters: variable|filter1:param|filter2
	parts := strings.Split(variable, "|")
	varName := strings.TrimSpace(parts[0])

	// Get the value
	value := e.getVariableValue(varName, file)

	// Apply filters
	for i := 1; i < len(parts); i++ {
		filter := strings.TrimSpace(parts[i])
		value = e.applyFilter(value, filter)
	}

	return value
}

// getVariableValue returns the value for a template variable
func (e *Engine) getVariableValue(varName string, file *vault.VaultFile) string {
	switch varName {
	case "current_date":
		return e.currentTime.Format("2006-01-02")
	case "current_datetime":
		return e.currentTime.Format("2006-01-02T15:04:05Z")
	case "filename":
		filename := filepath.Base(file.Path)
		return strings.TrimSuffix(filename, filepath.Ext(filename))
	case "filename_without_datestring":
		filename := filepath.Base(file.Path)
		filenameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))
		return e.extractFilenameWithoutDatestring(filenameWithoutExt)
	case "existing_datestring":
		filename := filepath.Base(file.Path)
		filenameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))
		return e.extractDatestring(filenameWithoutExt)
	case "relative_path":
		return file.RelativePath
	case "parent_dir":
		dir := filepath.Dir(file.RelativePath)
		if dir == "." || dir == "/" {
			return ""
		}
		return filepath.Base(dir)
	case "file_mtime":
		return file.Modified.Format("2006-01-02")
	case "file_mtime_iso":
		return file.Modified.Format("2006-01-02T15:04:05Z")
	case "uuid":
		return e.generateUUID()
	case "created":
		// Handle created field specially - check frontmatter first, then use file modified time
		if value, exists := file.GetField("created"); exists {
			if str, ok := value.(string); ok {
				return str
			}
			if t, ok := value.(time.Time); ok {
				return t.Format("2006-01-02")
			}
			return fmt.Sprintf("%v", value)
		}
		return file.Modified.Format("2006-01-02")
	default:
		// Try to get from frontmatter
		if value, exists := file.GetField(varName); exists {
			if str, ok := value.(string); ok {
				return str
			}
			if t, ok := value.(time.Time); ok {
				return t.Format("2006-01-02T15:04:05Z")
			}
			return fmt.Sprintf("%v", value)
		}
		return ""
	}
}

// applyFilter applies a filter to a value
func (e *Engine) applyFilter(value, filter string) string {
	// Split filter and parameters: filter:param1:param2
	parts := strings.Split(filter, ":")
	filterName := parts[0]

	switch filterName {
	case "upper":
		return strings.ToUpper(value)
	case "lower":
		return strings.ToLower(value)
	case "slug":
		return e.slugify(value)
	case "slug_underscore":
		return e.slugifyWithUnderscore(value)
	case "date":
		if len(parts) > 1 {
			return e.formatDate(value, parts[1])
		}
		return value
	default:
		return value
	}
}

// slugify converts a string to a URL-friendly slug
func (e *Engine) slugify(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Replace spaces and special characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	s = reg.ReplaceAllString(s, "-")

	// Remove leading/trailing hyphens
	s = strings.Trim(s, "-")

	return s
}

// slugifyWithUnderscore converts a string to a slug using underscores
func (e *Engine) slugifyWithUnderscore(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Replace spaces and special characters with underscores
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	s = reg.ReplaceAllString(s, "_")

	// Remove leading/trailing underscores
	s = strings.Trim(s, "_")

	return s
}

// formatDate formats a date string with the given layout
func (e *Engine) formatDate(dateStr, layout string) string {
	// Try to parse the date in common formats
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t.Format(layout)
		}
	}

	// If parsing fails, return original string
	return dateStr
}

// generateUUID generates a random UUID v4
func (e *Engine) generateUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to timestamp-based UUID if random fails
		return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
			e.currentTime.Unix(),
			e.currentTime.Nanosecond()&0xffff,
			0x4000|(e.currentTime.Nanosecond()>>16)&0x0fff,
			0x8000|(e.currentTime.Nanosecond()>>8)&0x3fff,
			e.currentTime.Nanosecond())
	}

	// Set version (4) and variant bits
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant 10

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ExtractDatestring extracts the datestring from the beginning of a filename
// Returns the datestring if found, empty string if not found
func (e *Engine) ExtractDatestring(filename string) string {
	// Match datestring pattern at the beginning: YYYYMMDDHHMMSS
	datestringPattern := regexp.MustCompile(`^(\d{14})-`)
	matches := datestringPattern.FindStringSubmatch(filename)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// ExtractFilenameWithoutDatestring removes the datestring prefix from a filename
// Returns the filename without the datestring prefix
func (e *Engine) ExtractFilenameWithoutDatestring(filename string) string {
	// Match datestring pattern at the beginning: YYYYMMDDHHMMSS-
	datestringPattern := regexp.MustCompile(`^(\d{14})-`)
	return datestringPattern.ReplaceAllString(filename, "")
}

// SlugifyWithUnderscore converts a string to a slug using underscores (public method)
func (e *Engine) SlugifyWithUnderscore(s string) string {
	return e.slugifyWithUnderscore(s)
}

// extractDatestring extracts the datestring from the beginning of a filename
// Returns the datestring if found, empty string if not found
func (e *Engine) extractDatestring(filename string) string {
	return e.ExtractDatestring(filename)
}

// extractFilenameWithoutDatestring removes the datestring prefix from a filename
// Returns the filename without the datestring prefix
func (e *Engine) extractFilenameWithoutDatestring(filename string) string {
	return e.ExtractFilenameWithoutDatestring(filename)
}

// processConditionals handles if-else constructs in templates
func (e *Engine) processConditionals(template string, file *vault.VaultFile) string {
	// Handle {{if condition}}...{{else}}...{{end}} and {{if condition}}...{{end}}
	ifElsePattern := regexp.MustCompile(`\{\{if\s+([^}]+)\}\}(.*?)\{\{else\}\}(.*?)\{\{end\}\}`)
	ifOnlyPattern := regexp.MustCompile(`\{\{if\s+([^}]+)\}\}(.*?)\{\{end\}\}`)

	result := template

	// First handle if-else constructs
	result = ifElsePattern.ReplaceAllStringFunc(result, func(match string) string {
		matches := ifElsePattern.FindStringSubmatch(match)
		if len(matches) >= 4 {
			condition := strings.TrimSpace(matches[1])
			trueBranch := matches[2]
			falseBranch := matches[3]

			if e.evaluateCondition(condition, file) {
				return trueBranch
			} else {
				return falseBranch
			}
		}
		return match
	})

	// Then handle if-only constructs
	result = ifOnlyPattern.ReplaceAllStringFunc(result, func(match string) string {
		matches := ifOnlyPattern.FindStringSubmatch(match)
		if len(matches) >= 3 {
			condition := strings.TrimSpace(matches[1])
			trueBranch := matches[2]

			if e.evaluateCondition(condition, file) {
				return trueBranch
			} else {
				return ""
			}
		}
		return match
	})

	return result
}

// evaluateCondition evaluates a template condition
func (e *Engine) evaluateCondition(condition string, file *vault.VaultFile) bool {
	// For now, just check if the variable exists and is non-empty
	value := e.getVariableValue(condition, file)
	return value != ""
}
