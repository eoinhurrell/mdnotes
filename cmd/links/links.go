package links

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/eoinhurrell/mdnotes/internal/processor"
	"github.com/eoinhurrell/mdnotes/internal/vault"
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
		Use:   "check [path]",
		Short: "Check for broken internal links",
		Long: `Check for broken internal links in markdown files.
Reports links that point to non-existent files.`,
		Args: cobra.ExactArgs(1),
		RunE: runCheck,
	}

	cmd.Flags().StringSlice("ignore", []string{".obsidian/*", "*.tmp"}, "Ignore patterns")

	return cmd
}

func runCheck(cmd *cobra.Command, args []string) error {
	path := args[0]
	
	// Get flags
	ignorePatterns, _ := cmd.Flags().GetStringSlice("ignore")
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")

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

	// Create a map of existing files
	existingFiles := make(map[string]bool)
	for _, file := range files {
		existingFiles[file.RelativePath] = true
		// Also add without .md extension for wiki links
		if strings.HasSuffix(file.RelativePath, ".md") {
			withoutExt := strings.TrimSuffix(file.RelativePath, ".md")
			existingFiles[withoutExt] = true
		}
	}

	// Check links
	linkParser := processor.NewLinkParser()
	brokenLinks := 0
	totalLinks := 0

	for _, file := range files {
		linkParser.UpdateFile(file)
		
		for _, link := range file.Links {
			totalLinks++
			
			// Normalize link target for checking
			target := link.Target
			if link.Type == vault.WikiLink && !strings.HasSuffix(target, ".md") && !strings.Contains(target, ".") {
				// Wiki links might not have extension
				if !existingFiles[target] && !existingFiles[target+".md"] {
					brokenLinks++
					fmt.Printf("✗ %s: broken link [[%s]]\n", file.RelativePath, target)
				} else if verbose {
					fmt.Printf("✓ %s: valid link [[%s]]\n", file.RelativePath, target)
				}
			} else {
				// Regular links with extensions
				if !existingFiles[target] {
					brokenLinks++
					linkText := formatLinkForDisplay(link)
					fmt.Printf("✗ %s: broken link %s\n", file.RelativePath, linkText)
				} else if verbose {
					linkText := formatLinkForDisplay(link)
					fmt.Printf("✓ %s: valid link %s\n", file.RelativePath, linkText)
				}
			}
		}
	}

	// Summary
	if brokenLinks > 0 {
		fmt.Printf("\nCheck completed: %d broken links found out of %d total links\n", brokenLinks, totalLinks)
		return fmt.Errorf("found %d broken links", brokenLinks)
	} else {
		fmt.Printf("\nCheck completed: all %d links are valid\n", totalLinks)
	}

	return nil
}

// NewConvertCommand creates the links convert command
func NewConvertCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convert [path]",
		Short: "Convert between link formats",
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
		IgnorePatterns: ignorePatterns,
		ProcessFile: func(file *vault.VaultFile) (bool, error) {
			return converter.ConvertFile(file, from, to), nil
		},
		OnFileProcessed: func(file *vault.VaultFile, modified bool) {
			if modified {
				if verbose {
					fmt.Printf("✓ %s: Converted links from %s to %s format\n", file.RelativePath, fromFormat, toFormat)
				} else {
					fmt.Printf("✓ Processed: %s\n", file.RelativePath)
				}
			} else if verbose {
				fmt.Printf("- Skipped: %s (no links to convert)\n", file.RelativePath)
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