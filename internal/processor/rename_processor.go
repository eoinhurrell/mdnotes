package processor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/eoinhurrell/mdnotes/internal/vault"
	"github.com/eoinhurrell/mdnotes/pkg/template"
)

// RenameProcessor handles file renaming with link updates in a performance-optimized way
type RenameProcessor struct {
	scanner     *vault.Scanner
	linkParser  *LinkParser
	linkUpdater *LinkUpdater
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

	return &RenameProcessor{
		scanner:     scanner,
		linkParser:  NewLinkParser(),
		linkUpdater: NewLinkUpdater(),
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

	// Use ripgrep for all rename operations
	modifiedFiles, errors := rp.processRenameWithRipgrep(ctx, move, options, result)

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

// processFileWorker processes files from the stream
func (rp *RenameProcessor) processFileWorker(ctx context.Context, fileStream <-chan *vault.VaultFile, results chan<- FileProcessResult, move FileMove, linkRegex *regexp.Regexp, dryRun bool) {
	for {
		select {
		case file, ok := <-fileStream:
			if !ok {
				return // Channel closed
			}

			result := rp.processFile(file, move, linkRegex, dryRun)
			
			select {
			case results <- result:
			case <-ctx.Done():
				return
			}

		case <-ctx.Done():
			return
		}
	}
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

	// Normalize paths for comparison
	moveFrom := filepath.ToSlash(move.From)
	target = filepath.ToSlash(target)

	// Get basename for comparison (for wiki links that might use just filename)
	moveFromBasename := filepath.Base(moveFrom)
	moveFromBasenameWithoutExt := strings.TrimSuffix(moveFromBasename, ".md")
	moveFromWithoutExt := strings.TrimSuffix(moveFrom, ".md")

	switch link.Type {
	case vault.WikiLink:
		// Wiki links can reference files by:
		// 1. Full path: "folder/file" or "folder/file.md"
		// 2. Basename: "file" or "file.md"
		if target == moveFromWithoutExt || target == moveFrom {
			return true
		}
		if target == moveFromBasenameWithoutExt || target == moveFromBasename {
			return true
		}
		// Also check if target with .md extension matches
		if !strings.HasSuffix(target, ".md") {
			return target+".md" == moveFrom || target+".md" == moveFromBasename
		}
		return false

	case vault.MarkdownLink, vault.EmbedLink:
		// Markdown links can be full path or basename
		return target == moveFrom || target == moveFromBasename

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


// processRenameWithRipgrep uses ripgrep to find candidate files and process them
func (rp *RenameProcessor) processRenameWithRipgrep(ctx context.Context, move FileMove, options RenameOptions, result *RenameResult) ([]*vault.VaultFile, []error) {
	var modifiedFiles []*vault.VaultFile
	var errors []error

	// Try to use ripgrep for faster file discovery
	candidateFiles, rgErr := rp.findCandidateFilesWithRipgrep(ctx, move, options)
	
	if rgErr != nil {
		if options.Verbose {
			fmt.Printf("Ripgrep unavailable, falling back to full scan: %v\n", rgErr)
		}
		// Fallback to original approach if ripgrep failed
		return rp.processFullRenameFallback(ctx, move, options, result)
	}

	if options.Verbose {
		fmt.Printf("Ripgrep found %d candidate files to examine\n", len(candidateFiles))
	}

	// Process only files that ripgrep found containing potential matches
	for _, filePath := range candidateFiles {
		vaultFile := &vault.VaultFile{Path: filePath}
		content, err := os.ReadFile(filePath)
		if err != nil {
			errors = append(errors, fmt.Errorf("reading %s: %w", filePath, err))
			continue
		}
		
		if err := vaultFile.Parse(content); err != nil {
			errors = append(errors, fmt.Errorf("parsing %s: %w", filePath, err))
			continue
		}
		
		// Set relative path
		if rel, err := filepath.Rel(options.VaultRoot, filePath); err == nil {
			vaultFile.RelativePath = rel
		}
		
		processResult := rp.processFile(vaultFile, move, nil, options.DryRun)
		result.FilesScanned++
		
		if processResult.Error != nil {
			errors = append(errors, processResult.Error)
			continue
		}
		
		if processResult.Modified {
			result.FilesModified++
			result.LinksUpdated += processResult.LinksUpdated
			result.ModifiedFiles = append(result.ModifiedFiles, vaultFile.RelativePath)
			modifiedFiles = append(modifiedFiles, vaultFile)
			
			if options.Verbose {
				fmt.Printf("Examining: %s - updated %d links\n", vaultFile.RelativePath, processResult.LinksUpdated)
			}
		} else if options.Verbose {
			fmt.Printf("Examining: %s - no changes needed\n", vaultFile.RelativePath)
		}
	}

	return modifiedFiles, errors
}

// findCandidateFilesWithRipgrep uses ripgrep to quickly find files that might contain references
func (rp *RenameProcessor) findCandidateFilesWithRipgrep(ctx context.Context, move FileMove, options RenameOptions) ([]string, error) {
	// Create search patterns for the file being renamed
	sourceFile := filepath.Base(move.From)
	sourceWithoutExt := strings.TrimSuffix(sourceFile, ".md")
	
	// Build ripgrep pattern that matches potential links
	// This is a broad pattern to catch wiki links, markdown links, and embeds
	pattern := fmt.Sprintf(`(\[\[%s(\]\]|\|)|\]?\(%s\)|!\[\[%s)`, 
		regexp.QuoteMeta(sourceWithoutExt),
		regexp.QuoteMeta(sourceFile), 
		regexp.QuoteMeta(sourceWithoutExt))
	
	// Use ripgrep to find files containing potential references
	cmd := exec.CommandContext(ctx, "rg", 
		"--files-with-matches",  // Only return filenames, not content
		"--type", "md",          // Only markdown files
		"--case-insensitive",    // Case insensitive search
		pattern, 
		options.VaultRoot)
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ripgrep execution failed: %w", err)
	}
	
	// Parse ripgrep output (one filename per line)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var files []string
	for _, line := range lines {
		if line != "" {
			files = append(files, strings.TrimSpace(line))
		}
	}
	
	return files, nil
}

// processFullRenameFallback is the original comprehensive approach
func (rp *RenameProcessor) processFullRenameFallback(ctx context.Context, move FileMove, options RenameOptions, result *RenameResult) ([]*vault.VaultFile, []error) {
	var modifiedFiles []*vault.VaultFile
	var errors []error

	// Pre-compile regex patterns for performance
	linkRegex := rp.compileOptimizedLinkRegex(move.From)

	// Scan vault for files - stream processing instead of loading all at once
	fileStream := make(chan *vault.VaultFile, 100)
	go func() {
		defer close(fileStream)
		err := rp.scanner.WalkWithCallback(options.VaultRoot, func(file *vault.VaultFile) error {
			select {
			case fileStream <- file:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
		if err != nil && options.Verbose {
			fmt.Printf("Warning: error during vault scan: %v\n", err)
		}
	}()

	// Process files in parallel with worker pool
	processResults := make(chan FileProcessResult, rp.workers)
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < rp.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rp.processFileWorker(ctx, fileStream, processResults, move, linkRegex, options.DryRun)
		}()
	}

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(processResults)
	}()

	// Collect results
	for processResult := range processResults {
		result.FilesScanned++

		if processResult.Error != nil {
			errors = append(errors, processResult.Error)
			if options.Verbose {
				fmt.Printf("Error processing %s: %v\n", processResult.File.RelativePath, processResult.Error)
			}
			continue
		}

		if processResult.Modified {
			result.FilesModified++
			result.LinksUpdated += processResult.LinksUpdated
			result.ModifiedFiles = append(result.ModifiedFiles, processResult.File.RelativePath)
			modifiedFiles = append(modifiedFiles, processResult.File)

			if options.Verbose {
				fmt.Printf("Examining: %s - updated %d links\n", processResult.File.RelativePath, processResult.LinksUpdated)
			}
		} else if options.Verbose {
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