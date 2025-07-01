package frontmatter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/eoinhurrell/mdnotes/internal/config"
	"github.com/eoinhurrell/mdnotes/internal/downloader"
	"github.com/eoinhurrell/mdnotes/internal/processor"
	"github.com/eoinhurrell/mdnotes/internal/query"
	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// NewFrontmatterCommand creates the frontmatter command
func NewFrontmatterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "frontmatter",
		Aliases: []string{"fm"},
		Short:   "Manage frontmatter in markdown files",
		Long:    "Commands for managing YAML frontmatter in Obsidian notes",
	}

	cmd.AddCommand(NewEnsureCommand())
	cmd.AddCommand(NewSetCommand())
	cmd.AddCommand(NewCastCommand())
	cmd.AddCommand(NewSyncCommand())
	cmd.AddCommand(NewCheckCommand())
	cmd.AddCommand(NewQueryCommand())
	cmd.AddCommand(NewDownloadCommand())

	return cmd
}

// NewEnsureCommand creates the frontmatter ensure command
func NewEnsureCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ensure [path]",
		Aliases: []string{"e"},
		Short:   "Ensure frontmatter fields exist with default values",
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
	quiet, _ := cmd.Root().PersistentFlags().GetBool("quiet")

	// Override verbose if quiet is specified
	if quiet {
		verbose = false
	}

	if len(fields) != len(defaults) {
		return fmt.Errorf("number of fields (%d) must match number of defaults (%d)", len(fields), len(defaults))
	}

	// Parse type rules
	types := make(map[string]string)
	for _, rule := range typeRules {
		parts := strings.Split(rule, ":")
		if len(parts) == 2 {
			// Standard format: field:type
			types[parts[0]] = parts[1]
		} else if len(parts) == 1 && len(fields) == 1 {
			// Implicit format: just type when there's only one field
			types[fields[0]] = parts[0]
		} else if len(parts) == 1 {
			return fmt.Errorf("type %s specified but multiple fields provided - use field:type format", rule)
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

			// Implicit array detection: if default value has bracket notation and no explicit type is set
			if _, hasExplicitType := types[field]; !hasExplicitType {
				trimmed := strings.TrimSpace(defaultValue)
				if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
					types[field] = "array"
					if verbose {
						fmt.Printf("Auto-detected array type for field '%s' due to bracket notation\n", field)
					}
				}
			}
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
		Quiet:          quiet,
		IgnorePatterns: ignorePatterns,
		ProcessFile: func(file *vault.VaultFile) (bool, error) {
			fileModified := false

			// Phase 1: Ensure fields exist with default values
			for field, defaultValue := range fieldDefaults {
				if frontmatterProcessor.Ensure(file, field, defaultValue) {
					fileModified = true
					if verbose {
						fmt.Printf("Examining: %s - Added field '%s' = %v\n", file.RelativePath, field, defaultValue)
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
									fmt.Printf("Examining: %s - Fixed type for '%s' (%T -> %T)\n", file.RelativePath, field, value, newValue)
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
			if modified && !verbose && !quiet {
				fmt.Printf("✓ Processed: %s\n", file.RelativePath)
			} else if !modified && verbose {
				fmt.Printf("Examining: %s - No changes needed\n", file.RelativePath)
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
		Use:     "set [path]",
		Aliases: []string{"s"},
		Short:   "Set frontmatter fields to specific values",
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
	quiet, _ := cmd.Root().PersistentFlags().GetBool("quiet")

	// Override verbose if quiet is specified
	if quiet {
		verbose = false
	}

	if len(fields) != len(values) {
		return fmt.Errorf("number of fields (%d) must match number of values (%d)", len(fields), len(values))
	}

	// Parse type rules
	types := make(map[string]string)
	for _, rule := range typeRules {
		parts := strings.Split(rule, ":")
		if len(parts) == 2 {
			// Standard format: field:type
			types[parts[0]] = parts[1]
		} else if len(parts) == 1 && len(fields) == 1 {
			// Implicit format: just type when there's only one field
			types[fields[0]] = parts[0]
		} else if len(parts) == 1 {
			return fmt.Errorf("type %s specified but multiple fields provided - use field:type format", rule)
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

			// Implicit array detection: if value has bracket notation and no explicit type is set
			if _, hasExplicitType := types[field]; !hasExplicitType {
				trimmed := strings.TrimSpace(value)
				if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
					types[field] = "array"
					if verbose {
						fmt.Printf("Auto-detected array type for field '%s' due to bracket notation\n", field)
					}
				}
			}
		}
	}

	// Create processors
	typeCaster := processor.NewTypeCaster()

	// Setup file processor
	fileProcessor := &processor.FileProcessor{
		DryRun:         dryRun,
		Verbose:        verbose,
		Quiet:          quiet,
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
							fmt.Printf("Examining: %s - Cast value for '%s' to %s\n", file.RelativePath, field, expectedType)
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
						fmt.Printf("Examining: %s - Updated field '%s': %v -> %v\n", file.RelativePath, field, currentValue, processedValue)
					} else {
						fmt.Printf("Examining: %s - Set field '%s' = %v\n", file.RelativePath, field, processedValue)
					}
				}
			}

			return fileModified, nil
		},
		OnFileProcessed: func(file *vault.VaultFile, modified bool) {
			if modified && !verbose && !quiet {
				fmt.Printf("✓ Processed: %s\n", file.RelativePath)
			} else if !modified && verbose {
				fmt.Printf("Examining: %s - No changes needed\n", file.RelativePath)
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
		Use:     "cast [path]",
		Aliases: []string{"c"},
		Short:   "Cast frontmatter fields to proper types",
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
	quiet, _ := cmd.Root().PersistentFlags().GetBool("quiet")

	// Override verbose if quiet is specified
	if quiet {
		verbose = false
	}

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
		Quiet:          quiet,
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
								fmt.Printf("Examining: %s - Cast '%s' from %T to %T\n", file.RelativePath, field, value, newValue)
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
									fmt.Printf("Examining: %s - Auto-cast '%s' to %s\n", file.RelativePath, field, detectedType)
								}
							}
						}
					}
				}
			}

			return fileModified, nil
		},
		OnFileProcessed: func(file *vault.VaultFile, modified bool) {
			if modified && !verbose && !quiet {
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
		Use:     "sync [path]",
		Aliases: []string{"sy"},
		Short:   "Sync frontmatter fields with file system data",
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
	quiet, _ := cmd.Root().PersistentFlags().GetBool("quiet")

	// Override verbose if quiet is specified
	if quiet {
		verbose = false
	}

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
		Quiet:          quiet,
		IgnorePatterns: ignorePatterns,
		ProcessFile: func(file *vault.VaultFile) (bool, error) {
			fileModified := false
			for field, source := range fieldSources {
				if sync.SyncField(file, field, source) {
					fileModified = true
					if verbose {
						value, _ := file.GetField(field)
						fmt.Printf("Examining: %s - Synced '%s' = %v\n", file.RelativePath, field, value)
					}
				}
			}
			return fileModified, nil
		},
		OnFileProcessed: func(file *vault.VaultFile, modified bool) {
			if modified && !verbose && !quiet {
				fmt.Printf("✓ Processed: %s\n", file.RelativePath)
			} else if !modified && verbose {
				fmt.Printf("Examining: %s - No changes needed\n", file.RelativePath)
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
		Use:     "check [path]",
		Aliases: []string{"ch"},
		Short:   "Check frontmatter for parsing issues and validate against rules",
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
	quiet, _ := cmd.Root().PersistentFlags().GetBool("quiet")

	// Override verbose if quiet is specified
	if quiet {
		verbose = false
	}

	// Parse type rules
	types := make(map[string]string)
	for _, rule := range typeRules {
		parts := strings.Split(rule, ":")
		if len(parts) == 2 {
			// Standard format: field:type
			types[parts[0]] = parts[1]
		} else if len(parts) == 1 {
			return fmt.Errorf("type %s specified but no field name provided - use field:type format", rule)
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
			fmt.Printf("Examining: %s - Parsing OK\n", file.RelativePath)
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
				fmt.Printf("Examining: %s - Validation OK\n", file.RelativePath)
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

// NewDownloadCommand creates the frontmatter download command
func NewDownloadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "download [path]",
		Aliases: []string{"d"},
		Short:   "Download web resources from frontmatter fields",
		Long: `Download web resources referenced in frontmatter fields and convert them to local references.

The command:
1. Scans frontmatter fields for HTTP/HTTPS URLs
2. Downloads the resources to the configured attachments directory
3. Renames the original field to <field>-original
4. Replaces the field value with a wiki link to the downloaded file

Example:
  # Download all web resources in frontmatter
  mdnotes frontmatter download /vault/path
  
  # Download only specific fields
  mdnotes frontmatter download --field cover --field image /vault/path
  
  # Preview what would be downloaded
  mdnotes frontmatter download --dry-run /vault/path`,
		Args: cobra.ExactArgs(1),
		RunE: runDownload,
	}

	cmd.Flags().StringSlice("field", nil, "Only download specific fields (default: all URL fields)")
	cmd.Flags().StringSlice("ignore", []string{".obsidian/*", "*.tmp"}, "Ignore patterns")
	cmd.Flags().String("config", "", "Config file path")

	return cmd
}

func runDownload(cmd *cobra.Command, args []string) error {
	path := args[0]

	// Get flags
	targetFields, _ := cmd.Flags().GetStringSlice("field")
	ignorePatterns, _ := cmd.Flags().GetStringSlice("ignore")
	configPath, _ := cmd.Flags().GetString("config")
	dryRun, _ := cmd.Root().PersistentFlags().GetBool("dry-run")
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
	quiet, _ := cmd.Root().PersistentFlags().GetBool("quiet")

	// Override verbose if quiet is specified
	if quiet {
		verbose = false
	}

	// Load configuration
	cfg, err := loadConfigWithPath(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Create downloader
	downloader, err := newDownloaderFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("creating downloader: %w", err)
	}

	// Load files (handle both files and directories)
	files, err := loadFilesForProcessing(path, ignorePatterns)
	if err != nil {
		return fmt.Errorf("loading files: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No markdown files found")
		return nil
	}

	if verbose {
		fmt.Printf("Scanned %d files\n", len(files))
	}

	// Process files
	totalDownloads := 0
	totalFiles := 0
	errors := []error{}

	for _, file := range files {
		downloads, fileErrors := processFileDownloads(file, downloader, targetFields, dryRun, verbose)
		if len(downloads) > 0 {
			totalFiles++
			totalDownloads += len(downloads)

			// Save file if not dry run and has modifications
			if !dryRun && len(downloads) > 0 {
				content, err := file.Serialize()
				if err != nil {
					errors = append(errors, fmt.Errorf("serializing %s: %w", file.RelativePath, err))
					continue
				}

				if err := os.WriteFile(file.Path, content, 0644); err != nil {
					errors = append(errors, fmt.Errorf("saving %s: %w", file.RelativePath, err))
					continue
				}
			}
		}

		errors = append(errors, fileErrors...)
	}

	// Print summary
	if len(errors) > 0 {
		for _, err := range errors {
			fmt.Printf("✗ %v\n", err)
		}
	}

	if dryRun {
		fmt.Printf("\nDry run completed. Would download %d resources from %d files.\n", totalDownloads, totalFiles)
	} else {
		fmt.Printf("\nCompleted. Downloaded %d resources from %d files.\n", totalDownloads, totalFiles)
	}

	if len(errors) > 0 {
		return fmt.Errorf("%d errors occurred during processing", len(errors))
	}

	return nil
}

// Helper functions for download command

func loadConfigWithPath(configPath string) (*config.Config, error) {
	if configPath != "" {
		return config.LoadConfigFromFile(configPath)
	}

	// Use default config search paths
	paths := config.GetDefaultConfigPaths()
	return config.LoadConfigWithFallback(paths)
}

func newDownloaderFromConfig(cfg *config.Config) (*downloader.Downloader, error) {
	return downloader.NewDownloader(cfg.Downloads)
}

func processFileDownloads(file *vault.VaultFile, dl *downloader.Downloader, targetFields []string, dryRun, verbose bool) ([]string, []error) {
	var downloads []string
	var errors []error

	// Get base filename for generating download names
	baseFilename := strings.TrimSuffix(filepath.Base(file.RelativePath), filepath.Ext(file.RelativePath))

	for field, value := range file.Frontmatter {
		// Skip if targeting specific fields and this isn't one of them
		if len(targetFields) > 0 {
			found := false
			for _, target := range targetFields {
				if field == target {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Check if value is a string URL
		urlStr, ok := value.(string)
		if !ok {
			continue
		}

		if !downloader.IsValidURL(urlStr) {
			continue
		}

		// Found a downloadable URL
		if dryRun {
			fmt.Printf("Would download: %s.%s = %s\n", file.RelativePath, field, urlStr)
			downloads = append(downloads, field)
			continue
		}

		if verbose {
			fmt.Printf("Downloading: %s.%s = %s\n", file.RelativePath, field, urlStr)
		}

		// Download the resource
		ctx := context.Background()
		result, err := dl.DownloadResource(ctx, urlStr, baseFilename, field)
		if err != nil {
			errors = append(errors, fmt.Errorf("%s.%s: %w", file.RelativePath, field, err))
			continue
		}

		if verbose {
			if result.Skipped {
				fmt.Printf("⚠ Skipped: %s (file already exists) -> %s\n", urlStr, result.LocalPath)
			} else {
				fmt.Printf("✓ Downloaded: %s (%d bytes) -> %s\n", urlStr, result.Size, result.LocalPath)
			}
		}

		// Update frontmatter
		originalField := field + "-original"
		file.Frontmatter[originalField] = urlStr
		file.Frontmatter[field] = downloader.GenerateWikiLink(result.LocalPath)

		downloads = append(downloads, field)
	}

	return downloads, errors
}

// loadFilesForProcessing loads files from the given path, handling both files and directories
func loadFilesForProcessing(path string, ignorePatterns []string) ([]*vault.VaultFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("path error: %w", err)
	}

	if info.IsDir() {
		// Use scanner for directories
		scanner := vault.NewScanner(vault.WithIgnorePatterns(ignorePatterns))
		return scanner.Walk(path)
	} else {
		// Handle single file
		if !strings.HasSuffix(path, ".md") {
			return nil, fmt.Errorf("file must have .md extension")
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading file: %w", err)
		}

		vf := &vault.VaultFile{
			Path:         path,
			RelativePath: filepath.Base(path),
			Modified:     info.ModTime(),
		}

		if err := vf.Parse(content); err != nil {
			return nil, fmt.Errorf("parsing file: %w", err)
		}

		return []*vault.VaultFile{vf}, nil
	}
}

// NewQueryCommand creates the frontmatter query command
func NewQueryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "query [path]",
		Aliases: []string{"q"},
		Short:   "Query and filter frontmatter fields",
		Long: `Query and filter markdown files based on frontmatter criteria.
Find files that match specific conditions, are missing fields, or have duplicate values.

Enhanced Query Language:
  Simple comparisons:
    --where "status = 'draft'"           # Exact match
    --where "priority > 3"               # Numeric comparison  
    --where "priority >= 5"              # Greater than or equal
    --where "status != 'done'"           # Not equal
    
  Contains operator:
    --where "tags contains 'urgent'"     # Array/string contains
    --where "title contains 'project'"   # Case-insensitive search
    
  Date comparisons:
    --where "created after '2024-01-01'"     # Date after
    --where "modified before '2024-12-01'"   # Date before  
    --where "updated within '7 days'"        # Within ±7 days from today
    --where "edited within '3 weeks'"        # Within ±3 weeks from today
    --where "accessed within '2 hours'"      # Supports minutes, hours, days, weeks, months, years
    
  Logical operators:
    --where "priority > 3 AND status != 'done'"           # Both conditions
    --where "tags contains 'work' OR tags contains 'project'"  # Either condition
    --where "(priority > 5 OR status = 'urgent') AND tags contains 'active'"

Other query types:
  # Find files missing specific fields
  mdnotes fm query . --missing "created"
  
  # Find files with duplicate field values
  mdnotes fm query . --duplicates "title"
  
  # Select specific fields and format output
  mdnotes fm query . --field "title,tags,status" --format table
  
  # Just count matching files
  mdnotes fm query . --where "status = 'draft'" --count
  
  # Auto-fix missing fields
  mdnotes fm query . --missing "created" --fix-with "{{current_date}}"
  
Piping support:
  # Output paths for piping to other commands
  mdnotes fm query . --where "status = 'draft'" --paths-only
  
  # Pipe to other mdnotes commands
  mdnotes fm query . --where "status = 'draft'" --paths-only | xargs -I {} mdnotes fm upsert --field status --default "published" "{}"`,
		Args: cobra.ExactArgs(1),
		RunE: runQuery,
	}

	// Query criteria flags
	cmd.Flags().String("where", "", "Filter expression (e.g., \"status = 'draft'\", \"priority > 3\")")
	cmd.Flags().String("missing", "", "Find files missing this field")
	cmd.Flags().String("duplicates", "", "Find files with duplicate values for this field")

	// Output control flags (consistent with other commands)
	cmd.Flags().StringSlice("field", nil, "Select specific fields to display (comma-separated)")
	cmd.Flags().String("format", "table", "Output format: table, json, csv, yaml, paths")
	cmd.Flags().Bool("count", false, "Show only the count of matching files")
	cmd.Flags().Bool("paths-only", false, "Output only file paths (for piping to other commands)")
	cmd.Flags().StringSlice("ignore", []string{".obsidian/*", "*.tmp"}, "Ignore patterns")

	// Auto-fix functionality (matches ensure command pattern)
	cmd.Flags().String("fix-with", "", "Auto-fix missing fields with this value (only with --missing)")

	return cmd
}

func runQuery(cmd *cobra.Command, args []string) error {
	path := args[0]

	// Get flags
	whereExpr, _ := cmd.Flags().GetString("where")
	missingField, _ := cmd.Flags().GetString("missing")
	duplicatesField, _ := cmd.Flags().GetString("duplicates")
	fields, _ := cmd.Flags().GetStringSlice("field")
	format, _ := cmd.Flags().GetString("format")
	count, _ := cmd.Flags().GetBool("count")
	pathsOnly, _ := cmd.Flags().GetBool("paths-only")
	ignorePatterns, _ := cmd.Flags().GetStringSlice("ignore")
	fixWith, _ := cmd.Flags().GetString("fix-with")
	dryRun, _ := cmd.Root().PersistentFlags().GetBool("dry-run")
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
	quiet, _ := cmd.Root().PersistentFlags().GetBool("quiet")

	// Validate flag combinations
	criteriaCount := 0
	if whereExpr != "" {
		criteriaCount++
	}
	if missingField != "" {
		criteriaCount++
	}
	if duplicatesField != "" {
		criteriaCount++
	}

	if criteriaCount == 0 {
		return fmt.Errorf("must specify one of: --where, --missing, or --duplicates")
	}
	if criteriaCount > 1 {
		return fmt.Errorf("can only specify one of: --where, --missing, or --duplicates")
	}

	if fixWith != "" && missingField == "" {
		return fmt.Errorf("--fix-with can only be used with --missing")
	}

	if pathsOnly && format != "table" {
		return fmt.Errorf("--paths-only cannot be used with --format (use --paths-only OR --format)")
	}

	if pathsOnly {
		format = "paths"
	}

	// Load files using existing helper
	files, err := loadFilesForProcessing(path, ignorePatterns)
	if err != nil {
		return fmt.Errorf("loading files: %w", err)
	}

	if len(files) == 0 {
		if !quiet {
			fmt.Println("No markdown files found")
		}
		return nil
	}

	if verbose {
		fmt.Printf("Scanning %d files...\n", len(files))
	}

	var matchingFiles []*vault.VaultFile
	var modifications int

	// Process files based on query type
	if whereExpr != "" {
		matchingFiles = processWhereQuery(files, whereExpr, verbose, quiet)
	} else if missingField != "" {
		matchingFiles, modifications = processMissingQuery(files, missingField, fixWith, dryRun, verbose, quiet)
	} else if duplicatesField != "" {
		matchingFiles = processDuplicatesQuery(files, duplicatesField, verbose, quiet)
	}

	// Handle count-only output
	if count {
		if !quiet {
			fmt.Printf("%d files match the criteria\n", len(matchingFiles))
		} else {
			fmt.Printf("%d\n", len(matchingFiles))
		}
		return nil
	}

	// Handle no matches
	if len(matchingFiles) == 0 {
		if !quiet {
			fmt.Println("No files match the criteria")
		}
		return nil
	}

	// Output results in requested format
	if err := outputResults(matchingFiles, fields, format, quiet); err != nil {
		return fmt.Errorf("outputting results: %w", err)
	}

	// Summary for modifications
	if modifications > 0 {
		if dryRun {
			fmt.Printf("\nDry run completed. Would modify %d files.\n", modifications)
		} else {
			fmt.Printf("\nCompleted. Modified %d files.\n", modifications)
		}
	}

	return nil
}

// Enhanced where expression parser using the new query language
func processWhereQuery(files []*vault.VaultFile, whereExpr string, verbose, quiet bool) []*vault.VaultFile {
	var matches []*vault.VaultFile

	// Parse the expression using the enhanced query parser
	parser := query.NewParser(whereExpr)
	expr, err := parser.Parse()
	if err != nil {
		if !quiet {
			fmt.Printf("Error parsing query expression: %v\n", err)
			fmt.Printf("Supported syntax:\n")
			fmt.Printf("  Simple comparisons: field = 'value', priority > 3, status != 'done'\n")
			fmt.Printf("  Contains operator: tags contains 'urgent', title contains 'project'\n")
			fmt.Printf("  Date comparisons: created after '2024-01-01', modified before '2024-12-01', updated within '7 days'\n")
			fmt.Printf("  Logical operators: priority > 3 AND status != 'done', tags contains 'work' OR tags contains 'project'\n")
		}
		return matches
	}

	// Evaluate the expression against each file
	for _, file := range files {
		if expr.Evaluate(file) {
			matches = append(matches, file)
			if verbose {
				fmt.Printf("Examining: %s - Matches query\n", file.RelativePath)
			}
		} else if verbose {
			fmt.Printf("Examining: %s - No match\n", file.RelativePath)
		}
	}

	return matches
}

func processMissingQuery(files []*vault.VaultFile, field, fixWith string, dryRun, verbose, quiet bool) ([]*vault.VaultFile, int) {
	var matches []*vault.VaultFile
	modifications := 0

	for _, file := range files {
		if _, exists := file.GetField(field); !exists {
			matches = append(matches, file)

			if verbose {
				fmt.Printf("Examining: %s - Missing field '%s'\n", file.RelativePath, field)
			}

			// Auto-fix if requested
			if fixWith != "" {
				if dryRun {
					if verbose {
						fmt.Printf("Would fix: %s - Would add field '%s' = %s\n", file.RelativePath, field, fixWith)
					}
				} else {
					// Process template variables
					processedValue := fixWith
					if strings.Contains(fixWith, "{{current_date}}") {
						processedValue = strings.ReplaceAll(processedValue, "{{current_date}}", "2024-12-18") // TODO: use actual date
					}

					file.SetField(field, processedValue)

					// Save file
					content, err := file.Serialize()
					if err == nil {
						err = os.WriteFile(file.Path, content, 0644)
						if err == nil {
							modifications++
							if verbose {
								fmt.Printf("Fixed: %s - Added field '%s' = %s\n", file.RelativePath, field, processedValue)
							}
						}
					}
				}
			}
		} else if verbose {
			fmt.Printf("Examining: %s - Has field '%s'\n", file.RelativePath, field)
		}
	}

	return matches, modifications
}

func processDuplicatesQuery(files []*vault.VaultFile, field string, verbose, quiet bool) []*vault.VaultFile {
	valueMap := make(map[string][]*vault.VaultFile)

	// Group files by field value
	for _, file := range files {
		if value, exists := file.GetField(field); exists {
			valueStr := fmt.Sprintf("%v", value)
			valueMap[valueStr] = append(valueMap[valueStr], file)
		}
	}

	// Find duplicates
	var duplicates []*vault.VaultFile
	for value, fileList := range valueMap {
		if len(fileList) > 1 {
			if verbose {
				fmt.Printf("Found %d files with %s = '%s'\n", len(fileList), field, value)
			}
			duplicates = append(duplicates, fileList...)
		}
	}

	return duplicates
}

func outputResults(files []*vault.VaultFile, fields []string, format string, quiet bool) error {
	switch format {
	case "table":
		return outputTable(files, fields, quiet)
	case "json":
		return outputJSON(files, fields)
	case "csv":
		return outputCSV(files, fields)
	case "yaml":
		return outputYAML(files, fields)
	case "paths":
		return outputPaths(files)
	default:
		return fmt.Errorf("unsupported format: %s (supported: table, json, csv, yaml, paths)", format)
	}
}

func outputTable(files []*vault.VaultFile, fields []string, quiet bool) error {
	if len(files) == 0 {
		return nil
	}

	// Default fields if none specified
	if len(fields) == 0 {
		fields = []string{"file", "title"}
	}

	// Calculate column widths for proper alignment
	colWidths := make([]int, len(fields))
	rows := make([][]string, len(files))

	// Initialize column widths with header lengths
	for i, field := range fields {
		colWidths[i] = len(cases.Title(language.English).String(field))
	}

	// Collect all data and calculate maximum width for each column
	for fileIdx, file := range files {
		row := make([]string, len(fields))
		for i, field := range fields {
			var cellValue string
			if field == "file" {
				cellValue = file.RelativePath
			} else {
				if value, exists := file.GetField(field); exists {
					cellValue = fmt.Sprintf("%v", value)
				} else {
					cellValue = ""
				}
			}
			row[i] = cellValue
			if len(cellValue) > colWidths[i] {
				colWidths[i] = len(cellValue)
			}
		}
		rows[fileIdx] = row
	}

	if !quiet {
		// Print header with proper alignment
		for i, field := range fields {
			if i > 0 {
				fmt.Print(" │ ")
			}
			header := cases.Title(language.English).String(field)
			fmt.Printf("%-*s", colWidths[i], header)
		}
		fmt.Println()

		// Print separator line
		for i, _ := range fields {
			if i > 0 {
				fmt.Print("─┼─")
			}
			fmt.Print(strings.Repeat("─", colWidths[i]))
		}
		fmt.Println()
	}

	// Print data rows with proper alignment
	for _, row := range rows {
		for i, cellValue := range row {
			if i > 0 {
				fmt.Print(" │ ")
			}
			fmt.Printf("%-*s", colWidths[i], cellValue)
		}
		fmt.Println()
	}

	return nil
}

func outputJSON(files []*vault.VaultFile, fields []string) error {
	var results []map[string]interface{}

	for _, file := range files {
		result := map[string]interface{}{
			"file": file.RelativePath,
		}

		if len(fields) == 0 {
			// Include all frontmatter
			for k, v := range file.Frontmatter {
				result[k] = v
			}
		} else {
			// Include only specified fields
			for _, field := range fields {
				if field == "file" {
					continue // already added
				}
				if value, exists := file.GetField(field); exists {
					result[field] = value
				}
			}
		}

		results = append(results, result)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

func outputCSV(files []*vault.VaultFile, fields []string) error {
	// Default fields if none specified
	if len(fields) == 0 {
		fields = []string{"file", "title"}
	}

	// Header
	for i, field := range fields {
		if i > 0 {
			fmt.Print(",")
		}
		fmt.Printf("\"%s\"", field)
	}
	fmt.Println()

	// Data
	for _, file := range files {
		for i, field := range fields {
			if i > 0 {
				fmt.Print(",")
			}

			var value string
			if field == "file" {
				value = file.RelativePath
			} else {
				if v, exists := file.GetField(field); exists {
					value = fmt.Sprintf("%v", v)
				}
			}
			fmt.Printf("\"%s\"", strings.ReplaceAll(value, "\"", "\"\""))
		}
		fmt.Println()
	}

	return nil
}

func outputYAML(files []*vault.VaultFile, fields []string) error {
	for i, file := range files {
		if i > 0 {
			fmt.Println("---")
		}

		fmt.Printf("file: %s\n", file.RelativePath)

		if len(fields) == 0 {
			// Include all frontmatter
			for k, v := range file.Frontmatter {
				fmt.Printf("%s: %v\n", k, v)
			}
		} else {
			// Include only specified fields
			for _, field := range fields {
				if field == "file" {
					continue // already added
				}
				if value, exists := file.GetField(field); exists {
					fmt.Printf("%s: %v\n", field, value)
				}
			}
		}
	}

	return nil
}

// outputPaths outputs only the file paths, one per line, for piping to other commands
func outputPaths(files []*vault.VaultFile) error {
	for _, file := range files {
		fmt.Println(file.Path)
	}
	return nil
}
