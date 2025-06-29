package rename

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/eoinhurrell/mdnotes/internal/processor"
	"github.com/eoinhurrell/mdnotes/internal/vault"
	"github.com/spf13/cobra"
)

// NewRenameCommand creates the rename command
func NewRenameCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rename [path] [template_or_target]",
		Aliases: []string{"r"},
		Short:   "Rename files and update all references",
		Long: `Rename markdown files and automatically update all references throughout the vault.
This command ensures vault integrity by updating both wiki links ([[file]]) and markdown links ([text](file.md)) 
that point to renamed files.

Works with both single files and entire directories.

Examples:
  # Rename a single file
  mdnotes rename note.md new-note.md
  
  # Rename file using default template
  mdnotes rename "Case Closed.md"
  # Results in: 20250620125421-case-closed.md
  
  # Rename all files in a directory using default template
  mdnotes rename /path/to/vault/
  
  # Rename all files in directory with custom template
  mdnotes rename /path/to/vault/ "{{created|date:2006-01-02}}-{{title|slug}}.md"
  
  # Preview changes without applying them
  mdnotes rename --dry-run /path/to/vault/
  
  # Rename with verbose output
  mdnotes rename --verbose /path/to/vault/`,
		Args: cobra.RangeArgs(1, 2),
		RunE: runRename,
	}

	cmd.Flags().StringSlice("ignore", []string{".obsidian/*", "*.tmp"}, "Ignore patterns for scanning vault")
	cmd.Flags().String("vault", ".", "Vault root directory for link updates")
	cmd.Flags().String("template", "{{created|date:20060102150405}}-{{filename|slug_underscore}}.md", "Template for default rename target")
	cmd.Flags().Int("workers", runtime.NumCPU(), "Number of worker goroutines for parallel processing")

	return cmd
}

func runRename(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	path := args[0]
	var templateOrTarget string
	if len(args) == 2 {
		templateOrTarget = args[1]
	}

	// Get flags
	ignorePatterns, _ := cmd.Flags().GetStringSlice("ignore")
	vaultRoot, _ := cmd.Flags().GetString("vault")
	defaultTemplate, _ := cmd.Flags().GetString("template")
	workers, _ := cmd.Flags().GetInt("workers")
	dryRun, _ := cmd.Root().PersistentFlags().GetBool("dry-run")
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
	quiet, _ := cmd.Root().PersistentFlags().GetBool("quiet")

	// Override verbose if quiet is specified
	if quiet {
		verbose = false
	}

	// Validate path exists
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", path)
	}
	if err != nil {
		return fmt.Errorf("accessing path: %w", err)
	}

	// Get absolute paths
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("getting absolute path: %w", err)
	}

	vaultAbs, err := filepath.Abs(vaultRoot)
	if err != nil {
		return fmt.Errorf("getting absolute path for vault: %w", err)
	}

	if info.IsDir() {
		// Directory mode: rename all markdown files using template
		return runDirectoryRename(ctx, pathAbs, vaultAbs, templateOrTarget, defaultTemplate, 
			ignorePatterns, workers, dryRun, verbose, quiet)
	} else {
		// Single file mode: existing logic
		return runSingleFileRename(ctx, pathAbs, vaultAbs, templateOrTarget, defaultTemplate,
			ignorePatterns, workers, dryRun, verbose, quiet)
	}
}

