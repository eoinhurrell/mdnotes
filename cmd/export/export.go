package export

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eoinhurrell/mdnotes/internal/processor"
	"github.com/spf13/cobra"
)

// NewExportCommand creates the export command
func NewExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export <output-folder> [vault-path]",
		Short: "Export markdown files from vault to another location",
		Long: `Export markdown files from your Obsidian vault to another location.

This command copies markdown files while preserving directory structure and
optionally filtering files based on queries and processing links. It supports
parallel processing for improved performance on large vaults.

BASIC USAGE:
  # Export entire vault to backup folder
  mdnotes export ./backup

  # Export specific vault to output folder
  mdnotes export ./output /path/to/vault

QUERY FILTERING:
  # Export files matching query criteria
  mdnotes export ./blog --query "tags contains 'published'"
  mdnotes export ./work --query "folder = 'projects/' AND status = 'active'"
  mdnotes export ./recent --query "created >= '2024-01-01'"

LINK PROCESSING:
  # Convert external links to plain text (default)
  mdnotes export ./output --link-strategy remove

  # Use frontmatter URLs for external links
  mdnotes export ./output --link-strategy url

  # Skip link processing entirely
  mdnotes export ./output --process-links=false

ADVANCED FEATURES:
  # Include referenced assets (images, PDFs, etc.)
  mdnotes export ./complete --include-assets

  # Include files that link to exported files (recursive)
  mdnotes export ./network --with-backlinks

  # Normalize filenames for web compatibility
  mdnotes export ./web --slugify --flatten

PERFORMANCE OPTIONS:
  # Use parallel processing (auto-detects CPU count)
  mdnotes export ./output --parallel 0

  # Optimize memory usage for large vaults
  mdnotes export ./large-vault --optimize-memory

  # Set timeout for large exports
  mdnotes export ./huge-vault --timeout 30m

PREVIEW AND DEBUGGING:
  # Preview what would be exported without copying
  mdnotes export ./output --dry-run

  # Show detailed progress information
  mdnotes export ./output --verbose

  # Minimize output (errors only)
  mdnotes export ./output --quiet

ERROR HANDLING:
  The export command provides clear error messages for common issues:
  - Invalid query syntax with suggestions
  - Missing vault directories with helpful paths
  - Permission errors with recommended fixes
  - Output directory conflicts with resolution options

PERFORMANCE GUIDELINES:
  - For vaults with <100 files: ~1 second processing time
  - For vaults with <1000 files: ~10 seconds processing time
  - Use --parallel flag for vaults with >50 files
  - Use --optimize-memory for vaults with >1000 files
  - Large vaults benefit from SSD storage and adequate RAM`,
		Args: cobra.RangeArgs(1, 2),
		RunE: runExport,
	}

	// Add export-specific flags
	cmd.Flags().String("query", "", "Query to filter which files are exported (uses frontmatter query syntax)")
	cmd.Flags().StringSlice("ignore", []string{".obsidian/*", "*.tmp"}, "Ignore patterns for scanning vault")
	cmd.Flags().String("link-strategy", "remove", "Strategy for handling external links: 'remove' (convert to plain text) or 'url' (use frontmatter URL field)")
	cmd.Flags().Bool("process-links", true, "Process and rewrite links in exported files")
	cmd.Flags().Bool("include-assets", false, "Copy referenced assets (images, PDFs, etc.) to output directory")
	cmd.Flags().Bool("with-backlinks", false, "Include files that link to exported files (recursive)")
	cmd.Flags().Bool("slugify", false, "Convert filenames to URL-safe slugs")
	cmd.Flags().Bool("flatten", false, "Put all files in a single directory")
	cmd.Flags().Duration("timeout", 10*time.Minute, "Maximum time to wait for export to complete")
	cmd.Flags().Int("parallel", 0, "Number of parallel workers for file processing (0 = auto-detect)")
	cmd.Flags().Bool("optimize-memory", false, "Use memory-optimized processing for large vaults")

	return cmd
}

