package plugins

import (
	"context"
	"fmt"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// HookType represents different plugin hook points
type HookType string

const (
	// HookPreCommand is executed before any command processing
	HookPreCommand HookType = "pre-command"

	// HookPerFile is executed for each file during processing
	HookPerFile HookType = "per-file"

	// HookPostCommand is executed after command processing completes
	HookPostCommand HookType = "post-command"

	// HookExportComplete is executed after export operations complete
	HookExportComplete HookType = "export-complete"
)

// HookContext provides context information to plugin hooks
type HookContext struct {
	Command    string                 // The command being executed (e.g., "frontmatter", "export")
	SubCommand string                 // The subcommand being executed (e.g., "ensure", "sync")
	VaultPath  string                 // Path to the vault being processed
	Config     map[string]interface{} // Plugin-specific configuration
	Metadata   map[string]interface{} // Additional metadata for the hook
}

// ProcessResult represents the result of a plugin's processing
type ProcessResult struct {
	Modified       bool                   // Whether the file was modified
	NewContent     string                 // New file content (if modified)
	NewFrontmatter map[string]interface{} // New frontmatter (if modified)
	Metadata       map[string]interface{} // Additional metadata to pass to next plugin
	Skip           bool                   // Whether to skip further processing of this file
}

// Plugin interface defines the contract for mdnotes plugins
type Plugin interface {
	// Name returns the plugin's name
	Name() string

	// Version returns the plugin's version
	Version() string

	// Description returns a brief description of what the plugin does
	Description() string

	// Init initializes the plugin with configuration
	Init(config map[string]interface{}) error

	// SupportedHooks returns the hook types this plugin supports
	SupportedHooks() []HookType

	// ExecuteHook executes the plugin for a specific hook type
	ExecuteHook(ctx context.Context, hookType HookType, hookCtx *HookContext, file *vault.VaultFile) (*ProcessResult, error)

	// Cleanup performs any necessary cleanup when the plugin is unloaded
	Cleanup() error
}

// PluginInfo contains metadata about a loaded plugin
type PluginInfo struct {
	Name           string                 `json:"name"`
	Version        string                 `json:"version"`
	Description    string                 `json:"description"`
	Enabled        bool                   `json:"enabled"`
	Config         map[string]interface{} `json:"config,omitempty"`
	SupportedHooks []HookType             `json:"supported_hooks"`
	LoadPath       string                 `json:"load_path,omitempty"`
}

// PluginError represents plugin-specific errors
type PluginError struct {
	PluginName string
	Operation  string
	Message    string
	Cause      error
}

func (e *PluginError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("plugin '%s' %s: %s: %v", e.PluginName, e.Operation, e.Message, e.Cause)
	}
	return fmt.Sprintf("plugin '%s' %s: %s", e.PluginName, e.Operation, e.Message)
}

func (e *PluginError) Unwrap() error {
	return e.Cause
}

// NewPluginError creates a new plugin error
func NewPluginError(pluginName, operation, message string) *PluginError {
	return &PluginError{
		PluginName: pluginName,
		Operation:  operation,
		Message:    message,
	}
}

// NewPluginErrorWithCause creates a new plugin error with a cause
func NewPluginErrorWithCause(pluginName, operation, message string, cause error) *PluginError {
	return &PluginError{
		PluginName: pluginName,
		Operation:  operation,
		Message:    message,
		Cause:      cause,
	}
}
