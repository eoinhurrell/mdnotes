package headings

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/eoinhurrell/mdnotes/internal/processor"
	"github.com/eoinhurrell/mdnotes/internal/vault"
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

	// Analyze headings
	headingProcessor := processor.NewHeadingProcessor()
	totalIssues := 0

	for _, file := range files {
		analysis := headingProcessor.Analyze(file)
		if len(analysis.Issues) > 0 {
			totalIssues += len(analysis.Issues)
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
		} else if verbose {
			fmt.Printf("✓ %s: valid heading structure\n", file.RelativePath)
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
		Use:   "fix [path]",
		Short: "Fix heading structure issues",
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
		IgnorePatterns: ignorePatterns,
		ProcessFile: func(file *vault.VaultFile) (bool, error) {
			originalBody := file.Body
			
			if err := headingProcessor.Fix(file, rules); err != nil {
				if verbose {
					fmt.Printf("✗ %s: Error fixing headings: %v\n", file.RelativePath, err)
				}
				return false, nil // Don't fail the entire operation
			}

			return file.Body != originalBody, nil
		},
		OnFileProcessed: func(file *vault.VaultFile, modified bool) {
			if modified {
				if verbose {
					fmt.Printf("✓ %s: Fixed heading structure\n", file.RelativePath)
				} else {
					fmt.Printf("✓ Processed: %s\n", file.RelativePath)
				}
			} else if verbose {
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