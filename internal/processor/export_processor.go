package processor

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/eoinhurrell/mdnotes/internal/query"
	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// ExportProgressReporter implements progress reporting for export operations
type ExportProgressReporter struct {
	reporter ProgressReporter
	quiet    bool
	verbose  bool
}

// NewExportProgressReporter creates a new export progress reporter
func NewExportProgressReporter(quiet, verbose bool) *ExportProgressReporter {
	var reporter ProgressReporter
	if quiet {
		reporter = NewSilentProgress()
	} else {
		reporter = NewTerminalProgress()
	}

	return &ExportProgressReporter{
		reporter: reporter,
		quiet:    quiet,
		verbose:  verbose,
	}
}

func (epr *ExportProgressReporter) StartPhase(total int, message string) {
	if !epr.quiet {
		fmt.Printf("%s\n", message)
	}
	epr.reporter.Start(total)
}

func (epr *ExportProgressReporter) UpdatePhase(current int, message string) {
	if epr.verbose {
		epr.reporter.Update(current, message)
	} else {
		epr.reporter.Update(current, "")
	}
}

func (epr *ExportProgressReporter) FinishPhase(message string) {
	epr.reporter.Finish()
	if !epr.quiet && message != "" {
		fmt.Printf("%s\n", message)
	}
}

// ExportProcessor handles exporting markdown files from a vault
type ExportProcessor struct {
	scanner  *vault.Scanner
	verbose  bool
	progress *ExportProgressReporter
}

// ExportOptions contains configuration for export operations
type ExportOptions struct {
	VaultPath       string
	OutputPath      string
	Query           string
	IgnorePatterns  []string
	DryRun          bool
	Verbose         bool
	ProcessLinks    bool
	LinkStrategy    string
	IncludeAssets   bool
	WithBacklinks   bool
	Slugify         bool
	Flatten         bool
	ParallelWorkers int  // Number of parallel workers (0 = auto-detect)
	OptimizeMemory  bool // Use memory-optimized processing
}

// ExportResult contains the results of an export operation
type ExportResult struct {
	VaultPath     string
	OutputPath    string
	FilesScanned  int
	FilesSelected int
	FilesExported int
	TotalSize     int64
	SelectedFiles []string
	Duration      time.Duration
	// Link processing statistics
	ExternalLinksRemoved    int
	ExternalLinksConverted  int
	InternalLinksUpdated    int
	FilesWithLinksProcessed int
	// Asset processing statistics
	AssetsCopied  int
	AssetsMissing int
	// Backlinks statistics
	BacklinksIncluded int
	// Filename processing statistics
	FilesRenamed int
	// Performance metrics
	Performance *PerformanceMetrics
}

// NewExportProcessor creates a new export processor
func NewExportProcessor(options ExportOptions) *ExportProcessor {
	scanner := vault.NewScanner(vault.WithIgnorePatterns(options.IgnorePatterns))

	return &ExportProcessor{
		scanner:  scanner,
		verbose:  options.Verbose,
		progress: NewExportProgressReporter(false, options.Verbose), // quiet=false for now
	}
}

