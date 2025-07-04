package processor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/eoinhurrell/mdnotes/internal/rgsearch"
	"github.com/eoinhurrell/mdnotes/internal/vault"
	"github.com/eoinhurrell/mdnotes/internal/workerpool"
	"github.com/eoinhurrell/mdnotes/pkg/template"
)

// RenameProcessor handles file renaming with link updates in a performance-optimized way
type RenameProcessor struct {
	scanner     *vault.Scanner
	linkParser  *LinkParser
	linkUpdater *LinkUpdater
	searcher    *rgsearch.Searcher
	pool        *workerpool.WorkerPool
	workers     int
	verbose     bool
}

// RenameOptions contains configuration for rename operations
type RenameOptions struct {
	VaultRoot      string
	IgnorePatterns []string
	Template       string
	DryRun         bool
	Verbose        bool
	Workers        int
}

// RenameResult contains the results of a rename operation
type RenameResult struct {
	SourcePath    string
	TargetPath    string
	FilesScanned  int
	FilesModified int
	LinksUpdated  int
	ModifiedFiles []string
	Duration      time.Duration
}

// FileProcessResult contains the result of processing a single file
type FileProcessResult struct {
	File         *vault.VaultFile
	Modified     bool
	LinksUpdated int
	Error        error
}

// NewRenameProcessor creates a new performance-optimized rename processor
func NewRenameProcessor(options RenameOptions) *RenameProcessor {
	workers := options.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	scanner := vault.NewScanner(vault.WithIgnorePatterns(options.IgnorePatterns))
	searcher := rgsearch.NewSearcher()

	// Create worker pool for parallel processing
	poolConfig := workerpool.Config{
		MaxWorkers:  workers,
		QueueSize:   workers * 10,
		TaskTimeout: 30 * time.Second,
		EnableStats: options.Verbose,
	}
	pool := workerpool.NewWorkerPool(poolConfig)

	return &RenameProcessor{
		scanner:     scanner,
		linkParser:  NewLinkParser(),
		linkUpdater: NewLinkUpdater(),
		searcher:    searcher,
		pool:        pool,
		workers:     workers,
		verbose:     options.Verbose,
	}
}

// ProcessRename performs the complete rename operation
func (rp *RenameProcessor) ProcessRename(ctx context.Context, sourcePath, targetPath string, options RenameOptions) (*RenameResult, error) {
	startTime := time.Now()

	// Create result object
	result := &RenameResult{
		SourcePath: sourcePath,
		TargetPath: targetPath,
	}

	// Get relative paths for tracking
	sourceRel, err := filepath.Rel(options.VaultRoot, sourcePath)
	if err != nil {
		return nil, fmt.Errorf("getting relative source path: %w", err)
	}

	targetRel, err := filepath.Rel(options.VaultRoot, targetPath)
	if err != nil {
		return nil, fmt.Errorf("getting relative target path: %w", err)
	}

	// Create file move record
	move := FileMove{From: sourceRel, To: targetRel}

	// Use optimized search and processing
	modifiedFiles, errors := rp.processRenameWithOptimizedSearch(ctx, move, options, result)

	// Check for context cancellation
	if ctx.Err() != nil {
		return result, ctx.Err()
	}

	// If not dry run, save modified files and perform rename
	if !options.DryRun {
		if err := rp.saveModifiedFiles(modifiedFiles); err != nil {
			return result, fmt.Errorf("saving modified files: %w", err)
		}

		if err := rp.performFileRename(sourcePath, targetPath); err != nil {
			return result, fmt.Errorf("renaming file: %w", err)
		}
	}

	result.Duration = time.Since(startTime)

	// Report any errors collected
	if len(errors) > 0 {
		if options.Verbose {
			fmt.Printf("Encountered %d errors during processing:\n", len(errors))
			for i, err := range errors {
				fmt.Printf("  %d: %v\n", i+1, err)
			}
		} else {
			fmt.Printf("Encountered %d errors during processing (use --verbose for details)\n", len(errors))
		}
	}

	return result, nil
}

// Cleanup properly shuts down the worker pool
func (rp *RenameProcessor) Cleanup() error {
	if rp.pool != nil {
		return rp.pool.Shutdown(10 * time.Second)
	}
	return nil
}

