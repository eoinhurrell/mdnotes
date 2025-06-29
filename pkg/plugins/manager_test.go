package plugins

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/eoinhurrell/mdnotes/internal/vault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPluginManager(t *testing.T) {
	config := ManagerConfig{
		Enabled:     true,
		SearchPaths: []string{"./plugins", "~/.mdnotes/plugins"},
		Plugins: map[string]interface{}{
			"test-plugin": map[string]interface{}{
				"setting": "value",
			},
		},
	}

	manager := NewPluginManager(config)

	assert.True(t, manager.enabled)
	assert.Len(t, manager.searchPaths, 2)
	assert.Contains(t, manager.searchPaths, "./plugins")
	assert.Equal(t, 0, manager.GetLoadedPluginCount())
	assert.True(t, manager.IsEnabled())
}

func TestPluginManagerDisabled(t *testing.T) {
	config := ManagerConfig{
		Enabled: false,
	}

	manager := NewPluginManager(config)

	// LoadPlugins should succeed but do nothing when disabled
	err := manager.LoadPlugins()
	assert.NoError(t, err)
	assert.Equal(t, 0, manager.GetLoadedPluginCount())

	// ExecuteHook should return empty result when disabled
	ctx := context.Background()
	hookCtx := &HookContext{Command: "test"}
	file := &vault.VaultFile{}

	result, err := manager.ExecuteHook(ctx, HookPerFile, hookCtx, file)
	assert.NoError(t, err)
	assert.False(t, result.Modified)
	assert.Empty(t, result.Metadata)
}

func TestPluginManagerSearchPaths(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()

	// Test with non-existent paths (should not error)
	config := ManagerConfig{
		Enabled:     true,
		SearchPaths: []string{filepath.Join(tmpDir, "nonexistent")},
	}

	manager := NewPluginManager(config)
	err := manager.LoadPlugins()
	assert.NoError(t, err) // Should not error for non-existent paths
}

func TestPluginManagerGetPluginConfig(t *testing.T) {
	config := ManagerConfig{
		Enabled: true,
		Plugins: map[string]interface{}{
			"test-plugin": map[string]interface{}{
				"setting1": "value1",
				"setting2": 42,
			},
			"other-plugin": "simple-string", // Invalid config type
		},
	}

	manager := NewPluginManager(config)

	// Test valid plugin config
	pluginConfig := manager.getPluginConfig("test-plugin")
	assert.Len(t, pluginConfig, 2)
	assert.Equal(t, "value1", pluginConfig["setting1"])
	assert.Equal(t, 42, pluginConfig["setting2"])

	// Test invalid config type (should return empty map)
	invalidConfig := manager.getPluginConfig("other-plugin")
	assert.Empty(t, invalidConfig)

	// Test non-existent plugin
	nonExistentConfig := manager.getPluginConfig("non-existent")
	assert.Empty(t, nonExistentConfig)
}

