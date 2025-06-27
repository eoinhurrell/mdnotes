package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sort"
	"strings"
	"sync"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// PluginManager manages the lifecycle and execution of plugins
type PluginManager struct {
	mu              sync.RWMutex
	plugins         map[string]Plugin
	pluginInfos     map[string]*PluginInfo
	hooks           map[HookType][]Plugin
	searchPaths     []string
	globalConfig    map[string]interface{}
	enabled         bool
}

// ManagerConfig contains configuration for the plugin manager
type ManagerConfig struct {
	Enabled     bool                   `yaml:"enabled" json:"enabled"`
	SearchPaths []string               `yaml:"paths" json:"paths"`
	Plugins     map[string]interface{} `yaml:"plugins" json:"plugins"`
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(config ManagerConfig) *PluginManager {
	// Expand home directory in search paths
	searchPaths := make([]string, len(config.SearchPaths))
	for i, path := range config.SearchPaths {
		if strings.HasPrefix(path, "~/") {
			homeDir, err := os.UserHomeDir()
			if err == nil {
				searchPaths[i] = filepath.Join(homeDir, path[2:])
			} else {
				searchPaths[i] = path
			}
		} else {
			searchPaths[i] = path
		}
	}

	return &PluginManager{
		plugins:      make(map[string]Plugin),
		pluginInfos:  make(map[string]*PluginInfo),
		hooks:        make(map[HookType][]Plugin),
		searchPaths:  searchPaths,
		globalConfig: config.Plugins,
		enabled:      config.Enabled,
	}
}

// LoadPlugins discovers and loads all plugins from search paths
func (pm *PluginManager) LoadPlugins() error {
	if !pm.enabled {
		return nil
	}

	var loadErrors []error

	for _, searchPath := range pm.searchPaths {
		if err := pm.loadPluginsFromPath(searchPath); err != nil {
			loadErrors = append(loadErrors, fmt.Errorf("loading from %s: %w", searchPath, err))
		}
	}

	// Return combined errors if any occurred
	if len(loadErrors) > 0 {
		var errMsgs []string
		for _, err := range loadErrors {
			errMsgs = append(errMsgs, err.Error())
		}
		return fmt.Errorf("plugin loading errors: %s", strings.Join(errMsgs, "; "))
	}

	return nil
}

// loadPluginsFromPath loads plugins from a specific directory
func (pm *PluginManager) loadPluginsFromPath(searchPath string) error {
	// Check if path exists
	if _, err := os.Stat(searchPath); os.IsNotExist(err) {
		return nil // Skip non-existent paths silently
	}

	// Find all .so files in the directory
	pattern := filepath.Join(searchPath, "*.so")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("globbing for plugins: %w", err)
	}

	var loadErrors []error
	for _, pluginPath := range matches {
		if err := pm.loadPlugin(pluginPath); err != nil {
			loadErrors = append(loadErrors, fmt.Errorf("loading %s: %w", pluginPath, err))
		}
	}

	if len(loadErrors) > 0 {
		var errMsgs []string
		for _, err := range loadErrors {
			errMsgs = append(errMsgs, err.Error())
		}
		return fmt.Errorf("plugin loading errors: %s", strings.Join(errMsgs, "; "))
	}

	return nil
}

// loadPlugin loads a single plugin from a .so file
func (pm *PluginManager) loadPlugin(pluginPath string) error {
	// Load the plugin shared object
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return fmt.Errorf("opening plugin: %w", err)
	}

	// Look for the required NewPlugin function
	sym, err := p.Lookup("NewPlugin")
	if err != nil {
		return fmt.Errorf("plugin missing NewPlugin function: %w", err)
	}

	// Type assert to the expected function signature
	newPluginFunc, ok := sym.(func() Plugin)
	if !ok {
		return fmt.Errorf("NewPlugin function has wrong signature")
	}

	// Create the plugin instance
	pluginInstance := newPluginFunc()
	if pluginInstance == nil {
		return fmt.Errorf("NewPlugin returned nil")
	}

	// Get plugin configuration
	pluginConfig := pm.getPluginConfig(pluginInstance.Name())

	// Initialize the plugin
	if err := pluginInstance.Init(pluginConfig); err != nil {
		return NewPluginErrorWithCause(pluginInstance.Name(), "initialization", "failed to initialize", err)
	}

	// Register the plugin
	pm.mu.Lock()
	defer pm.mu.Unlock()

	name := pluginInstance.Name()
	if _, exists := pm.plugins[name]; exists {
		return fmt.Errorf("plugin %s is already loaded", name)
	}

	pm.plugins[name] = pluginInstance
	pm.pluginInfos[name] = &PluginInfo{
		Name:           pluginInstance.Name(),
		Version:        pluginInstance.Version(),
		Description:    pluginInstance.Description(),
		Enabled:        true,
		Config:         pluginConfig,
		SupportedHooks: pluginInstance.SupportedHooks(),
		LoadPath:       pluginPath,
	}

	// Register hooks
	for _, hookType := range pluginInstance.SupportedHooks() {
		pm.hooks[hookType] = append(pm.hooks[hookType], pluginInstance)
	}

	return nil
}