func runExport(cmd *cobra.Command, args []string) error {
	// Parse arguments
	outputPath := args[0]
	vaultPath := "."
	if len(args) > 1 {
		vaultPath = args[1]
	}

	// Get flags
	query, _ := cmd.Flags().GetString("query")
	ignorePatterns, _ := cmd.Flags().GetStringSlice("ignore")
	linkStrategy, _ := cmd.Flags().GetString("link-strategy")
	processLinks, _ := cmd.Flags().GetBool("process-links")
	includeAssets, _ := cmd.Flags().GetBool("include-assets")
	withBacklinks, _ := cmd.Flags().GetBool("with-backlinks")
	slugify, _ := cmd.Flags().GetBool("slugify")
	flatten, _ := cmd.Flags().GetBool("flatten")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	parallelWorkers, _ := cmd.Flags().GetInt("parallel")
	optimizeMemory, _ := cmd.Flags().GetBool("optimize-memory")
	dryRun, _ := cmd.Root().PersistentFlags().GetBool("dry-run")
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
	quiet, _ := cmd.Root().PersistentFlags().GetBool("quiet")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Override verbose if quiet is specified
	if quiet {
		verbose = false
	}

	// Comprehensive input validation
	if err := validateExportInputs(outputPath, vaultPath, query, linkStrategy, processLinks); err != nil {
		return NewExportError(ErrInvalidInput, err.Error())
	}

	// Validate link strategy (already done in validateExportInputs)
	// This is kept for backward compatibility but validation is now centralized

	// Validate and resolve paths
	vaultAbs, outputAbs, err := validateAndResolvePaths(vaultPath, outputPath, dryRun)
	if err != nil {
		return err
	}

	if verbose {
		fmt.Printf("Exporting from: %s\n", vaultAbs)
		fmt.Printf("Exporting to: %s\n", outputAbs)
		if query != "" {
			fmt.Printf("Query filter: %s\n", query)
		}
	}

	// Create export processor
	options := processor.ExportOptions{
		VaultPath:        vaultAbs,
		OutputPath:       outputAbs,
		Query:            query,
		IgnorePatterns:   ignorePatterns,
		DryRun:           dryRun,
		Verbose:          verbose,
		ProcessLinks:     processLinks,
		LinkStrategy:     linkStrategy,
		IncludeAssets:    includeAssets,
		WithBacklinks:    withBacklinks,
		Slugify:          slugify,
		Flatten:          flatten,
		ParallelWorkers:  parallelWorkers,
		OptimizeMemory:   optimizeMemory,
	}

	exportProcessor := processor.NewExportProcessor(options)

	// Perform the export operation with enhanced error handling
	result, err := exportProcessor.ProcessExport(ctx, options)
	if err != nil {
		return handleExportError(err, options)
	}

	// Display results with enhanced summary
	if dryRun {
		displayDryRunSummary(result, verbose)
	} else {
		if !quiet {
			displayExportSummary(result, outputAbs, verbose)
		}
	}

	return nil
}

