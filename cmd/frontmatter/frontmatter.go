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
	cmd.AddCommand(NewSetCommand())
	cmd.AddCommand(NewCastCommand())
	cmd.AddCommand(NewSyncCommand())
	cmd.AddCommand(NewCheckCommand())

	return cmd
}

// NewEnsureCommand creates the frontmatter ensure command
func NewEnsureCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ensure [path]",
		Short: "Ensure frontmatter fields exist with default values",
		Long: `Ensure that specified frontmatter fields exist in all markdown files.
If a field is missing, it will be added with the provided default value.
Supports template variables like {{filename}} and {{current_date}}.

Special default values:
  null - Sets the field to null (not the string "null")`,
		Args: cobra.ExactArgs(1),
		RunE: runEnsure,
	}

	cmd.Flags().StringSlice("field", nil, "Field name to ensure (can be specified multiple times)")
	cmd.Flags().StringSlice("default", nil, "Default value for field (can be specified multiple times)")
	cmd.Flags().StringSlice("type", nil, "Type rules in format field:type (optional, for type checking)")
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
	typeRules, _ := cmd.Flags().GetStringSlice("type")
	ignorePatterns, _ := cmd.Flags().GetStringSlice("ignore")
	dryRun, _ := cmd.Root().PersistentFlags().GetBool("dry-run")
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")

	if len(fields) != len(defaults) {
		return fmt.Errorf("number of fields (%d) must match number of defaults (%d)", len(fields), len(defaults))
	}

	// Parse type rules
	types := make(map[string]string)
	for _, rule := range typeRules {
		parts := strings.Split(rule, ":")
		if len(parts) == 2 {
			types[parts[0]] = parts[1]
		}
	}

	// Create field-default pairs with null value support
	fieldDefaults := make(map[string]interface{})
	for i, field := range fields {
		defaultValue := defaults[i]
		// Handle special null value
		if defaultValue == "null" {
			fieldDefaults[field] = nil
		} else {
			fieldDefaults[field] = defaultValue
		}
	}

	// Create processors
	frontmatterProcessor := processor.NewFrontmatterProcessor()
	typeCaster := processor.NewTypeCaster()
	validator := processor.NewValidator(processor.ValidationRules{
		Types: types,
	})
	
	// Setup file processor
	fileProcessor := &processor.FileProcessor{
		DryRun:         dryRun,
		Verbose:        verbose,
		IgnorePatterns: ignorePatterns,
		ProcessFile: func(file *vault.VaultFile) (bool, error) {
			fileModified := false
			
			// Phase 1: Ensure fields exist with default values
			for field, defaultValue := range fieldDefaults {
				if frontmatterProcessor.Ensure(file, field, defaultValue) {
					fileModified = true
					if verbose {
						fmt.Printf("✓ %s: Added field '%s' = %v\n", file.RelativePath, field, defaultValue)
					}
				}
			}
			
			// Phase 2: Check and fix types
			for field, expectedType := range types {
				if value, exists := file.GetField(field); exists {
					// Check if field has correct type
					errors := validator.Validate(file)
					for _, err := range errors {
						if strings.Contains(err.Error(), fmt.Sprintf("field '%s' must be of type %s", field, expectedType)) {
							// Try to cast the field to the correct type
							if newValue, castErr := typeCaster.Cast(value, expectedType); castErr == nil {
								file.SetField(field, newValue)
								fileModified = true
								if verbose {
									fmt.Printf("✓ %s: Fixed type for '%s' (%T -> %T)\n", file.RelativePath, field, value, newValue)
								}
							} else {
								// Non-halting error: report but continue
								fmt.Printf("✗ %s: Field '%s' has wrong type (expected %s, got %T) and cannot be cast: %v\n", 
									file.RelativePath, field, expectedType, value, castErr)
							}
							break
						}
					}
				}
			}
			
			return fileModified, nil
		},
		OnFileProcessed: func(file *vault.VaultFile, modified bool) {
			if modified && !verbose {
				fmt.Printf("✓ Processed: %s\n", file.RelativePath)
			} else if !modified && verbose {
				fmt.Printf("- Skipped: %s (no changes needed)\n", file.RelativePath)
			}
		},
	}

	// Process files
	result, err := fileProcessor.ProcessPath(path)
	if err != nil {
		return err
	}

	// Print summary
	fileProcessor.PrintSummary(result)

	return nil
}

// NewSetCommand creates the frontmatter set command
func NewSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set [path]",
		Short: "Set frontmatter fields to specific values",
		Long: `Set frontmatter fields to specific values in all markdown files.
Unlike 'ensure', this command always updates the field to the specified value,
even if it already exists. Supports template variables and type casting.

Special values:
  null - Sets the field to null (not the string "null")`,
		Args: cobra.ExactArgs(1),
		RunE: runSet,
	}

	cmd.Flags().StringSlice("field", nil, "Field name to set (can be specified multiple times)")
	cmd.Flags().StringSlice("value", nil, "Value for field (can be specified multiple times)")
	cmd.Flags().StringSlice("type", nil, "Type rules in format field:type (optional, for type casting)")
	cmd.Flags().Bool("recursive", true, "Process subdirectories")
	cmd.Flags().StringSlice("ignore", []string{".obsidian/*", "*.tmp"}, "Ignore patterns")

	cmd.MarkFlagRequired("field")
	cmd.MarkFlagRequired("value")

	return cmd
}

