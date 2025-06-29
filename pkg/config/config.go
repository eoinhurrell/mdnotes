package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the complete mdnotes configuration
type Config struct {
	Vault       VaultConfig       `mapstructure:"vault" yaml:"vault"`
	Frontmatter FrontmatterConfig `mapstructure:"frontmatter" yaml:"frontmatter"`
	Linkding    LinkdingConfig    `mapstructure:"linkding" yaml:"linkding"`
	Export      ExportConfig      `mapstructure:"export" yaml:"export"`
	Watch       WatchConfig       `mapstructure:"watch" yaml:"watch"`
	Performance PerformanceConfig `mapstructure:"performance" yaml:"performance"`
	Plugins     PluginsConfig     `mapstructure:"plugins" yaml:"plugins"`
}

// VaultConfig contains vault-specific settings
type VaultConfig struct {
	Path           string   `mapstructure:"path" yaml:"path"`
	IgnorePatterns []string `mapstructure:"ignore_patterns" yaml:"ignore_patterns"`
}

// FrontmatterConfig contains frontmatter processing settings
type FrontmatterConfig struct {
	UpsertDefaults map[string]interface{} `mapstructure:"upsert_defaults" yaml:"upsert_defaults"`
	TypeRules      map[string]string      `mapstructure:"type_rules" yaml:"type_rules"`
}

// LinkdingConfig contains Linkding integration settings
type LinkdingConfig struct {
	APIURL    string `mapstructure:"api_url" yaml:"api_url"`
	APIToken  string `mapstructure:"api_token" yaml:"api_token"`
	SyncTitle bool   `mapstructure:"sync_title" yaml:"sync_title"`
	SyncTags  bool   `mapstructure:"sync_tags" yaml:"sync_tags"`
}

// ExportConfig contains export operation settings
type ExportConfig struct {
	DefaultStrategy string `mapstructure:"default_strategy" yaml:"default_strategy"`
	IncludeAssets   bool   `mapstructure:"include_assets" yaml:"include_assets"`
}

// WatchConfig contains file watching settings
type WatchConfig struct {
	Debounce string      `mapstructure:"debounce" yaml:"debounce"`
	Rules    []WatchRule `mapstructure:"rules" yaml:"rules"`
}

// WatchRule defines a file watching rule
type WatchRule struct {
	Name    string   `mapstructure:"name" yaml:"name"`
	Paths   []string `mapstructure:"paths" yaml:"paths"`
	Events  []string `mapstructure:"events" yaml:"events"`
	Actions []string `mapstructure:"actions" yaml:"actions"`
}

// PerformanceConfig contains performance optimization settings
type PerformanceConfig struct {
	MaxWorkers int `mapstructure:"max_workers" yaml:"max_workers"`
}

// PluginsConfig contains plugin system settings
type PluginsConfig struct {
	Enabled bool     `mapstructure:"enabled" yaml:"enabled"`
	Paths   []string `mapstructure:"paths" yaml:"paths"`
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Vault: VaultConfig{
			Path:           ".",
			IgnorePatterns: []string{".obsidian/*", "*.tmp", ".git/*"},
		},
		Frontmatter: FrontmatterConfig{
			UpsertDefaults: make(map[string]interface{}),
			TypeRules:      make(map[string]string),
		},
		Linkding: LinkdingConfig{
			SyncTitle: true,
			SyncTags:  true,
		},
		Export: ExportConfig{
			DefaultStrategy: "remove",
			IncludeAssets:   true,
		},
		Watch: WatchConfig{
			Debounce: "2s",
			Rules:    []WatchRule{},
		},
		Performance: PerformanceConfig{
			MaxWorkers: 0, // 0 = auto-detect
		},
		Plugins: PluginsConfig{
			Enabled: true,
			Paths:   []string{"~/.mdnotes/plugins"},
		},
	}
}

// Loader handles configuration loading and merging
type Loader struct {
	searchPaths []string
}

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
	return &Loader{
		searchPaths: []string{
			".",            // Current working directory
			"~",            // User home directory
			"/etc/mdnotes", // System-wide directory
		},
	}
}

