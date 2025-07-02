package security

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/eoinhurrell/mdnotes/internal/errors"
)

// PluginSandbox provides security restrictions for plugin execution
type PluginSandbox struct {
	allowedPaths     []string
	deniedPaths      []string
	maxMemoryMB      int64
	maxExecutionTime time.Duration
	allowNetworking  bool
	allowFileWrite   bool
	allowFileRead    bool
	tempDir          string
}

// SandboxConfig configures plugin sandbox restrictions
type SandboxConfig struct {
	AllowedPaths     []string
	DeniedPaths      []string
	MaxMemoryMB      int64
	MaxExecutionTime time.Duration
	AllowNetworking  bool
	AllowFileWrite   bool
	AllowFileRead    bool
	TempDir          string
}

// NewPluginSandbox creates a new plugin sandbox
func NewPluginSandbox(config SandboxConfig) *PluginSandbox {
	sandbox := &PluginSandbox{
		allowedPaths:     config.AllowedPaths,
		deniedPaths:      config.DeniedPaths,
		maxMemoryMB:      config.MaxMemoryMB,
		maxExecutionTime: config.MaxExecutionTime,
		allowNetworking:  config.AllowNetworking,
		allowFileWrite:   config.AllowFileWrite,
		allowFileRead:    config.AllowFileRead,
		tempDir:          config.TempDir,
	}

	// Set defaults
	if sandbox.maxMemoryMB <= 0 {
		sandbox.maxMemoryMB = 256 // 256MB default
	}
	if sandbox.maxExecutionTime <= 0 {
		sandbox.maxExecutionTime = 30 * time.Second
	}
	if sandbox.tempDir == "" {
		sandbox.tempDir = os.TempDir()
	}

	return sandbox
}

// ExecuteWithRestrictions executes a function with sandbox restrictions
func (ps *PluginSandbox) ExecuteWithRestrictions(ctx context.Context, fn func() error) error {
	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, ps.maxExecutionTime)
	defer cancel()

	// Channel to capture the result
	resultChan := make(chan error, 1)

	// Execute in goroutine to allow monitoring
	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultChan <- errors.NewErrorBuilder().
					WithOperation("plugin execution").
					WithError(fmt.Errorf("plugin panic: %v", r)).
					WithCode(errors.ErrCodePluginExecution).
					WithSuggestion("Check plugin code for runtime errors").
					Build()
			}
		}()

		// Monitor memory usage
		if ps.maxMemoryMB > 0 {
			go ps.monitorMemory(timeoutCtx, resultChan)
		}

		// Execute the function
		err := fn()
		select {
		case resultChan <- err:
			// Result sent successfully
		case <-timeoutCtx.Done():
			// Function completed but timeout context is done
		}
	}()

	// Wait for completion or timeout
	select {
	case err := <-resultChan:
		return err
	case <-timeoutCtx.Done():
		return errors.NewErrorBuilder().
			WithOperation("plugin execution").
			WithError(fmt.Errorf("plugin execution timeout after %v", ps.maxExecutionTime)).
			WithCode(errors.ErrCodeOperationTimeout).
			WithSuggestion("Optimize plugin performance or increase timeout").
			Build()
	}
}

// ValidateFileAccess checks if a file access is allowed
func (ps *PluginSandbox) ValidateFileAccess(path string, write bool) error {
	// Resolve absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return errors.NewErrorBuilder().
			WithOperation("file access validation").
			WithError(fmt.Errorf("cannot resolve path: %w", err)).
			WithCode(errors.ErrCodePathInvalid).
			WithSuggestion("Ensure the file path is valid").
			Build()
	}

	// Check if file operations are allowed
	if write && !ps.allowFileWrite {
		return errors.NewErrorBuilder().
			WithOperation("file access validation").
			WithError(fmt.Errorf("file write access denied: %s", absPath)).
			WithCode(errors.ErrCodeFilePermission).
			WithSuggestion("Enable file write permissions in plugin configuration").
			Build()
	}

	if !write && !ps.allowFileRead {
		return errors.NewErrorBuilder().
			WithOperation("file access validation").
			WithError(fmt.Errorf("file read access denied: %s", absPath)).
			WithCode(errors.ErrCodeFilePermission).
			WithSuggestion("Enable file read permissions in plugin configuration").
			Build()
	}

	// Check denied paths first
	for _, denied := range ps.deniedPaths {
		deniedAbs, err := filepath.Abs(denied)
		if err != nil {
			continue
		}

		rel, err := filepath.Rel(deniedAbs, absPath)
		if err != nil {
			continue
		}

		// If path is within denied directory
		if !strings.HasPrefix(rel, "..") {
			return errors.NewErrorBuilder().
				WithOperation("file access validation").
				WithError(fmt.Errorf("access denied to restricted path: %s", absPath)).
				WithCode(errors.ErrCodeFilePermission).
				WithSuggestion("Access to this path is restricted by security policy").
				Build()
		}
	}

	// Check allowed paths if specified
	if len(ps.allowedPaths) > 0 {
		allowed := false
		for _, allowedPath := range ps.allowedPaths {
			allowedAbs, err := filepath.Abs(allowedPath)
			if err != nil {
				continue
			}

			rel, err := filepath.Rel(allowedAbs, absPath)
			if err != nil {
				continue
			}

			// If path is within allowed directory
			if !strings.HasPrefix(rel, "..") {
				allowed = true
				break
			}
		}

		if !allowed {
			return errors.NewErrorBuilder().
				WithOperation("file access validation").
				WithError(fmt.Errorf("access denied: path not in allowed directories: %s", absPath)).
				WithCode(errors.ErrCodeFilePermission).
				WithSuggestion("Ensure the path is within allowed directories").
				Build()
		}
	}

	return nil
}