// ProcessExport performs the complete export operation
func (ep *ExportProcessor) ProcessExport(ctx context.Context, options ExportOptions) (*ExportResult, error) {
	startTime := time.Now()

	// Create result object
	result := &ExportResult{
		VaultPath:  options.VaultPath,
		OutputPath: options.OutputPath,
	}

	// Step 1: Scan vault for markdown files
	ep.progress.StartPhase(0, "🔍 Scanning vault for markdown files...")
	files, err := ep.scanVaultFiles(ctx, options.VaultPath)
	if err != nil {
		return nil, fmt.Errorf("scanning vault: %w", err)
	}
	result.FilesScanned = len(files)
	ep.progress.FinishPhase(fmt.Sprintf("✅ Scanned %d files in vault", result.FilesScanned))

	// Step 2: Filter files based on query (if provided)
	selectedFiles := files
	if options.Query != "" {
		ep.progress.StartPhase(0, fmt.Sprintf("📋 Filtering files with query: %s", options.Query))
		selectedFiles, err = ep.filterFilesByQuery(files, options.Query)
		if err != nil {
			return nil, fmt.Errorf("filtering files by query: %w", err)
		}
		ep.progress.FinishPhase(fmt.Sprintf("✅ Selected %d files after filtering", len(selectedFiles)))
	}
	result.FilesSelected = len(selectedFiles)

	// Step 3: Expand with backlinks (if requested)
	if options.WithBacklinks {
		ep.progress.StartPhase(0, "🔗 Discovering backlinks...")
		backlinkResult, err := ep.expandWithBacklinks(ctx, selectedFiles, files, options)
		if err != nil {
			return nil, fmt.Errorf("expanding with backlinks: %w", err)
		}
		selectedFiles = append(selectedFiles, backlinkResult.BacklinkFiles...)
		result.BacklinksIncluded = backlinkResult.TotalBacklinks
		ep.progress.FinishPhase(fmt.Sprintf("✅ Added %d backlink files", result.BacklinksIncluded))
	}

	// Step 4: Normalize filenames (if requested)
	var filenameMap map[string]string
	if options.Slugify || options.Flatten {
		normalizationResult, err := ep.normalizeFilenames(selectedFiles, options)
		if err != nil {
			return nil, fmt.Errorf("normalizing filenames: %w", err)
		}
		filenameMap = normalizationResult.FileMap
		result.FilesRenamed = normalizationResult.RenamedFiles

		if ep.verbose && result.FilesRenamed > 0 {
			fmt.Printf("Renamed %d files during normalization\n", result.FilesRenamed)
		}
	} else {
		// Create identity mapping when no normalization is requested
		filenameMap = make(map[string]string)
		for _, file := range selectedFiles {
			filenameMap[file.RelativePath] = file.RelativePath
		}
	}

	// Step 5: Calculate total size and collect file paths
	result.TotalSize = ep.calculateTotalSize(selectedFiles)
	result.SelectedFiles = make([]string, len(selectedFiles))
	for i, file := range selectedFiles {
		result.SelectedFiles[i] = filenameMap[file.RelativePath] // Use normalized paths
	}

	// Step 6: Copy files (if not dry run)
	if !options.DryRun {
		ep.progress.StartPhase(len(selectedFiles), "📄 Copying files...")

		// Determine if we should use parallel processing
		useParallel := len(selectedFiles) >= 10 && (options.ParallelWorkers > 1 || options.ParallelWorkers == 0)

		if options.ProcessLinks {
			// Copy files with link processing and filename normalization
			var linkResult *LinkProcessingResult
			if useParallel && !options.OptimizeMemory {
				linkResult, err = ep.copyFilesWithLinkProcessingParallel(ctx, selectedFiles, files, filenameMap, options)
			} else {
				linkResult, err = ep.copyFilesWithLinkProcessingAndNormalization(ctx, selectedFiles, files, filenameMap, options)
			}
			if err != nil {
				return nil, fmt.Errorf("copying files with link processing: %w", err)
			}
			// Update result with link processing statistics
			result.ExternalLinksRemoved = linkResult.ExternalLinksRemoved
			result.ExternalLinksConverted = linkResult.ExternalLinksConverted
			result.InternalLinksUpdated = linkResult.InternalLinksUpdated
			result.FilesWithLinksProcessed = linkResult.FilesWithLinksProcessed
		} else {
			// Copy files with filename normalization only
			if useParallel && !options.OptimizeMemory {
				err = ep.copyFilesWithNormalizationParallel(ctx, selectedFiles, filenameMap, options)
			} else {
				err = ep.copyFilesWithNormalization(ctx, selectedFiles, filenameMap, options)
			}
			if err != nil {
				return nil, fmt.Errorf("copying files: %w", err)
			}
		}
		result.FilesExported = len(selectedFiles)
		ep.progress.FinishPhase(fmt.Sprintf("✅ Copied %d files", result.FilesExported))

		// Step 7: Process assets (if requested and not dry run)
		if options.IncludeAssets {
			ep.progress.StartPhase(0, "🖼️ Processing assets...")
			assetResult, err := ep.processAssets(ctx, selectedFiles, options)
			if err != nil {
				return nil, fmt.Errorf("processing assets: %w", err)
			}
			result.AssetsCopied = assetResult.AssetsCopied
			result.AssetsMissing = assetResult.AssetsMissing
			ep.progress.FinishPhase(fmt.Sprintf("✅ Processed %d assets", result.AssetsCopied))
		}
	} else {
		// For dry run, analyze what would be processed

		// Analyze backlinks for dry run
		if options.WithBacklinks {
			backlinkResult, err := ep.expandWithBacklinks(ctx, selectedFiles, files, options)
			if err != nil {
				return nil, fmt.Errorf("analyzing backlinks: %w", err)
			}
			selectedFiles = append(selectedFiles, backlinkResult.BacklinkFiles...)
			result.BacklinksIncluded = backlinkResult.TotalBacklinks
		}

		// Analyze filename normalization for dry run
		if options.Slugify || options.Flatten {
			normalizationResult, err := ep.normalizeFilenames(selectedFiles, options)
			if err != nil {
				return nil, fmt.Errorf("analyzing filename normalization: %w", err)
			}
			filenameMap = normalizationResult.FileMap
			result.FilesRenamed = normalizationResult.RenamedFiles
		}

		if options.ProcessLinks {
			linkResult := ep.analyzeLinkProcessing(selectedFiles, files, options)
			result.ExternalLinksRemoved = linkResult.ExternalLinksRemoved
			result.ExternalLinksConverted = linkResult.ExternalLinksConverted
			result.InternalLinksUpdated = linkResult.InternalLinksUpdated
			result.FilesWithLinksProcessed = linkResult.FilesWithLinksProcessed
		}

		// For dry run with assets, analyze what would be copied
		if options.IncludeAssets {
			assetResult := ep.analyzeAssetProcessing(selectedFiles, options)
			result.AssetsCopied = assetResult.AssetsCopied
			result.AssetsMissing = assetResult.AssetsMissing
		}
	}

	result.Duration = time.Since(startTime)

	// Calculate performance metrics
	result.Performance = &PerformanceMetrics{}
	if result.FilesExported > 0 {
		*result.Performance = CalculatePerformanceMetrics(
			result.FilesExported,
			result.Duration.Seconds(),
			options.ParallelWorkers,
		)
	}

	return result, nil
}

