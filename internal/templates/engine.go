package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/google/uuid"
)

// Engine provides a centralized template processing system
type Engine struct {
	cache map[string]*template.Template
	mu    sync.RWMutex
	funcs template.FuncMap
}

// Context provides variables and functions available to templates
type Context struct {
	// File-related variables
	Filename     string
	Title        string
	RelativePath string
	ParentDir    string

	// Time-related variables
	FileModTime time.Time

	// Additional custom variables
	Variables map[string]interface{}
}

// NewEngine creates a new template engine with default functions
func NewEngine() *Engine {
	engine := &Engine{
		cache: make(map[string]*template.Template),
		funcs: make(template.FuncMap),
	}

	// Register default functions
	engine.registerDefaultFunctions()

	return engine
}

// registerDefaultFunctions registers all built-in template functions
func (e *Engine) registerDefaultFunctions() {
	e.funcs["current_date"] = func() string {
		return time.Now().Format("2006-01-02")
	}

	e.funcs["current_datetime"] = func() string {
		return time.Now().Format("2006-01-02T15:04:05Z")
	}

	e.funcs["uuid"] = func() string {
		return uuid.New().String()
	}

	// String transformation filters
	e.funcs["upper"] = strings.ToUpper
	e.funcs["lower"] = strings.ToLower
	e.funcs["slug"] = e.slugify

	// Date formatting filter
	e.funcs["date"] = func(format string, t time.Time) string {
		return t.Format(format)
	}
}

// slugify converts a string to a URL-friendly slug
func (e *Engine) slugify(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Replace spaces and common separators with hyphens
	re := regexp.MustCompile(`[\s_\.]+`)
	s = re.ReplaceAllString(s, "-")

	// Remove non-alphanumeric characters except hyphens
	re = regexp.MustCompile(`[^a-z0-9\-]`)
	s = re.ReplaceAllString(s, "")

	// Remove multiple consecutive hyphens
	re = regexp.MustCompile(`-+`)
	s = re.ReplaceAllString(s, "-")

	// Trim hyphens from start and end
	s = strings.Trim(s, "-")

	return s
}

// Process processes a template string with the given context
func (e *Engine) Process(templateStr string, ctx *Context) (string, error) {
	if ctx == nil {
		ctx = &Context{Variables: make(map[string]interface{})}
	}

	// Check cache first
	e.mu.RLock()
	tmpl, exists := e.cache[templateStr]
	e.mu.RUnlock()

	if !exists {
		// Compile template
		var err error
		tmpl, err = e.compileTemplate(templateStr)
		if err != nil {
			return "", fmt.Errorf("failed to compile template: %w", err)
		}

		// Cache compiled template
		e.mu.Lock()
		e.cache[templateStr] = tmpl
		e.mu.Unlock()
	}

	// Prepare template data
	data := e.prepareData(ctx)

	// Execute template
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// compileTemplate compiles a template string with proper function mapping
func (e *Engine) compileTemplate(templateStr string) (*template.Template, error) {
	// Validate template syntax
	if err := e.validateTemplate(templateStr); err != nil {
		return nil, err
	}

	// Create template with functions
	tmpl := template.New("template").Funcs(e.funcs)

	// Parse template
	tmpl, err := tmpl.Parse(templateStr)
	if err != nil {
		return nil, fmt.Errorf("template parse error: %w", err)
	}

	return tmpl, nil
}

// validateTemplate performs security and syntax validation
func (e *Engine) validateTemplate(templateStr string) error {
	// Check for potentially dangerous patterns
	dangerous := []string{
		"{{call",
		"{{range",
		"{{with",
		"{{block",
		"{{define",
		"{{template",
	}

	lowerTemplate := strings.ToLower(templateStr)
	for _, pattern := range dangerous {
		if strings.Contains(lowerTemplate, pattern) {
			return fmt.Errorf("template contains potentially unsafe directive: %s", pattern)
		}
	}

	// Basic syntax validation - check for unmatched braces
	openCount := strings.Count(templateStr, "{{")
	closeCount := strings.Count(templateStr, "}}")
	if openCount != closeCount {
		return fmt.Errorf("template contains unmatched braces")
	}

	return nil
}

// prepareData creates the data map available to templates
func (e *Engine) prepareData(ctx *Context) map[string]interface{} {
	data := make(map[string]interface{})

	// File-related variables
	if ctx != nil {
		data["filename"] = ctx.Filename
		data["title"] = ctx.Title
		data["relative_path"] = ctx.RelativePath
		data["parent_dir"] = ctx.ParentDir
	} else {
		data["filename"] = ""
		data["title"] = ""
		data["relative_path"] = ""
		data["parent_dir"] = ""
	}

	// Time-related variables
	if ctx != nil {
		data["file_mtime"] = ctx.FileModTime
	} else {
		data["file_mtime"] = time.Time{}
	}

	// Function shortcuts (for variables that are function calls)
	data["current_date"] = time.Now().Format("2006-01-02")
	data["current_datetime"] = time.Now().Format("2006-01-02T15:04:05Z")
	data["uuid"] = uuid.New().String()

	// Custom variables
	if ctx != nil && ctx.Variables != nil {
		for k, v := range ctx.Variables {
			data[k] = v
		}
	}

	return data
}

// ProcessFile processes a template for a specific file
func (e *Engine) ProcessFile(templateStr, filePath string) (string, error) {
	// Extract file information
	abs, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	filename := filepath.Base(filePath)
	relativePath := filePath
	parentDir := filepath.Dir(filePath)

	// Get file modification time
	var modTime time.Time
	if stat, err := os.Stat(abs); err == nil {
		modTime = stat.ModTime()
	}

	// Remove extension from filename for template variable
	filenameNoExt := strings.TrimSuffix(filename, filepath.Ext(filename))

	ctx := &Context{
		Filename:     filenameNoExt,
		Title:        filenameNoExt, // Default to filename, can be overridden
		RelativePath: relativePath,
		ParentDir:    parentDir,
		FileModTime:  modTime,
		Variables:    make(map[string]interface{}),
	}

	return e.Process(templateStr, ctx)
}

// RegisterFunction adds a custom function to the template engine
func (e *Engine) RegisterFunction(name string, fn interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.funcs[name] = fn

	// Clear cache to force recompilation with new function
	e.cache = make(map[string]*template.Template)
}

// ClearCache clears the template compilation cache
func (e *Engine) ClearCache() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.cache = make(map[string]*template.Template)
}

// IsTemplate checks if a string contains template syntax
func IsTemplate(s string) bool {
	return strings.Contains(s, "{{") && strings.Contains(s, "}}")
}

// ExtractVariables extracts all variable references from a template string
func ExtractVariables(templateStr string) []string {
	re := regexp.MustCompile(`\{\{\s*\.?([a-zA-Z_][a-zA-Z0-9_]*)\s*(?:\|[^}]+)?\s*\}\}`)
	matches := re.FindAllStringSubmatch(templateStr, -1)

	var variables []string
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 && !seen[match[1]] {
			variables = append(variables, match[1])
			seen[match[1]] = true
		}
	}

	return variables
}

// ValidateVariables checks if all required variables are available in context
func (e *Engine) ValidateVariables(templateStr string, ctx *Context) error {
	variables := ExtractVariables(templateStr)
	data := e.prepareData(ctx)

	var missing []string
	for _, variable := range variables {
		if _, exists := data[variable]; !exists {
			missing = append(missing, variable)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("template references undefined variables: %v", missing)
	}

	return nil
}
