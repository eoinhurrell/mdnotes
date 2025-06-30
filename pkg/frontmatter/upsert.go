package frontmatter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/eoinhurrell/mdnotes/internal/templates"
)

// UpsertResult represents the result of an upsert operation
type UpsertResult struct {
	FilePath      string
	FieldsAdded   []string
	FieldsUpdated []string
	Error         error
}

// UpsertStats tracks operation statistics
type UpsertStats struct {
	FilesProcessed int64
	FilesModified  int64
	FieldsAdded    int64
	FieldsUpdated  int64
	Errors         int64
}

// UpsertService handles batch upsert operations
type UpsertService struct {
	processor      *Processor
	templateEngine *templates.Engine
}

// TemplateEngineAdapter adapts templates.Engine to TemplateEngine interface
type TemplateEngineAdapter struct {
	engine *templates.Engine
}

func (t *TemplateEngineAdapter) Process(template string, ctx interface{}) (string, error) {
	// Convert to templates.Context if needed
	var templateCtx *templates.Context
	if ctx != nil {
		if tCtx, ok := ctx.(*templates.Context); ok {
			templateCtx = tCtx
		}
	}
	return t.engine.Process(template, templateCtx)
}

// NewUpsertService creates a new upsert service
func NewUpsertService() *UpsertService {
	templateEngine := templates.NewEngine()
	adapter := &TemplateEngineAdapter{engine: templateEngine}
	processor := NewProcessor(adapter)

	return &UpsertService{
		processor:      processor,
		templateEngine: templateEngine,
	}
}

// UpsertFile performs upsert operation on a single file
func (s *UpsertService) UpsertFile(filePath string, options UpsertOptions) (*UpsertResult, error) {
	result := &UpsertResult{
		FilePath: filePath,
	}

	// Parse the file
	doc, err := s.processor.Parse(filePath)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse file: %w", err)
		return result, result.Error
	}

	// Create template context
	ctx, err := s.createTemplateContext(filePath, doc)
	if err != nil {
		result.Error = fmt.Errorf("failed to create template context: %w", err)
		return result, result.Error
	}

	// Track original state
	originalFM := make(map[string]interface{})
	for k, v := range doc.Frontmatter {
		originalFM[k] = v
	}

	// Perform upsert
	err = s.upsertFields(doc, options, ctx, result)
	if err != nil {
		result.Error = err
		return result, result.Error
	}

	// Check if anything changed
	if len(result.FieldsAdded) == 0 && len(result.FieldsUpdated) == 0 {
		return result, nil
	}

	// Write back to file
	err = s.processor.Write(doc, filePath)
	if err != nil {
		result.Error = fmt.Errorf("failed to write file: %w", err)
		return result, result.Error
	}

	return result, nil
}

// UpsertDirectory performs upsert operation on all markdown files in a directory
func (s *UpsertService) UpsertDirectory(dirPath string, options UpsertOptions, ignorePatterns []string) (*UpsertStats, error) {
	stats := &UpsertStats{}
	var wg sync.WaitGroup

	// Channel for results
	resultCh := make(chan *UpsertResult, 100)

	// Start result collector
	go func() {
		for result := range resultCh {
			atomic.AddInt64(&stats.FilesProcessed, 1)

			if result.Error != nil {
				atomic.AddInt64(&stats.Errors, 1)
				continue
			}

			if len(result.FieldsAdded) > 0 || len(result.FieldsUpdated) > 0 {
				atomic.AddInt64(&stats.FilesModified, 1)
				atomic.AddInt64(&stats.FieldsAdded, int64(len(result.FieldsAdded)))
				atomic.AddInt64(&stats.FieldsUpdated, int64(len(result.FieldsUpdated)))
			}
		}
	}()

	// Walk directory and process files
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip non-markdown files
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		// Check ignore patterns
		relPath, _ := filepath.Rel(dirPath, path)
		if shouldIgnore(relPath, ignorePatterns) {
			return nil
		}

		// Process file
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()
			result, _ := s.UpsertFile(filePath, options)
			resultCh <- result
		}(path)

		return nil
	})

	if err != nil {
		return stats, fmt.Errorf("error walking directory: %w", err)
	}

	// Wait for all files to be processed
	wg.Wait()
	close(resultCh)

	return stats, nil
}

// upsertFields performs the actual field upsert logic
func (s *UpsertService) upsertFields(doc *Document, options UpsertOptions, ctx *templates.Context, result *UpsertResult) error {
	for i, field := range options.Fields {
		defaultValue := options.Defaults[i]

		// Check if field exists
		_, exists := doc.Frontmatter[field]

		// Skip if exists and not overwriting
		if exists && !options.Overwrite {
			continue
		}

		// Process template
		processedValue, err := s.templateEngine.Process(defaultValue, ctx)
		if err != nil {
			return fmt.Errorf("error processing template for field %s: %w", field, err)
		}

		// Convert value
		convertedValue := convertValue(processedValue)

		// Set the field
		doc.Frontmatter[field] = convertedValue

		// Track changes
		if exists {
			result.FieldsUpdated = append(result.FieldsUpdated, field)
		} else {
			result.FieldsAdded = append(result.FieldsAdded, field)
		}
	}

	return nil
}

// createTemplateContext creates template context from file and document
func (s *UpsertService) createTemplateContext(filePath string, doc *Document) (*templates.Context, error) {
	fileCtx, err := NewFileContext(filePath)
	if err != nil {
		return nil, err
	}

	// Extract title from frontmatter if available
	title := ""
	if titleVal, exists := doc.Frontmatter["title"]; exists {
		if titleStr, ok := titleVal.(string); ok {
			title = titleStr
		}
	}

	ctx := &templates.Context{
		Filename:     fileCtx.Filename,
		Title:        title,
		RelativePath: filePath, // TODO: Make this relative to vault root
		ParentDir:    filepath.Dir(filePath),
		FileModTime:  fileCtx.FileModTime,
	}

	return ctx, nil
}

// shouldIgnore checks if a path matches any ignore pattern
func shouldIgnore(path string, ignorePatterns []string) bool {
	for _, pattern := range ignorePatterns {
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}

		// Check if any parent directory matches the pattern
		// This handles patterns like ".obsidian/*"
		if strings.Contains(pattern, "/*") {
			prefix := strings.TrimSuffix(pattern, "/*")
			if strings.HasPrefix(path, prefix+"/") || path == prefix {
				return true
			}
		}
	}
	return false
}

// ValidateOptions validates upsert options
func ValidateOptions(options UpsertOptions) error {
	if len(options.Fields) == 0 {
		return fmt.Errorf("at least one field must be specified")
	}

	if len(options.Fields) != len(options.Defaults) {
		return fmt.Errorf("number of fields (%d) must match number of defaults (%d)",
			len(options.Fields), len(options.Defaults))
	}

	// Validate field names
	for _, field := range options.Fields {
		if strings.TrimSpace(field) == "" {
			return fmt.Errorf("field name cannot be empty")
		}
	}

	return nil
}
