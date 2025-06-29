package plugins

import (
	"context"
	"testing"

	"github.com/eoinhurrell/mdnotes/internal/vault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginError(t *testing.T) {
	t.Run("ErrorWithoutCause", func(t *testing.T) {
		err := NewPluginError("test-plugin", "initialization", "failed to start")
		expected := "plugin 'test-plugin' initialization: failed to start"
		assert.Equal(t, expected, err.Error())
		assert.Nil(t, err.Unwrap())
	})

	t.Run("ErrorWithCause", func(t *testing.T) {
		cause := assert.AnError
		err := NewPluginErrorWithCause("test-plugin", "execution", "hook failed", cause)
		expected := "plugin 'test-plugin' execution: hook failed: assert.AnError general error for testing"
		assert.Equal(t, expected, err.Error())
		assert.Equal(t, cause, err.Unwrap())
	})
}

func TestHookTypes(t *testing.T) {
	tests := []struct {
		name     string
		hookType HookType
		expected string
	}{
		{"PreCommand", HookPreCommand, "pre-command"},
		{"PerFile", HookPerFile, "per-file"},
		{"PostCommand", HookPostCommand, "post-command"},
		{"ExportComplete", HookExportComplete, "export-complete"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.hookType))
		})
	}
}

func TestHookContext(t *testing.T) {
	ctx := &HookContext{
		Command:    "frontmatter",
		SubCommand: "ensure",
		VaultPath:  "/test/vault",
		Config: map[string]interface{}{
			"key": "value",
		},
		Metadata: map[string]interface{}{
			"meta": "data",
		},
	}

	assert.Equal(t, "frontmatter", ctx.Command)
	assert.Equal(t, "ensure", ctx.SubCommand)
	assert.Equal(t, "/test/vault", ctx.VaultPath)
	assert.Equal(t, "value", ctx.Config["key"])
	assert.Equal(t, "data", ctx.Metadata["meta"])
}

func TestProcessResult(t *testing.T) {
	result := &ProcessResult{
		Modified:   true,
		NewContent: "new content",
		NewFrontmatter: map[string]interface{}{
			"title": "New Title",
		},
		Metadata: map[string]interface{}{
			"processed_by": "test-plugin",
		},
		Skip: false,
	}

	assert.True(t, result.Modified)
	assert.Equal(t, "new content", result.NewContent)
	assert.Equal(t, "New Title", result.NewFrontmatter["title"])
	assert.Equal(t, "test-plugin", result.Metadata["processed_by"])
	assert.False(t, result.Skip)
}

func TestPluginInfo(t *testing.T) {
	info := &PluginInfo{
		Name:        "test-plugin",
		Version:     "1.0.0",
		Description: "A test plugin",
		Enabled:     true,
		Config: map[string]interface{}{
			"setting": "value",
		},
		SupportedHooks: []HookType{HookPerFile, HookPostCommand},
		LoadPath:       "/path/to/plugin.so",
	}

	assert.Equal(t, "test-plugin", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.Equal(t, "A test plugin", info.Description)
	assert.True(t, info.Enabled)
	assert.Equal(t, "value", info.Config["setting"])
	assert.Len(t, info.SupportedHooks, 2)
	assert.Contains(t, info.SupportedHooks, HookPerFile)
	assert.Contains(t, info.SupportedHooks, HookPostCommand)
	assert.Equal(t, "/path/to/plugin.so", info.LoadPath)
}

func TestValidatePlugin(t *testing.T) {
	t.Run("ValidPlugin", func(t *testing.T) {
		plugin := NewExampleFrontmatterPlugin()
		err := ValidatePlugin(plugin)
		assert.NoError(t, err)
	})

	t.Run("EmptyName", func(t *testing.T) {
		plugin := &mockPlugin{name: ""}
		err := ValidatePlugin(plugin)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "plugin name cannot be empty")
	})

	t.Run("EmptyVersion", func(t *testing.T) {
		plugin := &mockPlugin{name: "test", version: ""}
		err := ValidatePlugin(plugin)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "plugin version cannot be empty")
	})

	t.Run("NoSupportedHooks", func(t *testing.T) {
		plugin := &mockPlugin{name: "test", version: "1.0.0", hooks: []HookType{}}
		err := ValidatePlugin(plugin)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must support at least one hook type")
	})

	t.Run("InvalidHookType", func(t *testing.T) {
		plugin := &mockPlugin{
			name:    "test",
			version: "1.0.0",
			hooks:   []HookType{"invalid-hook"},
		}
		err := ValidatePlugin(plugin)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported hook type")
	})
}

// mockPlugin is a test implementation of the Plugin interface
type mockPlugin struct {
	name        string
	version     string
	description string
	hooks       []HookType
	initErr     error
	executeErr  error
	cleanupErr  error
}

