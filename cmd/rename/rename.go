package rename

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/eoinhurrell/mdnotes/internal/processor"
	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// NewRenameCommand creates the rename command
func NewRenameCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rename [source_file] [new_name]",
		Short: "Rename a file and update all references",
		Long: `Rename a markdown file and automatically update all references to it throughout the vault.
This command ensures vault integrity by updating both wiki links ([[file]]) and markdown links ([text](file.md)) 
that point to the renamed file.

Examples:
  # Rename a file
  mdnotes rename note.md new-note.md
  
  # Rename with verbose output
  mdnotes rename --verbose old-name.md better-name.md
  
  # Preview changes without applying them
  mdnotes rename --dry-run test.md renamed-test.md`,
		Args: cobra.ExactArgs(2),
		RunE: runRename,
	}

	cmd.Flags().StringSlice("ignore", []string{".obsidian/*", "*.tmp"}, "Ignore patterns for scanning vault")
	cmd.Flags().String("vault", ".", "Vault root directory for link updates")

	return cmd
}

func runRename(cmd *cobra.Command, args []string) error {
	sourceFile := args[0]
	newName := args[1]
	
	// Get flags
	ignorePatterns, _ := cmd.Flags().GetStringSlice("ignore")
	vaultRoot, _ := cmd.Flags().GetString("vault")
	dryRun, _ := cmd.Root().PersistentFlags().GetBool("dry-run")
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")

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

	// Check if target already exists (unless it's the same file)
	if sourceAbs != targetAbs {
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
	}

	// If dry-run, just show what would happen
	if dryRun {
		fmt.Printf("Would rename: %s -> %s\n", sourceRel, targetRel)
		if err := showLinkReferences(sourceRel, vaultAbs, ignorePatterns, verbose); err != nil {
			return fmt.Errorf("analyzing link references: %w", err)
		}
		return nil
	}

	// Scan vault for all markdown files to update links
	scanner := vault.NewScanner(vault.WithIgnorePatterns(ignorePatterns))
	files, err := scanner.Walk(vaultAbs)
	if err != nil {
		return fmt.Errorf("scanning vault: %w", err)
	}

	if verbose {
		fmt.Printf("Scanned %d files in vault\n", len(files))
	}

	// Create file move record
	move := processor.FileMove{
		From: sourceRel,
		To:   targetRel,
	}

	// Create link updater
	linkUpdater := processor.NewLinkUpdater()
	
	// Update links in all files
	modifiedFiles := linkUpdater.UpdateBatch(files, []processor.FileMove{move})
	
	if verbose && len(modifiedFiles) > 0 {
		fmt.Printf("Updated links in %d files:\n", len(modifiedFiles))
		for _, file := range modifiedFiles {
			fmt.Printf("  - %s\n", file.RelativePath)
		}
	}

	// Save modified files
	for _, file := range modifiedFiles {
		content, err := file.Serialize()
		if err != nil {
			return fmt.Errorf("serializing updated file %s: %w", file.RelativePath, err)
		}
		
		if err := os.WriteFile(file.Path, content, 0644); err != nil {
			return fmt.Errorf("saving updated file %s: %w", file.RelativePath, err)
		}
	}

	// Create target directory if needed
	targetDir := filepath.Dir(targetAbs)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("creating target directory: %w", err)
	}

	// Perform the actual file rename
	if err := os.Rename(sourceAbs, targetAbs); err != nil {
		return fmt.Errorf("renaming file: %w", err)
	}

	fmt.Printf("✓ Renamed: %s -> %s\n", sourceRel, targetRel)
	if len(modifiedFiles) > 0 {
		fmt.Printf("✓ Updated %d files with references\n", len(modifiedFiles))
	}

	return nil
}

// showLinkReferences shows what files reference the source file (for dry-run)
func showLinkReferences(sourceRel, vaultRoot string, ignorePatterns []string, verbose bool) error {
	scanner := vault.NewScanner(vault.WithIgnorePatterns(ignorePatterns))
	files, err := scanner.Walk(vaultRoot)
	if err != nil {
		return err
	}

	linkParser := processor.NewLinkParser()
	referencingFiles := 0
	totalReferences := 0

	for _, file := range files {
		linkParser.UpdateFile(file)
		fileReferences := 0
		
		for _, link := range file.Links {
			// Check if this link references our source file
			if linksToFile(link, sourceRel) {
				if fileReferences == 0 {
					fmt.Printf("Would update references in: %s\n", file.RelativePath)
					referencingFiles++
				}
				if verbose {
					linkText := formatLinkForDisplay(link)
					fmt.Printf("  - %s\n", linkText)
				}
				fileReferences++
				totalReferences++
			}
		}
	}

	if referencingFiles > 0 {
		fmt.Printf("Would update %d references in %d files\n", totalReferences, referencingFiles)
	} else {
		fmt.Println("No references found to update")
	}

	return nil
}

// linksToFile checks if a link references the given file
func linksToFile(link vault.Link, targetFile string) bool {
	target := link.Target
	
	// Normalize paths for comparison
	targetFile = filepath.ToSlash(targetFile)
	target = filepath.ToSlash(target)
	
	// Remove .md extension from target file for wiki link comparison
	targetWithoutExt := strings.TrimSuffix(targetFile, ".md")
	
	switch link.Type {
	case vault.WikiLink:
		// Wiki links might not have .md extension
		if target == targetWithoutExt || target == targetFile {
			return true
		}
		// Also check if target with .md extension matches
		if !strings.HasSuffix(target, ".md") {
			return target+".md" == targetFile
		}
		return target == targetFile
		
	case vault.MarkdownLink, vault.EmbedLink:
		// Direct comparison for markdown and embed links
		return target == targetFile
		
	default:
		return false
	}
}

func formatLinkForDisplay(link vault.Link) string {
	switch link.Type {
	case vault.WikiLink:
		if link.Text == link.Target {
			return fmt.Sprintf("[[%s]]", link.Target)
		}
		return fmt.Sprintf("[[%s|%s]]", link.Target, link.Text)
	case vault.MarkdownLink:
		return fmt.Sprintf("[%s](%s)", link.Text, link.Target)
	case vault.EmbedLink:
		return fmt.Sprintf("![[%s]]", link.Target)
	default:
		return link.Target
	}
}