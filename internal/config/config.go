package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	Version     string            `yaml:"version"`
	Vault       VaultConfig       `yaml:"vault"`
	Frontmatter FrontmatterConfig `yaml:"frontmatter"`
	Linkding    LinkdingConfig    `yaml:"linkding"`
	Batch       BatchConfig       `yaml:"batch"`
	Safety      SafetyConfig      `yaml:"safety"`
	Downloads   DownloadConfig    `yaml:"downloads"`
	Watch       WatchConfig       `yaml:"watch"`
}

// VaultConfig contains vault-specific settings
type VaultConfig struct {
	Path           string   `yaml:"path"`
	IgnorePatterns []string `yaml:"ignore_patterns"`
}

// FrontmatterConfig contains frontmatter processing settings
type FrontmatterConfig struct {
	RequiredFields []string  `yaml:"required_fields"`
	TypeRules      TypeRules `yaml:"type_rules"`
}

// TypeRules defines field type validation rules
type TypeRules struct {
	Fields map[string]string `yaml:"fields"`
}

// LinkdingConfig contains linkding integration settings
type LinkdingConfig struct {
	APIURL    string `yaml:"api_url"`
	APIToken  string `yaml:"api_token"`
	SyncTitle bool   `yaml:"sync_title"`
	SyncTags  bool   `yaml:"sync_tags"`
}

// BatchConfig contains batch processing settings
type BatchConfig struct {
	StopOnError  bool `yaml:"stop_on_error"`
	CreateBackup bool `yaml:"create_backup"`
	MaxWorkers   int  `yaml:"max_workers"`
}

// SafetyConfig contains safety and backup settings
type SafetyConfig struct {
	BackupRetention string `yaml:"backup_retention"`
	MaxBackups      int    `yaml:"max_backups"`
}

// DownloadConfig contains settings for downloading resources
type DownloadConfig struct {
	AttachmentsDir string `yaml:"attachments_dir"`
	Timeout        string `yaml:"timeout"`
	UserAgent      string `yaml:"user_agent"`
	MaxFileSize    int64  `yaml:"max_file_size"`
}

// WatchConfig contains file watching settings
type WatchConfig struct {
	Enabled         bool                `yaml:"enabled"`
	DebounceTimeout string              `yaml:"debounce_timeout"`
	Rules           []WatchRule         `yaml:"rules"`
	IgnorePatterns  []string            `yaml:"ignore_patterns"`
}

// WatchRule defines a file watching rule
type WatchRule struct {
	Name    string   `yaml:"name"`
	Paths   []string `yaml:"paths"`
	Events  []string `yaml:"events"`
	Actions []string `yaml:"actions"`
}

// LoadConfig loads configuration from a reader with environment variable expansion
func LoadConfig(reader io.Reader) (*Config, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	// Expand environment variables
	expandedContent := expandEnvVars(string(content))

	var config Config
	if err := yaml.Unmarshal([]byte(expandedContent), &config); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	return &config, nil
}

// LoadConfigFromFile loads configuration from a file
func LoadConfigFromFile(filepath string) (*Config, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("opening config file %s: %w", filepath, err)
	}
	defer file.Close()

	return LoadConfig(file)
}

// LoadConfigWithFallback tries to load config from multiple paths, returns default if none found
func LoadConfigWithFallback(paths []string) (*Config, error) {
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return LoadConfigFromFile(path)
		}
	}

	// Return default config if no file found
	return DefaultConfig(), nil
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Version: "1.0",
		Vault: VaultConfig{
			Path: "",
			IgnorePatterns: []string{
				".obsidian/*",
				"*.tmp",
				"*.bak",
				".DS_Store",
			},
		},
		Frontmatter: FrontmatterConfig{
			RequiredFields: []string{},
			TypeRules: TypeRules{
				Fields: make(map[string]string),
			},
		},
		Linkding: LinkdingConfig{
			APIURL:    "",
			APIToken:  "",
			SyncTitle: false,
			SyncTags:  false,
		},
		Batch: BatchConfig{
			StopOnError:  false,
			CreateBackup: true,
			MaxWorkers:   4,
		},
		Safety: SafetyConfig{
			BackupRetention: "24h",
			MaxBackups:      50,
		},
		Downloads: DownloadConfig{
			AttachmentsDir: "./resources/attachments",
			Timeout:        "30s",
			UserAgent:      "mdnotes/1.0",
			MaxFileSize:    10 * 1024 * 1024, // 10MB
		},
		Watch: WatchConfig{
			Enabled:         false,
			DebounceTimeout: "2s",
			Rules:           []WatchRule{},
			IgnorePatterns: []string{
				".obsidian/*",
				".git/*",
				"node_modules/*",
				"*.tmp",
				"*.bak",
				"*.swp",
				".DS_Store",
			},
		},
	}
}

