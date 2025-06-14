package processor

import (
	"path/filepath"
	"strings"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// FrontmatterProcessor handles frontmatter operations
type FrontmatterProcessor struct {
	preserveOrder bool
}

// NewFrontmatterProcessor creates a new frontmatter processor
func NewFrontmatterProcessor() *FrontmatterProcessor {
	return &FrontmatterProcessor{
		preserveOrder: true,
	}
}

// Ensure adds a field with default value if it doesn't exist
// Returns true if the field was added or modified
func (p *FrontmatterProcessor) Ensure(file *vault.VaultFile, field string, defaultValue interface{}) bool {
	// Initialize frontmatter if nil
	if file.Frontmatter == nil {
		file.Frontmatter = make(map[string]interface{})
	}

	// Check if field already exists
	if _, exists := file.Frontmatter[field]; exists {
		return false
	}

	// Process template if string
	if strVal, ok := defaultValue.(string); ok {
		processedValue := p.processTemplate(strVal, file)
		file.SetField(field, processedValue)
	} else {
		file.SetField(field, defaultValue)
	}

	return true
}

// processTemplate replaces template variables in a string
func (p *FrontmatterProcessor) processTemplate(template string, file *vault.VaultFile) string {
	result := template

	// Replace {{filename}} with base filename without extension
	if strings.Contains(result, "{{filename}}") {
		filename := filepath.Base(file.Path)
		filename = strings.TrimSuffix(filename, filepath.Ext(filename))
		result = strings.ReplaceAll(result, "{{filename}}", filename)
	}

	// Replace {{title}} with title field if it exists
	if strings.Contains(result, "{{title}}") {
		if title, exists := file.GetField("title"); exists {
			if titleStr, ok := title.(string); ok {
				result = strings.ReplaceAll(result, "{{title}}", titleStr)
			}
		}
	}

	// Add more template variables as needed
	// {{current_date}}, {{uuid}}, etc. can be added here

	return result
}