// scanVaultFiles scans the vault and returns all markdown files
func (ep *ExportProcessor) scanVaultFiles(ctx context.Context, vaultPath string) ([]*vault.VaultFile, error) {
	var files []*vault.VaultFile

	err := ep.scanner.WalkWithCallback(vaultPath, func(file *vault.VaultFile) error {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		files = append(files, file)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

// filterFilesByQuery filters files based on the provided query
func (ep *ExportProcessor) filterFilesByQuery(files []*vault.VaultFile, queryStr string) ([]*vault.VaultFile, error) {
	if queryStr == "" {
		return files, nil
	}

	// Parse the query using the existing query engine
	parser := query.NewParser(queryStr)
	expression, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("parsing query: %w", err)
	}

	var filteredFiles []*vault.VaultFile
	for _, file := range files {
		// Evaluate the expression against this file
		matches := expression.Evaluate(file)

		if matches {
			filteredFiles = append(filteredFiles, file)
		}
	}

	return filteredFiles, nil
}

// calculateTotalSize calculates the total size of all selected files
func (ep *ExportProcessor) calculateTotalSize(files []*vault.VaultFile) int64 {
	var totalSize int64
	for _, file := range files {
		if info, err := os.Stat(file.Path); err == nil {
			totalSize += info.Size()
		}
	}
	return totalSize
}

// copyFiles copies the selected files to the output directory
func (ep *ExportProcessor) copyFiles(ctx context.Context, files []*vault.VaultFile, options ExportOptions) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(options.OutputPath, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	for _, file := range files {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Determine output file path
		outputFilePath := filepath.Join(options.OutputPath, file.RelativePath)

		// Create output directory for this file
		outputDir := filepath.Dir(outputFilePath)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("creating output directory %s: %w", outputDir, err)
		}

		// Copy the file
		err := ep.copyFile(file.Path, outputFilePath)
		if err != nil {
			return fmt.Errorf("copying file %s: %w", file.RelativePath, err)
		}

		if ep.verbose {
			fmt.Printf("Copied: %s\n", file.RelativePath)
		}
	}

	return nil
}

// copyFile copies a single file from source to destination
func (ep *ExportProcessor) copyFile(srcPath, dstPath string) error {
	// Open source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy content
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	// Copy file mode
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return err
	}

	return os.Chmod(dstPath, srcInfo.Mode())
}