// runSingleFileRename handles renaming a single file
func runSingleFileRename(ctx context.Context, sourceAbs, vaultAbs, templateOrTarget, defaultTemplate string,
	ignorePatterns []string, workers int, dryRun, verbose, quiet bool) error {
	
	var newName string
	if templateOrTarget != "" {
		// Use provided target name
		newName = templateOrTarget
	} else {
		// Generate name using template
		generatedName, err := processor.GenerateNameFromTemplate(sourceAbs, defaultTemplate)
		if err != nil {
			return fmt.Errorf("generating name from template: %w", err)
		}
		newName = generatedName
	}

	// Determine target path
	var targetAbs string
	if filepath.IsAbs(newName) {
		targetAbs = newName
	} else {
		// If newName is just a filename, put it in the same directory as source
		if !strings.Contains(newName, string(filepath.Separator)) {
			targetAbs = filepath.Join(filepath.Dir(sourceAbs), newName)
		} else {
			// newName contains path components, resolve relative to vault root
			targetAbs = filepath.Join(vaultAbs, newName)
		}
	}

	// Ensure target has .md extension if source does
	if strings.HasSuffix(sourceAbs, ".md") && !strings.HasSuffix(targetAbs, ".md") {
		targetAbs += ".md"
	}

	// Check if target already exists (unless it's the same file or case-only change)
	if !isSameFile(sourceAbs, targetAbs) {
		if _, err := os.Stat(targetAbs); err == nil {
			return fmt.Errorf("target file already exists: %s", targetAbs)
		}
	}

	// Get relative paths for move tracking
	sourceRel, err := filepath.Rel(vaultAbs, sourceAbs)
	if err != nil {
		return fmt.Errorf("getting relative path for source: %w", err)
	}

	targetRel, err := filepath.Rel(vaultAbs, targetAbs)
	if err != nil {
		return fmt.Errorf("getting relative path for target: %w", err)
	}

	if verbose {
		fmt.Printf("Renaming: %s -> %s\n", sourceRel, targetRel)
		if templateOrTarget == "" {
			fmt.Printf("Using template-generated name\n")
		}
	}

	// Create rename processor with optimized settings
	options := processor.RenameOptions{
		VaultRoot:      vaultAbs,
		IgnorePatterns: ignorePatterns,
		DryRun:         dryRun,
		Verbose:        verbose,
		Workers:        workers,
	}

	renameProcessor := processor.NewRenameProcessor(options)
	defer func() {
		if cleanupErr := renameProcessor.Cleanup(); cleanupErr != nil && verbose {
			fmt.Printf("Warning: error during cleanup: %v\n", cleanupErr)
		}
	}()

	// Perform the rename operation
	result, err := renameProcessor.ProcessRename(ctx, sourceAbs, targetAbs, options)
	if err != nil {
		return fmt.Errorf("processing rename: %w", err)
	}

	// Display results
	if dryRun {
		fmt.Printf("Would rename: %s -> %s\n", sourceRel, targetRel)
		if result.FilesModified > 0 {
			fmt.Printf("Would update %d links in %d files\n", result.LinksUpdated, result.FilesModified)
			if verbose {
				for _, file := range result.ModifiedFiles {
					fmt.Printf("  - %s\n", file)
				}
			}
		} else {
			fmt.Println("No references found to update")
		}
	} else {
		if !quiet {
			fmt.Printf("✓ Renamed: %s -> %s\n", sourceRel, targetRel)
			if result.FilesModified > 0 {
				fmt.Printf("✓ Updated %d links in %d files\n", result.LinksUpdated, result.FilesModified)
			}
			if verbose {
				fmt.Printf("Processed %d files in %v\n", result.FilesScanned, result.Duration)
			}
		}
	}

	return nil
}

