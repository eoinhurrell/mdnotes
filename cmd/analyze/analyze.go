package analyze

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/eoinhurrell/mdnotes/internal/analyzer"
	"github.com/eoinhurrell/mdnotes/internal/config"
	"github.com/eoinhurrell/mdnotes/internal/errors"
	"github.com/eoinhurrell/mdnotes/internal/processor"
	"github.com/eoinhurrell/mdnotes/internal/selector"
	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// NewAnalyzeCommand creates the analyze command
func NewAnalyzeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "analyze",
		Aliases: []string{"a"},
		Short:   "Analyze vault statistics and health",
		Long:    `Generate comprehensive statistics and health reports for your vault`,
	}

	// Add subcommands
	cmd.AddCommand(newStatsCommand())
	cmd.AddCommand(newDuplicatesCommand())
	cmd.AddCommand(newHealthCommand())
	cmd.AddCommand(newLinksCommand())
	cmd.AddCommand(newContentCommand())
	cmd.AddCommand(newTrendsCommand())
	cmd.AddCommand(newInboxCommand())

	return cmd
}

func newStatsCommand() *cobra.Command {
	var (
		outputFormat string
		outputFile   string
	)

	cmd := &cobra.Command{
		Use:   "stats [vault-path]",
		Short: "Generate vault statistics",
		Long:  `Generate comprehensive statistics about your vault including file counts, frontmatter usage, and tag distribution`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vaultPath := "."
			if len(args) > 0 {
				vaultPath = args[0]
			}

			// Load configuration
			cfg, err := loadConfig(cmd)
			if err != nil {
				return errors.NewConfigError("", err.Error())
			}

			// Get file selection configuration from global flags
			mode, fileSelector, err := selector.GetGlobalSelectionConfig(cmd)
			if err != nil {
				return errors.WrapError(err, "file selection config", "")
			}

			// Merge config ignore patterns with global ignore patterns if needed
			if len(fileSelector.IgnorePatterns) == 0 {
				fileSelector = fileSelector.WithIgnorePatterns(cfg.Vault.IgnorePatterns)
			}

			// Select files using unified architecture
			selection, err := fileSelector.SelectFiles(vaultPath, mode)
			if err != nil {
				if os.IsNotExist(err) {
					return errors.NewFileNotFoundError(vaultPath,
						"Ensure the vault path exists and contains markdown files. Use 'ls' to verify the directory structure.")
				}
				if os.IsPermission(err) {
					return errors.NewPermissionError(vaultPath, "vault scanning")
				}
				return errors.WrapError(err, "vault scanning", vaultPath)
			}

			// Report any parsing errors encountered
			if len(selection.ParseErrors) > 0 {
				_, _ = fmt.Fprintf(os.Stderr, "Warning: %d files had parsing errors:\n", len(selection.ParseErrors))
				for _, parseErr := range selection.ParseErrors {
					_, _ = fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", parseErr.Path, parseErr.Error)
				}
				_, _ = fmt.Fprintf(os.Stderr, "\n")
			}

			files := selection.Files

			// Generate statistics
			ana := analyzer.NewAnalyzer()
			stats := ana.GenerateStats(files)

			// Output results
			if outputFormat == "json" {
				data, err := json.MarshalIndent(stats, "", "  ")
				if err != nil {
					return fmt.Errorf("marshaling JSON: %w", err)
				}

				if outputFile != "" {
					return os.WriteFile(outputFile, data, 0644)
				}
				fmt.Println(string(data))
			} else {
				output := formatStatsText(stats)
				if outputFile != "" {
					return os.WriteFile(outputFile, []byte(output), 0644)
				}
				_, _ = fmt.Print(output)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "Output format (text, json)")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (default: stdout)")

	return cmd
}

func newDuplicatesCommand() *cobra.Command {
	var (
		outputFormat  string
		minSimilarity float64
		duplicateType string
	)

	cmd := &cobra.Command{
		Use:   "duplicates [vault-path]",
		Short: "Find duplicate files",
		Long: `Find duplicate files in your vault including:
  - Content duplicates (identical file content)
  - Obsidian copies (files with ' 1', ' 2' suffixes)
  - Sync conflicts (syncthing, dropbox, etc.)
  
Example:
  mdnotes analyze duplicates --type obsidian
  mdnotes analyze duplicates --type sync-conflicts
  mdnotes analyze duplicates --type content`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vaultPath := "."
			if len(args) > 0 {
				vaultPath = args[0]
			}

			// Load configuration
			cfg, err := loadConfig(cmd)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			// Get file selection configuration from global flags
			mode, fileSelector, err := selector.GetGlobalSelectionConfig(cmd)
			if err != nil {
				return errors.WrapError(err, "file selection config", "")
			}

			// Merge config ignore patterns with global ignore patterns if needed
			if len(fileSelector.IgnorePatterns) == 0 {
				fileSelector = fileSelector.WithIgnorePatterns(cfg.Vault.IgnorePatterns)
			}

			// Select files using unified architecture
			selection, err := fileSelector.SelectFiles(vaultPath, mode)
			if err != nil {
				if os.IsNotExist(err) {
					return errors.NewFileNotFoundError(vaultPath,
						"Ensure the vault path exists and contains markdown files. Use 'ls' to verify the directory structure.")
				}
				if os.IsPermission(err) {
					return errors.NewPermissionError(vaultPath, "vault scanning")
				}
				return errors.WrapError(err, "vault scanning", vaultPath)
			}

			files := selection.Files

			// Report any parsing errors encountered
			if len(selection.ParseErrors) > 0 {
				_, _ = fmt.Fprintf(os.Stderr, "Warning: %d files had parsing errors:\n", len(selection.ParseErrors))
				for _, parseErr := range selection.ParseErrors {
					_, _ = fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", parseErr.Path, parseErr.Error)
				}
				_, _ = fmt.Fprintf(os.Stderr, "\n")
			}

			ana := analyzer.NewAnalyzer()

			// Find different types of duplicates based on flag
			switch duplicateType {
			case "obsidian":
				obsidianCopies := ana.FindObsidianCopies(files)
				if outputFormat == "json" {
					data, err := json.MarshalIndent(obsidianCopies, "", "  ")
					if err != nil {
						return fmt.Errorf("marshaling JSON: %w", err)
					}
					fmt.Println(string(data))
				} else {
					output := formatObsidianCopiesText(obsidianCopies)
					_, _ = fmt.Print(output)
				}
			case "sync-conflicts":
				syncConflicts := ana.FindSyncConflictFiles(files)
				if outputFormat == "json" {
					data, err := json.MarshalIndent(syncConflicts, "", "  ")
					if err != nil {
						return fmt.Errorf("marshaling JSON: %w", err)
					}
					fmt.Println(string(data))
				} else {
					output := formatSyncConflictsText(syncConflicts)
					_, _ = fmt.Print(output)
				}
			case "content":
				contentDuplicates := ana.FindContentDuplicates(files, analyzer.ExactMatch)
				if outputFormat == "json" {
					data, err := json.MarshalIndent(contentDuplicates, "", "  ")
					if err != nil {
						return fmt.Errorf("marshaling JSON: %w", err)
					}
					fmt.Println(string(data))
				} else {
					output := formatContentDuplicatesText(contentDuplicates)
					_, _ = fmt.Print(output)
				}
			default:
				// Show all types by default
				obsidianCopies := ana.FindObsidianCopies(files)
				syncConflicts := ana.FindSyncConflictFiles(files)
				contentDuplicates := ana.FindContentDuplicates(files, analyzer.ExactMatch)

				if outputFormat == "json" {
					result := map[string]interface{}{
						"obsidian_copies":    obsidianCopies,
						"sync_conflicts":     syncConflicts,
						"content_duplicates": contentDuplicates,
					}
					data, err := json.MarshalIndent(result, "", "  ")
					if err != nil {
						return fmt.Errorf("marshaling JSON: %w", err)
					}
					fmt.Println(string(data))
				} else {
					output := formatAllDuplicatesText(obsidianCopies, syncConflicts, contentDuplicates)
					_, _ = fmt.Print(output)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "Output format (text, json)")
	cmd.Flags().Float64Var(&minSimilarity, "similarity", 0.8, "Minimum similarity threshold (0.0-1.0)")
	cmd.Flags().StringVarP(&duplicateType, "type", "t", "all", "Type of duplicates to find (all, obsidian, sync-conflicts, content)")

	return cmd
}

func newHealthCommand() *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "health [vault-path]",
		Short: "Check vault health",
		Long:  `Generate a comprehensive health report for your vault`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vaultPath := "."
			if len(args) > 0 {
				vaultPath = args[0]
			}

			// Load configuration
			cfg, err := loadConfig(cmd)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			// Get file selection configuration from global flags
			mode, fileSelector, err := selector.GetGlobalSelectionConfig(cmd)
			if err != nil {
				return errors.WrapError(err, "file selection config", "")
			}

			// Merge config ignore patterns with global ignore patterns if needed
			if len(fileSelector.IgnorePatterns) == 0 {
				fileSelector = fileSelector.WithIgnorePatterns(cfg.Vault.IgnorePatterns)
			}

			// Select files using unified architecture
			selection, err := fileSelector.SelectFiles(vaultPath, mode)
			files := selection.Files
			if err != nil {
				if os.IsNotExist(err) {
					return errors.NewFileNotFoundError(vaultPath,
						"Ensure the vault path exists and contains markdown files. Use 'ls' to verify the directory structure.")
				}
				if os.IsPermission(err) {
					return errors.NewPermissionError(vaultPath, "vault scanning")
				}
				return errors.WrapError(err, "vault scanning", vaultPath)
			}

			// Report any parsing errors encountered
			if len(selection.ParseErrors) > 0 {
				_, _ = fmt.Fprintf(os.Stderr, "Warning: %d files had parsing errors:\n", len(selection.ParseErrors))
				for _, parseErr := range selection.ParseErrors {
					_, _ = fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", parseErr.Path, parseErr.Error)
				}
				_, _ = fmt.Fprintf(os.Stderr, "\n")
			}

			// Generate health report
			ana := analyzer.NewAnalyzer()
			stats := ana.GenerateStats(files)
			health := ana.GetHealthScore(stats)

			// Output results
			if outputFormat == "json" {
				data, err := json.MarshalIndent(health, "", "  ")
				if err != nil {
					return fmt.Errorf("marshaling JSON: %w", err)
				}
				fmt.Println(string(data))
			} else {
				output := formatHealthText(health)
				_, _ = fmt.Print(output)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "Output format (text, json)")

	return cmd
}

func loadConfig(cmd *cobra.Command) (*config.Config, error) {
	configPath, _ := cmd.Flags().GetString("config")

	if configPath != "" {
		return config.LoadConfigFromFile(configPath)
	}

	return config.LoadConfigWithFallback(config.GetDefaultConfigPaths())
}

func formatStatsText(stats analyzer.VaultStats) string {
	output := fmt.Sprintf(`Vault Statistics
================

Files:
  Total files: %d
  Files with frontmatter: %d
  Files without frontmatter: %d

Content:
  Total size: %d bytes
  Average file size: %.1f bytes

Frontmatter Fields:
`, stats.TotalFiles, stats.FilesWithFrontmatter, stats.FilesWithoutFrontmatter,
		stats.TotalSize, float64(stats.TotalSize)/float64(stats.TotalFiles))

	for field, count := range stats.FieldPresence {
		percentage := float64(count) / float64(stats.TotalFiles) * 100
		output += fmt.Sprintf("  %s: %d files (%.1f%%)\n", field, count, percentage)
	}

	if len(stats.TagDistribution) > 0 {
		output += "\nTop Tags:\n"
		for tag, count := range stats.TagDistribution {
			output += fmt.Sprintf("  #%s: %d files\n", tag, count)
		}
	}

	return output
}

func formatHealthText(health analyzer.HealthScore) string {
	return fmt.Sprintf(`Vault Health Report
==================

Health Level: %s
Score: %.1f/100

Issues Found:
%s

Suggestions:
%s
`, health.Level, health.Score,
		formatIssues(health.Issues),
		formatSuggestions(health.Suggestions))
}

func formatIssues(issues []string) string {
	if len(issues) == 0 {
		return "  No issues found. Great job!"
	}

	output := ""
	for _, issue := range issues {
		output += fmt.Sprintf("  - %s\n", issue)
	}
	return output
}

func formatSuggestions(suggestions []string) string {
	if len(suggestions) == 0 {
		return "  No suggestions at this time."
	}

	output := ""
	for _, suggestion := range suggestions {
		output += fmt.Sprintf("  - %s\n", suggestion)
	}
	return output
}

// newLinksCommand creates the links analysis command
func newLinksCommand() *cobra.Command {
	var (
		outputFormat   string
		showGraph      bool
		maxDepth       int
		minConnections int
	)

	cmd := &cobra.Command{
		Use:     "links [vault-path]",
		Aliases: []string{"l"},
		Short:   "Analyze link structure and connectivity",
		Long:    `Analyze the link structure of your vault, including connectivity graphs and orphaned files`,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vaultPath := "."
			if len(args) > 0 {
				vaultPath = args[0]
			}

			// Load configuration
			cfg, err := loadConfig(cmd)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			// Scan vault files
			scanner := vault.NewScanner(
				vault.WithIgnorePatterns(cfg.Vault.IgnorePatterns),
				vault.WithContinueOnErrors(),
			)
			files, err := scanner.Walk(vaultPath)
			if err != nil {
				return fmt.Errorf("scanning vault: %w", err)
			}

			// Generate link analysis
			ana := analyzer.NewAnalyzer()
			linkParser := processor.NewLinkParser()
			ana.SetLinkParser(linkParser)
			linkAnalysis := ana.AnalyzeLinks(files)

			// Output results
			if outputFormat == "json" {
				data, err := json.MarshalIndent(linkAnalysis, "", "  ")
				if err != nil {
					return fmt.Errorf("marshaling JSON: %w", err)
				}
				fmt.Println(string(data))
			} else {
				output := formatLinkAnalysisText(linkAnalysis, showGraph, maxDepth, minConnections)
				_, _ = fmt.Print(output)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "Output format (text, json)")
	cmd.Flags().BoolVar(&showGraph, "graph", false, "Show text-based link graph visualization")
	cmd.Flags().IntVar(&maxDepth, "depth", 3, "Maximum depth for graph visualization")
	cmd.Flags().IntVar(&minConnections, "min-connections", 1, "Minimum connections to show in graph")

	return cmd
}

// newContentCommand creates the content quality analysis command
func newContentCommand() *cobra.Command {
	var (
		outputFormat  string
		includeScores bool
		minScore      float64
	)

	cmd := &cobra.Command{
		Use:     "content [vault-path]",
		Aliases: []string{"c"},
		Short:   "Analyze content quality and completeness",
		Long:    `Analyze the quality of content in your vault, including completeness scores and suggestions`,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vaultPath := "."
			if len(args) > 0 {
				vaultPath = args[0]
			}

			// Check for verbose flag from global flags
			verbose, _ := cmd.Flags().GetBool("verbose")

			// Load configuration
			cfg, err := loadConfig(cmd)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			// Scan vault files
			scanner := vault.NewScanner(
				vault.WithIgnorePatterns(cfg.Vault.IgnorePatterns),
				vault.WithContinueOnErrors(),
			)
			files, err := scanner.Walk(vaultPath)
			if err != nil {
				return fmt.Errorf("scanning vault: %w", err)
			}

			// Generate content analysis
			ana := analyzer.NewAnalyzer()
			contentAnalysis := ana.AnalyzeContentQuality(files)

			// Output results
			if outputFormat == "json" {
				data, err := json.MarshalIndent(contentAnalysis, "", "  ")
				if err != nil {
					return fmt.Errorf("marshaling JSON: %w", err)
				}
				fmt.Println(string(data))
			} else {
				output := formatContentAnalysisText(contentAnalysis, includeScores, minScore, verbose)
				_, _ = fmt.Print(output)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "Output format (text, json, table, csv)")
	cmd.Flags().BoolVar(&includeScores, "scores", false, "Include individual file quality scores")
	cmd.Flags().Float64Var(&minScore, "min-score", 0.0, "Minimum quality score to display (0.0-100)")

	return cmd
}

// newTrendsCommand creates the vault growth trends analysis command
func newTrendsCommand() *cobra.Command {
	var (
		outputFormat string
		timespan     string
		granularity  string
	)

	cmd := &cobra.Command{
		Use:     "trends [vault-path]",
		Aliases: []string{"t"},
		Short:   "Analyze vault growth trends and patterns",
		Long:    `Analyze growth trends, writing patterns, and temporal statistics for your vault`,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vaultPath := "."
			if len(args) > 0 {
				vaultPath = args[0]
			}

			// Load configuration
			cfg, err := loadConfig(cmd)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			// Scan vault files
			scanner := vault.NewScanner(
				vault.WithIgnorePatterns(cfg.Vault.IgnorePatterns),
				vault.WithContinueOnErrors(),
			)
			files, err := scanner.Walk(vaultPath)
			if err != nil {
				return fmt.Errorf("scanning vault: %w", err)
			}

			// Generate trends analysis
			ana := analyzer.NewAnalyzer()
			trendsAnalysis := ana.AnalyzeTrends(files, timespan, granularity)

			// Output results
			if outputFormat == "json" {
				data, err := json.MarshalIndent(trendsAnalysis, "", "  ")
				if err != nil {
					return fmt.Errorf("marshaling JSON: %w", err)
				}
				fmt.Println(string(data))
			} else {
				output := formatTrendsAnalysisText(trendsAnalysis)
				_, _ = fmt.Print(output)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "Output format (text, json)")
	cmd.Flags().StringVar(&timespan, "timespan", "1y", "Time span to analyze (1w, 1m, 3m, 6m, 1y, all)")
	cmd.Flags().StringVar(&granularity, "granularity", "month", "Time granularity (day, week, month, quarter)")

	return cmd
}

// Formatting functions for the new analysis types

func formatLinkAnalysisText(analysis analyzer.LinkAnalysis, showGraph bool, maxDepth, minConnections int) string {
	output := fmt.Sprintf(`Link Structure Analysis
=======================

Overview:
  Total files: %d
  Files with outbound links: %d
  Files with inbound links: %d
  Orphaned files: %d
  Total links: %d
  Broken links: %d

Connectivity:
  Average outbound links per file: %.1f
  Average inbound links per file: %.1f
  Most connected file: %s (%d connections)
  Link density: %.3f

`, analysis.TotalFiles, analysis.FilesWithOutboundLinks, analysis.FilesWithInboundLinks,
		len(analysis.OrphanedFiles), analysis.TotalLinks, analysis.BrokenLinks,
		analysis.AvgOutboundLinks, analysis.AvgInboundLinks,
		analysis.MostConnectedFile, analysis.MaxConnections, analysis.LinkDensity)

	if len(analysis.OrphanedFiles) > 0 {
		output += "Orphaned Files:\n"
		for _, file := range analysis.OrphanedFiles {
			output += fmt.Sprintf("  - %s\n", file)
		}
		output += "\n"
	}

	if showGraph && len(analysis.LinkGraph) > 0 {
		output += "Link Graph (text visualization):\n"
		output += formatLinkGraph(analysis.LinkGraph, maxDepth, minConnections)
		output += "\n"
	}

	if len(analysis.CentralFiles) > 0 {
		output += "Most Central Files:\n"
		for i, file := range analysis.CentralFiles {
			if i >= 10 { // Show top 10
				break
			}
			output += fmt.Sprintf("  %d. %s (score: %.3f)\n", i+1, file.Path, file.CentralityScore)
		}
	}

	return output
}

func formatContentAnalysisText(analysis analyzer.ContentAnalysis, includeScores bool, minScore float64, verbose bool) string {
	output := fmt.Sprintf(`Zettelkasten Content Quality Analysis
====================================

Overall Quality Score: %.1f/100

Scoring based on Zettelkasten principles:
  1. Readability (Flesch-Kincaid Reading Ease)
  2. Link Density (outbound links per 100 words)
  3. Completeness (title, summary, word count)
  4. Atomicity (one concept per note)
  5. Recency (recently modified content)

Distribution:
  Excellent (90-100): %d files
  Good (75-89): %d files  
  Fair (60-74): %d files
  Poor (40-59): %d files
  Critical (0-39): %d files

Content Metrics:
  Average content length: %.0f characters
  Average word count: %.0f words
  Files with frontmatter: %d
  Files with headings: %d
  Files with links: %d

`, analysis.OverallScore,
		analysis.ScoreDistribution["excellent"], analysis.ScoreDistribution["good"],
		analysis.ScoreDistribution["fair"], analysis.ScoreDistribution["poor"],
		analysis.ScoreDistribution["critical"],
		analysis.AvgContentLength, analysis.AvgWordCount,
		analysis.FilesWithFrontmatter, analysis.FilesWithHeadings, analysis.FilesWithLinks)

	// Show worst-scoring files in the summary
	if len(analysis.FileScores) > 0 {
		worstFiles := getWorstScoringFiles(analysis.FileScores, 5)
		if len(worstFiles) > 0 {
			output += "⚠️  Files Needing Attention (lowest scores):\n"
			for i, score := range worstFiles {
				output += fmt.Sprintf("  %d. %.1f  %s\n", i+1, score.Score, score.Path)
				if len(score.SuggestedFixes) > 0 && len(score.SuggestedFixes[0]) > 0 {
					output += fmt.Sprintf("      → %s\n", score.SuggestedFixes[0])
				}
			}
			output += "\n"
		}
	}

	if len(analysis.QualityIssues) > 0 {
		output += "Quality Issues Found:\n"
		for _, issue := range analysis.QualityIssues {
			output += fmt.Sprintf("  - %s\n", issue)
		}
		output += "\n"
	}

	if len(analysis.Suggestions) > 0 {
		output += "Improvement Suggestions:\n"
		for _, suggestion := range analysis.Suggestions {
			output += fmt.Sprintf("  - %s\n", suggestion)
		}
		output += "\n"
	}

	// Show individual file scores
	if includeScores && len(analysis.FileScores) > 0 {
		if verbose {
			output += fmt.Sprintf("📊 Individual File Scores (showing files >= %.1f):\n", minScore)
			output += "====================================================================\n"
			output += "Score  File                                    Read Link Comp Atom Rec\n"
			output += "--------------------------------------------------------------------\n"
			for _, score := range analysis.FileScores {
				if score.Score >= minScore {
					// Truncate path if too long
					displayPath := score.Path
					if len(displayPath) > 35 {
						displayPath = "..." + displayPath[len(displayPath)-32:]
					}

					output += fmt.Sprintf("%-6.1f %-35s %4.0f %4.0f %4.0f %4.0f %4.0f\n",
						score.Score, displayPath,
						score.ReadabilityScore*100, score.LinkDensityScore*100,
						score.CompletenessScore*100, score.AtomicityScore*100, score.RecencyScore*100)

					if verbose && len(score.SuggestedFixes) > 0 {
						output += fmt.Sprintf("       Improvements: %s\n", strings.Join(score.SuggestedFixes, "; "))
					}
				}
			}
			output += "\nMetrics: Read=Readability, Link=Link Density, Comp=Completeness, Atom=Atomicity, Rec=Recency\n"
		} else {
			output += fmt.Sprintf("Individual File Scores (showing files >= %.1f):\n", minScore)
			output += "================================================================\n"
			for _, score := range analysis.FileScores {
				if score.Score >= minScore {
					output += fmt.Sprintf("%.1f  %s\n", score.Score, score.Path)
					if len(score.SuggestedFixes) > 0 {
						output += "     → " + strings.Join(score.SuggestedFixes, "; ") + "\n"
					}
					output += "\n"
				}
			}
		}
	}

	return output
}

// getWorstScoringFiles returns the N worst-scoring files
func getWorstScoringFiles(fileScores []analyzer.FileQualityScore, n int) []analyzer.FileQualityScore {
	if len(fileScores) == 0 {
		return nil
	}

	// Create a copy and sort by score ascending (worst first)
	scores := make([]analyzer.FileQualityScore, len(fileScores))
	copy(scores, fileScores)

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score < scores[j].Score
	})

	// Return first N (worst) files
	if len(scores) < n {
		n = len(scores)
	}

	return scores[:n]
}

func formatTrendsAnalysisText(analysis analyzer.TrendsAnalysis) string {
	output := fmt.Sprintf(`Vault Growth Trends Analysis
============================

Time Period: %s to %s
Total Duration: %s

Growth Statistics:
  Files created: %d
  Peak creation period: %s (%d files)
  Average files per %s: %.1f
  Growth rate: %.1f%% per %s

Activity Patterns:
  Most active day: %s
  Most active month: %s
  Writing streak: %d days
  Days with activity: %d/%d (%.1f%%)

`, analysis.StartDate.Format("2006-01-02"), analysis.EndDate.Format("2006-01-02"), analysis.TotalDuration,
		analysis.TotalFilesCreated, analysis.PeakPeriod, analysis.PeakFiles,
		analysis.Granularity, analysis.AvgFilesPerPeriod,
		analysis.GrowthRate, analysis.Granularity,
		analysis.MostActiveDay, analysis.MostActiveMonth,
		analysis.WritingStreak, analysis.ActiveDays, analysis.TotalDays, analysis.ActivityPercentage)

	if len(analysis.Timeline) > 0 {
		output += "Timeline (last 12 periods):\n"
		for i, point := range analysis.Timeline {
			if i >= 12 {
				break
			}
			output += fmt.Sprintf("  %s: %d files\n", point.Period, point.Count)
		}
		output += "\n"
	}

	if len(analysis.TagTrends) > 0 {
		output += "Trending Tags:\n"
		for tag, trend := range analysis.TagTrends {
			if trend.Count >= 3 { // Only show tags with at least 3 uses
				output += fmt.Sprintf("  #%s: %d files (%.1f%% growth)\n", tag, trend.Count, trend.GrowthRate)
			}
		}
	}

	return output
}

func formatLinkGraph(graph map[string][]string, maxDepth, minConnections int) string {
	output := ""
	visited := make(map[string]bool)

	// Find files with enough connections to display
	for file, connections := range graph {
		if len(connections) >= minConnections && !visited[file] {
			output += formatGraphNode(file, connections, graph, visited, 0, maxDepth, minConnections)
		}
	}

	if output == "" {
		output = "  No files meet the minimum connection criteria\n"
	}

	return output
}

func formatGraphNode(file string, connections []string, graph map[string][]string, visited map[string]bool, depth, maxDepth, minConnections int) string {
	if depth >= maxDepth || visited[file] {
		return ""
	}

	visited[file] = true
	indent := strings.Repeat("  ", depth)
	output := fmt.Sprintf("%s├─ %s (%d connections)\n", indent, file, len(connections))

	// Show connected files
	for i, connection := range connections {
		if i >= 5 { // Limit to 5 connections per node to avoid clutter
			output += fmt.Sprintf("%s│  └─ ... (%d more)\n", indent, len(connections)-5)
			break
		}

		connectionCount := len(graph[connection])
		if connectionCount >= minConnections {
			output += fmt.Sprintf("%s│  ├─ %s (%d)\n", indent, connection, connectionCount)
		} else {
			output += fmt.Sprintf("%s│  └─ %s\n", indent, connection)
		}
	}

	return output
}

// formatObsidianCopiesText formats Obsidian copy analysis results
func formatObsidianCopiesText(copies []analyzer.ObsidianCopy) string {
	if len(copies) == 0 {
		return "No Obsidian copy files found.\n"
	}

	output := fmt.Sprintf("Found %d Obsidian copy files:\n\n", len(copies))

	currentOriginal := ""
	for _, copy := range copies {
		if copy.OriginalFile != currentOriginal {
			currentOriginal = copy.OriginalFile
			output += fmt.Sprintf("Original: %s\n", copy.OriginalFile)
		}
		output += fmt.Sprintf("  └─ Copy %d: %s\n", copy.CopyNumber, copy.CopyFile)
	}

	output += "\n💡 Suggestion: Review these copies and consider merging or removing duplicates.\n"
	output += "   Use 'mdnotes rename' to organize files or manually review content.\n"

	return output
}

// formatSyncConflictsText formats sync conflict analysis results
func formatSyncConflictsText(conflicts []analyzer.SyncConflictFile) string {
	if len(conflicts) == 0 {
		return "No sync conflict files found.\n"
	}

	output := fmt.Sprintf("Found %d sync conflict files:\n\n", len(conflicts))

	// Group by conflict type
	conflictTypes := make(map[string][]analyzer.SyncConflictFile)
	for _, conflict := range conflicts {
		conflictTypes[conflict.ConflictType] = append(conflictTypes[conflict.ConflictType], conflict)
	}

	for conflictType, typeConflicts := range conflictTypes {
		output += fmt.Sprintf("\n%s conflicts (%d):\n", cases.Title(language.English).String(conflictType), len(typeConflicts))
		currentOriginal := ""
		for _, conflict := range typeConflicts {
			if conflict.OriginalFile != currentOriginal {
				currentOriginal = conflict.OriginalFile
				output += fmt.Sprintf("  Original: %s\n", conflict.OriginalFile)
			}
			output += fmt.Sprintf("    └─ Conflict: %s\n", conflict.ConflictFile)
		}
	}

	output += "\n💡 Suggestion: Review and resolve sync conflicts by comparing content.\n"
	output += "   Keep the most recent version and delete conflict files after verification.\n"

	return output
}

// formatContentDuplicatesText formats content duplicate analysis results
func formatContentDuplicatesText(duplicates []analyzer.ContentDuplicate) string {
	if len(duplicates) == 0 {
		return "No content duplicates found.\n"
	}

	output := fmt.Sprintf("Found %d content duplicate groups:\n\n", len(duplicates))

	for i, dup := range duplicates {
		output += fmt.Sprintf("Group %d (%d bytes, %d files):\n", i+1, dup.Size, dup.Count)
		for _, file := range dup.Files {
			output += fmt.Sprintf("  - %s\n", file)
		}
		output += "\n"
	}

	output += "💡 Suggestion: Review duplicate content and consider merging or removing redundant files.\n"

	return output
}

// formatAllDuplicatesText formats all duplicate types in a single report
func formatAllDuplicatesText(obsidianCopies []analyzer.ObsidianCopy, syncConflicts []analyzer.SyncConflictFile, contentDuplicates []analyzer.ContentDuplicate) string {
	output := "# Duplicate Analysis Report\n\n"

	// Summary
	totalIssues := len(obsidianCopies) + len(syncConflicts) + len(contentDuplicates)
	if totalIssues == 0 {
		return "✅ No duplicate files found. Your vault is clean!\n"
	}

	output += fmt.Sprintf("Found %d duplicate issues:\n", totalIssues)
	output += fmt.Sprintf("  - %d Obsidian copies\n", len(obsidianCopies))
	output += fmt.Sprintf("  - %d sync conflicts\n", len(syncConflicts))
	output += fmt.Sprintf("  - %d content duplicates\n\n", len(contentDuplicates))

	// Obsidian copies
	if len(obsidianCopies) > 0 {
		output += "## Obsidian Copies\n\n"
		output += formatObsidianCopiesText(obsidianCopies)
		output += "\n"
	}

	// Sync conflicts
	if len(syncConflicts) > 0 {
		output += "## Sync Conflicts\n\n"
		output += formatSyncConflictsText(syncConflicts)
		output += "\n"
	}

	// Content duplicates
	if len(contentDuplicates) > 0 {
		output += "## Content Duplicates\n\n"
		output += formatContentDuplicatesText(contentDuplicates)
	}

	return output
}

// newInboxCommand creates the INBOX triage analysis command
func newInboxCommand() *cobra.Command {
	var (
		outputFormat string
		sortBy       string
		minItems     int
	)

	cmd := &cobra.Command{
		Use:     "inbox [vault-path]",
		Aliases: []string{"i"},
		Short:   "Analyze INBOX content that needs processing",
		Long:    `Find content under INBOX headings and other temporary sections that need processing, sorted by content volume for prioritization`,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vaultPath := "."
			if len(args) > 0 {
				vaultPath = args[0]
			}

			// Load configuration
			cfg, err := loadConfig(cmd)
			if err != nil {
				return errors.NewConfigError("", err.Error())
			}

			// Get file selection configuration from global flags
			mode, fileSelector, err := selector.GetGlobalSelectionConfig(cmd)
			if err != nil {
				return errors.WrapError(err, "file selection config", "")
			}

			// Merge config ignore patterns with global ignore patterns if needed
			if len(fileSelector.IgnorePatterns) == 0 {
				fileSelector = fileSelector.WithIgnorePatterns(cfg.Vault.IgnorePatterns)
			}

			// Select files using unified architecture
			selection, err := fileSelector.SelectFiles(vaultPath, mode)
			if err != nil {
				if os.IsNotExist(err) {
					return errors.NewFileNotFoundError(vaultPath,
						"Ensure the vault path exists and contains markdown files. Use 'ls' to verify the directory structure.")
				}
				if os.IsPermission(err) {
					return errors.NewPermissionError(vaultPath, "vault scanning")
				}
				return errors.WrapError(err, "vault scanning", vaultPath)
			}

			// Report any parsing errors encountered
			if len(selection.ParseErrors) > 0 {
				_, _ = fmt.Fprintf(os.Stderr, "Warning: %d files had parsing errors:\n", len(selection.ParseErrors))
				for _, parseErr := range selection.ParseErrors {
					_, _ = fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", parseErr.Path, parseErr.Error)
				}
				_, _ = fmt.Fprintf(os.Stderr, "\n")
			}

			files := selection.Files

			// Generate inbox analysis using configured headings
			ana := analyzer.NewAnalyzer()
			inboxAnalysis := ana.AnalyzeInbox(files, cfg.Analysis.InboxHeadings, sortBy, minItems)

			// Output results
			if outputFormat == "json" {
				data, err := json.MarshalIndent(inboxAnalysis, "", "  ")
				if err != nil {
					return fmt.Errorf("marshaling JSON: %w", err)
				}
				fmt.Println(string(data))
			} else {
				output := formatInboxAnalysisText(inboxAnalysis)
				_, _ = fmt.Print(output)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "Output format (text, json)")
	cmd.Flags().StringVar(&sortBy, "sort", "size", "Sort inbox items by: size, count, urgency")
	cmd.Flags().IntVar(&minItems, "min-items", 1, "Minimum number of items to show section")

	return cmd
}

// formatInboxAnalysisText formats inbox analysis results as text
func formatInboxAnalysisText(analysis *analyzer.InboxAnalysis) string {
	var output strings.Builder

	output.WriteString("INBOX Triage Analysis\n")
	output.WriteString("====================\n\n")

	if len(analysis.InboxSections) == 0 {
		output.WriteString("No INBOX sections found!\n\n")
		output.WriteString("This is great - your vault appears to be well-organized without pending tasks.\n")
		return output.String()
	}

	output.WriteString(fmt.Sprintf("Found %d INBOX sections with pending content:\n\n", len(analysis.InboxSections)))

	// Summary statistics
	totalItems := 0
	totalSize := 0
	for _, section := range analysis.InboxSections {
		totalItems += section.ItemCount
		totalSize += section.ContentSize
	}

	output.WriteString(fmt.Sprintf("Total items to process: %d\n", totalItems))
	output.WriteString(fmt.Sprintf("Total content size: %d characters\n\n", totalSize))

	// Priority recommendations
	output.WriteString("Priority Recommendations:\n")
	output.WriteString("------------------------\n")
	if len(analysis.InboxSections) > 0 {
		output.WriteString(fmt.Sprintf("🔥 Start with: %s (%d items, %d chars)\n",
			analysis.InboxSections[0].File,
			analysis.InboxSections[0].ItemCount,
			analysis.InboxSections[0].ContentSize))
	}
	output.WriteString("\n")

	// Detailed sections
	output.WriteString("Inbox Sections by Priority:\n")
	output.WriteString("---------------------------\n")
	for i, section := range analysis.InboxSections {
		priority := "📝"
		if i == 0 {
			priority = "🔥"
		} else if i < 3 {
			priority = "⚡"
		}

		output.WriteString(fmt.Sprintf("%s %s\n", priority, section.File))
		output.WriteString(fmt.Sprintf("   Heading: %s\n", section.Heading))
		output.WriteString(fmt.Sprintf("   Items: %d | Size: %d chars | Urgency: %s\n",
			section.ItemCount, section.ContentSize, section.UrgencyLevel))

		if len(section.ActionSuggestions) > 0 {
			output.WriteString("   Suggestions: ")
			output.WriteString(strings.Join(section.ActionSuggestions, ", "))
			output.WriteString("\n")
		}
		output.WriteString("\n")
	}

	// Action plan
	if len(analysis.InboxSections) > 0 {
		output.WriteString("Suggested Action Plan:\n")
		output.WriteString("---------------------\n")
		output.WriteString("1. Process the 🔥 high-priority section first\n")
		output.WriteString("2. Break down large items into smaller, actionable tasks\n")
		output.WriteString("3. Move completed items to appropriate permanent locations\n")
		output.WriteString("4. Archive or delete obsolete items\n")
		output.WriteString("5. Re-run this analysis to track progress\n")
	}

	return output.String()
}