// LinkProcessingResult contains the aggregate results of link processing
type LinkProcessingResult struct {
	ExternalLinksRemoved    int
	ExternalLinksConverted  int
	InternalLinksUpdated    int
	FilesWithLinksProcessed int
}

// copyFilesWithLinkProcessing copies files and processes links during the copy
func (ep *ExportProcessor) copyFilesWithLinkProcessing(ctx context.Context, selectedFiles, allFiles []*vault.VaultFile, options ExportOptions) (*LinkProcessingResult, error) {
	// Create output directory
	if err := os.MkdirAll(options.OutputPath, 0755); err != nil {
		return nil, fmt.Errorf("creating output directory: %w", err)
	}

	// Create link analyzer and rewriter
	analyzer := NewExportLinkAnalyzer(selectedFiles, allFiles)
	strategy := LinkRewriteStrategy(options.LinkStrategy)
	rewriter := NewExportLinkRewriter(analyzer, strategy)

	result := &LinkProcessingResult{}

	for _, file := range selectedFiles {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Determine output file path
		outputFilePath := filepath.Join(options.OutputPath, file.RelativePath)

		// Create output directory for this file
		outputDir := filepath.Dir(outputFilePath)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return nil, fmt.Errorf("creating output directory %s: %w", outputDir, err)
		}

		// Process links in the file
		linkResult := rewriter.RewriteFileContent(file)

		// Aggregate link processing statistics
		result.ExternalLinksRemoved += linkResult.ExternalLinksRemoved
		result.ExternalLinksConverted += linkResult.ExternalLinksConverted
		result.InternalLinksUpdated += linkResult.InternalLinksUpdated
		if len(linkResult.ChangedLinks) > 0 {
			result.FilesWithLinksProcessed++
		}

		// Write the processed content to the output file
		err := ep.writeProcessedFile(linkResult.RewrittenContent, file, outputFilePath)
		if err != nil {
			return nil, fmt.Errorf("writing processed file %s: %w", file.RelativePath, err)
		}

		if ep.verbose {
			if len(linkResult.ChangedLinks) > 0 {
				fmt.Printf("Copied with link processing: %s (%d links processed)\n",
					file.RelativePath, len(linkResult.ChangedLinks))
			} else {
				fmt.Printf("Copied: %s\n", file.RelativePath)
			}
		}
	}

	return result, nil
}

// analyzeLinkProcessing analyzes what link processing would be done (for dry run)
func (ep *ExportProcessor) analyzeLinkProcessing(selectedFiles, allFiles []*vault.VaultFile, options ExportOptions) *LinkProcessingResult {
	// Create link analyzer and rewriter
	analyzer := NewExportLinkAnalyzer(selectedFiles, allFiles)
	strategy := LinkRewriteStrategy(options.LinkStrategy)
	rewriter := NewExportLinkRewriter(analyzer, strategy)

	result := &LinkProcessingResult{}

	for _, file := range selectedFiles {
		// Process links in the file
		linkResult := rewriter.RewriteFileContent(file)

		// Aggregate link processing statistics
		result.ExternalLinksRemoved += linkResult.ExternalLinksRemoved
		result.ExternalLinksConverted += linkResult.ExternalLinksConverted
		result.InternalLinksUpdated += linkResult.InternalLinksUpdated
		if len(linkResult.ChangedLinks) > 0 {
			result.FilesWithLinksProcessed++
		}
	}

	return result
}

