package frontmatter

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/eoinhurrell/mdnotes/internal/processor"
	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// NewFrontmatterCommand creates the frontmatter command
func NewFrontmatterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "frontmatter",
		Short: "Manage frontmatter in markdown files",
		Long:  "Commands for managing YAML frontmatter in Obsidian notes",
	}

	cmd.AddCommand(NewEnsureCommand())

	return cmd
}

// NewEnsureCommand creates the frontmatter ensure command
func NewEnsureCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ensure [path]",
		Short: "Ensure frontmatter fields exist with default values",
		Long: `Ensure that specified frontmatter fields exist in all markdown files.
If a field is missing, it will be added with the provided default value.
Supports template variables like {{filename}} and {{current_date}}.`,
		Args: cobra.ExactArgs(1),
		RunE: runEnsure,
	}

	cmd.Flags().StringSlice("field", nil, "Field name to ensure (can be specified multiple times)")
	cmd.Flags().StringSlice("default", nil, "Default value for field (can be specified multiple times)")
	cmd.Flags().Bool("recursive", true, "Process subdirectories")
	cmd.Flags().StringSlice("ignore", []string{".obsidian/*", "*.tmp"}, "Ignore patterns")

	cmd.MarkFlagRequired("field")
	cmd.MarkFlagRequired("default")

	return cmd
}

func runEnsure(cmd *cobra.Command, args []string) error {
	path := args[0]
	
	// Get flags
	fields, _ := cmd.Flags().GetStringSlice("field")
	defaults, _ := cmd.Flags().GetStringSlice("default")
	ignorePatterns, _ := cmd.Flags().GetStringSlice("ignore")
	dryRun, _ := cmd.Root().PersistentFlags().GetBool("dry-run")
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")

	if len(fields) != len(defaults) {
		return fmt.Errorf("number of fields (%d) must match number of defaults (%d)", len(fields), len(defaults))
	}

	// Create field-default pairs
	fieldDefaults := make(map[string]interface{})
	for i, field := range fields {
		fieldDefaults[field] = defaults[i]
	}

	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path error: %w", err)
	}

	var files []*vault.VaultFile

	if info.IsDir() {
		// Scan directory
		scanner := vault.NewScanner(vault.WithIgnorePatterns(ignorePatterns))
		files, err = scanner.Walk(path)
		if err != nil {
			return fmt.Errorf("scanning directory: %w", err)
		}
	} else {
		// Single file
		if !strings.HasSuffix(path, ".md") {
			return fmt.Errorf("file must have .md extension")
		}
		
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}

		vf := &vault.VaultFile{Path: path}
		if err := vf.Parse(content); err != nil {
			return fmt.Errorf("parsing file: %w", err)
		}
		files = []*vault.VaultFile{vf}
	}

	if len(files) == 0 {
		fmt.Println("No markdown files found")
		return nil
	}

	// Process files
	processor := processor.NewFrontmatterProcessor()
	
	processedCount := 0
	for _, file := range files {
		fileModified := false

		for field, defaultValue := range fieldDefaults {
			if processor.Ensure(file, field, defaultValue) {
				fileModified = true
				if verbose {
					fmt.Printf("✓ %s: Added field '%s' = %v\n", file.RelativePath, field, defaultValue)
				}
			}
		}

		if fileModified {
			processedCount++

			if !dryRun {
				// Write the file back
				content, err := file.Serialize()
				if err != nil {
					return fmt.Errorf("serializing %s: %w", file.Path, err)
				}

				if err := os.WriteFile(file.Path, content, 0644); err != nil {
					return fmt.Errorf("writing %s: %w", file.Path, err)
				}
			}

			if !verbose {
				fmt.Printf("✓ Processed: %s\n", file.RelativePath)
			}
		} else if verbose {
			fmt.Printf("- Skipped: %s (no changes needed)\n", file.RelativePath)
		}
	}

	// Summary
	if dryRun {
		fmt.Printf("\nDry run completed. Would modify %d files.\n", processedCount)
	} else {
		fmt.Printf("\nCompleted. Modified %d files.\n", processedCount)
	}

	return nil
}