package processor

import (
	"github.com/eoinhurrell/mdnotes/internal/vault"
	"github.com/eoinhurrell/mdnotes/pkg/template"
)

// FrontmatterProcessor handles frontmatter operations
type FrontmatterProcessor struct {
	preserveOrder  bool
	templateEngine *template.Engine
}

// NewFrontmatterProcessor creates a new frontmatter processor
func NewFrontmatterProcessor() *FrontmatterProcessor {
	return &FrontmatterProcessor{
		preserveOrder:  true,
		templateEngine: template.NewEngine(),
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
		processedValue := p.templateEngine.Process(strVal, file)
		file.SetField(field, processedValue)
	} else {
		file.SetField(field, defaultValue)
	}

	return true
}