// writeProcessedFile writes processed content to a file, preserving frontmatter
func (ep *ExportProcessor) writeProcessedFile(processedBody string, originalFile *vault.VaultFile, outputPath string) error {
	// Create a copy of the original file with the processed body
	processedFile := &vault.VaultFile{
		Path:         outputPath,
		RelativePath: originalFile.RelativePath,
		Frontmatter:  originalFile.Frontmatter,
		Body:         processedBody,
		Modified:     originalFile.Modified,
	}

	// Serialize the file (this will include frontmatter + processed body)
	content, err := processedFile.Serialize()
	if err != nil {
		return fmt.Errorf("serializing processed file: %w", err)
	}

	// Write to output file
	err = os.WriteFile(outputPath, content, 0644)
	if err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

// processAssets handles asset discovery and copying for exported files
func (ep *ExportProcessor) processAssets(ctx context.Context, selectedFiles []*vault.VaultFile, options ExportOptions) (*AssetProcessingResult, error) {
	// Create asset handler
	assetHandler := NewExportAssetHandler(options.VaultPath, options.OutputPath, ep.verbose)

	// Discover assets referenced by exported files
	discovery := assetHandler.DiscoverAssets(selectedFiles)

	if ep.verbose && discovery.TotalAssets > 0 {
		fmt.Printf("Found %d asset references in exported files\n", discovery.TotalAssets)
	}

	// Copy assets
	result := assetHandler.ProcessAssets(discovery)

	return result, nil
}

// analyzeAssetProcessing analyzes what asset processing would be done (for dry run)
func (ep *ExportProcessor) analyzeAssetProcessing(selectedFiles []*vault.VaultFile, options ExportOptions) *AssetProcessingResult {
	// Create asset handler
	assetHandler := NewExportAssetHandler(options.VaultPath, options.OutputPath, ep.verbose)

	// Discover assets that would be copied
	discovery := assetHandler.DiscoverAssets(selectedFiles)

	return &AssetProcessingResult{
		AssetsCopied:  len(discovery.AssetFiles),
		AssetsMissing: len(discovery.MissingAssets),
		CopiedAssets:  make([]string, 0), // Don't populate for dry run
		MissingAssets: discovery.MissingAssets,
	}
}

// expandWithBacklinks finds and includes files that link to exported files
func (ep *ExportProcessor) expandWithBacklinks(ctx context.Context, selectedFiles, allFiles []*vault.VaultFile, options ExportOptions) (*BacklinksDiscoveryResult, error) {
	// Create backlinks handler
	backlinksHandler := NewExportBacklinksHandler(allFiles, ep.verbose)

	// Discover backlinks
	result := backlinksHandler.DiscoverBacklinks(ctx, selectedFiles)

	if ep.verbose && result.TotalBacklinks > 0 {
		fmt.Printf("Found %d backlink files\n", result.TotalBacklinks)
	}

	return result, nil
}

// normalizeFilenames handles filename normalization for exported files
func (ep *ExportProcessor) normalizeFilenames(selectedFiles []*vault.VaultFile, options ExportOptions) (*FilenameNormalizationResult, error) {
	normalizationOptions := FilenameNormalizationOptions{
		Slugify: options.Slugify,
		Flatten: options.Flatten,
	}

	normalizer := NewExportFilenameNormalizer(normalizationOptions, ep.verbose)
	result := normalizer.NormalizeFilenames(selectedFiles)

	return result, nil
}

// copyFilesWithNormalization copies files with filename normalization only
func (ep *ExportProcessor) copyFilesWithNormalization(ctx context.Context, files []*vault.VaultFile, filenameMap map[string]string, options ExportOptions) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(options.OutputPath, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	for i, file := range files {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Determine output file path using filename mapping
		outputFilePath := filepath.Join(options.OutputPath, filenameMap[file.RelativePath])

		// Create output directory for this file
		outputDir := filepath.Dir(outputFilePath)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("creating output directory %s: %w", outputDir, err)
		}

		// Update links in file content if filename normalization occurred
		content := file.Body
		if filenameMap[file.RelativePath] != file.RelativePath {
			normalizer := NewExportFilenameNormalizer(FilenameNormalizationOptions{
				Slugify: options.Slugify,
				Flatten: options.Flatten,
			}, ep.verbose)
			content = normalizer.UpdateFileLinks(file, filenameMap)
		}

		// Write the file with updated content
		err := ep.writeNormalizedFile(content, file, outputFilePath)
		if err != nil {
			return fmt.Errorf("writing normalized file %s: %w", file.RelativePath, err)
		}

		// Update progress
		ep.progress.UpdatePhase(i+1, fmt.Sprintf("Copied: %s", file.RelativePath))
	}

	return nil
}