// displayDryRunSummary shows what would be exported without doing it
func displayDryRunSummary(result *processor.ExportResult, verbose bool) {
	fmt.Printf("\nExport Summary (Dry Run)\n")
	fmt.Printf("========================\n\n")
	
	// Summary statistics
	fmt.Printf("Source:         %s\n", result.VaultPath)
	fmt.Printf("Destination:    %s\n", result.OutputPath)
	fmt.Printf("Files scanned:  %d\n", result.FilesScanned)
	fmt.Printf("Files selected: %d\n", result.FilesSelected)
	fmt.Printf("Total size:     %s\n", formatSize(result.TotalSize))
	
	if result.FilesSelected == 0 {
		fmt.Printf("\n⚠️  No files match the criteria. Nothing would be exported.\n")
		return
	}
	
	fmt.Printf("\n✅ Would export %d files (%s)\n", 
		result.FilesSelected, formatSize(result.TotalSize))
	
	// Show link processing statistics if any
	if result.ExternalLinksRemoved > 0 || result.ExternalLinksConverted > 0 || result.InternalLinksUpdated > 0 {
		fmt.Printf("\nLink processing (would be performed):\n")
		if result.ExternalLinksRemoved > 0 {
			fmt.Printf("  • External links removed: %d\n", result.ExternalLinksRemoved)
		}
		if result.ExternalLinksConverted > 0 {
			fmt.Printf("  • External links converted: %d\n", result.ExternalLinksConverted)
		}
		if result.InternalLinksUpdated > 0 {
			fmt.Printf("  • Internal links updated: %d\n", result.InternalLinksUpdated)
		}
		fmt.Printf("  • Files with links processed: %d\n", result.FilesWithLinksProcessed)
	}
	
	// Show asset processing statistics if any
	if result.AssetsCopied > 0 || result.AssetsMissing > 0 {
		fmt.Printf("\nAsset processing (would be performed):\n")
		if result.AssetsCopied > 0 {
			fmt.Printf("  • Assets to copy: %d\n", result.AssetsCopied)
		}
		if result.AssetsMissing > 0 {
			fmt.Printf("  • Missing assets: %d\n", result.AssetsMissing)
		}
	}
	
	// Show backlinks statistics if any
	if result.BacklinksIncluded > 0 {
		fmt.Printf("\nBacklinks (would be included):\n")
		fmt.Printf("  • Additional files via backlinks: %d\n", result.BacklinksIncluded)
	}
	
	// Show filename normalization statistics if any
	if result.FilesRenamed > 0 {
		fmt.Printf("\nFilename normalization (would be performed):\n")
		fmt.Printf("  • Files to rename: %d\n", result.FilesRenamed)
	}
	
	// Show individual files if verbose
	if verbose && len(result.SelectedFiles) > 0 {
		fmt.Printf("\nFiles that would be exported:\n")
		for _, file := range result.SelectedFiles {
			fmt.Printf("  ✓ %s\n", file)
		}
	}
}

// displayExportSummary shows the results of a completed export
func displayExportSummary(result *processor.ExportResult, outputPath string, verbose bool) {
	fmt.Printf("\nExport Summary\n")
	fmt.Printf("==============\n\n")
	
	fmt.Printf("✅ Export completed successfully\n")
	fmt.Printf("✅ Exported %d files (%s)\n", 
		result.FilesExported, formatSize(result.TotalSize))
	fmt.Printf("✅ Destination: %s\n", outputPath)
	fmt.Printf("⏱️  Processing time: %v\n", result.Duration.Round(time.Millisecond))
	
	// Show link processing statistics if any
	if result.ExternalLinksRemoved > 0 || result.ExternalLinksConverted > 0 || result.InternalLinksUpdated > 0 {
		fmt.Printf("\nLink processing:\n")
		if result.ExternalLinksRemoved > 0 {
			fmt.Printf("  • External links removed: %d\n", result.ExternalLinksRemoved)
		}
		if result.ExternalLinksConverted > 0 {
			fmt.Printf("  • External links converted: %d\n", result.ExternalLinksConverted)
		}
		if result.InternalLinksUpdated > 0 {
			fmt.Printf("  • Internal links updated: %d\n", result.InternalLinksUpdated)
		}
		fmt.Printf("  • Files with links processed: %d\n", result.FilesWithLinksProcessed)
	}
	
	// Show asset processing statistics if any
	if result.AssetsCopied > 0 || result.AssetsMissing > 0 {
		fmt.Printf("\nAsset processing:\n")
		if result.AssetsCopied > 0 {
			fmt.Printf("  • Assets copied: %d\n", result.AssetsCopied)
		}
		if result.AssetsMissing > 0 {
			fmt.Printf("  • Missing assets: %d\n", result.AssetsMissing)
		}
	}
	
	// Show backlinks statistics if any
	if result.BacklinksIncluded > 0 {
		fmt.Printf("\nBacklinks:\n")
		fmt.Printf("  • Additional files via backlinks: %d\n", result.BacklinksIncluded)
	}
	
	// Show filename normalization statistics if any
	if result.FilesRenamed > 0 {
		fmt.Printf("\nFilename normalization:\n")
		fmt.Printf("  • Files renamed: %d\n", result.FilesRenamed)
	}
	
	if verbose {
		fmt.Printf("\nProcessing details:\n")
		fmt.Printf("  Files scanned: %d\n", result.FilesScanned)
		fmt.Printf("  Files exported: %d\n", result.FilesExported)
		fmt.Printf("  Processing time: %v\n", result.Duration)
		
		// Show performance metrics if available
		if result.Performance != nil {
			fmt.Printf("\nPerformance metrics:\n")
			fmt.Printf("  Processing speed: %.1f files/second\n", result.Performance.FilesPerSecond)
			fmt.Printf("  Memory usage: %.1f MB\n", result.Performance.MemoryUsageMB)
			if result.Performance.ParallelWorkers > 0 {
				fmt.Printf("  Parallel workers: %d\n", result.Performance.ParallelWorkers)
			}
		}
		
		if len(result.SelectedFiles) > 0 {
			fmt.Printf("\nExported files:\n")
			for _, file := range result.SelectedFiles {
				fmt.Printf("  ✓ %s\n", file)
			}
		}
	}
}

