package config

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Load(t *testing.T) {
	configYAML := `
version: "1.0"
vault:
  ignore_patterns:
    - "*.tmp"
    - ".obsidian/*"
frontmatter:
  required_fields: ["id", "title"]
  type_rules:
    fields:
      created: date
      tags: array
      priority: number
linkding:
  api_url: "${LINKDING_URL}"
  api_token: "${LINKDING_TOKEN}"
  sync_title: true
  sync_tags: true
batch:
  stop_on_error: false
  create_backup: true
  max_workers: 4
safety:
  backup_retention: "72h"
  max_backups: 100
`

	// Set environment variables
	os.Setenv("LINKDING_URL", "https://linkding.example.com")
	os.Setenv("LINKDING_TOKEN", "secret-token")
	defer func() {
		os.Unsetenv("LINKDING_URL")
		os.Unsetenv("LINKDING_TOKEN")
	}()

	cfg, err := LoadConfig(strings.NewReader(configYAML))
	require.NoError(t, err)

	// Test basic structure
	assert.Equal(t, "1.0", cfg.Version)

	// Test vault config
	assert.Contains(t, cfg.Vault.IgnorePatterns, "*.tmp")
	assert.Contains(t, cfg.Vault.IgnorePatterns, ".obsidian/*")

	// Test frontmatter config
	assert.Contains(t, cfg.Frontmatter.RequiredFields, "id")
	assert.Contains(t, cfg.Frontmatter.RequiredFields, "title")
	assert.Equal(t, "date", cfg.Frontmatter.TypeRules.Fields["created"])
	assert.Equal(t, "array", cfg.Frontmatter.TypeRules.Fields["tags"])
	assert.Equal(t, "number", cfg.Frontmatter.TypeRules.Fields["priority"])

	// Test linkding config with environment variable expansion
	assert.Equal(t, "https://linkding.example.com", cfg.Linkding.APIURL)
	assert.Equal(t, "secret-token", cfg.Linkding.APIToken)
	assert.True(t, cfg.Linkding.SyncTitle)
	assert.True(t, cfg.Linkding.SyncTags)

	// Test batch config
	assert.False(t, cfg.Batch.StopOnError)
	assert.True(t, cfg.Batch.CreateBackup)
	assert.Equal(t, 4, cfg.Batch.MaxWorkers)

	// Test safety config
	assert.Equal(t, "72h", cfg.Safety.BackupRetention)
	assert.Equal(t, 100, cfg.Safety.MaxBackups)
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: Config{
				Version: "1.0",
				Frontmatter: FrontmatterConfig{
					TypeRules: TypeRules{
						Fields: map[string]string{
							"date": "date",
							"tags": "array",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid type in type rules",
			config: Config{
				Version: "1.0",
				Frontmatter: FrontmatterConfig{
					TypeRules: TypeRules{
						Fields: map[string]string{
							"date": "invalid-type",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "invalid type",
		},
		{
			name: "missing version",
			config: Config{
				Frontmatter: FrontmatterConfig{},
			},
			expectError: true,
			errorMsg:    "version is required",
		},
		{
			name: "invalid backup retention",
			config: Config{
				Version: "1.0",
				Safety: SafetyConfig{
					BackupRetention: "invalid-duration",
				},
			},
			expectError: true,
			errorMsg:    "invalid backup retention",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_LoadFromFile(t *testing.T) {
	// Create temporary config file
	tmpFile := t.TempDir() + "/config.yaml"
	configContent := `
version: "1.0"
vault:
  ignore_patterns: ["*.tmp"]
frontmatter:
  required_fields: ["title"]
`
	err := os.WriteFile(tmpFile, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := LoadConfigFromFile(tmpFile)
	require.NoError(t, err)

	assert.Equal(t, "1.0", cfg.Version)
	assert.Contains(t, cfg.Vault.IgnorePatterns, "*.tmp")
	assert.Contains(t, cfg.Frontmatter.RequiredFields, "title")
}

func TestConfig_LoadNonExistentFile(t *testing.T) {
	_, err := LoadConfigFromFile("/nonexistent/config.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}

func TestConfig_DefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "1.0", cfg.Version)
	assert.Contains(t, cfg.Vault.IgnorePatterns, ".obsidian/*")
	assert.Contains(t, cfg.Vault.IgnorePatterns, "*.tmp")
	assert.False(t, cfg.Batch.StopOnError)
	assert.True(t, cfg.Batch.CreateBackup)
	assert.Equal(t, "24h", cfg.Safety.BackupRetention)
	assert.Equal(t, 50, cfg.Safety.MaxBackups)
}

func TestConfig_EnvironmentVariableExpansion(t *testing.T) {
	// Set test environment variables
	os.Setenv("TEST_URL", "https://test.example.com")
	os.Setenv("TEST_TOKEN", "test-token-123")
	os.Setenv("TEST_NUMBER", "42")
	defer func() {
		os.Unsetenv("TEST_URL")
		os.Unsetenv("TEST_TOKEN")
		os.Unsetenv("TEST_NUMBER")
	}()

	configYAML := `
version: "1.0"
linkding:
  api_url: "${TEST_URL}/api"
  api_token: "${TEST_TOKEN}"
batch:
  max_workers: ${TEST_NUMBER}
vault:
  path: "${HOME}/vault"
`

	cfg, err := LoadConfig(strings.NewReader(configYAML))
	require.NoError(t, err)

	assert.Equal(t, "https://test.example.com/api", cfg.Linkding.APIURL)
	assert.Equal(t, "test-token-123", cfg.Linkding.APIToken)
	assert.Equal(t, 42, cfg.Batch.MaxWorkers)
	// HOME should be expanded from actual environment
	assert.NotContains(t, cfg.Vault.Path, "${HOME}")
}

func TestConfig_MissingEnvironmentVariable(t *testing.T) {
	configYAML := `
version: "1.0"
linkding:
  api_url: "${MISSING_VAR}"
`

	cfg, err := LoadConfig(strings.NewReader(configYAML))
	require.NoError(t, err)

	// Missing environment variables should be left as empty strings
	assert.Equal(t, "", cfg.Linkding.APIURL)
}

func TestConfig_NestedEnvironmentVariables(t *testing.T) {
	os.Setenv("BASE_URL", "https://api.example.com")
	os.Setenv("API_VERSION", "v1")
	defer func() {
		os.Unsetenv("BASE_URL")
		os.Unsetenv("API_VERSION")
	}()

	configYAML := `
version: "1.0"
linkding:
  api_url: "${BASE_URL}/${API_VERSION}/bookmarks"
`

	cfg, err := LoadConfig(strings.NewReader(configYAML))
	require.NoError(t, err)

	assert.Equal(t, "https://api.example.com/v1/bookmarks", cfg.Linkding.APIURL)
}

func TestConfig_SaveConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Vault.Path = "/test/vault"
	cfg.Linkding.APIURL = "https://test.example.com"

	// Save to temporary file
	tmpFile := t.TempDir() + "/saved-config.yaml"
	err := cfg.SaveToFile(tmpFile)
	require.NoError(t, err)

	// Load it back
	loadedCfg, err := LoadConfigFromFile(tmpFile)
	require.NoError(t, err)

	assert.Equal(t, cfg.Version, loadedCfg.Version)
	assert.Equal(t, cfg.Vault.Path, loadedCfg.Vault.Path)
	assert.Equal(t, cfg.Linkding.APIURL, loadedCfg.Linkding.APIURL)
}

func TestConfig_MergeConfig(t *testing.T) {
	base := DefaultConfig()
	base.Vault.Path = "/base/vault"
	base.Linkding.APIURL = "https://base.example.com"

	override := Config{
		Vault: VaultConfig{
			Path: "/override/vault",
		},
		Frontmatter: FrontmatterConfig{
			RequiredFields: []string{"override-field"},
		},
	}

	merged := base.Merge(override)

	// Overridden values
	assert.Equal(t, "/override/vault", merged.Vault.Path)
	assert.Contains(t, merged.Frontmatter.RequiredFields, "override-field")

	// Preserved values
	assert.Equal(t, "https://base.example.com", merged.Linkding.APIURL)
	assert.Equal(t, "1.0", merged.Version)
}

func TestConfig_GetConfigPaths(t *testing.T) {
	paths := GetDefaultConfigPaths()

	// Should include standard config locations
	assert.Greater(t, len(paths), 0)

	// Should include current directory
	found := false
	for _, path := range paths {
		if strings.Contains(path, ".obsidian-admin.yaml") {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestConfig_LoadWithFallback(t *testing.T) {
	// Create config file only in temp directory
	tmpDir := t.TempDir()
	configFile := tmpDir + "/.obsidian-admin.yaml"
	configContent := `
version: "1.0"
vault:
  path: "/test/vault"
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load with custom search paths
	cfg, err := LoadConfigWithFallback([]string{configFile, "/nonexistent/config.yaml"})
	require.NoError(t, err)

	assert.Equal(t, "1.0", cfg.Version)
	assert.Equal(t, "/test/vault", cfg.Vault.Path)
}

func TestConfig_LoadWithFallbackDefault(t *testing.T) {
	// Load with all nonexistent paths - should return default config
	cfg, err := LoadConfigWithFallback([]string{"/nonexistent/config1.yaml", "/nonexistent/config2.yaml"})
	require.NoError(t, err)

	// Should be default config
	defaultCfg := DefaultConfig()
	assert.Equal(t, defaultCfg.Version, cfg.Version)
	assert.Equal(t, defaultCfg.Vault.IgnorePatterns, cfg.Vault.IgnorePatterns)
}