// copyFilesWithLinkProcessingAndNormalization copies files with both link processing and filename normalization
func (ep *ExportProcessor) copyFilesWithLinkProcessingAndNormalization(ctx context.Context, selectedFiles, allFiles []*vault.VaultFile, filenameMap map[string]string, options ExportOptions) (*LinkProcessingResult, error) {
	// Create output directory
	if err := os.MkdirAll(options.OutputPath, 0755); err != nil {
		return nil, fmt.Errorf("creating output directory: %w", err)
	}

	// Create link analyzer and rewriter
	analyzer := NewExportLinkAnalyzer(selectedFiles, allFiles)
	strategy := LinkRewriteStrategy(options.LinkStrategy)
	rewriter := NewExportLinkRewriter(analyzer, strategy)

	result := &LinkProcessingResult{}

	for i, file := range selectedFiles {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Determine output file path using filename mapping
		outputFilePath := filepath.Join(options.OutputPath, filenameMap[file.RelativePath])

		// Create output directory for this file
		outputDir := filepath.Dir(outputFilePath)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return nil, fmt.Errorf("creating output directory %s: %w", outputDir, err)
		}

		// Process links in the file
		linkResult := rewriter.RewriteFileContent(file)

		// Update links for filename normalization
		processedContent := linkResult.RewrittenContent
		if filenameMap[file.RelativePath] != file.RelativePath {
			normalizer := NewExportFilenameNormalizer(FilenameNormalizationOptions{
				Slugify: options.Slugify,
				Flatten: options.Flatten,
			}, ep.verbose)

			// Create a temporary file with the link-processed content for link normalization
			tempFile := &vault.VaultFile{
				Path:         file.Path,
				RelativePath: file.RelativePath,
				Frontmatter:  file.Frontmatter,
				Body:         processedContent,
				Modified:     file.Modified,
			}
			processedContent = normalizer.UpdateFileLinks(tempFile, filenameMap)
		}

		// Aggregate link processing statistics
		result.ExternalLinksRemoved += linkResult.ExternalLinksRemoved
		result.ExternalLinksConverted += linkResult.ExternalLinksConverted
		result.InternalLinksUpdated += linkResult.InternalLinksUpdated
		if len(linkResult.ChangedLinks) > 0 {
			result.FilesWithLinksProcessed++
		}

		// Write the processed content to the output file
		err := ep.writeNormalizedFile(processedContent, file, outputFilePath)
		if err != nil {
			return nil, fmt.Errorf("writing processed file %s: %w", file.RelativePath, err)
		}

		// Update progress
		message := fmt.Sprintf("Processed: %s", file.RelativePath)
		if len(linkResult.ChangedLinks) > 0 {
			message = fmt.Sprintf("Processed: %s (%d links)", file.RelativePath, len(linkResult.ChangedLinks))
		}
		ep.progress.UpdatePhase(i+1, message)
	}

	return result, nil
}