func TestPluginManagerExecuteHook(t *testing.T) {
	// Create manager with mock plugins
	manager := &PluginManager{
		enabled:     true,
		plugins:     make(map[string]Plugin),
		pluginInfos: make(map[string]*PluginInfo),
		hooks:       make(map[HookType][]Plugin),
	}

	// Create mock plugins
	plugin1 := &mockPlugin{
		name:    "plugin1",
		version: "1.0.0",
		hooks:   []HookType{HookPerFile},
	}
	plugin2 := &mockPlugin{
		name:    "plugin2",
		version: "1.0.0",
		hooks:   []HookType{HookPerFile},
	}

	// Register plugins
	manager.plugins["plugin1"] = plugin1
	manager.plugins["plugin2"] = plugin2
	manager.hooks[HookPerFile] = []Plugin{plugin1, plugin2}

	// Test hook execution
	ctx := context.Background()
	hookCtx := &HookContext{Command: "test"}
	file := &vault.VaultFile{}

	result, err := manager.ExecuteHook(ctx, HookPerFile, hookCtx, file)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPluginManagerExecuteHookWithError(t *testing.T) {
	manager := &PluginManager{
		enabled:     true,
		plugins:     make(map[string]Plugin),
		pluginInfos: make(map[string]*PluginInfo),
		hooks:       make(map[HookType][]Plugin),
	}

	// Create mock plugin that returns error
	plugin := &mockPlugin{
		name:       "error-plugin",
		version:    "1.0.0",
		hooks:      []HookType{HookPerFile},
		executeErr: assert.AnError,
	}

	manager.plugins["error-plugin"] = plugin
	manager.hooks[HookPerFile] = []Plugin{plugin}

	// Test hook execution with error
	ctx := context.Background()
	hookCtx := &HookContext{Command: "test"}
	file := &vault.VaultFile{}

	result, err := manager.ExecuteHook(ctx, HookPerFile, hookCtx, file)
	assert.Error(t, err)
	assert.IsType(t, &PluginError{}, err)
	assert.Contains(t, err.Error(), "error-plugin")
	assert.NotNil(t, result)
}

func TestPluginManagerExecuteHookWithSkip(t *testing.T) {
	manager := &PluginManager{
		enabled:     true,
		plugins:     make(map[string]Plugin),
		pluginInfos: make(map[string]*PluginInfo),
		hooks:       make(map[HookType][]Plugin),
	}

	// Create mock plugin that requests skip
	plugin1 := &mockPluginWithResult{
		mockPlugin: mockPlugin{
			name:    "skip-plugin",
			version: "1.0.0",
			hooks:   []HookType{HookPerFile},
		},
		result: &ProcessResult{
			Skip: true,
			Metadata: map[string]interface{}{
				"skipped": true,
			},
		},
	}
	plugin2 := &mockPlugin{
		name:    "second-plugin",
		version: "1.0.0",
		hooks:   []HookType{HookPerFile},
	}

	manager.plugins["skip-plugin"] = plugin1
	manager.plugins["second-plugin"] = plugin2
	manager.hooks[HookPerFile] = []Plugin{plugin1, plugin2}

	// Test hook execution - should stop after first plugin
	ctx := context.Background()
	hookCtx := &HookContext{Command: "test"}
	file := &vault.VaultFile{}

	result, err := manager.ExecuteHook(ctx, HookPerFile, hookCtx, file)
	require.NoError(t, err)
	assert.True(t, result.Skip)
	assert.Equal(t, true, result.Metadata["skipped"])
}

func TestPluginManagerEnableDisablePlugin(t *testing.T) {
	manager := &PluginManager{
		enabled:     true,
		plugins:     make(map[string]Plugin),
		pluginInfos: make(map[string]*PluginInfo),
		hooks:       make(map[HookType][]Plugin),
	}

	// Add plugin info
	manager.pluginInfos["test-plugin"] = &PluginInfo{
		Name:    "test-plugin",
		Enabled: true,
	}

	// Test disabling plugin
	err := manager.DisablePlugin("test-plugin")
	assert.NoError(t, err)
	assert.False(t, manager.pluginInfos["test-plugin"].Enabled)

	// Test enabling plugin
	err = manager.EnablePlugin("test-plugin")
	assert.NoError(t, err)
	assert.True(t, manager.pluginInfos["test-plugin"].Enabled)

	// Test non-existent plugin
	err = manager.EnablePlugin("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	err = manager.DisablePlugin("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPluginManagerUnloadPlugin(t *testing.T) {
	manager := &PluginManager{
		enabled:     true,
		plugins:     make(map[string]Plugin),
		pluginInfos: make(map[string]*PluginInfo),
		hooks:       make(map[HookType][]Plugin),
	}

	// Create mock plugin
	plugin := &mockPlugin{
		name:    "test-plugin",
		version: "1.0.0",
		hooks:   []HookType{HookPerFile},
	}

	// Register plugin
	manager.plugins["test-plugin"] = plugin
	manager.pluginInfos["test-plugin"] = &PluginInfo{Name: "test-plugin"}
	manager.hooks[HookPerFile] = []Plugin{plugin}

	// Test unloading plugin
	err := manager.UnloadPlugin("test-plugin")
	assert.NoError(t, err)
	assert.Empty(t, manager.plugins)
	assert.Empty(t, manager.pluginInfos)
	assert.Empty(t, manager.hooks[HookPerFile])

	// Test unloading non-existent plugin
	err = manager.UnloadPlugin("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not loaded")
}

func TestPluginManagerUnloadPluginWithCleanupError(t *testing.T) {
	manager := &PluginManager{
		enabled:     true,
		plugins:     make(map[string]Plugin),
		pluginInfos: make(map[string]*PluginInfo),
		hooks:       make(map[HookType][]Plugin),
	}

	// Create mock plugin that fails cleanup
	plugin := &mockPlugin{
		name:       "test-plugin",
		version:    "1.0.0",
		hooks:      []HookType{HookPerFile},
		cleanupErr: assert.AnError,
	}

	manager.plugins["test-plugin"] = plugin
	manager.pluginInfos["test-plugin"] = &PluginInfo{Name: "test-plugin"}

	// Test unloading plugin with cleanup error
	err := manager.UnloadPlugin("test-plugin")
	assert.Error(t, err)
	assert.IsType(t, &PluginError{}, err)
}

func TestPluginManagerListPlugins(t *testing.T) {
	manager := &PluginManager{
		enabled:     true,
		plugins:     make(map[string]Plugin),
		pluginInfos: make(map[string]*PluginInfo),
		hooks:       make(map[HookType][]Plugin),
	}

	// Add some plugin infos
	manager.pluginInfos["plugin-b"] = &PluginInfo{Name: "plugin-b"}
	manager.pluginInfos["plugin-a"] = &PluginInfo{Name: "plugin-a"}
	manager.pluginInfos["plugin-c"] = &PluginInfo{Name: "plugin-c"}

	// Test listing plugins (should be sorted by name)
	plugins := manager.ListPlugins()
	require.Len(t, plugins, 3)
	assert.Equal(t, "plugin-a", plugins[0].Name)
	assert.Equal(t, "plugin-b", plugins[1].Name)
	assert.Equal(t, "plugin-c", plugins[2].Name)
}

func TestPluginManagerGetPluginInfo(t *testing.T) {
	manager := &PluginManager{
		enabled:     true,
		plugins:     make(map[string]Plugin),
		pluginInfos: make(map[string]*PluginInfo),
		hooks:       make(map[HookType][]Plugin),
	}

	// Add plugin info
	expectedInfo := &PluginInfo{
		Name:    "test-plugin",
		Version: "1.0.0",
	}
	manager.pluginInfos["test-plugin"] = expectedInfo

	// Test getting existing plugin info
	info, exists := manager.GetPluginInfo("test-plugin")
	assert.True(t, exists)
	assert.Equal(t, expectedInfo, info)

	// Test getting non-existent plugin info
	info, exists = manager.GetPluginInfo("non-existent")
	assert.False(t, exists)
	assert.Nil(t, info)
}

func TestPluginManagerHasHooksForType(t *testing.T) {
	manager := &PluginManager{
		enabled:     true,
		plugins:     make(map[string]Plugin),
		pluginInfos: make(map[string]*PluginInfo),
		hooks:       make(map[HookType][]Plugin),
	}

	// Initially no hooks
	assert.False(t, manager.HasHooksForType(HookPerFile))

	// Add a plugin to hooks
	plugin := &mockPlugin{}
	manager.hooks[HookPerFile] = []Plugin{plugin}

	assert.True(t, manager.HasHooksForType(HookPerFile))
	assert.False(t, manager.HasHooksForType(HookPostCommand))
}

func TestPluginManagerCleanup(t *testing.T) {
	manager := &PluginManager{
		enabled:     true,
		plugins:     make(map[string]Plugin),
		pluginInfos: make(map[string]*PluginInfo),
		hooks:       make(map[HookType][]Plugin),
	}

	// Add mock plugins
	plugin1 := &mockPlugin{name: "plugin1"}
	plugin2 := &mockPlugin{name: "plugin2", cleanupErr: assert.AnError}

	manager.plugins["plugin1"] = plugin1
	manager.plugins["plugin2"] = plugin2
	manager.pluginInfos["plugin1"] = &PluginInfo{Name: "plugin1"}
	manager.pluginInfos["plugin2"] = &PluginInfo{Name: "plugin2"}
	manager.hooks[HookPerFile] = []Plugin{plugin1, plugin2}

	// Test cleanup
	err := manager.Cleanup()
	assert.Error(t, err) // Should return error from plugin2
	assert.Contains(t, err.Error(), "plugin cleanup errors")

	// All data structures should be cleared
	assert.Empty(t, manager.plugins)
	assert.Empty(t, manager.pluginInfos)
	assert.Empty(t, manager.hooks)
}

func TestPluginManagerHomeDirectoryExpansion(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	config := ManagerConfig{
		Enabled:     true,
		SearchPaths: []string{"~/plugins", "./local"},
	}

	manager := NewPluginManager(config)

	assert.Len(t, manager.searchPaths, 2)
	assert.Equal(t, filepath.Join(homeDir, "plugins"), manager.searchPaths[0])
	assert.Equal(t, "./local", manager.searchPaths[1])
}

// mockPluginWithResult allows customizing the result returned by ExecuteHook
type mockPluginWithResult struct {
	mockPlugin
	result *ProcessResult
}

func (m *mockPluginWithResult) ExecuteHook(ctx context.Context, hookType HookType, hookCtx *HookContext, file *vault.VaultFile) (*ProcessResult, error) {
	if m.executeErr != nil {
		return nil, m.executeErr
	}
	return m.result, nil
}