func runSet(cmd *cobra.Command, args []string) error {
	path := args[0]
	
	// Get flags
	fields, _ := cmd.Flags().GetStringSlice("field")
	values, _ := cmd.Flags().GetStringSlice("value")
	typeRules, _ := cmd.Flags().GetStringSlice("type")
	ignorePatterns, _ := cmd.Flags().GetStringSlice("ignore")
	dryRun, _ := cmd.Root().PersistentFlags().GetBool("dry-run")
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")

	if len(fields) != len(values) {
		return fmt.Errorf("number of fields (%d) must match number of values (%d)", len(fields), len(values))
	}

	// Parse type rules
	types := make(map[string]string)
	for _, rule := range typeRules {
		parts := strings.Split(rule, ":")
		if len(parts) == 2 {
			types[parts[0]] = parts[1]
		}
	}

	// Create field-value pairs with null value support
	fieldValues := make(map[string]interface{})
	for i, field := range fields {
		value := values[i]
		// Handle special null value
		if value == "null" {
			fieldValues[field] = nil
		} else {
			fieldValues[field] = value
		}
	}

	// Create processors
	typeCaster := processor.NewTypeCaster()
	
	// Setup file processor
	fileProcessor := &processor.FileProcessor{
		DryRun:         dryRun,
		Verbose:        verbose,
		IgnorePatterns: ignorePatterns,
		ProcessFile: func(file *vault.VaultFile) (bool, error) {
			fileModified := false
			
			for field, value := range fieldValues {
				// Get current value for comparison
				currentValue, exists := file.GetField(field)
				
				// Set the new value
				processedValue := value
				
				// Apply type casting if specified
				if expectedType, hasType := types[field]; hasType && value != nil {
					if castValue, err := typeCaster.Cast(value, expectedType); err == nil {
						processedValue = castValue
						if verbose {
							fmt.Printf("✓ %s: Cast value for '%s' to %s\n", file.RelativePath, field, expectedType)
						}
					} else {
						// Non-halting error: report but continue with original value
						fmt.Printf("✗ %s: Cannot cast value for '%s' to %s: %v (using original value)\n", 
							file.RelativePath, field, expectedType, err)
					}
				}
				
				// Set the field value
				file.SetField(field, processedValue)
				fileModified = true
				
				if verbose {
					if exists {
						fmt.Printf("✓ %s: Updated field '%s': %v -> %v\n", file.RelativePath, field, currentValue, processedValue)
					} else {
						fmt.Printf("✓ %s: Set field '%s' = %v\n", file.RelativePath, field, processedValue)
					}
				}
			}
			
			return fileModified, nil
		},
		OnFileProcessed: func(file *vault.VaultFile, modified bool) {
			if modified && !verbose {
				fmt.Printf("✓ Processed: %s\n", file.RelativePath)
			} else if !modified && verbose {
				fmt.Printf("- Skipped: %s (no changes needed)\n", file.RelativePath)
			}
		},
	}

	// Process files
	result, err := fileProcessor.ProcessPath(path)
	if err != nil {
		return err
	}

	// Print summary
	fileProcessor.PrintSummary(result)

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

	// Create processor
	typeCaster := processor.NewTypeCaster()
	
	// Setup file processor
	fileProcessor := &processor.FileProcessor{
		DryRun:         dryRun,
		Verbose:        verbose,
		IgnorePatterns: ignorePatterns,
		ProcessFile: func(file *vault.VaultFile) (bool, error) {
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
					detectedType := typeCaster.AutoDetect(value)
					if detectedType != "string" {
						if newValue, err := typeCaster.Cast(value, detectedType); err == nil {
							// Only modify if the cast actually changed the value type
							if fmt.Sprintf("%T", newValue) != fmt.Sprintf("%T", value) {
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

			return fileModified, nil
		},
		OnFileProcessed: func(file *vault.VaultFile, modified bool) {
			if modified && !verbose {
				fmt.Printf("✓ Processed: %s\n", file.RelativePath)
			}
		},
	}

	// Process files
	result, err := fileProcessor.ProcessPath(path)
	if err != nil {
		return err
	}

	// Print summary
	fileProcessor.PrintSummary(result)

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

	// Create processor
	sync := processor.NewFrontmatterSync()
	
	// Setup file processor
	fileProcessor := &processor.FileProcessor{
		DryRun:         dryRun,
		Verbose:        verbose,
		IgnorePatterns: ignorePatterns,
		ProcessFile: func(file *vault.VaultFile) (bool, error) {
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
			return fileModified, nil
		},
		OnFileProcessed: func(file *vault.VaultFile, modified bool) {
			if modified && !verbose {
				fmt.Printf("✓ Processed: %s\n", file.RelativePath)
			} else if !modified && verbose {
				fmt.Printf("- Skipped: %s (no changes needed)\n", file.RelativePath)
			}
		},
	}

	// Process files
	result, err := fileProcessor.ProcessPath(path)
	if err != nil {
		return err
	}

	// Print summary
	fileProcessor.PrintSummary(result)

	return nil
}

// NewCheckCommand creates the frontmatter check command
func NewCheckCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check [path]",
		Short: "Check frontmatter for parsing issues and validate against rules",
		Long: `Check all markdown files for frontmatter parsing issues and validate against rules.
This command identifies files with malformed YAML frontmatter and can also validate
that frontmatter meets specified requirements like required fields and type constraints.`,
		Args: cobra.ExactArgs(1),
		RunE: runCheck,
	}

	cmd.Flags().StringSlice("required", nil, "Required field names")
	cmd.Flags().StringSlice("type", nil, "Type rules in format field:type")
	cmd.Flags().StringSlice("ignore", []string{".obsidian/*", "*.tmp"}, "Ignore patterns")
	cmd.Flags().Bool("parsing-only", false, "Only check for YAML parsing issues, skip validation rules")

	return cmd
}

func runCheck(cmd *cobra.Command, args []string) error {
	path := args[0]
	
	// Get flags
	required, _ := cmd.Flags().GetStringSlice("required")
	typeRules, _ := cmd.Flags().GetStringSlice("type")
	ignorePatterns, _ := cmd.Flags().GetStringSlice("ignore")
	parsingOnly, _ := cmd.Flags().GetBool("parsing-only")
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")

	// Parse type rules
	types := make(map[string]string)
	for _, rule := range typeRules {
		parts := strings.Split(rule, ":")
		if len(parts) == 2 {
			types[parts[0]] = parts[1]
		}
	}

	// Scan files using the proper scanner with ignore patterns
	scanner := vault.NewScanner(vault.WithIgnorePatterns(ignorePatterns))
	files, err := scanner.Walk(path)
	if err != nil {
		return fmt.Errorf("scanning directory: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No markdown files found")
		return nil
	}

	// Phase 1: Check for parsing issues
	var parsingIssues []string
	var validFiles []*vault.VaultFile
	
	for _, file := range files {
		// Files from scanner are already parsed, but check if there were errors
		if file.Frontmatter == nil {
			// Try to parse again to get the specific error
			content, readErr := os.ReadFile(file.Path)
			if readErr != nil {
				parsingIssues = append(parsingIssues, fmt.Sprintf("✗ %s: Failed to read file - %v", file.RelativePath, readErr))
				continue
			}
			
			parseErr := file.Parse(content)
			if parseErr != nil {
				parsingIssues = append(parsingIssues, fmt.Sprintf("✗ %s: %v", file.RelativePath, parseErr))
				if verbose {
					fmt.Printf("✗ %s: %v\n", file.RelativePath, parseErr)
				}
				continue
			}
		}
		
		validFiles = append(validFiles, file)
		if verbose {
			fmt.Printf("✓ %s: Parsing OK\n", file.RelativePath)
		}
	}

	// Report parsing issues
	if len(parsingIssues) > 0 {
		if !verbose {
			for _, issue := range parsingIssues {
				fmt.Println(issue)
			}
		}
		fmt.Printf("\nFound %d files with parsing issues out of %d total files\n", len(parsingIssues), len(files))
		
		// If only checking parsing, return here
		if parsingOnly {
			return fmt.Errorf("frontmatter parsing issues found")
		}
	}

	// Phase 2: Validate against rules (if not parsing-only and rules are specified)
	if !parsingOnly && (len(required) > 0 || len(types) > 0) {
		validator := processor.NewValidator(processor.ValidationRules{
			Required: required,
			Types:    types,
		})

		totalValidationErrors := 0
		for _, file := range validFiles {
			errors := validator.Validate(file)
			if len(errors) > 0 {
				totalValidationErrors += len(errors)
				fmt.Printf("✗ %s (validation):\n", file.RelativePath)
				for _, err := range errors {
					fmt.Printf("  - %s\n", err.Error())
				}
			} else if verbose {
				fmt.Printf("✓ %s: Validation OK\n", file.RelativePath)
			}
		}

		if totalValidationErrors > 0 {
			fmt.Printf("\nValidation failed: %d validation errors in %d files\n", totalValidationErrors, len(validFiles))
			if len(parsingIssues) > 0 {
				return fmt.Errorf("found both parsing issues and validation errors")
			}
			return fmt.Errorf("validation failed")
		} else {
			fmt.Printf("\nValidation passed: %d files validated\n", len(validFiles))
		}
	}

	// Final summary
	if len(parsingIssues) == 0 {
		if parsingOnly || (len(required) == 0 && len(types) == 0) {
			fmt.Printf("✓ All %d files have valid frontmatter\n", len(files))
		}
	} else {
		return fmt.Errorf("frontmatter issues found")
	}
	
	return nil
}