// GetDefaultConfigPaths returns default configuration file paths to search
func GetDefaultConfigPaths() []string {
	homeDir, _ := os.UserHomeDir()
	currentDir, _ := os.Getwd()

	return []string{
		filepath.Join(currentDir, ".obsidian-admin.yaml"),
		filepath.Join(currentDir, "obsidian-admin.yaml"),
		filepath.Join(homeDir, ".config", "obsidian-admin", "config.yaml"),
		filepath.Join(homeDir, ".obsidian-admin.yaml"),
		"/etc/obsidian-admin/config.yaml",
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Version == "" {
		return fmt.Errorf("version is required")
	}

	// Validate type rules
	validTypes := map[string]bool{
		"string":  true,
		"number":  true,
		"boolean": true,
		"array":   true,
		"date":    true,
		"object":  true,
	}

	for field, fieldType := range c.Frontmatter.TypeRules.Fields {
		if !validTypes[fieldType] {
			return fmt.Errorf("invalid type '%s' for field '%s'", fieldType, field)
		}
	}

	// Validate backup retention duration
	if c.Safety.BackupRetention != "" {
		if _, err := time.ParseDuration(c.Safety.BackupRetention); err != nil {
			return fmt.Errorf("invalid backup retention duration: %w", err)
		}
	}

	// Validate watch debounce timeout
	if c.Watch.DebounceTimeout != "" {
		if _, err := time.ParseDuration(c.Watch.DebounceTimeout); err != nil {
			return fmt.Errorf("invalid watch debounce timeout: %w", err)
		}
	}

	// Validate watch rule events
	validEvents := map[string]bool{
		"create": true,
		"write":  true,
		"remove": true,
		"rename": true,
		"chmod":  true,
	}

	for _, rule := range c.Watch.Rules {
		for _, event := range rule.Events {
			if !validEvents[event] {
				return fmt.Errorf("invalid watch event '%s' in rule '%s'", event, rule.Name)
			}
		}
	}

	return nil
}

// SaveToFile saves the configuration to a file
func (c *Config) SaveToFile(filepath string) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath[:strings.LastIndex(filepath, "/")], 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

// Merge merges another config into this one, with the other config taking precedence
func (c *Config) Merge(other Config) *Config {
	result := *c // Start with base config

	// Merge non-zero values from other config
	if other.Version != "" {
		result.Version = other.Version
	}

	// Vault config
	if other.Vault.Path != "" {
		result.Vault.Path = other.Vault.Path
	}
	if len(other.Vault.IgnorePatterns) > 0 {
		result.Vault.IgnorePatterns = other.Vault.IgnorePatterns
	}

	// Frontmatter config
	if len(other.Frontmatter.RequiredFields) > 0 {
		result.Frontmatter.RequiredFields = other.Frontmatter.RequiredFields
	}
	if len(other.Frontmatter.TypeRules.Fields) > 0 {
		if result.Frontmatter.TypeRules.Fields == nil {
			result.Frontmatter.TypeRules.Fields = make(map[string]string)
		}
		for k, v := range other.Frontmatter.TypeRules.Fields {
			result.Frontmatter.TypeRules.Fields[k] = v
		}
	}

	// Linkding config
	if other.Linkding.APIURL != "" {
		result.Linkding.APIURL = other.Linkding.APIURL
	}
	if other.Linkding.APIToken != "" {
		result.Linkding.APIToken = other.Linkding.APIToken
	}

	// Batch config
	if other.Batch.MaxWorkers != 0 {
		result.Batch.MaxWorkers = other.Batch.MaxWorkers
	}

	// Safety config
	if other.Safety.BackupRetention != "" {
		result.Safety.BackupRetention = other.Safety.BackupRetention
	}
	if other.Safety.MaxBackups != 0 {
		result.Safety.MaxBackups = other.Safety.MaxBackups
	}

	// Downloads config
	if other.Downloads.AttachmentsDir != "" {
		result.Downloads.AttachmentsDir = other.Downloads.AttachmentsDir
	}
	if other.Downloads.Timeout != "" {
		result.Downloads.Timeout = other.Downloads.Timeout
	}
	if other.Downloads.UserAgent != "" {
		result.Downloads.UserAgent = other.Downloads.UserAgent
	}
	if other.Downloads.MaxFileSize != 0 {
		result.Downloads.MaxFileSize = other.Downloads.MaxFileSize
	}

	return &result
}

// expandEnvVars expands environment variables in the format ${VAR_NAME}
func expandEnvVars(content string) string {
	// Pattern to match ${VAR_NAME}
	pattern := regexp.MustCompile(`\$\{([^}]+)\}`)

	return pattern.ReplaceAllStringFunc(content, func(match string) string {
		// Extract variable name (remove ${ and })
		varName := match[2 : len(match)-1]

		// Get environment variable value
		envValue := os.Getenv(varName)

		// Return the environment variable value or empty string if not found
		return envValue
	})
}
