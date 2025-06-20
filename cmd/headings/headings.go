package headings

import (
	"fmt"
	"strings"

	"github.com/eoinhurrell/mdnotes/internal/processor"
	"github.com/eoinhurrell/mdnotes/internal/vault"
	"github.com/spf13/cobra"
)

// NewHeadingsCommand creates the headings command
func NewHeadingsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "headings",
		Short: "Manage heading structure in markdown files",
		Long:  "Commands for analyzing and fixing heading structure in Obsidian notes",
	}

	cmd.AddCommand(NewAnalyzeCommand())
	cmd.AddCommand(NewFixCommand())
	cmd.AddCommand(NewCleanCommand())

	return cmd
}

// NewAnalyzeCommand creates the headings analyze command
func NewAnalyzeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze [path]",
		Short: "Analyze heading structure and report issues",
		Long: `Analyze heading structure in markdown files and report issues like:
- Multiple H1 headings
- H1 not matching title field
- Skipped heading levels`,
		Args: cobra.ExactArgs(1),
		RunE: runAnalyze,
	}

	cmd.Flags().StringSlice("ignore", []string{".obsidian/*", "*.tmp"}, "Ignore patterns")

	return cmd
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	path := args[0]

	// Get flags
	ignorePatterns, _ := cmd.Flags().GetStringSlice("ignore")
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
	quiet, _ := cmd.Root().PersistentFlags().GetBool("quiet")

	// Override verbose if quiet is specified
	if quiet {
		verbose = false
	}

	// Scan files
	scanner := vault.NewScanner(vault.WithIgnorePatterns(ignorePatterns))
	files, err := scanner.Walk(path)
	if err != nil {
		return fmt.Errorf("scanning directory: %w", err)
	}

	if len(files) == 0 {
		if !quiet {
			fmt.Println("No markdown files found")
		}
		return nil
	}

	// Analyze headings
	headingProcessor := processor.NewHeadingProcessor()
	totalIssues := 0

	for _, file := range files {
		analysis := headingProcessor.Analyze(file)
		if len(analysis.Issues) > 0 {
			totalIssues += len(analysis.Issues)
			if verbose {
				fmt.Printf("Examining: %s - Found %d heading issues\n", file.RelativePath, len(analysis.Issues))
			}
			if !quiet {
				fmt.Printf("✗ %s:\n", file.RelativePath)
				for _, issue := range analysis.Issues {
					fmt.Printf("  - Line %d: %s", issue.Line, formatIssue(issue))
					if issue.Expected != "" {
						fmt.Printf(" (expected: %s", issue.Expected)
						if issue.Actual != "" {
							fmt.Printf(", actual: %s", issue.Actual)
						}
						fmt.Printf(")")
					}
					fmt.Println()
				}
			}
		} else {
			if verbose {
				fmt.Printf("Examining: %s - Valid heading structure\n", file.RelativePath)
			}
		}
	}

	if totalIssues > 0 {
		fmt.Printf("\nAnalysis completed: %d issues found in %d files\n", totalIssues, len(files))
	} else {
		fmt.Printf("\nAnalysis completed: all %d files have valid heading structure\n", len(files))
	}

	return nil
}

// NewFixCommand creates the headings fix command
func NewFixCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "fix [path]",
		Aliases: []string{"f"},
		Short:   "Fix heading structure issues",
		Long: `Fix heading structure issues in markdown files according to rules:
- Ensure H1 matches title field
- Convert multiple H1s to H2s
- Fix skipped heading levels`,
		Args: cobra.ExactArgs(1),
		RunE: runFix,
	}

	cmd.Flags().Bool("ensure-h1-title", true, "Ensure H1 matches title field")
	cmd.Flags().Bool("single-h1", true, "Convert extra H1s to H2s")
	cmd.Flags().Bool("fix-sequence", false, "Fix skipped heading levels")
	cmd.Flags().StringSlice("ignore", []string{".obsidian/*", "*.tmp"}, "Ignore patterns")

	return cmd
}

