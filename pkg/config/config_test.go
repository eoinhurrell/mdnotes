package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	assert.Equal(t, ".", config.Vault.Path)
	assert.Contains(t, config.Vault.IgnorePatterns, ".obsidian/*")
	assert.Contains(t, config.Vault.IgnorePatterns, "*.tmp")
	assert.Contains(t, config.Vault.IgnorePatterns, ".git/*")
	
	assert.True(t, config.Linkding.SyncTitle)
	assert.True(t, config.Linkding.SyncTags)
	
	assert.Equal(t, "remove", config.Export.DefaultStrategy)
	assert.True(t, config.Export.IncludeAssets)
	
	assert.Equal(t, "2s", config.Watch.Debounce)
	assert.Empty(t, config.Watch.Rules)
	
	assert.Equal(t, 0, config.Performance.MaxWorkers)
	
	assert.True(t, config.Plugins.Enabled)
	assert.Contains(t, config.Plugins.Paths, "~/.mdnotes/plugins")
}

func TestLoader_Load_NoConfigFile(t *testing.T) {
	// Test loading when no config file exists
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)
	
	loader := NewLoader()
	config, err := loader.Load()
	
	require.NoError(t, err)
	require.NotNil(t, config)
	
	// Should return defaults
	assert.Equal(t, ".", config.Vault.Path)
	assert.Equal(t, "remove", config.Export.DefaultStrategy)
}

func TestLoader_Load_ValidConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "mdnotes.yaml")
	
	configContent := `
vault:
  path: "/custom/vault"
  ignore_patterns: [".custom/*", "*.bak"]
linkding:
  api_url: "https://example.com"
  api_token: "test-token"
  sync_title: false
export:
  default_strategy: "url"
performance:
  max_workers: 4
`
	
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)
	
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)
	
	loader := NewLoader()
	config, err := loader.Load()
	
	require.NoError(t, err)
	require.NotNil(t, config)
	
	assert.Equal(t, "/custom/vault", config.Vault.Path)
	assert.Equal(t, []string{".custom/*", "*.bak"}, config.Vault.IgnorePatterns)
	assert.Equal(t, "https://example.com", config.Linkding.APIURL)
	assert.Equal(t, "test-token", config.Linkding.APIToken)
	assert.False(t, config.Linkding.SyncTitle)
	assert.Equal(t, "url", config.Export.DefaultStrategy)
	assert.Equal(t, 4, config.Performance.MaxWorkers)
}

func TestLoader_Load_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "mdnotes.yaml")
	
	invalidYAML := `
vault:
  path: "/vault"
  invalid_yaml: [unclosed list
`
	
	err := os.WriteFile(configFile, []byte(invalidYAML), 0644)
	require.NoError(t, err)
	
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)
	
	loader := NewLoader()
	_, err = loader.Load()
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error reading config file")
}

