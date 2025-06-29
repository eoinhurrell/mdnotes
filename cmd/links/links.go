package links

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/eoinhurrell/mdnotes/internal/processor"
	"github.com/eoinhurrell/mdnotes/internal/selector"
	"github.com/eoinhurrell/mdnotes/internal/vault"
	"github.com/spf13/cobra"
)

// NewLinksCommand creates the links command
func NewLinksCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "links",
		Short: "Manage links in markdown files",
		Long:  "Commands for checking, converting, and updating links in Obsidian notes",
	}

	cmd.AddCommand(NewCheckCommand())
	cmd.AddCommand(NewConvertCommand())

	return cmd
}

// NewCheckCommand creates the links check command
func NewCheckCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "check [path]",
		Aliases: []string{"c"},
		Short:   "Check for broken internal links",
		Long: `Check for broken internal links in markdown files.
Reports links that point to non-existent files.

By default, markdown links are checked relative to the vault root (Obsidian behavior).
Wiki links are always checked relative to the vault root.

Examples:
  # Check links (default: vault-relative)
  mdnotes links check /path/to/vault
  
  # Check links relative to each file's directory
  mdnotes links check --file-relative /path/to/vault`,
		Args: cobra.ExactArgs(1),
		RunE: runCheck,
	}

	cmd.Flags().StringSlice("ignore", []string{".obsidian/*", "*.tmp"}, "Ignore patterns")
	cmd.Flags().Bool("file-relative", false, "Check markdown links relative to each file's directory instead of vault root")

	return cmd
}

func runCheck(cmd *cobra.Command, args []string) error {
	path := args[0]

	// Get flags
	ignorePatterns, _ := cmd.Flags().GetStringSlice("ignore")
	fileRelative, _ := cmd.Flags().GetBool("file-relative")
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
	quiet, _ := cmd.Root().PersistentFlags().GetBool("quiet")

	// Override verbose if quiet is specified
	if quiet {
		verbose = false
	}

	// Get file selection configuration from global flags
	mode, fileSelector, err := selector.GetGlobalSelectionConfig(cmd)
	if err != nil {
		return fmt.Errorf("getting file selection config: %w", err)
	}
	
	// Merge local ignore patterns with global ignore patterns
	localIgnore := ignorePatterns
	if len(fileSelector.IgnorePatterns) > 0 {
		// Combine both sets of ignore patterns
		combinedIgnore := append(fileSelector.IgnorePatterns, localIgnore...)
		fileSelector = fileSelector.WithIgnorePatterns(combinedIgnore)
	} else {
		fileSelector = fileSelector.WithIgnorePatterns(localIgnore)
	}
	
	// Select files using unified architecture
	selection, err := fileSelector.SelectFiles(path, mode)
	if err != nil {
		return fmt.Errorf("selecting files: %w", err)
	}
	
	// Print selection summary if verbose
	if verbose {
		fmt.Printf("%s\n", selection.GetSelectionSummary())
	}
	
	// Print parse errors if any
	if len(selection.ParseErrors) > 0 && verbose {
		selection.PrintParseErrors()
	}
	
	files := selection.Files

	if len(files) == 0 {
		if !quiet {
			fmt.Println("No markdown files found")
		}
		return nil
	}

	// Get vault root absolute path for resolving links
	vaultRoot, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("getting absolute path for vault: %w", err)
	}

	// Create maps for different types of file lookups
	existingFiles := make(map[string]bool)           // vault-relative paths
	baseNameFiles := make(map[string][]string)       // basename -> list of full paths
	for _, file := range files {
		// Normalize path separators for consistent lookup
		normalizedPath := filepath.ToSlash(file.RelativePath)
		existingFiles[normalizedPath] = true
		
		// Also add without .md extension for exact matches
		if strings.HasSuffix(normalizedPath, ".md") {
			withoutExt := strings.TrimSuffix(normalizedPath, ".md")
			existingFiles[withoutExt] = true
			
			// For wiki links: map basename to full paths (Obsidian behavior)
			baseName := filepath.Base(withoutExt)
			baseNameFiles[baseName] = append(baseNameFiles[baseName], normalizedPath)
		}
	}

	// Check links
	linkParser := processor.NewLinkParser()
	brokenLinks := 0
	totalLinks := 0

	for _, file := range files {
		linkParser.UpdateFile(file)

		fileHasBrokenLinks := false
		fileLinksCount := 0

		for _, link := range file.Links {
			totalLinks++
			fileLinksCount++

			// Determine the target path to check based on link type and flags
			targetToCheck := resolveTargetPath(link, file, vaultRoot, fileRelative)
			linkExists := checkLinkExists(targetToCheck, existingFiles, baseNameFiles, link.Type)

			if !linkExists {
				brokenLinks++
				fileHasBrokenLinks = true
				linkText := formatLinkForDisplay(link)
				if fileRelative && link.Type == vault.MarkdownLink {
					fmt.Printf("✗ %s: broken link %s (checked relative to file)\n", file.RelativePath, linkText)
				} else {
					fmt.Printf("✗ %s: broken link %s\n", file.RelativePath, linkText)
				}
			} else if verbose {
				linkText := formatLinkForDisplay(link)
				if fileRelative && link.Type == vault.MarkdownLink {
					fmt.Printf("✓ %s: valid link %s (checked relative to file)\n", file.RelativePath, linkText)
				} else {
					fmt.Printf("✓ %s: valid link %s\n", file.RelativePath, linkText)
				}
			}
		}

		// Show examining message for verbose mode
		if verbose {
			if fileLinksCount == 0 {
				fmt.Printf("Examining: %s - No links found\n", file.RelativePath)
			} else if fileHasBrokenLinks {
				fmt.Printf("Examining: %s - Found broken links\n", file.RelativePath)
			} else {
				fmt.Printf("Examining: %s - All %d links valid\n", file.RelativePath, fileLinksCount)
			}
		}
	}

	// Summary
	if brokenLinks > 0 {
		if !quiet {
			fmt.Printf("\nCheck completed: %d broken links found out of %d total links\n", brokenLinks, totalLinks)
		}
		return fmt.Errorf("found %d broken links", brokenLinks)
	} else {
		if !quiet {
			fmt.Printf("\nCheck completed: all %d links are valid\n", totalLinks)
		}
	}

	return nil
}