// runDirectoryRename handles renaming all markdown files in a directory
func runDirectoryRename(ctx context.Context, pathAbs, vaultAbs, templateOrTarget, defaultTemplate string,
	ignorePatterns []string, workers int, dryRun, verbose, quiet bool) error {
	
	// Determine template to use
	template := defaultTemplate
	if templateOrTarget != "" {
		template = templateOrTarget
	}

	// Use Scanner to find all markdown files
	scanner := vault.NewScanner(vault.WithIgnorePatterns(ignorePatterns))
	files, err := scanner.Walk(pathAbs)
	if err != nil {
		return fmt.Errorf("scanning directory: %w", err)
	}

	if len(files) == 0 {
		if !quiet {
			fmt.Println("No markdown files found")
		}
		return nil
	}

	if verbose {
		fmt.Printf("Found %d markdown files to process\n", len(files))
		if templateOrTarget != "" {
			fmt.Printf("Using custom template: %s\n", template)
		} else {
			fmt.Printf("Using default template: %s\n", template)
		}
	}

	// Prepare rename operations with conflict detection
	type renameOperation struct {
		sourceFile   *vault.VaultFile
		sourcePath   string
		targetPath   string
		targetRel    string
		shouldRename bool
		error        error
	}

	operations := make([]renameOperation, 0, len(files))
	targetPaths := make(map[string]*vault.VaultFile) // For conflict detection

	for _, file := range files {
		op := renameOperation{
			sourceFile: file,
			sourcePath: file.Path,
		}

		// Generate target name using template
		generatedName, err := processor.GenerateNameFromTemplate(file.Path, template)
		if err != nil {
			op.error = fmt.Errorf("generating name from template: %w", err)
			operations = append(operations, op)
			continue
		}

		// Construct target path (in same directory as source)
		targetPath := filepath.Join(filepath.Dir(file.Path), generatedName)
		
		// Ensure .md extension
		if !strings.HasSuffix(targetPath, ".md") {
			targetPath += ".md"
		}

		op.targetPath = targetPath

		// Get relative path for display
		targetRel, err := filepath.Rel(vaultAbs, targetPath)
		if err != nil {
			op.error = fmt.Errorf("getting relative path for target: %w", err)
			operations = append(operations, op)
			continue
		}
		op.targetRel = targetRel

		// Check if rename is needed (file name would change)
		if !isSameFile(file.Path, targetPath) {
			// Check for conflicts with other files being renamed
			if conflictFile, exists := targetPaths[targetPath]; exists {
				op.error = fmt.Errorf("target name conflict: both %s and %s would be renamed to %s", 
					file.RelativePath, conflictFile.RelativePath, targetRel)
				operations = append(operations, op)
				continue
			}

			// Check if target already exists
			if _, err := os.Stat(targetPath); err == nil {
				op.error = fmt.Errorf("target file already exists: %s", targetRel)
				operations = append(operations, op)
				continue
			}

			op.shouldRename = true
			targetPaths[targetPath] = file
		}

		operations = append(operations, op)
	}

	// Count operations and report
	var renameCount, errorCount, skipCount int
	for _, op := range operations {
		if op.error != nil {
			errorCount++
		} else if op.shouldRename {
			renameCount++
		} else {
			skipCount++
		}
	}

	if verbose || dryRun {
		fmt.Printf("Operations planned:\n")
		fmt.Printf("  - Files to rename: %d\n", renameCount)
		fmt.Printf("  - Files to skip: %d\n", skipCount)
		fmt.Printf("  - Errors: %d\n", errorCount)
	}

	// Show errors
	if errorCount > 0 {
		fmt.Printf("Errors encountered:\n")
		for _, op := range operations {
			if op.error != nil {
				fmt.Printf("  - %s: %v\n", op.sourceFile.RelativePath, op.error)
			}
		}
		if !dryRun {
			return fmt.Errorf("cannot proceed with %d errors", errorCount)
		}
	}

	if renameCount == 0 {
		if !quiet {
			fmt.Println("No files need renaming")
		}
		return nil
	}

	if dryRun {
		fmt.Printf("\nWould rename %d files:\n", renameCount)
		for _, op := range operations {
			if op.shouldRename {
				sourceRel, _ := filepath.Rel(vaultAbs, op.sourcePath)
				fmt.Printf("  %s -> %s\n", sourceRel, op.targetRel)
			}
		}
		return nil
	}

	// Execute renames
	var successCount, failureCount int
	allModifiedFiles := make(map[string]bool)
	totalLinksUpdated := 0

	for i, op := range operations {
		if !op.shouldRename {
			continue
		}

		if verbose {
			sourceRel, _ := filepath.Rel(vaultAbs, op.sourcePath)
			fmt.Printf("Renaming [%d/%d]: %s -> %s\n", i+1, renameCount, sourceRel, op.targetRel)
		}

		// Create rename processor for this operation
		options := processor.RenameOptions{
			VaultRoot:      vaultAbs,
			IgnorePatterns: ignorePatterns,
			DryRun:         false, // Already checked above
			Verbose:        false, // Control output at this level
			Workers:        workers,
		}

		renameProcessor := processor.NewRenameProcessor(options)
		result, err := renameProcessor.ProcessRename(ctx, op.sourcePath, op.targetPath, options)
		
		// Clean up processor
		if cleanupErr := renameProcessor.Cleanup(); cleanupErr != nil && verbose {
			fmt.Printf("Warning: error during cleanup: %v\n", cleanupErr)
		}

		if err != nil {
			failureCount++
			if !quiet {
				fmt.Printf("✗ Failed to rename %s: %v\n", op.sourceFile.RelativePath, err)
			}
			continue
		}

		successCount++
		if !quiet && !verbose {
			fmt.Printf("✓ Renamed: %s -> %s\n", op.sourceFile.RelativePath, op.targetRel)
		}

		// Track link updates
		totalLinksUpdated += result.LinksUpdated
		for _, file := range result.ModifiedFiles {
			allModifiedFiles[file] = true
		}
	}

	// Final summary
	if !quiet {
		fmt.Printf("\nRename Summary:\n")
		fmt.Printf("✓ Successfully renamed: %d files\n", successCount)
		if failureCount > 0 {
			fmt.Printf("✗ Failed to rename: %d files\n", failureCount)
		}
		if totalLinksUpdated > 0 {
			fmt.Printf("✓ Updated %d links in %d files\n", totalLinksUpdated, len(allModifiedFiles))
		}
	}

	if failureCount > 0 {
		return fmt.Errorf("completed with %d failures out of %d operations", failureCount, renameCount)
	}

	return nil
}

// isSameFile checks if two paths refer to the same file, handling case-insensitive filesystems
func isSameFile(path1, path2 string) bool {
	// Quick check for exact match
	if path1 == path2 {
		return true
	}
	
	// Get file info for both paths
	info1, err1 := os.Stat(path1)
	info2, err2 := os.Stat(path2)
	
	// If either file doesn't exist, they're not the same
	if os.IsNotExist(err1) || os.IsNotExist(err2) {
		return false
	}
	
	// If we can't stat either file, fall back to case-insensitive string comparison
	if err1 != nil || err2 != nil {
		return strings.EqualFold(path1, path2)
	}
	
	// On most filesystems, if the inodes are the same, it's the same file
	// This works for case-insensitive renames and also handles hard links
	return os.SameFile(info1, info2)
}