func (m *mockPlugin) Name() string {
	return m.name
}

func (m *mockPlugin) Version() string {
	return m.version
}

func (m *mockPlugin) Description() string {
	return m.description
}

func (m *mockPlugin) Init(config map[string]interface{}) error {
	return m.initErr
}

func (m *mockPlugin) SupportedHooks() []HookType {
	return m.hooks
}

func (m *mockPlugin) ExecuteHook(ctx context.Context, hookType HookType, hookCtx *HookContext, file *vault.VaultFile) (*ProcessResult, error) {
	if m.executeErr != nil {
		return nil, m.executeErr
	}
	return &ProcessResult{}, nil
}

func (m *mockPlugin) Cleanup() error {
	return m.cleanupErr
}

func TestExampleFrontmatterPlugin(t *testing.T) {
	plugin := NewExampleFrontmatterPlugin()

	// Test plugin metadata
	assert.Equal(t, "auto-frontmatter", plugin.Name())
	assert.Equal(t, "1.0.0", plugin.Version())
	assert.NotEmpty(t, plugin.Description())
	assert.Contains(t, plugin.SupportedHooks(), HookPerFile)

	// Test initialization
	config := map[string]interface{}{
		"test": "value",
	}
	err := plugin.Init(config)
	assert.NoError(t, err)

	// Test hook execution
	ctx := context.Background()
	hookCtx := &HookContext{
		Command:    "frontmatter",
		SubCommand: "ensure",
		VaultPath:  "/test/vault",
	}
	file := &vault.VaultFile{
		Path: "test.md",
		Frontmatter: map[string]interface{}{
			"title": "Test Note",
		},
		Body: "Test content",
	}

	result, err := plugin.ExecuteHook(ctx, HookPerFile, hookCtx, file)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Modified)
	assert.Contains(t, result.NewFrontmatter, "created")
	assert.Contains(t, result.NewFrontmatter, "modified")
	assert.Contains(t, result.NewFrontmatter, "tags")

	// Test cleanup
	err = plugin.Cleanup()
	assert.NoError(t, err)
}

func TestExampleContentPlugin(t *testing.T) {
	plugin := NewExampleContentPlugin()

	// Test plugin metadata
	assert.Equal(t, "content-enhancer", plugin.Name())
	assert.Equal(t, "1.0.0", plugin.Version())
	assert.NotEmpty(t, plugin.Description())
	assert.Contains(t, plugin.SupportedHooks(), HookPerFile)

	// Test initialization
	err := plugin.Init(map[string]interface{}{})
	assert.NoError(t, err)

	// Test hook execution with content that needs fixing
	ctx := context.Background()
	hookCtx := &HookContext{
		Command:   "export",
		VaultPath: "/test/vault",
	}
	file := &vault.VaultFile{
		Path:        "test.md",
		Frontmatter: map[string]interface{}{},
		Body:        "Test  content\n\n\nwith   multiple   spaces",
	}

	result, err := plugin.ExecuteHook(ctx, HookPerFile, hookCtx, file)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Modified)
	assert.NotEqual(t, file.Body, result.NewContent)
	assert.Contains(t, result.Metadata, "content_enhancements")

	// Test cleanup
	err = plugin.Cleanup()
	assert.NoError(t, err)
}

func TestExampleExportPlugin(t *testing.T) {
	plugin := NewExampleExportPlugin()

	// Test plugin metadata
	assert.Equal(t, "export-processor", plugin.Name())
	assert.Equal(t, "1.0.0", plugin.Version())
	assert.NotEmpty(t, plugin.Description())
	assert.Contains(t, plugin.SupportedHooks(), HookPerFile)
	assert.Contains(t, plugin.SupportedHooks(), HookExportComplete)

	// Test initialization
	err := plugin.Init(map[string]interface{}{})
	assert.NoError(t, err)

	// Test per-file hook
	ctx := context.Background()
	hookCtx := &HookContext{
		Command:   "export",
		VaultPath: "/test/vault",
	}
	file := &vault.VaultFile{
		Path:        "test.md",
		Frontmatter: map[string]interface{}{},
		Body:        "Test content",
	}

	result, err := plugin.ExecuteHook(ctx, HookPerFile, hookCtx, file)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Modified)
	assert.Contains(t, result.NewContent, "Exported by mdnotes")
	assert.Contains(t, result.Metadata, "export_timestamp")

	// Test export complete hook
	result, err = plugin.ExecuteHook(ctx, HookExportComplete, hookCtx, nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, result.Metadata, "export_completed")

	// Test cleanup
	err = plugin.Cleanup()
	assert.NoError(t, err)
}