// processFile processes a single file for link updates
func (rp *RenameProcessor) processFile(file *vault.VaultFile, move FileMove, linkRegex *regexp.Regexp, dryRun bool, verbose bool) FileProcessResult {
	result := FileProcessResult{File: file}

	// Quick check if file body contains any potential links using regex (if provided)
	if linkRegex != nil && !linkRegex.MatchString(file.Body) {
		if verbose {
			fmt.Printf("Examining: %s - no potential links found by regex\n", file.RelativePath)
		}
		return result // No potential links found, skip expensive parsing
	}

	// Parse links
	rp.linkParser.UpdateFile(file)

	if verbose && len(file.Links) > 0 {
		fmt.Printf("Examining: %s - found %d links total\n", file.RelativePath, len(file.Links))
	}

	// Count matching links and collect details for verbose output
	matchingLinks := 0
	var matchedTargets []string
	for _, link := range file.Links {
		if rp.linkMatchesMove(link, move) {
			matchingLinks++
			if verbose {
				matchedTargets = append(matchedTargets, link.Target)
			}
		}
	}

	if matchingLinks == 0 {
		if verbose && len(file.Links) > 0 {
			fmt.Printf("Examining: %s - no links match the moved file\n", file.RelativePath)
		}
		return result // No matching links found
	}

	if verbose {
		fmt.Printf("Examining: %s - found %d matching links: %v\n", file.RelativePath, matchingLinks, matchedTargets)
	}

	// Update links if not dry run
	if !dryRun {
		if rp.linkUpdater.UpdateFile(file, []FileMove{move}) {
			result.Modified = true
			result.LinksUpdated = matchingLinks
		}
	} else {
		// For dry run, just count what would be updated
		result.Modified = true
		result.LinksUpdated = matchingLinks
	}

	return result
}

// compileOptimizedLinkRegex creates an optimized regex for quick link detection
func (rp *RenameProcessor) compileOptimizedLinkRegex(sourceFile string) *regexp.Regexp {
	// We use a simple permissive pattern for performance
	// The detailed link matching happens later with proper parsing

	// Use a permissive pattern that catches any potential link
	// The exact matching will be done later with proper link parsing
	pattern := `(?i)(\[\[|\]\(.*\.md)`

	regex, err := regexp.Compile(pattern)
	if err != nil {
		// Fallback to simple pattern if compilation fails
		return regexp.MustCompile(`\[\[|\]\(.*\.md\)`)
	}

	return regex
}

// linkMatchesMove checks if a link matches the file move
func (rp *RenameProcessor) linkMatchesMove(link vault.Link, move FileMove) bool {
	// Use the Link.ShouldUpdate method which handles all the complex matching logic
	// including case-insensitive matching and underscore/space conversion
	return link.ShouldUpdate(move.From, move.To)
}

// saveModifiedFiles saves all modified files atomically
func (rp *RenameProcessor) saveModifiedFiles(files []*vault.VaultFile) error {
	var errors []error

	for _, file := range files {
		content, err := file.Serialize()
		if err != nil {
			errors = append(errors, fmt.Errorf("serializing %s: %w", file.RelativePath, err))
			continue
		}

		// Write atomically by writing to temp file then renaming
		tempPath := file.Path + ".tmp"
		if err := os.WriteFile(tempPath, content, 0644); err != nil {
			errors = append(errors, fmt.Errorf("writing temp file for %s: %w", file.RelativePath, err))
			continue
		}

		if err := os.Rename(tempPath, file.Path); err != nil {
			// Clean up temp file on failure
			os.Remove(tempPath)
			errors = append(errors, fmt.Errorf("renaming temp file for %s: %w", file.RelativePath, err))
			continue
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors saving files: %v", errors)
	}

	return nil
}

// performFileRename renames the source file to target path
func (rp *RenameProcessor) performFileRename(sourcePath, targetPath string) error {
	// Create target directory if needed
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("creating target directory: %w", err)
	}

	// Perform the atomic rename
	if err := os.Rename(sourcePath, targetPath); err != nil {
		return fmt.Errorf("renaming file: %w", err)
	}

	return nil
}

