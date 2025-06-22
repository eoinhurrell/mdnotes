package rename

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/eoinhurrell/mdnotes/internal/processor"
	"github.com/spf13/cobra"
)

// NewRenameCommand creates the rename command
func NewRenameCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rename [source_file] [new_name]",
		Aliases: []string{"r"},
		Short:   "Rename a file and update all references",
		Long: `Rename a markdown file and automatically update all references to it throughout the vault.
This command ensures vault integrity by updating both wiki links ([[file]]) and markdown links ([text](file.md)) 
that point to the renamed file.

If no new_name is provided, uses the default template pattern.

Examples:
  # Rename a file
  mdnotes rename note.md new-note.md
  
  # Rename using default template
  mdnotes rename "Case Closed.md"
  # Results in: 20250620125421-case-closed.md
  
  # Rename with verbose output
  mdnotes rename --verbose old-name.md better-name.md
  
  # Preview changes without applying them
  mdnotes rename --dry-run test.md renamed-test.md
  
  # Custom template
  mdnotes rename --template "{{created|date:2006-01-02}}-{{filename|slug}}.md" note.md`,
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
	
	sourceFile := args[0]
	var newName string
	if len(args) == 2 {
		newName = args[1]
	} else {
		// Generate name using template
		template, _ := cmd.Flags().GetString("template")
		generatedName, err := processor.GenerateNameFromTemplate(sourceFile, template)
		if err != nil {
			return fmt.Errorf("generating name from template: %w", err)
		}
		newName = generatedName
	}

	// Get flags
	ignorePatterns, _ := cmd.Flags().GetStringSlice("ignore")
	vaultRoot, _ := cmd.Flags().GetString("vault")
	workers, _ := cmd.Flags().GetInt("workers")
	dryRun, _ := cmd.Root().PersistentFlags().GetBool("dry-run")
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
	quiet, _ := cmd.Root().PersistentFlags().GetBool("quiet")

	// Override verbose if quiet is specified
	if quiet {
		verbose = false
	}

	// Validate source file exists
	if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
		return fmt.Errorf("source file does not exist: %s", sourceFile)
	}

	// Get absolute paths
	sourceAbs, err := filepath.Abs(sourceFile)
	if err != nil {
		return fmt.Errorf("getting absolute path for source: %w", err)
	}

	vaultAbs, err := filepath.Abs(vaultRoot)
	if err != nil {
		return fmt.Errorf("getting absolute path for vault: %w", err)
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
		if len(args) == 1 {
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





