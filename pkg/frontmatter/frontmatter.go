package frontmatter

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Document represents a markdown file with frontmatter
type Document struct {
	Frontmatter map[string]interface{}
	Body        string
	Raw         []byte
}

// UpsertOptions contains options for the upsert operation
type UpsertOptions struct {
	Fields    []string
	Defaults  []string
	Overwrite bool
}

// Processor handles frontmatter operations
type Processor struct {
	templateEngine TemplateEngine
}

// TemplateEngine interface for template processing
type TemplateEngine interface {
	Process(template string, ctx interface{}) (string, error)
}

// NewProcessor creates a new frontmatter processor
func NewProcessor(templateEngine TemplateEngine) *Processor {
	return &Processor{
		templateEngine: templateEngine,
	}
}

// Parse parses a markdown file and extracts frontmatter
func (p *Processor) Parse(filePath string) (*Document, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	return p.ParseBytes(data)
}

// ParseBytes parses frontmatter from raw bytes
func (p *Processor) ParseBytes(data []byte) (*Document, error) {
	doc := &Document{
		Raw:         data,
		Frontmatter: make(map[string]interface{}),
	}

	// Check if file starts with frontmatter delimiter
	if !bytes.HasPrefix(data, []byte("---\n")) && !bytes.HasPrefix(data, []byte("---\r\n")) {
		// No frontmatter, entire content is body
		doc.Body = string(data)
		return doc, nil
	}

	// Find the closing delimiter
	reader := bytes.NewReader(data)
	scanner := bufio.NewScanner(reader)

	var frontmatterLines []string
	var bodyLines []string
	inFrontmatter := false
	frontmatterClosed := false

	for scanner.Scan() {
		line := scanner.Text()

		if !inFrontmatter && (line == "---" || line == "---\r") {
			inFrontmatter = true
			continue
		}

		if inFrontmatter && (line == "---" || line == "---\r") {
			frontmatterClosed = true
			inFrontmatter = false
			continue
		}

		if inFrontmatter {
			frontmatterLines = append(frontmatterLines, line)
		} else if frontmatterClosed {
			bodyLines = append(bodyLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning file: %w", err)
	}

	// Parse YAML frontmatter
	if len(frontmatterLines) > 0 {
		frontmatterYAML := strings.Join(frontmatterLines, "\n")
		if err := yaml.Unmarshal([]byte(frontmatterYAML), &doc.Frontmatter); err != nil {
			return nil, fmt.Errorf("error parsing frontmatter YAML: %w", err)
		}
	}

	doc.Body = strings.Join(bodyLines, "\n")
	return doc, nil
}

// Upsert updates or inserts frontmatter fields
func (p *Processor) Upsert(doc *Document, options UpsertOptions, templateContext interface{}) error {
	if len(options.Fields) != len(options.Defaults) {
		return fmt.Errorf("number of fields (%d) must match number of defaults (%d)",
			len(options.Fields), len(options.Defaults))
	}

	for i, field := range options.Fields {
		defaultValue := options.Defaults[i]

		// Check if field exists and we're not overwriting
		if _, exists := doc.Frontmatter[field]; exists && !options.Overwrite {
			continue
		}

		// Process template in default value
		processedValue, err := p.templateEngine.Process(defaultValue, templateContext)
		if err != nil {
			return fmt.Errorf("error processing template for field %s: %w", field, err)
		}

		// Convert value to appropriate type
		doc.Frontmatter[field] = convertValue(processedValue)
	}

	return nil
}

// Write writes the document back to a file
func (p *Processor) Write(doc *Document, filePath string) error {
	content, err := p.Serialize(doc)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, content, 0644)
}

// Serialize converts a document back to markdown with frontmatter
func (p *Processor) Serialize(doc *Document) ([]byte, error) {
	var buf bytes.Buffer

	// Write frontmatter if it exists
	if len(doc.Frontmatter) > 0 {
		buf.WriteString("---\n")

		yamlData, err := yaml.Marshal(doc.Frontmatter)
		if err != nil {
			return nil, fmt.Errorf("error marshaling frontmatter: %w", err)
		}

		buf.Write(yamlData)
		buf.WriteString("---\n")
	}

	// Write body
	buf.WriteString(doc.Body)

	return buf.Bytes(), nil
}

// convertValue attempts to convert string values to appropriate types
func convertValue(value string) interface{} {
	// Handle special template values
	if value == "[]" {
		return []string{}
	}
	if value == "{}" {
		return map[string]interface{}{}
	}

	// Try to parse as various types
	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}

	// Try to parse as date
	if matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}`, value); matched {
		if t, err := time.Parse("2006-01-02", value[:10]); err == nil {
			return t.Format("2006-01-02")
		}
	}

	// Default to string
	return value
}

// FileContext provides context for template processing
type FileContext struct {
	Filename     string
	Title        string
	RelativePath string
	ParentDir    string
	FileModTime  time.Time
}

// NewFileContext creates context from a file path
func NewFileContext(filePath string) (*FileContext, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	filename := strings.TrimSuffix(info.Name(), ".md")

	return &FileContext{
		Filename:    filename,
		FileModTime: info.ModTime(),
		// TODO: Extract other fields like title from frontmatter
	}, nil
}
