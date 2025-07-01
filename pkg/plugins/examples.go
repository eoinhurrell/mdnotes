package plugins

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// ExampleFrontmatterPlugin demonstrates a plugin that automatically ensures frontmatter fields
type ExampleFrontmatterPlugin struct {
	config map[string]interface{}
}

// NewExampleFrontmatterPlugin creates a new frontmatter plugin instance
func NewExampleFrontmatterPlugin() Plugin {
	return &ExampleFrontmatterPlugin{}
}

func (p *ExampleFrontmatterPlugin) Name() string {
	return "auto-frontmatter"
}

func (p *ExampleFrontmatterPlugin) Version() string {
	return "1.0.0"
}

func (p *ExampleFrontmatterPlugin) Description() string {
	return "Automatically ensures required frontmatter fields are present"
}

func (p *ExampleFrontmatterPlugin) Init(config map[string]interface{}) error {
	p.config = config
	return nil
}

func (p *ExampleFrontmatterPlugin) SupportedHooks() []HookType {
	return []HookType{HookPerFile}
}

func (p *ExampleFrontmatterPlugin) ExecuteHook(ctx context.Context, hookType HookType, hookCtx *HookContext, file *vault.VaultFile) (*ProcessResult, error) {
	if hookType != HookPerFile {
		return &ProcessResult{}, nil
	}

	// Only process files during frontmatter operations
	if hookCtx.Command != "frontmatter" {
		return &ProcessResult{}, nil
	}

	result := &ProcessResult{
		NewFrontmatter: make(map[string]interface{}),
	}

	// Copy existing frontmatter
	for k, v := range file.Frontmatter {
		result.NewFrontmatter[k] = v
	}

	modified := false

	// Ensure created field
	if _, exists := result.NewFrontmatter["created"]; !exists {
		result.NewFrontmatter["created"] = time.Now().Format("2006-01-02")
	}

	// Ensure modified field (always updated)
	result.NewFrontmatter["modified"] = time.Now().Format("2006-01-02")
	modified = true

	// Ensure tags field exists
	if _, exists := result.NewFrontmatter["tags"]; !exists {
		result.NewFrontmatter["tags"] = []string{}
	}

	result.Modified = modified
	return result, nil
}

func (p *ExampleFrontmatterPlugin) Cleanup() error {
	return nil
}

// ExampleContentPlugin demonstrates a plugin that processes file content
type ExampleContentPlugin struct {
	config map[string]interface{}
}

// NewExampleContentPlugin creates a new content processing plugin instance
func NewExampleContentPlugin() Plugin {
	return &ExampleContentPlugin{}
}

func (p *ExampleContentPlugin) Name() string {
	return "content-enhancer"
}

func (p *ExampleContentPlugin) Version() string {
	return "1.0.0"
}

func (p *ExampleContentPlugin) Description() string {
	return "Enhances content by fixing common formatting issues"
}

func (p *ExampleContentPlugin) Init(config map[string]interface{}) error {
	p.config = config
	return nil
}

func (p *ExampleContentPlugin) SupportedHooks() []HookType {
	return []HookType{HookPerFile}
}

func (p *ExampleContentPlugin) ExecuteHook(ctx context.Context, hookType HookType, hookCtx *HookContext, file *vault.VaultFile) (*ProcessResult, error) {
	if hookType != HookPerFile {
		return &ProcessResult{}, nil
	}

	content := file.Body
	originalContent := content

	// Fix double spaces
	content = regexp.MustCompile(`  +`).ReplaceAllString(content, " ")

	// Fix multiple newlines (more than 2)
	content = regexp.MustCompile(`\n{3,}`).ReplaceAllString(content, "\n\n")

	// Fix trailing whitespace
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	content = strings.Join(lines, "\n")

	// Add final newline if missing
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	result := &ProcessResult{
		Modified:   content != originalContent,
		NewContent: content,
		Metadata: map[string]interface{}{
			"content_enhancements": map[string]bool{
				"fixed_spacing":      true,
				"fixed_newlines":     true,
				"trimmed_whitespace": true,
			},
		},
	}

	return result, nil
}

func (p *ExampleContentPlugin) Cleanup() error {
	return nil
}

// ExampleExportPlugin demonstrates a plugin that runs during export operations
type ExampleExportPlugin struct {
	config map[string]interface{}
}

// NewExampleExportPlugin creates a new export plugin instance
func NewExampleExportPlugin() Plugin {
	return &ExampleExportPlugin{}
}

func (p *ExampleExportPlugin) Name() string {
	return "export-processor"
}

func (p *ExampleExportPlugin) Version() string {
	return "1.0.0"
}

func (p *ExampleExportPlugin) Description() string {
	return "Processes files during export with custom transformations"
}

func (p *ExampleExportPlugin) Init(config map[string]interface{}) error {
	p.config = config
	return nil
}

func (p *ExampleExportPlugin) SupportedHooks() []HookType {
	return []HookType{HookPerFile, HookExportComplete}
}

func (p *ExampleExportPlugin) ExecuteHook(ctx context.Context, hookType HookType, hookCtx *HookContext, file *vault.VaultFile) (*ProcessResult, error) {
	switch hookType {
	case HookPerFile:
		return p.processFile(hookCtx, file)
	case HookExportComplete:
		return p.onExportComplete(hookCtx)
	default:
		return &ProcessResult{}, nil
	}
}

func (p *ExampleExportPlugin) processFile(hookCtx *HookContext, file *vault.VaultFile) (*ProcessResult, error) {
	// Only process during export operations
	if hookCtx.Command != "export" {
		return &ProcessResult{}, nil
	}

	content := file.Body

	// Add export timestamp comment
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	exportComment := fmt.Sprintf("\n<!-- Exported by mdnotes on %s -->\n", timestamp)
	content += exportComment

	result := &ProcessResult{
		Modified:   true,
		NewContent: content,
		Metadata: map[string]interface{}{
			"export_timestamp": timestamp,
			"export_plugin":    p.Name(),
		},
	}

	return result, nil
}

func (p *ExampleExportPlugin) onExportComplete(hookCtx *HookContext) (*ProcessResult, error) {
	// Log export completion (in a real plugin, this might update a database,
	// send a notification, generate reports, etc.)
	fmt.Printf("Export completed for vault: %s\n", hookCtx.VaultPath)

	return &ProcessResult{
		Metadata: map[string]interface{}{
			"export_completed": true,
			"completion_time":  time.Now(),
		},
	}, nil
}

func (p *ExampleExportPlugin) Cleanup() error {
	return nil
}

// ValidatePlugin validates that a plugin implements the Plugin interface correctly
func ValidatePlugin(p Plugin) error {
	if p.Name() == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	if p.Version() == "" {
		return fmt.Errorf("plugin version cannot be empty")
	}

	if len(p.SupportedHooks()) == 0 {
		return fmt.Errorf("plugin must support at least one hook type")
	}

	// Validate hook types
	validHooks := map[HookType]bool{
		HookPreCommand:     true,
		HookPerFile:        true,
		HookPostCommand:    true,
		HookExportComplete: true,
	}

	for _, hook := range p.SupportedHooks() {
		if !validHooks[hook] {
			return fmt.Errorf("unsupported hook type: %s", hook)
		}
	}

	return nil
}
