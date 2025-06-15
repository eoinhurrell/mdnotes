package analyze

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/eoinhurrell/mdnotes/internal/analyzer"
	"github.com/eoinhurrell/mdnotes/internal/config"
	"github.com/eoinhurrell/mdnotes/internal/errors"
	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// NewAnalyzeCommand creates the analyze command
func NewAnalyzeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze vault statistics and health",
		Long:  `Generate comprehensive statistics and health reports for your vault`,
	}

	// Add subcommands
	cmd.AddCommand(newStatsCommand())
	cmd.AddCommand(newDuplicatesCommand())
	cmd.AddCommand(newHealthCommand())

	return cmd
}

func newStatsCommand() *cobra.Command {
	var (
		outputFormat string
		outputFile   string
	)

	cmd := &cobra.Command{
		Use:   "stats [vault-path]",
		Short: "Generate vault statistics",
		Long:  `Generate comprehensive statistics about your vault including file counts, frontmatter usage, and tag distribution`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vaultPath := "."
			if len(args) > 0 {
				vaultPath = args[0]
			}

			// Load configuration
			cfg, err := loadConfig(cmd)
			if err != nil {
				return errors.NewConfigError("", err.Error())
			}

			// Scan vault files
			scanner := vault.NewScanner(vault.WithIgnorePatterns(cfg.Vault.IgnorePatterns))
			files, err := scanner.Walk(vaultPath)
			if err != nil {
				if os.IsNotExist(err) {
					return errors.NewFileNotFoundError(vaultPath, 
						"Ensure the vault path exists and contains markdown files. Use 'ls' to verify the directory structure.")
				}
				if os.IsPermission(err) {
					return errors.NewPermissionError(vaultPath, "vault scanning")
				}
				return errors.WrapError(err, "vault scanning", vaultPath)
			}

			// Generate statistics
			analyzer := analyzer.NewAnalyzer()
			stats := analyzer.GenerateStats(files)

			// Output results
			if outputFormat == "json" {
				data, err := json.MarshalIndent(stats, "", "  ")
				if err != nil {
					return fmt.Errorf("marshaling JSON: %w", err)
				}

				if outputFile != "" {
					return os.WriteFile(outputFile, data, 0644)
				}
				fmt.Println(string(data))
			} else {
				output := formatStatsText(stats)
				if outputFile != "" {
					return os.WriteFile(outputFile, []byte(output), 0644)
				}
				fmt.Print(output)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "Output format (text, json)")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (default: stdout)")

	return cmd
}

func newDuplicatesCommand() *cobra.Command {
	var (
		outputFormat string
		minSimilarity float64
	)

	cmd := &cobra.Command{
		Use:   "duplicates [vault-path]",
		Short: "Find duplicate files",
		Long:  `Find duplicate files in your vault based on content similarity`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vaultPath := "."
			if len(args) > 0 {
				vaultPath = args[0]
			}

			// Load configuration
			cfg, err := loadConfig(cmd)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			// Scan vault files
			scanner := vault.NewScanner(vault.WithIgnorePatterns(cfg.Vault.IgnorePatterns))
			files, err := scanner.Walk(vaultPath)
			if err != nil {
				return fmt.Errorf("scanning vault: %w", err)
			}

			// Find duplicates
			analyzer := analyzer.NewAnalyzer()
			duplicates := analyzer.FindDuplicates(files, "content")

			// Output results
			if outputFormat == "json" {
				data, err := json.MarshalIndent(duplicates, "", "  ")
				if err != nil {
					return fmt.Errorf("marshaling JSON: %w", err)
				}
				fmt.Println(string(data))
			} else {
				output := formatDuplicatesText(duplicates)
				fmt.Print(output)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "Output format (text, json)")
	cmd.Flags().Float64Var(&minSimilarity, "similarity", 0.8, "Minimum similarity threshold (0.0-1.0)")

	return cmd
}

func newHealthCommand() *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "health [vault-path]",
		Short: "Check vault health",
		Long:  `Generate a comprehensive health report for your vault`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vaultPath := "."
			if len(args) > 0 {
				vaultPath = args[0]
			}

			// Load configuration
			cfg, err := loadConfig(cmd)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			// Scan vault files
			scanner := vault.NewScanner(vault.WithIgnorePatterns(cfg.Vault.IgnorePatterns))
			files, err := scanner.Walk(vaultPath)
			if err != nil {
				return fmt.Errorf("scanning vault: %w", err)
			}

			// Generate health report
			analyzer := analyzer.NewAnalyzer()
			stats := analyzer.GenerateStats(files)
			health := analyzer.GetHealthScore(stats)

			// Output results
			if outputFormat == "json" {
				data, err := json.MarshalIndent(health, "", "  ")
				if err != nil {
					return fmt.Errorf("marshaling JSON: %w", err)
				}
				fmt.Println(string(data))
			} else {
				output := formatHealthText(health)
				fmt.Print(output)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "Output format (text, json)")

	return cmd
}

func loadConfig(cmd *cobra.Command) (*config.Config, error) {
	configPath, _ := cmd.Flags().GetString("config")
	
	if configPath != "" {
		return config.LoadConfigFromFile(configPath)
	}
	
	return config.LoadConfigWithFallback(config.GetDefaultConfigPaths())
}

func formatStatsText(stats analyzer.VaultStats) string {
	output := fmt.Sprintf(`Vault Statistics
================

Files:
  Total files: %d
  Files with frontmatter: %d
  Files without frontmatter: %d

Content:
  Total size: %d bytes
  Average file size: %.1f bytes

Frontmatter Fields:
`, stats.TotalFiles, stats.FilesWithFrontmatter, stats.FilesWithoutFrontmatter,
		stats.TotalSize, float64(stats.TotalSize)/float64(stats.TotalFiles))

	for field, count := range stats.FieldPresence {
		percentage := float64(count) / float64(stats.TotalFiles) * 100
		output += fmt.Sprintf("  %s: %d files (%.1f%%)\n", field, count, percentage)
	}

	if len(stats.TagDistribution) > 0 {
		output += "\nTop Tags:\n"
		for tag, count := range stats.TagDistribution {
			output += fmt.Sprintf("  #%s: %d files\n", tag, count)
		}
	}

	return output
}

func formatDuplicatesText(duplicates []analyzer.Duplicate) string {
	if len(duplicates) == 0 {
		return "No duplicates found.\n"
	}

	output := fmt.Sprintf("Found %d duplicate groups:\n\n", len(duplicates))
	
	for i, dup := range duplicates {
		output += fmt.Sprintf("Group %d (field: %s, value: %v):\n", i+1, dup.Field, dup.Value)
		for _, file := range dup.Files {
			output += fmt.Sprintf("  - %s\n", file)
		}
		output += "\n"
	}

	return output
}

func formatHealthText(health analyzer.HealthScore) string {
	return fmt.Sprintf(`Vault Health Report
==================

Health Level: %s
Score: %.1f/100

Issues Found:
%s

Suggestions:
%s
`, health.Level, health.Score,
		formatIssues(health.Issues),
		formatSuggestions(health.Suggestions))
}

func formatIssues(issues []string) string {
	if len(issues) == 0 {
		return "  No issues found. Great job!"
	}
	
	output := ""
	for _, issue := range issues {
		output += fmt.Sprintf("  - %s\n", issue)
	}
	return output
}

func formatSuggestions(suggestions []string) string {
	if len(suggestions) == 0 {
		return "  No suggestions at this time."
	}
	
	output := ""
	for _, suggestion := range suggestions {
		output += fmt.Sprintf("  - %s\n", suggestion)
	}
	return output
}