// getPluginConfig extracts configuration for a specific plugin
func (pm *PluginManager) getPluginConfig(pluginName string) map[string]interface{} {
	if pm.globalConfig == nil {
		return make(map[string]interface{})
	}

	if config, exists := pm.globalConfig[pluginName]; exists {
		if configMap, ok := config.(map[string]interface{}); ok {
			return configMap
		}
	}

	return make(map[string]interface{})
}

// ExecuteHook executes all plugins registered for a specific hook
func (pm *PluginManager) ExecuteHook(ctx context.Context, hookType HookType, hookCtx *HookContext, file *vault.VaultFile) (*ProcessResult, error) {
	if !pm.enabled {
		return &ProcessResult{}, nil
	}

	pm.mu.RLock()
	hookPlugins := pm.hooks[hookType]
	pm.mu.RUnlock()

	if len(hookPlugins) == 0 {
		return &ProcessResult{}, nil
	}

	// Start with empty result
	result := &ProcessResult{
		Metadata: make(map[string]interface{}),
	}

	// Execute plugins in order
	for _, plugin := range hookPlugins {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		// Execute the plugin
		pluginResult, err := plugin.ExecuteHook(ctx, hookType, hookCtx, file)
		if err != nil {
			return result, NewPluginErrorWithCause(plugin.Name(), "execution", 
				fmt.Sprintf("hook %s failed", hookType), err)
		}

		// Merge results
		if pluginResult != nil {
			if pluginResult.Modified {
				result.Modified = true
				if pluginResult.NewContent != "" {
					result.NewContent = pluginResult.NewContent
				}
				if pluginResult.NewFrontmatter != nil {
					result.NewFrontmatter = pluginResult.NewFrontmatter
				}
			}

			// Merge metadata
			for key, value := range pluginResult.Metadata {
				result.Metadata[key] = value
			}

			// If plugin requests to skip further processing, stop here
			if pluginResult.Skip {
				result.Skip = true
				break
			}
		}
	}

	return result, nil
}

// GetPluginInfo returns information about a specific plugin
func (pm *PluginManager) GetPluginInfo(name string) (*PluginInfo, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	info, exists := pm.pluginInfos[name]
	return info, exists
}

// ListPlugins returns information about all loaded plugins
func (pm *PluginManager) ListPlugins() []*PluginInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var plugins []*PluginInfo
	for _, info := range pm.pluginInfos {
		plugins = append(plugins, info)
	}

	// Sort by name for consistent output
	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].Name < plugins[j].Name
	})

	return plugins
}

// EnablePlugin enables a specific plugin
func (pm *PluginManager) EnablePlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	info, exists := pm.pluginInfos[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	info.Enabled = true
	return nil
}

// DisablePlugin disables a specific plugin
func (pm *PluginManager) DisablePlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	info, exists := pm.pluginInfos[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	info.Enabled = false
	return nil
}

// UnloadPlugin unloads a specific plugin
func (pm *PluginManager) UnloadPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not loaded", name)
	}

	// Clean up the plugin
	if err := plugin.Cleanup(); err != nil {
		return NewPluginErrorWithCause(name, "cleanup", "cleanup failed", err)
	}

	// Remove from all data structures
	delete(pm.plugins, name)
	delete(pm.pluginInfos, name)

	// Remove from hooks
	for hookType, hookPlugins := range pm.hooks {
		var newHookPlugins []Plugin
		for _, p := range hookPlugins {
			if p.Name() != name {
				newHookPlugins = append(newHookPlugins, p)
			}
		}
		pm.hooks[hookType] = newHookPlugins
	}

	return nil
}

// Cleanup unloads all plugins and cleans up resources
func (pm *PluginManager) Cleanup() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var cleanupErrors []error

	// Clean up all plugins
	for name, plugin := range pm.plugins {
		if err := plugin.Cleanup(); err != nil {
			cleanupErrors = append(cleanupErrors, 
				NewPluginErrorWithCause(name, "cleanup", "cleanup failed", err))
		}
	}

	// Clear all data structures
	pm.plugins = make(map[string]Plugin)
	pm.pluginInfos = make(map[string]*PluginInfo)
	pm.hooks = make(map[HookType][]Plugin)

	// Return combined errors if any occurred
	if len(cleanupErrors) > 0 {
		var errMsgs []string
		for _, err := range cleanupErrors {
			errMsgs = append(errMsgs, err.Error())
		}
		return fmt.Errorf("plugin cleanup errors: %s", strings.Join(errMsgs, "; "))
	}

	return nil
}

// IsEnabled returns whether the plugin system is enabled
func (pm *PluginManager) IsEnabled() bool {
	return pm.enabled
}

// GetLoadedPluginCount returns the number of loaded plugins
func (pm *PluginManager) GetLoadedPluginCount() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return len(pm.plugins)
}

// HasHooksForType returns whether any plugins are registered for a hook type
func (pm *PluginManager) HasHooksForType(hookType HookType) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return len(pm.hooks[hookType]) > 0
}