// processRenameWithOptimizedSearch uses rgsearch and workerpool for efficient processing
func (rp *RenameProcessor) processRenameWithOptimizedSearch(ctx context.Context, move FileMove, options RenameOptions, result *RenameResult) ([]*vault.VaultFile, []error) {
	var modifiedFiles []*vault.VaultFile
	var errors []error

	// Try to use rgsearch for faster file discovery
	candidateFiles, rgErr := rp.findCandidateFilesWithRgsearch(ctx, move, options)

	if rgErr != nil {
		if options.Verbose {
			fmt.Printf("Ripgrep unavailable, falling back to full scan: %v\n", rgErr)
		}
		// Fallback to original approach if ripgrep failed
		return rp.processFullRenameFallback(ctx, move, options, result)
	}

	if options.Verbose {
		fmt.Printf("Found %d candidate files to examine\n", len(candidateFiles))
	}

	if len(candidateFiles) == 0 {
		return modifiedFiles, errors
	}

	// Create tasks for parallel processing
	tasks := make([]workerpool.Task, len(candidateFiles))
	taskResults := make([]*FileProcessResult, len(candidateFiles))

	for i, filePath := range candidateFiles {
		i, filePath := i, filePath // Capture loop variables
		tasks[i] = func(ctx context.Context) error {
			vaultFile := &vault.VaultFile{Path: filePath}
			content, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("reading %s: %w", filePath, err)
			}

			if err := vaultFile.Parse(content); err != nil {
				return fmt.Errorf("parsing %s: %w", filePath, err)
			}

			// Set relative path
			if rel, err := filepath.Rel(options.VaultRoot, filePath); err == nil {
				vaultFile.RelativePath = rel
			}

			processResult := rp.processFile(vaultFile, move, nil, options.DryRun, options.Verbose)
			processResult.File = vaultFile
			taskResults[i] = &processResult

			return processResult.Error
		}
	}

	// Process all tasks in parallel
	results := rp.pool.ProcessBatch(tasks)

	// Collect results
	for i, taskResult := range results {
		result.FilesScanned++

		if taskResult.Error != nil {
			errors = append(errors, taskResult.Error)
			if options.Verbose {
				fmt.Printf("Error processing file: %v\n", taskResult.Error)
			}
			continue
		}

		processResult := taskResults[i]
		if processResult != nil && processResult.Modified {
			result.FilesModified++
			result.LinksUpdated += processResult.LinksUpdated
			result.ModifiedFiles = append(result.ModifiedFiles, processResult.File.RelativePath)
			modifiedFiles = append(modifiedFiles, processResult.File)
		}
	}

	return modifiedFiles, errors
}