// formatSize formats file size in a human-readable format
func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d bytes", bytes)
	}
}

// ExportErrorType represents different categories of export errors
type ExportErrorType int

const (
	ErrInvalidInput ExportErrorType = iota + 1
	ErrFileSystem
	ErrPermission
	ErrProcessing
	ErrQuery
	ErrCancellation
)

// ExportError represents a structured export error with type and user-friendly message
type ExportError struct {
	Type    ExportErrorType
	Message string
	Cause   error
}

func (e *ExportError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *ExportError) Unwrap() error {
	return e.Cause
}

// NewExportError creates a new export error
func NewExportError(errType ExportErrorType, message string) *ExportError {
	return &ExportError{
		Type:    errType,
		Message: message,
	}
}

// NewExportErrorWithCause creates a new export error with a cause
func NewExportErrorWithCause(errType ExportErrorType, message string, cause error) *ExportError {
	return &ExportError{
		Type:    errType,
		Message: message,
		Cause:   cause,
	}
}

// validateExportInputs performs comprehensive validation of export inputs
func validateExportInputs(outputPath, vaultPath, query, linkStrategy string, processLinks bool) error {
	// Validate output path is not empty
	if strings.TrimSpace(outputPath) == "" {
		return fmt.Errorf("output path cannot be empty")
	}
	
	// Validate vault path is not empty
	if strings.TrimSpace(vaultPath) == "" {
		return fmt.Errorf("vault path cannot be empty")
	}
	
	// Validate query syntax if provided
	if query != "" {
		if err := validateQuerySyntax(query); err != nil {
			return fmt.Errorf("invalid query syntax: %w", err)
		}
	}
	
	// Validate link strategy
	if processLinks && !processor.IsValidStrategy(linkStrategy) {
		return fmt.Errorf("invalid link strategy '%s' - valid options are: remove, url", linkStrategy)
	}
	
	// Validate output path safety (prevent writing to dangerous locations)
	if err := validateOutputPathSafety(outputPath); err != nil {
		return fmt.Errorf("unsafe output path: %w", err)
	}
	
	return nil
}

// validateQuerySyntax validates query syntax without executing it
func validateQuerySyntax(queryStr string) error {
	// Basic syntax validation for common issues
	if strings.Contains(queryStr, "\\") {
		return fmt.Errorf("backslashes are not supported in queries")
	}
	
	// Check for unmatched quotes
	quoteCount := strings.Count(queryStr, "\"")
	if quoteCount%2 != 0 {
		return fmt.Errorf("unmatched quotes in query")
	}
	
	// Check for empty query
	if strings.TrimSpace(queryStr) == "" {
		return fmt.Errorf("query cannot be empty")
	}
	
	return nil
}

// validateOutputPathSafety ensures output path is safe to write to
func validateOutputPathSafety(outputPath string) error {
	// Resolve to absolute path for safety checks
	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("cannot resolve output path: %w", err)
	}
	
	// Prevent writing to critical system directories
	unsafePaths := []string{
		"/bin", "/sbin", "/usr/bin", "/usr/sbin", "/etc", "/var/log", "/sys", "/proc",
		"/System", "/Library/System", "/Applications/Utilities", 
		"/Program Files", "/Windows", "/Windows/System32",
	}
	
	for _, unsafePath := range unsafePaths {
		if absPath == unsafePath || strings.HasPrefix(absPath+"/", unsafePath+"/") {
			return fmt.Errorf("cannot write to system directory: %s", absPath)
		}
	}
	
	return nil
}