// ValidateNetworkAccess checks if network access is allowed
func (ps *PluginSandbox) ValidateNetworkAccess(host string, port int) error {
	if !ps.allowNetworking {
		return errors.NewErrorBuilder().
			WithOperation("network access validation").
			WithError(fmt.Errorf("network access denied to %s:%d", host, port)).
			WithCode(errors.ErrCodeNetworkError).
			WithSuggestion("Enable network permissions in plugin configuration").
			Build()
	}

	// Additional network restrictions could be added here
	// e.g., whitelist/blacklist of hosts, port restrictions, etc.

	return nil
}

// GetTempDir returns a secure temporary directory for plugin use
func (ps *PluginSandbox) GetTempDir() (string, error) {
	// Create plugin-specific temp directory
	pluginTempDir := filepath.Join(ps.tempDir, "mdnotes-plugin")

	if err := CreateSecureDir(pluginTempDir); err != nil {
		return "", err
	}

	return pluginTempDir, nil
}

// CleanupTempDir removes temporary files created by plugins
func (ps *PluginSandbox) CleanupTempDir() error {
	pluginTempDir := filepath.Join(ps.tempDir, "mdnotes-plugin")

	if _, err := os.Stat(pluginTempDir); os.IsNotExist(err) {
		return nil // Directory doesn't exist, nothing to clean
	}

	if err := os.RemoveAll(pluginTempDir); err != nil {
		return errors.NewErrorBuilder().
			WithOperation("temp directory cleanup").
			WithError(fmt.Errorf("failed to cleanup temp directory: %w", err)).
			WithCode(errors.ErrCodeFilePermission).
			WithSuggestion("Check permissions on temp directory").
			Build()
	}

	return nil
}

// monitorMemory monitors memory usage and terminates if exceeded
func (ps *PluginSandbox) monitorMemory(ctx context.Context, resultChan chan<- error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)

			// Check allocated memory in MB
			allocMB := int64(memStats.Alloc / 1024 / 1024)
			if allocMB > ps.maxMemoryMB {
				select {
				case resultChan <- errors.NewErrorBuilder().
					WithOperation("memory monitoring").
					WithError(fmt.Errorf("memory limit exceeded: %dMB > %dMB", allocMB, ps.maxMemoryMB)).
					WithCode(errors.ErrCodeResourceExhausted).
					WithSuggestion("Optimize plugin memory usage or increase limit").
					Build():
				case <-ctx.Done():
				}
				return
			}
		}
	}
}

// RestrictedFileOperations provides file operations with sandbox validation
type RestrictedFileOperations struct {
	sandbox *PluginSandbox
}

// NewRestrictedFileOperations creates restricted file operations
func (ps *PluginSandbox) NewRestrictedFileOperations() *RestrictedFileOperations {
	return &RestrictedFileOperations{
		sandbox: ps,
	}
}

// ReadFile reads a file with access validation
func (rfo *RestrictedFileOperations) ReadFile(filename string) ([]byte, error) {
	if err := rfo.sandbox.ValidateFileAccess(filename, false); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, errors.NewErrorBuilder().
			WithOperation("restricted file read").
			WithFile(filename).
			WithError(fmt.Errorf("failed to read file: %w", err)).
			WithCode(errors.ErrCodeFileNotFound).
			WithSuggestion("Check file exists and permissions are correct").
			Build()
	}

	return data, nil
}

// WriteFile writes a file with access validation
func (rfo *RestrictedFileOperations) WriteFile(filename string, data []byte, perm os.FileMode) error {
	if err := rfo.sandbox.ValidateFileAccess(filename, true); err != nil {
		return err
	}

	if err := os.WriteFile(filename, data, perm); err != nil {
		return errors.NewErrorBuilder().
			WithOperation("restricted file write").
			WithFile(filename).
			WithError(fmt.Errorf("failed to write file: %w", err)).
			WithCode(errors.ErrCodeFilePermission).
			WithSuggestion("Check directory exists and permissions are correct").
			Build()
	}

	return nil
}