// findCandidateFilesWithRgsearch uses rgsearch to quickly find files that might contain references
func (rp *RenameProcessor) findCandidateFilesWithRgsearch(ctx context.Context, move FileMove, options RenameOptions) ([]string, error) {
	if !rp.searcher.IsAvailable() {
		return nil, fmt.Errorf("ripgrep not available")
	}

	// Create search patterns for the file being renamed
	sourceFile := filepath.Base(move.From)
	sourceWithoutExt := strings.TrimSuffix(sourceFile, ".md")
	moveFromWithoutExt := strings.TrimSuffix(move.From, ".md")

	// URL encode versions for matching encoded links (Obsidian-style)
	sourceFileEncoded := rp.obsidianURLEncode(sourceFile)
	moveFromEncoded := rp.obsidianURLEncode(move.From)

	// Also try with spaces instead of underscores (common mismatch)
	sourceFileWithSpaces := strings.ReplaceAll(sourceFile, "_", " ")
	sourceFileWithSpacesEncoded := rp.obsidianURLEncode(sourceFileWithSpaces)
	moveFromWithSpaces := strings.ReplaceAll(move.From, "_", " ")
	moveFromWithSpacesEncoded := rp.obsidianURLEncode(moveFromWithSpaces)

	// Build pattern that matches potential links - include full path patterns
	var patterns []string

	// Wiki links: [[basename]] or [[path/basename]] with optional fragments
	patterns = append(patterns, fmt.Sprintf(`\[\[%s(#[^|\]]*)?(\]\]|\|)`, rp.regexQuoteWithURLPreserve(sourceWithoutExt)))
	if move.From != sourceFile {
		patterns = append(patterns, fmt.Sprintf(`\[\[%s(#[^|\]]*)?(\]\]|\|)`, rp.regexQuoteWithURLPreserve(moveFromWithoutExt)))
	}

	// Markdown links: [text](basename.md) or [text](path/basename.md) with optional fragments
	patterns = append(patterns, fmt.Sprintf(`\]\(%s(#[^)]*)?`, rp.regexQuoteWithURLPreserve(sourceFile)))
	if move.From != sourceFile {
		patterns = append(patterns, fmt.Sprintf(`\]\(%s(#[^)]*)?`, rp.regexQuoteWithURLPreserve(move.From)))
	}

	// URL-encoded markdown links (handle %20, etc.) with optional fragments
	if sourceFileEncoded != sourceFile {
		patterns = append(patterns, fmt.Sprintf(`\]\(%s(#[^)]*)?`, rp.regexQuoteWithURLPreserve(sourceFileEncoded)))
	}
	if moveFromEncoded != move.From && move.From != sourceFile {
		patterns = append(patterns, fmt.Sprintf(`\]\(%s(#[^)]*)?`, rp.regexQuoteWithURLPreserve(moveFromEncoded)))
	}

	// Space-based variants (common mismatch: file uses underscore, link uses space) with optional fragments
	if sourceFileWithSpaces != sourceFile {
		patterns = append(patterns, fmt.Sprintf(`\]\(%s(#[^)]*)?`, rp.regexQuoteWithURLPreserve(sourceFileWithSpaces)))
	}
	if sourceFileWithSpacesEncoded != sourceFileWithSpaces && sourceFileWithSpacesEncoded != sourceFileEncoded {
		patterns = append(patterns, fmt.Sprintf(`\]\(%s(#[^)]*)?`, rp.regexQuoteWithURLPreserve(sourceFileWithSpacesEncoded)))
	}
	if moveFromWithSpaces != move.From && move.From != sourceFile {
		patterns = append(patterns, fmt.Sprintf(`\]\(%s(#[^)]*)?`, rp.regexQuoteWithURLPreserve(moveFromWithSpaces)))
	}
	if moveFromWithSpacesEncoded != moveFromWithSpaces && moveFromWithSpacesEncoded != moveFromEncoded {
		patterns = append(patterns, fmt.Sprintf(`\]\(%s(#[^)]*)?`, rp.regexQuoteWithURLPreserve(moveFromWithSpacesEncoded)))
	}

	// Embed links: ![[basename]] or ![[path/basename]] with optional fragments
	patterns = append(patterns, fmt.Sprintf(`!\[\[%s(#[^|\]]*)?`, rp.regexQuoteWithURLPreserve(sourceWithoutExt)))
	if move.From != sourceFile {
		patterns = append(patterns, fmt.Sprintf(`!\[\[%s(#[^|\]]*)?`, rp.regexQuoteWithURLPreserve(moveFromWithoutExt)))
	}

	// Combine all patterns
	pattern := "(" + strings.Join(patterns, "|") + ")"

	// Configure search options
	searchOptions := rgsearch.SearchOptions{
		Pattern:         pattern,
		Path:            options.VaultRoot,
		CaseSensitive:   false,
		Regex:           true,
		IncludePatterns: []string{"*.md"},
		MaxMatches:      1000,
		Timeout:         30 * time.Second,
		AdditionalArgs:  []string{"--no-ignore"}, // Disable gitignore to search test/test-vault
	}

	// Use rgsearch to find files containing potential references
	files, err := rp.searcher.SearchFiles(ctx, searchOptions)
	if err != nil {
		return nil, fmt.Errorf("rgsearch execution failed: %w", err)
	}

	return files, nil
}

// obsidianURLEncode encodes a path the way Obsidian does for markdown links
func (rp *RenameProcessor) obsidianURLEncode(path string) string {
	// Based on actual observation, Obsidian only encodes spaces to %20
	// Other characters like () are left as-is in the links
	result := strings.ReplaceAll(path, " ", "%20")
	return result
}

// regexQuoteWithURLPreserve escapes regex special characters but preserves URL encoding
func (rp *RenameProcessor) regexQuoteWithURLPreserve(path string) string {
	// First check if this looks like a URL-encoded path
	if strings.Contains(path, "%") {
		// For URL-encoded paths, we need to be more careful about escaping
		// Split by % and handle each part separately
		parts := strings.Split(path, "%")
		result := regexp.QuoteMeta(parts[0])

		for i := 1; i < len(parts); i++ {
			// Add back the % character (it was removed by split)
			result += "%"
			// For the hex part after %, don't escape if it's a valid hex sequence
			if len(parts[i]) >= 2 {
				hexPart := parts[i][:2]
				remainder := parts[i][2:]
				result += hexPart + regexp.QuoteMeta(remainder)
			} else {
				result += regexp.QuoteMeta(parts[i])
			}
		}
		return result
	}

	// For non-URL-encoded paths, use normal escaping
	return regexp.QuoteMeta(path)
}