// writeNormalizedFile writes processed content to a file, preserving frontmatter
func (ep *ExportProcessor) writeNormalizedFile(processedBody string, originalFile *vault.VaultFile, outputPath string) error {
	// Create a copy of the original file with the processed body
	processedFile := &vault.VaultFile{
		Path:         outputPath,
		RelativePath: filepath.Base(outputPath), // Use just the filename for relative path
		Frontmatter:  originalFile.Frontmatter,
		Body:         processedBody,
		Modified:     originalFile.Modified,
	}

	// Serialize the file (this will include frontmatter + processed body)
	content, err := processedFile.Serialize()
	if err != nil {
		return fmt.Errorf("serializing processed file: %w", err)
	}

	// Write to output file
	err = os.WriteFile(outputPath, content, 0644)
	if err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

// copyFilesWithNormalizationParallel copies files with parallel processing
func (ep *ExportProcessor) copyFilesWithNormalizationParallel(ctx context.Context, files []*vault.VaultFile, filenameMap map[string]string, options ExportOptions) error {
	parallelProcessor := NewParallelFileProcessor(options.ParallelWorkers, options.OptimizeMemory, ep.progress)

	// Create a processor function for individual files
	fileProcessor := func(file *vault.VaultFile, outputPath string, opts ExportOptions) (*FileProcessingResult, error) {
		// Create output directory for this file
		outputDir := filepath.Dir(filepath.Join(opts.OutputPath, outputPath))
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return nil, fmt.Errorf("creating output directory %s: %w", outputDir, err)
		}

		// Update links in file content if filename normalization occurred
		content := file.Body
		if filenameMap[file.RelativePath] != file.RelativePath {
			normalizer := NewExportFilenameNormalizer(FilenameNormalizationOptions{
				Slugify: opts.Slugify,
				Flatten: opts.Flatten,
			}, ep.verbose)
			content = normalizer.UpdateFileLinks(file, filenameMap)
		}

		// Write the file with updated content
		fullOutputPath := filepath.Join(opts.OutputPath, outputPath)
		err := ep.writeNormalizedFile(content, file, fullOutputPath)
		if err != nil {
			return nil, fmt.Errorf("writing normalized file %s: %w", file.RelativePath, err)
		}

		return &FileProcessingResult{
			File:    file,
			Success: true,
		}, nil
	}

	_, err := parallelProcessor.ProcessFilesInParallel(ctx, files, filenameMap, options, fileProcessor)
	return err
}

// copyFilesWithLinkProcessingParallel copies files with link processing using parallel workers
func (ep *ExportProcessor) copyFilesWithLinkProcessingParallel(ctx context.Context, selectedFiles, allFiles []*vault.VaultFile, filenameMap map[string]string, options ExportOptions) (*LinkProcessingResult, error) {
	parallelProcessor := NewParallelFileProcessor(options.ParallelWorkers, options.OptimizeMemory, ep.progress)

	// Create link analyzer and rewriter (shared across workers)
	analyzer := NewExportLinkAnalyzer(selectedFiles, allFiles)
	strategy := LinkRewriteStrategy(options.LinkStrategy)
	rewriter := NewExportLinkRewriter(analyzer, strategy)

	// Create a processor function for individual files
	fileProcessor := func(file *vault.VaultFile, outputPath string, opts ExportOptions) (*FileProcessingResult, error) {
		// Create output directory for this file
		outputDir := filepath.Dir(filepath.Join(opts.OutputPath, outputPath))
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return nil, fmt.Errorf("creating output directory %s: %w", outputDir, err)
		}

		// Process links in the file
		linkResult := rewriter.RewriteFileContent(file)

		// Update links for filename normalization
		processedContent := linkResult.RewrittenContent
		if filenameMap[file.RelativePath] != file.RelativePath {
			normalizer := NewExportFilenameNormalizer(FilenameNormalizationOptions{
				Slugify: opts.Slugify,
				Flatten: opts.Flatten,
			}, ep.verbose)

			// Create a temporary file with the link-processed content for link normalization
			tempFile := &vault.VaultFile{
				Path:         file.Path,
				RelativePath: file.RelativePath,
				Frontmatter:  file.Frontmatter,
				Body:         processedContent,
				Modified:     file.Modified,
			}
			processedContent = normalizer.UpdateFileLinks(tempFile, filenameMap)
		}

		// Write the processed content to the output file
		fullOutputPath := filepath.Join(opts.OutputPath, outputPath)
		err := ep.writeNormalizedFile(processedContent, file, fullOutputPath)
		if err != nil {
			return nil, fmt.Errorf("writing processed file %s: %w", file.RelativePath, err)
		}

		return &FileProcessingResult{
			File:                   file,
			Success:                true,
			ExternalLinksRemoved:   linkResult.ExternalLinksRemoved,
			ExternalLinksConverted: linkResult.ExternalLinksConverted,
			InternalLinksUpdated:   linkResult.InternalLinksUpdated,
			LinksProcessed:         len(linkResult.ChangedLinks),
		}, nil
	}

	return parallelProcessor.ProcessFilesInParallel(ctx, selectedFiles, filenameMap, options, fileProcessor)
}