func TestLoader_Validate(t *testing.T) {
	loader := NewLoader()
	
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid config",
			config:      DefaultConfig(),
			expectError: false,
		},
		{
			name: "empty vault path",
			config: &Config{
				Vault: VaultConfig{Path: ""},
			},
			expectError: true,
			errorMsg:    "vault.path cannot be empty",
		},
		{
			name: "linkding url without token",
			config: &Config{
				Vault:    VaultConfig{Path: "."},
				Linkding: LinkdingConfig{APIURL: "https://example.com"},
			},
			expectError: true,
			errorMsg:    "linkding.api_token is required",
		},
		{
			name: "invalid export strategy",
			config: &Config{
				Vault:  VaultConfig{Path: "."},
				Export: ExportConfig{DefaultStrategy: "invalid"},
			},
			expectError: true,
			errorMsg:    "invalid export.default_strategy",
		},
		{
			name: "negative max workers",
			config: &Config{
				Vault:       VaultConfig{Path: "."},
				Performance: PerformanceConfig{MaxWorkers: -1},
			},
			expectError: true,
			errorMsg:    "performance.max_workers cannot be negative",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := loader.Validate(tt.config)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoader_ExpandPath(t *testing.T) {
	loader := NewLoader()
	
	tests := []struct {
		name     string
		input    string
		expected func(string) bool // Function to validate result
	}{
		{
			name:  "empty path",
			input: "",
			expected: func(result string) bool {
				return result == ""
			},
		},
		{
			name:  "current directory",
			input: ".",
			expected: func(result string) bool {
				abs, _ := filepath.Abs(".")
				return result == abs
			},
		},
		{
			name:  "home directory expansion",
			input: "~",
			expected: func(result string) bool {
				home, _ := os.UserHomeDir()
				return result == home
			},
		},
		{
			name:  "home subdirectory expansion",
			input: "~/Documents",
			expected: func(result string) bool {
				home, _ := os.UserHomeDir()
				return result == filepath.Join(home, "Documents")
			},
		},
		{
			name:  "absolute path",
			input: "/tmp",
			expected: func(result string) bool {
				return result == "/tmp"
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := loader.expandPath(tt.input)
			assert.True(t, tt.expected(result), "Expected validation failed for result: %s", result)
		})
	}
}

func TestLoader_MigrateLegacy(t *testing.T) {
	tempDir := t.TempDir()
	legacyConfigFile := filepath.Join(tempDir, ".obsidian-admin.yaml")
	
	legacyContent := `
vault:
  path: "/legacy/vault"
  ignore_patterns: [".obsidian/*", "*.old"]
linkding:
  api_url: "https://legacy.example.com"
  api_token: "legacy-token"
  sync_title: false
  sync_tags: true
`
	
	err := os.WriteFile(legacyConfigFile, []byte(legacyContent), 0644)
	require.NoError(t, err)
	
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)
	
	loader := NewLoader()
	config, err := loader.MigrateLegacy()
	
	require.NoError(t, err)
	require.NotNil(t, config)
	
	// Check migrated values
	assert.Equal(t, "/legacy/vault", config.Vault.Path)
	assert.Equal(t, []string{".obsidian/*", "*.old"}, config.Vault.IgnorePatterns)
	assert.Equal(t, "https://legacy.example.com", config.Linkding.APIURL)
	assert.Equal(t, "legacy-token", config.Linkding.APIToken)
	assert.False(t, config.Linkding.SyncTitle)
	assert.True(t, config.Linkding.SyncTags)
	
	// Check that defaults are preserved for unmigrated fields
	assert.Equal(t, "remove", config.Export.DefaultStrategy)
	assert.True(t, config.Export.IncludeAssets)
}

func TestLoader_MigrateLegacy_NoLegacyFile(t *testing.T) {
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)
	
	loader := NewLoader()
	config, err := loader.MigrateLegacy()
	
	require.NoError(t, err)
	assert.Nil(t, config) // No legacy config found
}

func TestLoader_MigrateLegacy_InvalidLegacyFile(t *testing.T) {
	tempDir := t.TempDir()
	legacyConfigFile := filepath.Join(tempDir, ".obsidian-admin.yaml")
	
	invalidContent := `
vault:
  path: "/vault"
  invalid: [unclosed
`
	
	err := os.WriteFile(legacyConfigFile, []byte(invalidContent), 0644)
	require.NoError(t, err)
	
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)
	
	loader := NewLoader()
	_, err = loader.MigrateLegacy()
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error reading legacy config")
}

// TestEnvironmentVariableOverrides tests that environment variables properly override config
func TestLoader_EnvironmentVariableOverrides(t *testing.T) {
	// Set environment variables
	os.Setenv("MDNOTES_VAULT_PATH", "/env/vault")
	os.Setenv("MDNOTES_LINKDING_API_URL", "https://env.example.com")
	os.Setenv("MDNOTES_PERFORMANCE_MAX_WORKERS", "8")
	defer func() {
		os.Unsetenv("MDNOTES_VAULT_PATH")
		os.Unsetenv("MDNOTES_LINKDING_API_URL")
		os.Unsetenv("MDNOTES_PERFORMANCE_MAX_WORKERS")
	}()
	
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)
	
	loader := NewLoader()
	config, err := loader.Load()
	
	require.NoError(t, err)
	require.NotNil(t, config)
	
	// Environment variables should override defaults
	assert.Equal(t, "/env/vault", config.Vault.Path)
	assert.Equal(t, "https://env.example.com", config.Linkding.APIURL)
	assert.Equal(t, 8, config.Performance.MaxWorkers)
}