// Load loads configuration from multiple sources with precedence
func (l *Loader) Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	config := DefaultConfig()

	// Configure viper
	v.SetConfigName("mdnotes")
	v.SetConfigType("yaml")

	// Add search paths
	for _, path := range l.searchPaths {
		expandedPath := l.expandPath(path)
		v.AddConfigPath(expandedPath)
	}

	// Enable environment variable support
	v.SetEnvPrefix("MDNOTES")
	v.AutomaticEnv()

	// Try to read configuration
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found is OK, we'll use defaults
	}

	// Unmarshal into struct
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate configuration
	if err := l.Validate(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Expand paths in configuration
	config.Vault.Path = l.expandPath(config.Vault.Path)
	for i, path := range config.Plugins.Paths {
		config.Plugins.Paths[i] = l.expandPath(path)
	}

	return config, nil
}

// expandPath expands ~ to home directory and resolves relative paths
func (l *Loader) expandPath(path string) string {
	if path == "" {
		return path
	}

	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path // Return original if can't expand
		}
		return filepath.Join(home, path[1:])
	}

	// Convert to absolute path
	abs, err := filepath.Abs(path)
	if err != nil {
		return path // Return original if can't resolve
	}

	return abs
}

// Validate performs basic validation on the configuration
func (l *Loader) Validate(config *Config) error {
	// Validate vault path
	if config.Vault.Path == "" {
		return fmt.Errorf("vault.path cannot be empty")
	}

	// Validate Linkding configuration if API URL is provided
	if config.Linkding.APIURL != "" && config.Linkding.APIToken == "" {
		return fmt.Errorf("linkding.api_token is required when api_url is specified")
	}

	// Validate export strategy
	validStrategies := map[string]bool{"remove": true, "url": true, "stub": true}
	if !validStrategies[config.Export.DefaultStrategy] {
		return fmt.Errorf("invalid export.default_strategy: %s", config.Export.DefaultStrategy)
	}

	// Validate performance settings
	if config.Performance.MaxWorkers < 0 {
		return fmt.Errorf("performance.max_workers cannot be negative")
	}

	return nil
}

// MigrateLegacy attempts to migrate from legacy obsidian-admin.yaml
func (l *Loader) MigrateLegacy() (*Config, error) {
	// Look for legacy config files
	legacyFiles := []string{
		".obsidian-admin.yaml",
		"obsidian-admin.yaml",
	}

	for _, legacyFile := range legacyFiles {
		if _, err := os.Stat(legacyFile); err == nil {
			// Found legacy config, attempt migration
			return l.migrateLegacyFile(legacyFile)
		}
	}

	return nil, nil // No legacy config found
}

// migrateLegacyFile migrates a specific legacy config file
func (l *Loader) migrateLegacyFile(filename string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(filename)

	var legacyConfig map[string]interface{}
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading legacy config %s: %w", filename, err)
	}

	if err := v.Unmarshal(&legacyConfig); err != nil {
		return nil, fmt.Errorf("error parsing legacy config %s: %w", filename, err)
	}

	// Start with defaults and merge legacy settings
	config := DefaultConfig()

	// Migrate vault settings
	if vault, ok := legacyConfig["vault"].(map[string]interface{}); ok {
		if path, ok := vault["path"].(string); ok {
			config.Vault.Path = path
		}
		if patterns, ok := vault["ignore_patterns"].([]interface{}); ok {
			config.Vault.IgnorePatterns = make([]string, len(patterns))
			for i, p := range patterns {
				if str, ok := p.(string); ok {
					config.Vault.IgnorePatterns[i] = str
				}
			}
		}
	}

	// Migrate linkding settings
	if linkding, ok := legacyConfig["linkding"].(map[string]interface{}); ok {
		if apiURL, ok := linkding["api_url"].(string); ok {
			config.Linkding.APIURL = apiURL
		}
		if apiToken, ok := linkding["api_token"].(string); ok {
			config.Linkding.APIToken = apiToken
		}
		if syncTitle, ok := linkding["sync_title"].(bool); ok {
			config.Linkding.SyncTitle = syncTitle
		}
		if syncTags, ok := linkding["sync_tags"].(bool); ok {
			config.Linkding.SyncTags = syncTags
		}
	}

	return config, nil
}