// NewConvertCommand creates the links convert command
func NewConvertCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "convert [path]",
		Aliases: []string{"co"},
		Short:   "Convert between link formats",
		Long: `Convert links between wiki and markdown formats.
Wiki format: [[note]] or [[note|alias]]
Markdown format: [text](note.md)`,
		Args: cobra.ExactArgs(1),
		RunE: runConvert,
	}

	cmd.Flags().String("from", "wiki", "Source format (wiki, markdown)")
	cmd.Flags().String("to", "markdown", "Target format (wiki, markdown)")
	cmd.Flags().StringSlice("ignore", []string{".obsidian/*", "*.tmp"}, "Ignore patterns")

	return cmd
}

func runConvert(cmd *cobra.Command, args []string) error {
	path := args[0]

	// Get flags
	fromFormat, _ := cmd.Flags().GetString("from")
	toFormat, _ := cmd.Flags().GetString("to")
	ignorePatterns, _ := cmd.Flags().GetStringSlice("ignore")
	dryRun, _ := cmd.Root().PersistentFlags().GetBool("dry-run")
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
	quiet, _ := cmd.Root().PersistentFlags().GetBool("quiet")

	// Override verbose if quiet is specified
	if quiet {
		verbose = false
	}

	// Parse formats
	var from, to processor.LinkFormat
	switch fromFormat {
	case "wiki":
		from = processor.WikiFormat
	case "markdown":
		from = processor.MarkdownFormat
	default:
		return fmt.Errorf("invalid source format: %s (must be wiki or markdown)", fromFormat)
	}

	switch toFormat {
	case "wiki":
		to = processor.WikiFormat
	case "markdown":
		to = processor.MarkdownFormat
	default:
		return fmt.Errorf("invalid target format: %s (must be wiki or markdown)", toFormat)
	}

	if from == to {
		fmt.Println("Source and target formats are the same, no conversion needed")
		return nil
	}

	// Create processor
	converter := processor.NewLinkConverter()

	// Setup file processor
	fileProcessor := &processor.FileProcessor{
		DryRun:         dryRun,
		Verbose:        verbose,
		Quiet:          quiet,
		IgnorePatterns: ignorePatterns,
		ProcessFile: func(file *vault.VaultFile) (bool, error) {
			modified := converter.ConvertFile(file, from, to)
			if verbose {
				if modified {
					fmt.Printf("Examining: %s - Converted links from %s to %s format\n", file.RelativePath, fromFormat, toFormat)
				} else {
					fmt.Printf("Examining: %s - No links to convert\n", file.RelativePath)
				}
			}
			return modified, nil
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

	// Print custom summary for link conversion
	if len(result.Errors) > 0 {
		for _, err := range result.Errors {
			fmt.Printf("✗ %v\n", err)
		}
	}

	if dryRun {
		fmt.Printf("\nDry run completed. Would modify %d files.\n", result.ProcessedFiles)
	} else {
		fmt.Printf("\nCompleted. Converted links in %d files from %s to %s format.\n", result.ProcessedFiles, fromFormat, toFormat)
	}

	return nil
}

// resolveTargetPath determines the actual path to check based on link type and settings
func resolveTargetPath(link vault.Link, file *vault.VaultFile, vaultRoot string, fileRelative bool) string {
	target := link.Target
	
	// Remove any fragment identifiers (e.g., file.md#heading)
	if idx := strings.Index(target, "#"); idx != -1 {
		target = target[:idx]
	}
	
	// Normalize path separators
	target = filepath.ToSlash(target)
	
	switch link.Type {
	case vault.WikiLink, vault.EmbedLink:
		// Wiki links and embeds are always relative to vault root in Obsidian
		return target
		
	case vault.MarkdownLink:
		if fileRelative {
			// Check relative to the file's directory
			fileDir := filepath.Dir(file.RelativePath)
			if fileDir == "." {
				return target
			}
			return filepath.ToSlash(filepath.Join(fileDir, target))
		} else {
			// Default: check relative to vault root (Obsidian behavior)
			// This is the key fix - markdown links should be vault-relative by default
			return target
		}
		
	default:
		return target
	}
}

// checkLinkExists checks if a target path exists in the files map
func checkLinkExists(target string, existingFiles map[string]bool, baseNameFiles map[string][]string, linkType vault.LinkType) bool {
	// Normalize path separators
	target = filepath.ToSlash(target)
	
	// Remove any fragment identifiers (e.g., file.md#heading)
	if idx := strings.Index(target, "#"); idx != -1 {
		target = target[:idx]
	}
	
	// Check direct match first
	if existingFiles[target] {
		return true
	}
	
	// For wiki links and embeds, use Obsidian's basename resolution
	if linkType == vault.WikiLink || linkType == vault.EmbedLink {
		// Try adding .md extension if not present
		if !strings.HasSuffix(target, ".md") && !strings.Contains(target, ".") {
			if existingFiles[target+".md"] {
				return true
			}
		}
		
		// For wiki links, also check by basename (Obsidian behavior)
		// This allows [[filename]] to match files in subdirectories
		baseName := filepath.Base(target)
		if paths, exists := baseNameFiles[baseName]; exists && len(paths) > 0 {
			return true
		}
		
		// Try basename with .md removed
		if strings.HasSuffix(baseName, ".md") {
			baseNameWithoutExt := strings.TrimSuffix(baseName, ".md")
			if paths, exists := baseNameFiles[baseNameWithoutExt]; exists && len(paths) > 0 {
				return true
			}
		}
	}
	
	// For markdown links, also check without .md extension (for wiki-style references)
	if strings.HasSuffix(target, ".md") {
		withoutExt := strings.TrimSuffix(target, ".md")
		if existingFiles[withoutExt] {
			return true
		}
	}
	
	return false
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