// processFullRenameFallback is the original comprehensive approach using workerpool
func (rp *RenameProcessor) processFullRenameFallback(ctx context.Context, move FileMove, options RenameOptions, result *RenameResult) ([]*vault.VaultFile, []error) {
	var modifiedFiles []*vault.VaultFile
	var errors []error

	// Pre-compile regex patterns for performance
	linkRegex := rp.compileOptimizedLinkRegex(move.From)

	// Collect all files first
	var allFiles []*vault.VaultFile
	err := rp.scanner.WalkWithCallback(options.VaultRoot, func(file *vault.VaultFile) error {
		allFiles = append(allFiles, file)
		return nil
	})

	if err != nil {
		return modifiedFiles, []error{fmt.Errorf("scanning vault: %w", err)}
	}

	if len(allFiles) == 0 {
		return modifiedFiles, errors
	}

	// Create tasks for parallel processing
	tasks := make([]workerpool.Task, len(allFiles))
	taskResults := make([]*FileProcessResult, len(allFiles))

	for i, file := range allFiles {
		i, file := i, file // Capture loop variables
		tasks[i] = func(ctx context.Context) error {
			processResult := rp.processFile(file, move, linkRegex, options.DryRun, options.Verbose)
			processResult.File = file
			taskResults[i] = &processResult

			return processResult.Error
		}
	}

	// Process all tasks in parallel
	results := rp.pool.ProcessBatch(tasks)

	// Collect results
	for i, taskResult := range results {
		result.FilesScanned++

		if taskResult.Error != nil {
			errors = append(errors, taskResult.Error)
			if options.Verbose {
				fmt.Printf("Error processing file: %v\n", taskResult.Error)
			}
			continue
		}

		processResult := taskResults[i]
		if processResult != nil && processResult.Modified {
			result.FilesModified++
			result.LinksUpdated += processResult.LinksUpdated
			result.ModifiedFiles = append(result.ModifiedFiles, processResult.File.RelativePath)
			modifiedFiles = append(modifiedFiles, processResult.File)
		}
	}

	return modifiedFiles, errors
}

// GenerateNameFromTemplate generates a new filename using the template system
func GenerateNameFromTemplate(sourcePath, templateStr string) (string, error) {
	// Get file info
	fileInfo, err := os.Stat(sourcePath)
	if err != nil {
		return "", fmt.Errorf("getting file info: %w", err)
	}

	// Parse the source file to get frontmatter data
	vaultFile := &vault.VaultFile{Path: sourcePath}
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}

	if err := vaultFile.Parse(content); err != nil {
		return "", fmt.Errorf("parsing file: %w", err)
	}

	// Get creation time from frontmatter or file system
	var createdTime time.Time
	if created, exists := vaultFile.GetField("created"); exists {
		if createdTime, err = parseTimeField(created); err != nil {
			createdTime = fileInfo.ModTime()
		}
	} else {
		createdTime = fileInfo.ModTime()
	}

	// Extract filename without extension
	baseName := filepath.Base(sourcePath)
	filename := strings.TrimSuffix(baseName, filepath.Ext(baseName))

	// Check if filename already has a datestring prefix
	engine := template.NewEngine()
	existingDatestring := engine.ExtractDatestring(filename)

	// If filename already has a datestring, use it and remove it from the filename
	if existingDatestring != "" {
		// Use existing datestring and extract filename without it
		filenameWithoutDatestring := engine.ExtractFilenameWithoutDatestring(filename)

		// Create slug with underscores
		slugified := engine.SlugifyWithUnderscore(filenameWithoutDatestring)

		return fmt.Sprintf("%s-%s.md", existingDatestring, slugified), nil
	}

	// Create a temporary VaultFile for template processing with proper setup
	tempFile := &vault.VaultFile{
		Path:         sourcePath,
		RelativePath: filepath.Base(sourcePath),
		Modified:     createdTime,
		Frontmatter:  make(map[string]interface{}),
	}

	// Copy all existing frontmatter
	for k, v := range vaultFile.Frontmatter {
		tempFile.Frontmatter[k] = v
	}

	// Add template data to frontmatter
	tempFile.Frontmatter["filename"] = filename
	if _, exists := tempFile.Frontmatter["created"]; !exists {
		tempFile.Frontmatter["created"] = createdTime.Format("2006-01-02")
	}

	// Use the template engine
	result := engine.Process(templateStr, tempFile)

	return result, nil
}

// parseTimeField attempts to parse various time field formats
func parseTimeField(field interface{}) (time.Time, error) {
	switch v := field.(type) {
	case string:
		// Try different time formats
		formats := []string{
			"2006-01-02",
			time.RFC3339,
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05",
		}

		for _, format := range formats {
			if t, err := time.Parse(format, v); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("unrecognized time format: %s", v)

	case time.Time:
		return v, nil

	default:
		return time.Time{}, fmt.Errorf("unsupported time field type: %T", field)
	}
}