// MkdirAll creates directories with access validation
func (rfo *RestrictedFileOperations) MkdirAll(path string, perm os.FileMode) error {
	if err := rfo.sandbox.ValidateFileAccess(path, true); err != nil {
		return err
	}

	if err := os.MkdirAll(path, perm); err != nil {
		return errors.NewErrorBuilder().
			WithOperation("restricted directory creation").
			WithFile(path).
			WithError(fmt.Errorf("failed to create directory: %w", err)).
			WithCode(errors.ErrCodeFilePermission).
			WithSuggestion("Check parent directory permissions").
			Build()
	}

	return nil
}

// Remove removes a file or directory with access validation
func (rfo *RestrictedFileOperations) Remove(name string) error {
	if err := rfo.sandbox.ValidateFileAccess(name, true); err != nil {
		return err
	}

	if err := os.Remove(name); err != nil {
		return errors.NewErrorBuilder().
			WithOperation("restricted file removal").
			WithFile(name).
			WithError(fmt.Errorf("failed to remove file: %w", err)).
			WithCode(errors.ErrCodeFilePermission).
			WithSuggestion("Check file exists and permissions are correct").
			Build()
	}

	return nil
}

// Stat gets file info with access validation
func (rfo *RestrictedFileOperations) Stat(name string) (os.FileInfo, error) {
	if err := rfo.sandbox.ValidateFileAccess(name, false); err != nil {
		return nil, err
	}

	info, err := os.Stat(name)
	if err != nil {
		return nil, errors.NewErrorBuilder().
			WithOperation("restricted file stat").
			WithFile(name).
			WithError(fmt.Errorf("failed to get file info: %w", err)).
			WithCode(errors.ErrCodeFileNotFound).
			WithSuggestion("Check file exists and permissions are correct").
			Build()
	}

	return info, nil
}

// DefaultSandboxConfig returns a default sandbox configuration
func DefaultSandboxConfig(vaultPath string) SandboxConfig {
	return SandboxConfig{
		AllowedPaths: []string{
			vaultPath,    // Allow access to vault
			os.TempDir(), // Allow temp directory access
		},
		DeniedPaths: []string{
			"/etc",              // System config
			"/bin",              // System binaries
			"/usr/bin",          // User binaries
			"/System",           // macOS system (if applicable)
			"C:\\Windows",       // Windows system (if applicable)
			"C:\\Program Files", // Windows programs (if applicable)
		},
		MaxMemoryMB:      256,              // 256MB limit
		MaxExecutionTime: 30 * time.Second, // 30 second timeout
		AllowNetworking:  false,            // No network by default
		AllowFileWrite:   true,             // Allow file writes in allowed paths
		AllowFileRead:    true,             // Allow file reads in allowed paths
		TempDir:          os.TempDir(),
	}
}

// ApplySystemLimits applies OS-level resource limits (Unix only)
func ApplySystemLimits(maxMemoryMB int64, maxCPUTime time.Duration) error {
	if runtime.GOOS == "windows" {
		// Windows doesn't support rlimit, skip
		return nil
	}

	// Set memory limit (AS - Address Space on systems that support it)
	if maxMemoryMB > 0 {
		memLimit := maxMemoryMB * 1024 * 1024 // Convert MB to bytes
		if err := syscall.Setrlimit(syscall.RLIMIT_AS, &syscall.Rlimit{
			Cur: uint64(memLimit),
			Max: uint64(memLimit),
		}); err != nil {
			return fmt.Errorf("failed to set memory limit: %w", err)
		}
	}

	// Set CPU time limit
	if maxCPUTime > 0 {
		cpuLimit := uint64(maxCPUTime.Seconds())
		if err := syscall.Setrlimit(syscall.RLIMIT_CPU, &syscall.Rlimit{
			Cur: cpuLimit,
			Max: cpuLimit,
		}); err != nil {
			return fmt.Errorf("failed to set CPU time limit: %w", err)
		}
	}

	return nil
}

// SecurityAudit performs a security audit of plugin configuration
func SecurityAudit(config SandboxConfig) []string {
	var warnings []string

	// Check for overly permissive settings
	if config.AllowNetworking {
		warnings = append(warnings, "Network access is enabled - ensure plugins are trusted")
	}

	if len(config.AllowedPaths) == 0 {
		warnings = append(warnings, "No path restrictions - plugins can access any file")
	}

	if config.MaxMemoryMB > 1024 {
		warnings = append(warnings, "High memory limit (>1GB) - may impact system performance")
	}

	if config.MaxExecutionTime > 5*time.Minute {
		warnings = append(warnings, "Long execution timeout (>5min) - may cause UI freezing")
	}

	// Check for dangerous allowed paths
	dangerousPaths := []string{"/", "C:\\", "/etc", "/bin", "/usr", "/System"}
	for _, allowed := range config.AllowedPaths {
		for _, dangerous := range dangerousPaths {
			if strings.HasPrefix(allowed, dangerous) {
				warnings = append(warnings, fmt.Sprintf("Potentially dangerous allowed path: %s", allowed))
			}
		}
	}

	return warnings
}
