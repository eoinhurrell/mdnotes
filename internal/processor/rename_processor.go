package processor

import (
	"context"
	"fmt"
	"net/url"
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
	VaultRoot       string
	IgnorePatterns  []string
	Template        string
	DryRun          bool
	Verbose         bool
	Workers         int
}

// RenameResult contains the results of a rename operation
type RenameResult struct {
	SourcePath      string
	TargetPath      string
	FilesScanned    int
	FilesModified   int
	LinksUpdated    int
	ModifiedFiles   []string
	Duration        time.Duration
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
	if len(errors) > 0 && options.Verbose {
		fmt.Printf("Encountered %d errors during processing\n", len(errors))
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
func (rp *RenameProcessor) processFile(file *vault.VaultFile, move FileMove, linkRegex *regexp.Regexp, dryRun bool) FileProcessResult {
	result := FileProcessResult{File: file}

	// Quick check if file body contains any potential links using regex
	if !linkRegex.MatchString(file.Body) {
		return result // No potential links found, skip expensive parsing
	}

	// Parse links only if quick check passed
	rp.linkParser.UpdateFile(file)

	// Count matching links before processing
	matchingLinks := 0
	for _, link := range file.Links {
		if rp.linkMatchesMove(link, move) {
			matchingLinks++
		}
	}

	if matchingLinks == 0 {
		return result // No matching links found
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
	target := link.Target
	
	// Remove any fragment identifiers (e.g., file.md#heading)
	if idx := strings.Index(target, "#"); idx != -1 {
		target = target[:idx]
	}

	// Normalize paths for comparison - all paths should be vault-relative
	moveFrom := filepath.ToSlash(move.From)
	target = filepath.ToSlash(target)
	
	// URL decode the target for proper comparison (handles %20, etc.)
	decodedTarget, err := url.QueryUnescape(target)
	if err != nil {
		// If decoding fails, use original target
		decodedTarget = target
	}

	// Get basename for comparison (for wiki links that might use just filename)
	moveFromBasename := filepath.Base(moveFrom)
	moveFromBasenameWithoutExt := strings.TrimSuffix(moveFromBasename, ".md")
	moveFromWithoutExt := strings.TrimSuffix(moveFrom, ".md")

	switch link.Type {
	case vault.WikiLink:
		// Wiki links can reference files by:
		// 1. Full vault-relative path: "folder/file" or "folder/file.md"
		// 2. Basename only: "file" or "file.md" (Obsidian behavior)
		if decodedTarget == moveFromWithoutExt || decodedTarget == moveFrom {
			return true
		}
		if decodedTarget == moveFromBasenameWithoutExt || decodedTarget == moveFromBasename {
			return true
		}
		// Also check if target with .md extension matches
		if !strings.HasSuffix(decodedTarget, ".md") {
			return decodedTarget+".md" == moveFrom || decodedTarget+".md" == moveFromBasename
		}
		return false

	case vault.MarkdownLink, vault.EmbedLink:
		// Markdown links should be vault-relative paths
		// Compare both original and decoded target
		if target == moveFrom || decodedTarget == moveFrom {
			return true
		}
		// Check if target without extension matches moveFrom without extension
		targetWithoutExt := strings.TrimSuffix(target, ".md")
		decodedTargetWithoutExt := strings.TrimSuffix(decodedTarget, ".md")
		if targetWithoutExt == moveFromWithoutExt || decodedTargetWithoutExt == moveFromWithoutExt {
			return true
		}
		// Also check if adding .md to target matches the move path
		if !strings.HasSuffix(target, ".md") && target+".md" == moveFrom {
			return true
		}
		if !strings.HasSuffix(decodedTarget, ".md") && decodedTarget+".md" == moveFrom {
			return true
		}
		return false

	default:
		return false
	}
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
			
			processResult := rp.processFile(vaultFile, move, nil, options.DryRun)
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
			
			if options.Verbose {
				fmt.Printf("Examining: %s - updated %d links\n", processResult.File.RelativePath, processResult.LinksUpdated)
			}
		} else if options.Verbose && processResult != nil {
			fmt.Printf("Examining: %s - no changes needed\n", processResult.File.RelativePath)
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
	
	// Build pattern that matches potential links - include full path patterns
	var patterns []string
	
	// Wiki links: [[basename]] or [[path/basename]]
	patterns = append(patterns, fmt.Sprintf(`\[\[%s(\]\]|\|)`, regexp.QuoteMeta(sourceWithoutExt)))
	if move.From != sourceFile {
		patterns = append(patterns, fmt.Sprintf(`\[\[%s(\]\]|\|)`, regexp.QuoteMeta(moveFromWithoutExt)))
	}
	
	// Markdown links: [text](basename.md) or [text](path/basename.md)
	patterns = append(patterns, fmt.Sprintf(`\]\(%s\)`, regexp.QuoteMeta(sourceFile)))
	if move.From != sourceFile {
		patterns = append(patterns, fmt.Sprintf(`\]\(%s\)`, regexp.QuoteMeta(move.From)))
	}
	
	// URL-encoded markdown links (handle %20, etc.)
	if sourceFileEncoded != sourceFile {
		patterns = append(patterns, fmt.Sprintf(`\]\(%s\)`, regexp.QuoteMeta(sourceFileEncoded)))
	}
	if moveFromEncoded != move.From && move.From != sourceFile {
		patterns = append(patterns, fmt.Sprintf(`\]\(%s\)`, regexp.QuoteMeta(moveFromEncoded)))
	}
	
	// Embed links: ![[basename]] or ![[path/basename]]
	patterns = append(patterns, fmt.Sprintf(`!\[\[%s\]\]`, regexp.QuoteMeta(sourceWithoutExt)))
	if move.From != sourceFile {
		patterns = append(patterns, fmt.Sprintf(`!\[\[%s\]\]`, regexp.QuoteMeta(moveFromWithoutExt)))
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
	// Replace specific characters that Obsidian encodes
	result := strings.ReplaceAll(path, " ", "%20")
	result = strings.ReplaceAll(result, "'", "%27")
	result = strings.ReplaceAll(result, "\"", "%22")
	result = strings.ReplaceAll(result, "(", "%28")
	result = strings.ReplaceAll(result, ")", "%29")
	result = strings.ReplaceAll(result, "[", "%5B")
	result = strings.ReplaceAll(result, "]", "%5D")
	result = strings.ReplaceAll(result, "{", "%7B")
	result = strings.ReplaceAll(result, "}", "%7D")
	result = strings.ReplaceAll(result, "#", "%23")
	return result
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
			processResult := rp.processFile(file, move, linkRegex, options.DryRun)
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
			
			if options.Verbose {
				fmt.Printf("Examining: %s - updated %d links\n", processResult.File.RelativePath, processResult.LinksUpdated)
			}
		} else if options.Verbose && processResult != nil {
			fmt.Printf("Examining: %s - no changes needed\n", processResult.File.RelativePath)
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