func runFix(cmd *cobra.Command, args []string) error {
	path := args[0]

	// Get flags
	ensureH1Title, _ := cmd.Flags().GetBool("ensure-h1-title")
	singleH1, _ := cmd.Flags().GetBool("single-h1")
	fixSequence, _ := cmd.Flags().GetBool("fix-sequence")
	ignorePatterns, _ := cmd.Flags().GetStringSlice("ignore")
	dryRun, _ := cmd.Root().PersistentFlags().GetBool("dry-run")
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
	quiet, _ := cmd.Root().PersistentFlags().GetBool("quiet")

	// Override verbose if quiet is specified
	if quiet {
		verbose = false
	}

	// Create heading rules
	rules := processor.HeadingRules{
		EnsureH1Title: ensureH1Title,
		SingleH1:      singleH1,
		FixSequence:   fixSequence,
	}

	// Create processor
	headingProcessor := processor.NewHeadingProcessor()

	// Setup file processor
	fileProcessor := &processor.FileProcessor{
		DryRun:         dryRun,
		Verbose:        verbose,
		Quiet:          quiet,
		IgnorePatterns: ignorePatterns,
		ProcessFile: func(file *vault.VaultFile) (bool, error) {
			originalBody := file.Body

			if err := headingProcessor.Fix(file, rules); err != nil {
				if verbose {
					fmt.Printf("Examining: %s - Error fixing headings: %v\n", file.RelativePath, err)
				}
				return false, nil // Don't fail the entire operation
			}

			modified := file.Body != originalBody
			if verbose {
				if modified {
					fmt.Printf("Examining: %s - Fixed heading structure\n", file.RelativePath)
				} else {
					fmt.Printf("Examining: %s - No changes needed\n", file.RelativePath)
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

	// Print summary
	fileProcessor.PrintSummary(result)

	return nil
}

// NewCleanCommand creates the headings clean command
func NewCleanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "clean [path]",
		Aliases: []string{"cl"},
		Short:   "Clean headings for Obsidian compatibility",
		Long: `Clean headings to ensure Obsidian compatibility:
- Convert [X] to <X> in headings
- Convert headings containing links to list items`,
		Args: cobra.ExactArgs(1),
		RunE: runClean,
	}

	cmd.Flags().Bool("square-brackets", true, "Enable square bracket cleaning")
	cmd.Flags().Bool("link-headers", true, "Enable link header conversion")
	cmd.Flags().StringSlice("ignore", []string{".obsidian/*", "*.tmp"}, "Ignore patterns")

	return cmd
}

func runClean(cmd *cobra.Command, args []string) error {
	path := args[0]

	// Get flags
	squareBrackets, _ := cmd.Flags().GetBool("square-brackets")
	linkHeaders, _ := cmd.Flags().GetBool("link-headers")
	ignorePatterns, _ := cmd.Flags().GetStringSlice("ignore")
	dryRun, _ := cmd.Root().PersistentFlags().GetBool("dry-run")
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
	quiet, _ := cmd.Root().PersistentFlags().GetBool("quiet")

	// Override verbose if quiet is specified
	if quiet {
		verbose = false
	}

	// Create clean rules
	rules := processor.CleanRules{
		SquareBrackets: squareBrackets,
		LinkHeaders:    linkHeaders,
	}

	// Create processor
	headingProcessor := processor.NewHeadingProcessor()

	// Track total statistics
	totalSquareBracketsFixed := 0
	totalLinkHeadersConverted := 0

	// Setup file processor
	fileProcessor := &processor.FileProcessor{
		DryRun:         dryRun,
		Verbose:        verbose,
		Quiet:          quiet,
		IgnorePatterns: ignorePatterns,
		ProcessFile: func(file *vault.VaultFile) (bool, error) {
			originalBody := file.Body

			stats, err := headingProcessor.Clean(file, rules)
			if err != nil {
				if verbose {
					fmt.Printf("Examining: %s - Error cleaning headings: %v\n", file.RelativePath, err)
				}
				return false, nil // Don't fail the entire operation
			}

			// Accumulate statistics
			totalSquareBracketsFixed += stats.SquareBracketsFixed
			totalLinkHeadersConverted += stats.LinkHeadersConverted

			modified := file.Body != originalBody
			if verbose {
				if modified {
					changes := []string{}
					if stats.SquareBracketsFixed > 0 {
						changes = append(changes, fmt.Sprintf("Fixed %d square brackets", stats.SquareBracketsFixed))
					}
					if stats.LinkHeadersConverted > 0 {
						changes = append(changes, fmt.Sprintf("converted %d link headers", stats.LinkHeadersConverted))
					}
					fmt.Printf("Examining: %s - %s\n", file.RelativePath, strings.Join(changes, ", "))
				} else {
					fmt.Printf("Examining: %s - No changes needed\n", file.RelativePath)
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

	// Print summary with cleaning statistics
	if !quiet {
		if dryRun {
			fmt.Printf("\nDry run summary: %d files would be examined, %d would be modified\n", result.TotalFiles, result.ProcessedFiles)
		} else {
			fmt.Printf("\nSummary: %d files examined, %d modified\n", result.TotalFiles, result.ProcessedFiles)
		}
		
		if totalSquareBracketsFixed > 0 || totalLinkHeadersConverted > 0 {
			if totalSquareBracketsFixed > 0 {
				fmt.Printf("  - Square brackets fixed: %d\n", totalSquareBracketsFixed)
			}
			if totalLinkHeadersConverted > 0 {
				fmt.Printf("  - Link headers converted: %d\n", totalLinkHeadersConverted)
			}
		}
	}

	return nil
}

func formatIssue(issue processor.HeadingIssue) string {
	switch issue.Type {
	case "multiple_h1":
		return "Multiple H1 headings found"
	case "h1_title_mismatch":
		return "H1 doesn't match title field"
	case "skipped_level":
		return "Skipped heading level"
	case "missing_h1":
		return "Missing H1 heading"
	default:
		return issue.Type
	}
}
