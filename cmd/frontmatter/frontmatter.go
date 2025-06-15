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
	cmd.AddCommand(NewValidateCommand())
	cmd.AddCommand(NewCastCommand())
	cmd.AddCommand(NewSyncCommand())

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

// NewValidateCommand creates the frontmatter validate command
func NewValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate [path]",
		Short: "Validate frontmatter against rules",
		Long: `Validate that frontmatter in markdown files meets specified requirements.
Check for required fields, type validation, and other constraints.`,
		Args: cobra.ExactArgs(1),
		RunE: runValidate,
	}

	cmd.Flags().StringSlice("required", nil, "Required field names")
	cmd.Flags().StringSlice("type", nil, "Type rules in format field:type")
	cmd.Flags().StringSlice("ignore", []string{".obsidian/*", "*.tmp"}, "Ignore patterns")

	return cmd
}

func runValidate(cmd *cobra.Command, args []string) error {
	path := args[0]
	
	// Get flags
	required, _ := cmd.Flags().GetStringSlice("required")
	typeRules, _ := cmd.Flags().GetStringSlice("type")
	ignorePatterns, _ := cmd.Flags().GetStringSlice("ignore")
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")

	// Parse type rules
	types := make(map[string]string)
	for _, rule := range typeRules {
		parts := strings.Split(rule, ":")
		if len(parts) == 2 {
			types[parts[0]] = parts[1]
		}
	}

	// Scan files
	scanner := vault.NewScanner(vault.WithIgnorePatterns(ignorePatterns))
	files, err := scanner.Walk(path)
	if err != nil {
		return fmt.Errorf("scanning directory: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No markdown files found")
		return nil
	}

	// Validate files
	validator := processor.NewValidator(processor.ValidationRules{
		Required: required,
		Types:    types,
	})

	totalErrors := 0
	for _, file := range files {
		errors := validator.Validate(file)
		if len(errors) > 0 {
			totalErrors += len(errors)
			fmt.Printf("✗ %s:\n", file.RelativePath)
			for _, err := range errors {
				fmt.Printf("  - %s\n", err.Error())
			}
		} else if verbose {
			fmt.Printf("✓ %s: valid\n", file.RelativePath)
		}
	}

	if totalErrors > 0 {
		fmt.Printf("\nValidation failed: %d errors in %d files\n", totalErrors, len(files))
		return fmt.Errorf("validation failed")
	} else {
		fmt.Printf("\nValidation passed: %d files checked\n", len(files))
	}

	return nil
}

// NewCastCommand creates the frontmatter cast command
func NewCastCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cast [path]",
		Short: "Cast frontmatter fields to proper types",
		Long: `Convert frontmatter field values to appropriate types.
Supports auto-detection or explicit type specification.`,
		Args: cobra.ExactArgs(1),
		RunE: runCast,
	}

	cmd.Flags().StringSlice("field", nil, "Field names to cast")
	cmd.Flags().StringSlice("type", nil, "Target types for fields (field:type)")
	cmd.Flags().Bool("auto-detect", false, "Automatically detect and cast types")
	cmd.Flags().StringSlice("ignore", []string{".obsidian/*", "*.tmp"}, "Ignore patterns")

	return cmd
}

