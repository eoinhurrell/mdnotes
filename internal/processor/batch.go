package processor

import (
	"context"
	"fmt"
	"time"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// Vault represents a collection of vault files
type Vault struct {
	Files []*vault.VaultFile
	Path  string
}

// BatchConfig configures a batch operation
type BatchConfig struct {
	Operations   []Operation `yaml:"operations"`
	StopOnError  bool        `yaml:"stop_on_error"`
	CreateBackup bool        `yaml:"create_backup"`
	DryRun       bool        `yaml:"dry_run"`
	Parallel     bool        `yaml:"parallel"`
	MaxWorkers   int         `yaml:"max_workers"`
}

// Operation represents a single operation in a batch
type Operation struct {
	Name       string                 `yaml:"name"`
	Command    string                 `yaml:"command"`
	Parameters map[string]interface{} `yaml:"parameters"`
	Condition  string                 `yaml:"condition,omitempty"`
	OnError    string                 `yaml:"on_error,omitempty"` // skip, stop, retry
	RetryCount int                    `yaml:"retry_count,omitempty"`
	RetryDelay string                 `yaml:"retry_delay,omitempty"`
}

// OperationResult represents the result of an operation
type OperationResult struct {
	Operation     string        `json:"operation"`
	Success       bool          `json:"success"`
	Duration      time.Duration `json:"duration"`
	FilesAffected int           `json:"files_affected"`
	Message       string        `json:"message"`
	Error         error         `json:"error,omitempty"`
}

// String returns a formatted string representation of the result
func (or OperationResult) String() string {
	status := "✓"
	if !or.Success {
		status = "✗"
	}

	msg := fmt.Sprintf("%s %s (%s)", status, or.Operation, or.Duration.String())
	if or.Message != "" {
		msg += ": " + or.Message
	}
	if or.Error != nil {
		msg += " - " + or.Error.Error()
	}

	return msg
}

// Processor defines the interface for batch processors
type Processor interface {
	Process(ctx context.Context, vault *Vault, params map[string]interface{}) error
	Name() string
}

// Backup represents a backup of vault state
type Backup struct {
	ID        string
	Timestamp time.Time
	Files     map[string][]byte // path -> content
}

// BatchProcessor handles batch operations on vaults
type BatchProcessor struct {
	processors map[string]Processor
	lastBackup *Backup
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor() *BatchProcessor {
	processor := &BatchProcessor{
		processors: make(map[string]Processor),
	}

	// Register built-in processors
	processor.registerBuiltins()

	return processor
}

// RegisterProcessor registers a processor for a command
func (bp *BatchProcessor) RegisterProcessor(command string, processor Processor) {
	bp.processors[command] = processor
}

// Execute executes a batch configuration
func (bp *BatchProcessor) Execute(ctx context.Context, vault *Vault, config BatchConfig) ([]OperationResult, error) {
	var results []OperationResult

	// Create backup if requested
	if config.CreateBackup && !config.DryRun {
		backup, err := bp.createBackup(vault)
		if err != nil {
			return nil, fmt.Errorf("creating backup: %w", err)
		}
		bp.lastBackup = backup
	}

	// Execute operations
	for i, operation := range config.Operations {
		select {
		case <-ctx.Done():
			// Rollback on cancellation
			if bp.lastBackup != nil && !config.DryRun {
				bp.rollback(vault, bp.lastBackup)
			}
			return results, ctx.Err()
		default:
		}

		result := bp.executeOperation(ctx, vault, operation, config.DryRun)
		results = append(results, result)

		if !result.Success && config.StopOnError {
			if bp.lastBackup != nil && !config.DryRun {
				bp.rollback(vault, bp.lastBackup)
			}
			return results, fmt.Errorf("operation %d (%s) failed: %w", i, operation.Name, result.Error)
		}
	}

	return results, nil
}

// executeOperation executes a single operation
func (bp *BatchProcessor) executeOperation(ctx context.Context, vault *Vault, op Operation, dryRun bool) OperationResult {
	start := time.Now()

	result := OperationResult{
		Operation: op.Command,
	}

	// Find processor
	processor, exists := bp.processors[op.Command]
	if !exists {
		result.Success = false
		result.Error = fmt.Errorf("unknown command: %s", op.Command)
		result.Duration = time.Since(start)
		return result
	}

	// Add dry run flag to parameters
	params := make(map[string]interface{})
	for k, v := range op.Parameters {
		params[k] = v
	}
	if dryRun {
		params["dry_run"] = true
	}

	// Execute with retry logic
	retryCount := op.RetryCount
	if retryCount == 0 {
		retryCount = 1 // At least one attempt
	}

	var lastErr error
	for attempt := 0; attempt < retryCount; attempt++ {
		if attempt > 0 {
			// Parse retry delay
			delay := 1 * time.Second
			if op.RetryDelay != "" {
				if d, err := time.ParseDuration(op.RetryDelay); err == nil {
					delay = d
				}
			}

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				result.Success = false
				result.Error = ctx.Err()
				result.Duration = time.Since(start)
				return result
			}
		}

		err := processor.Process(ctx, vault, params)
		if err == nil {
			result.Success = true
			if dryRun {
				result.Message = fmt.Sprintf("would execute %s", op.Name)
			} else {
				result.Message = fmt.Sprintf("executed %s", op.Name)
			}
			break
		}

		lastErr = err
		if attempt == retryCount-1 {
			result.Success = false
			result.Error = lastErr
		}
	}

	result.Duration = time.Since(start)
	return result
}

// createBackup creates a backup of the current vault state
func (bp *BatchProcessor) createBackup(vault *Vault) (*Backup, error) {
	backup := &Backup{
		ID:        fmt.Sprintf("backup_%d", time.Now().Unix()),
		Timestamp: time.Now(),
		Files:     make(map[string][]byte),
	}

	for _, file := range vault.Files {
		content, err := file.Serialize()
		if err != nil {
			return nil, fmt.Errorf("serializing file %s: %w", file.Path, err)
		}
		backup.Files[file.Path] = content
	}

	return backup, nil
}

// rollback restores the vault to a previous state
func (bp *BatchProcessor) rollback(vault *Vault, backup *Backup) error {
	for _, file := range vault.Files {
		if originalContent, exists := backup.Files[file.Path]; exists {
			if err := file.Parse(originalContent); err != nil {
				return fmt.Errorf("restoring file %s: %w", file.Path, err)
			}
		}
	}
	return nil
}

// registerBuiltins registers built-in processors
func (bp *BatchProcessor) registerBuiltins() {
	bp.RegisterProcessor("frontmatter.ensure", &FrontmatterEnsureProcessor{})
	bp.RegisterProcessor("frontmatter.cast", &FrontmatterCastProcessor{})
	bp.RegisterProcessor("frontmatter.validate", &FrontmatterValidateProcessor{})
	bp.RegisterProcessor("frontmatter.sync", &FrontmatterSyncProcessor{})
	bp.RegisterProcessor("headings.fix", &HeadingsFixProcessor{})
	bp.RegisterProcessor("headings.analyze", &HeadingsAnalyzeProcessor{})
	bp.RegisterProcessor("links.convert", &LinksConvertProcessor{})
	bp.RegisterProcessor("links.check", &LinksCheckProcessor{})
}

// Built-in processors

// FrontmatterEnsureProcessor implements frontmatter.ensure
type FrontmatterEnsureProcessor struct{}

func (p *FrontmatterEnsureProcessor) Process(ctx context.Context, vault *Vault, params map[string]interface{}) error {
	field, ok := params["field"].(string)
	if !ok {
		return fmt.Errorf("field parameter is required")
	}

	defaultValue := params["default"]
	dryRun, _ := params["dry_run"].(bool)

	processor := NewFrontmatterProcessor()

	for _, file := range vault.Files {
		if !dryRun {
			processor.Ensure(file, field, defaultValue)
		}
	}

	return nil
}

func (p *FrontmatterEnsureProcessor) Name() string {
	return "frontmatter.ensure"
}

// FrontmatterCastProcessor implements frontmatter.cast
type FrontmatterCastProcessor struct{}

func (p *FrontmatterCastProcessor) Process(ctx context.Context, vault *Vault, params map[string]interface{}) error {
	field, ok := params["field"].(string)
	if !ok {
		return fmt.Errorf("field parameter is required")
	}

	toType, ok := params["type"].(string)
	if !ok {
		return fmt.Errorf("type parameter is required")
	}

	dryRun, _ := params["dry_run"].(bool)

	if dryRun {
		return nil
	}

	caster := NewTypeCaster()

	for _, file := range vault.Files {
		if value, exists := file.Frontmatter[field]; exists {
			if casted, err := caster.Cast(value, toType); err == nil {
				file.Frontmatter[field] = casted
			}
		}
	}

	return nil
}

func (p *FrontmatterCastProcessor) Name() string {
	return "frontmatter.cast"
}

// FrontmatterValidateProcessor implements frontmatter.validate
type FrontmatterValidateProcessor struct{}

func (p *FrontmatterValidateProcessor) Process(ctx context.Context, vault *Vault, params map[string]interface{}) error {
	// This would implement validation logic
	return nil
}

func (p *FrontmatterValidateProcessor) Name() string {
	return "frontmatter.validate"
}

// FrontmatterSyncProcessor implements frontmatter.sync
type FrontmatterSyncProcessor struct{}

func (p *FrontmatterSyncProcessor) Process(ctx context.Context, vault *Vault, params map[string]interface{}) error {
	// This would implement sync logic
	return nil
}

func (p *FrontmatterSyncProcessor) Name() string {
	return "frontmatter.sync"
}

// HeadingsFixProcessor implements headings.fix
type HeadingsFixProcessor struct{}

func (p *HeadingsFixProcessor) Process(ctx context.Context, vault *Vault, params map[string]interface{}) error {
	dryRun, _ := params["dry_run"].(bool)

	if dryRun {
		return nil
	}

	processor := NewHeadingProcessor()
	rules := HeadingRules{
		EnsureH1Title: true,
		SingleH1:      true,
	}

	for _, file := range vault.Files {
		processor.Fix(file, rules)
	}

	return nil
}

func (p *HeadingsFixProcessor) Name() string {
	return "headings.fix"
}

// HeadingsAnalyzeProcessor implements headings.analyze
type HeadingsAnalyzeProcessor struct{}

func (p *HeadingsAnalyzeProcessor) Process(ctx context.Context, vault *Vault, params map[string]interface{}) error {
	// This would implement analysis logic
	return nil
}

func (p *HeadingsAnalyzeProcessor) Name() string {
	return "headings.analyze"
}

// LinksConvertProcessor implements links.convert
type LinksConvertProcessor struct{}

func (p *LinksConvertProcessor) Process(ctx context.Context, vault *Vault, params map[string]interface{}) error {
	dryRun, _ := params["dry_run"].(bool)

	if dryRun {
		return nil
	}

	converter := NewLinkConverter()

	for _, file := range vault.Files {
		converter.ConvertFile(file, WikiFormat, MarkdownFormat)
	}

	return nil
}

func (p *LinksConvertProcessor) Name() string {
	return "links.convert"
}

// LinksCheckProcessor implements links.check
type LinksCheckProcessor struct{}

func (p *LinksCheckProcessor) Process(ctx context.Context, vault *Vault, params map[string]interface{}) error {
	// This would implement link checking logic
	return nil
}

func (p *LinksCheckProcessor) Name() string {
	return "links.check"
}