// validateAndResolvePaths validates and resolves both vault and output paths
func validateAndResolvePaths(vaultPath, outputPath string, dryRun bool) (string, string, error) {
	// Resolve vault path
	vaultAbs, err := filepath.Abs(vaultPath)
	if err != nil {
		return "", "", NewExportErrorWithCause(ErrInvalidInput, 
			fmt.Sprintf("Invalid vault path '%s'", vaultPath), err)
	}
	
	// Check vault exists and is accessible
	vaultInfo, err := os.Stat(vaultAbs)
	if os.IsNotExist(err) {
		return "", "", NewExportError(ErrFileSystem, 
			fmt.Sprintf("Vault path does not exist: %s", vaultAbs))
	}
	if err != nil {
		return "", "", NewExportErrorWithCause(ErrPermission, 
			fmt.Sprintf("Cannot access vault path: %s", vaultAbs), err)
	}
	if !vaultInfo.IsDir() {
		return "", "", NewExportError(ErrInvalidInput, 
			fmt.Sprintf("Vault path is not a directory: %s", vaultAbs))
	}
	
	// Resolve output path
	outputAbs, err := filepath.Abs(outputPath)
	if err != nil {
		return "", "", NewExportErrorWithCause(ErrInvalidInput, 
			fmt.Sprintf("Invalid output path '%s'", outputPath), err)
	}
	
	// Check output path constraints
	if err := validateOutputPath(outputAbs, dryRun); err != nil {
		return "", "", err
	}
	
	return vaultAbs, outputAbs, nil
}

// validateOutputPath validates the output path constraints
func validateOutputPath(outputAbs string, dryRun bool) error {
	if info, err := os.Stat(outputAbs); err == nil {
		if !info.IsDir() {
			return NewExportError(ErrInvalidInput, 
				fmt.Sprintf("Output path exists and is not a directory: %s", outputAbs))
		}
		
		// Check if directory is empty (only for non-dry-run)
		if !dryRun {
			entries, err := os.ReadDir(outputAbs)
			if err != nil {
				return NewExportErrorWithCause(ErrPermission, 
					fmt.Sprintf("Cannot read output directory: %s", outputAbs), err)
			}
			if len(entries) > 0 {
				return NewExportError(ErrInvalidInput, 
					fmt.Sprintf("Output directory is not empty: %s\\n\\nUse --dry-run to preview or choose an empty directory", outputAbs))
			}
		}
	} else if !os.IsNotExist(err) {
		return NewExportErrorWithCause(ErrPermission, 
			fmt.Sprintf("Cannot access output path: %s", outputAbs), err)
	}
	
	return nil
}

// handleExportError provides enhanced error handling for export operations
func handleExportError(err error, options processor.ExportOptions) error {
	// Check for context cancellation
	if err == context.Canceled {
		return NewExportError(ErrCancellation, "Export operation was cancelled")
	}
	if err == context.DeadlineExceeded {
		return NewExportError(ErrCancellation, "Export operation timed out")
	}
	
	// Handle specific error types
	errMsg := err.Error()
	
	// Query parsing errors
	if strings.Contains(errMsg, "parsing query") || strings.Contains(errMsg, "query") {
		return NewExportErrorWithCause(ErrQuery, 
			fmt.Sprintf("Query error with '%s'", options.Query), err)
	}
	
	// File system errors
	if os.IsPermission(err) {
		return NewExportErrorWithCause(ErrPermission, 
			"Permission denied - check file/directory permissions", err)
	}
	if os.IsNotExist(err) {
		return NewExportErrorWithCause(ErrFileSystem, 
			"File or directory not found", err)
	}
	
	// Processing errors
	if strings.Contains(errMsg, "link processing") || strings.Contains(errMsg, "asset") {
		return NewExportErrorWithCause(ErrProcessing, 
			"Error during content processing", err)
	}
	
	// Generic processing error
	return NewExportErrorWithCause(ErrProcessing, 
		"Export processing failed", err)
}