func runCast(cmd *cobra.Command, args []string) error {
	path := args[0]
	
	// Get flags
	fields, _ := cmd.Flags().GetStringSlice("field")
	typeSpecs, _ := cmd.Flags().GetStringSlice("type")
	autoDetect, _ := cmd.Flags().GetBool("auto-detect")
	ignorePatterns, _ := cmd.Flags().GetStringSlice("ignore")
	dryRun, _ := cmd.Root().PersistentFlags().GetBool("dry-run")
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")

	// Parse type specifications
	fieldTypes := make(map[string]string)
	for _, spec := range typeSpecs {
		parts := strings.Split(spec, ":")
		if len(parts) == 2 {
			fieldTypes[parts[0]] = parts[1]
		} else if len(parts) == 1 && len(fields) == 1 {
			// If only one field is specified and type doesn't contain ":", 
			// assume user wants to cast that field to this type
			fieldTypes[fields[0]] = parts[0]
		}
	}

	// Scan files
	scanner := vault.NewScanner(vault.WithIgnorePatterns(ignorePatterns))
	files, err := scanner.Walk(path)
	if err != nil {
		return fmt.Errorf("scanning directory: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No markdown files found")
		return nil
	}

	// Cast fields
	typeCaster := processor.NewTypeCaster()
	processedCount := 0

	for _, file := range files {
		fileModified := false

		// Process specified fields
		for _, field := range fields {
			if value, exists := file.GetField(field); exists {
				targetType := fieldTypes[field]
				if targetType == "" && autoDetect {
					targetType = typeCaster.AutoDetect(value)
				}
				
				if targetType != "" {
					if newValue, err := typeCaster.Cast(value, targetType); err == nil {
						file.SetField(field, newValue)
						fileModified = true
						if verbose {
							fmt.Printf("✓ %s: Cast '%s' from %T to %T\n", file.RelativePath, field, value, newValue)
						}
					} else if verbose {
						fmt.Printf("✗ %s: Failed to cast '%s': %v\n", file.RelativePath, field, err)
					}
				}
			}
		}

		// Auto-detect all fields if requested and no specific fields given
		if autoDetect && len(fields) == 0 {
			for field, value := range file.Frontmatter {
				if strVal, ok := value.(string); ok {
					detectedType := typeCaster.AutoDetect(strVal)
					if detectedType != "string" {
						if newValue, err := typeCaster.Cast(strVal, detectedType); err == nil {
							file.SetField(field, newValue)
							fileModified = true
							if verbose {
								fmt.Printf("✓ %s: Auto-cast '%s' to %s\n", file.RelativePath, field, detectedType)
							}
						}
					}
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

// NewSyncCommand creates the frontmatter sync command
func NewSyncCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync [path]",
		Short: "Sync frontmatter fields with file system data",
		Long: `Synchronize frontmatter fields with file system metadata.
Update fields based on filename patterns, modification times, or path structure.`,
		Args: cobra.ExactArgs(1),
		RunE: runSync,
	}

	cmd.Flags().StringSlice("field", nil, "Field names to sync")
	cmd.Flags().StringSlice("source", nil, "Data sources for fields (field:source)")
	cmd.Flags().StringSlice("ignore", []string{".obsidian/*", "*.tmp"}, "Ignore patterns")

	cmd.MarkFlagRequired("field")
	cmd.MarkFlagRequired("source")

	return cmd
}

func runSync(cmd *cobra.Command, args []string) error {
	path := args[0]
	
	// Get flags
	fields, _ := cmd.Flags().GetStringSlice("field")
	sources, _ := cmd.Flags().GetStringSlice("source")
	ignorePatterns, _ := cmd.Flags().GetStringSlice("ignore")
	dryRun, _ := cmd.Root().PersistentFlags().GetBool("dry-run")
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")

	if len(fields) != len(sources) {
		return fmt.Errorf("number of fields (%d) must match number of sources (%d)", len(fields), len(sources))
	}

	// Create field-source pairs
	fieldSources := make(map[string]string)
	for i, field := range fields {
		fieldSources[field] = sources[i]
	}

	// Scan files
	scanner := vault.NewScanner(vault.WithIgnorePatterns(ignorePatterns))
	files, err := scanner.Walk(path)
	if err != nil {
		return fmt.Errorf("scanning directory: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No markdown files found")
		return nil
	}

	// Sync fields
	sync := processor.NewFrontmatterSync()
	processedCount := 0

	for _, file := range files {
		fileModified := false

		for field, source := range fieldSources {
			if sync.SyncField(file, field, source) {
				fileModified = true
				if verbose {
					value, _ := file.GetField(field)
					fmt.Printf("✓ %s: Synced '%s' = %v\n", file.RelativePath, field, value)
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