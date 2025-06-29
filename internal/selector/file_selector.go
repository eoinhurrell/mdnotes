package selector

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/eoinhurrell/mdnotes/internal/query"
	"github.com/eoinhurrell/mdnotes/internal/vault"
	"github.com/spf13/cobra"
)

// SelectionMode determines how files are selected
type SelectionMode int

const (
	// AutoDetect automatically detects if path is file or directory
	AutoDetect SelectionMode = iota
	// FilesFromQuery uses query results as file list
	FilesFromQuery
	// FilesFromStdin reads file paths from stdin
	FilesFromStdin
	// FilesFromFile reads file paths from a text file
	FilesFromFile
)

// FileSelector provides unified file selection across all commands
type FileSelector struct {
	IgnorePatterns []string
	QueryFilter    string // Optional query to filter files
	SourceFile     string // File path for FilesFromFile mode
}

// SelectionResult contains the results of file selection
type SelectionResult struct {
	Files       []*vault.VaultFile
	ParseErrors []vault.ParseError
	Mode        SelectionMode
	Source      string // Description of selection source
}

// NewFileSelector creates a new file selector with default settings
func NewFileSelector() *FileSelector {
	return &FileSelector{
		IgnorePatterns: []string{".obsidian/*", "*.tmp"},
	}
}

// WithIgnorePatterns sets ignore patterns for scanning
func (fs *FileSelector) WithIgnorePatterns(patterns []string) *FileSelector {
	fs.IgnorePatterns = patterns
	return fs
}

// WithQuery sets a query filter
func (fs *FileSelector) WithQuery(query string) *FileSelector {
	fs.QueryFilter = query
	return fs
}

// WithSourceFile sets the source file for FilesFromFile mode
func (fs *FileSelector) WithSourceFile(path string) *FileSelector {
	fs.SourceFile = path
	return fs
}

// SelectFiles selects files based on the specified mode and input
func (fs *FileSelector) SelectFiles(input string, mode SelectionMode) (*SelectionResult, error) {
	switch mode {
	case AutoDetect:
		return fs.selectAutoDetect(input)
	case FilesFromQuery:
		return fs.selectFromQuery(input)
	case FilesFromStdin:
		return fs.selectFromStdin()
	case FilesFromFile:
		return fs.selectFromFile(fs.SourceFile)
	default:
		return nil, fmt.Errorf("unknown selection mode: %d", mode)
	}
}

// selectAutoDetect automatically detects if input is file or directory
func (fs *FileSelector) selectAutoDetect(path string) (*SelectionResult, error) {
	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("path error: %w", err)
	}

	var files []*vault.VaultFile
	var parseErrors []vault.ParseError
	var source string

	if info.IsDir() {
		// Scan directory
		scanner := vault.NewScanner(
			vault.WithIgnorePatterns(fs.IgnorePatterns),
			vault.WithContinueOnErrors(),
		)
		files, err = scanner.Walk(path)
		if err != nil {
			return nil, fmt.Errorf("scanning directory: %w", err)
		}
		parseErrors = scanner.GetParseErrors()
		source = fmt.Sprintf("directory: %s", path)
	} else {
		// Single file
		if !strings.HasSuffix(path, ".md") {
			return nil, fmt.Errorf("file must have .md extension")
		}

		file, err := fs.loadSingleFile(path)
		if err != nil {
			return nil, err
		}
		files = []*vault.VaultFile{file}
		source = fmt.Sprintf("file: %s", path)
	}

	// Apply query filter if specified
	if fs.QueryFilter != "" {
		filteredFiles, err := fs.applyQueryFilter(files)
		if err != nil {
			return nil, fmt.Errorf("applying query filter: %w", err)
		}
		files = filteredFiles
		source += fmt.Sprintf(" (filtered by query: %s)", fs.QueryFilter)
	}

	return &SelectionResult{
		Files:       files,
		ParseErrors: parseErrors,
		Mode:        AutoDetect,
		Source:      source,
	}, nil
}

// selectFromQuery selects files by running a query on the specified path
func (fs *FileSelector) selectFromQuery(path string) (*SelectionResult, error) {
	if fs.QueryFilter == "" {
		return nil, fmt.Errorf("query filter is required for FilesFromQuery mode")
	}

	// First scan all files in the path
	scanner := vault.NewScanner(
		vault.WithIgnorePatterns(fs.IgnorePatterns),
		vault.WithContinueOnErrors(),
	)
	allFiles, err := scanner.Walk(path)
	if err != nil {
		return nil, fmt.Errorf("scanning directory for query: %w", err)
	}

	// Apply query filter
	filteredFiles, err := fs.applyQueryFilter(allFiles)
	if err != nil {
		return nil, fmt.Errorf("applying query: %w", err)
	}

	return &SelectionResult{
		Files:       filteredFiles,
		ParseErrors: scanner.GetParseErrors(),
		Mode:        FilesFromQuery,
		Source:      fmt.Sprintf("query: %s on %s", fs.QueryFilter, path),
	}, nil
}

// selectFromStdin reads file paths from stdin
func (fs *FileSelector) selectFromStdin() (*SelectionResult, error) {
	return fs.selectFromReader(os.Stdin, "stdin", FilesFromStdin)
}

// selectFromFile reads file paths from a text file
func (fs *FileSelector) selectFromFile(filePath string) (*SelectionResult, error) {
	if filePath == "" {
		return nil, fmt.Errorf("source file path is required for FilesFromFile mode")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening source file: %w", err)
	}
	defer file.Close()

	return fs.selectFromReader(file, fmt.Sprintf("file: %s", filePath), FilesFromFile)
}

// selectFromReader reads file paths from any reader
func (fs *FileSelector) selectFromReader(reader io.Reader, sourceName string, mode SelectionMode) (*SelectionResult, error) {
	var files []*vault.VaultFile
	var parseErrors []vault.ParseError

	scanner := bufio.NewScanner(reader)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Validate and load the file
		if !strings.HasSuffix(line, ".md") {
			parseErrors = append(parseErrors, vault.ParseError{
				Path:  line,
				Error: fmt.Errorf("line %d: file must have .md extension", lineNum),
			})
			continue
		}

		// Check if file exists
		if _, err := os.Stat(line); os.IsNotExist(err) {
			parseErrors = append(parseErrors, vault.ParseError{
				Path:  line,
				Error: fmt.Errorf("line %d: file does not exist", lineNum),
			})
			continue
		}

		file, err := fs.loadSingleFile(line)
		if err != nil {
			parseErrors = append(parseErrors, vault.ParseError{
				Path:  line,
				Error: fmt.Errorf("line %d: %w", lineNum, err),
			})
			continue
		}

		files = append(files, file)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading from %s: %w", sourceName, err)
	}

	// Apply query filter if specified
	if fs.QueryFilter != "" {
		filteredFiles, err := fs.applyQueryFilter(files)
		if err != nil {
			return nil, fmt.Errorf("applying query filter: %w", err)
		}
		files = filteredFiles
	}

	return &SelectionResult{
		Files:       files,
		ParseErrors: parseErrors,
		Mode:        mode,
		Source:      sourceName,
	}, nil
}

// loadSingleFile loads and parses a single markdown file
func (fs *FileSelector) loadSingleFile(path string) (*vault.VaultFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	// Get file info for modification time
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("getting file info: %w", err)
	}

	// Determine relative path - use basename if we can't determine a better one
	relativePath := filepath.Base(path)

	// Try to make it relative to current working directory
	if cwd, err := os.Getwd(); err == nil {
		if relPath, err := filepath.Rel(cwd, path); err == nil && !strings.HasPrefix(relPath, "..") {
			relativePath = relPath
		}
	}

	vf := &vault.VaultFile{
		Path:         path,
		RelativePath: relativePath,
		Modified:     info.ModTime(),
	}

	if err := vf.Parse(content); err != nil {
		return nil, fmt.Errorf("parsing file: %w", err)
	}

	return vf, nil
}

// applyQueryFilter applies the query filter to a list of files
func (fs *FileSelector) applyQueryFilter(files []*vault.VaultFile) ([]*vault.VaultFile, error) {
	if fs.QueryFilter == "" {
		return files, nil
	}

	// Parse the query expression
	parser := query.NewParser(fs.QueryFilter)
	expr, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("parsing query expression: %w", err)
	}

	// Filter files that match the query
	var filteredFiles []*vault.VaultFile
	for _, file := range files {
		if expr.Evaluate(file) {
			filteredFiles = append(filteredFiles, file)
		}
	}

	return filteredFiles, nil
}

// GetSelectionSummary returns a human-readable summary of the selection
func (result *SelectionResult) GetSelectionSummary() string {
	summary := fmt.Sprintf("Selected %d files from %s", len(result.Files), result.Source)

	if len(result.ParseErrors) > 0 {
		summary += fmt.Sprintf(" (%d parse errors)", len(result.ParseErrors))
	}

	return summary
}

// PrintParseErrors prints any parse errors encountered during selection
func (result *SelectionResult) PrintParseErrors() {
	if len(result.ParseErrors) == 0 {
		return
	}

	fmt.Fprintf(os.Stderr, "Warning: %d files had errors during selection:\n", len(result.ParseErrors))
	for _, parseErr := range result.ParseErrors {
		fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", parseErr.Path, parseErr.Error)
	}
	fmt.Fprintf(os.Stderr, "\n")
}

// GetGlobalSelectionConfig extracts global file selection flags from a cobra command
// and returns the appropriate selection mode and configured FileSelector
func GetGlobalSelectionConfig(cmd *cobra.Command) (SelectionMode, *FileSelector, error) {
	// Get global flags - check both the command and its root for persistent flags
	query, _ := cmd.Root().PersistentFlags().GetString("query")
	fromFile, _ := cmd.Root().PersistentFlags().GetString("from-file")
	fromStdin, _ := cmd.Root().PersistentFlags().GetBool("from-stdin")
	ignorePatterns, _ := cmd.Root().PersistentFlags().GetStringSlice("ignore")

	// Determine selection mode based on flags
	mode := AutoDetect
	if fromStdin {
		mode = FilesFromStdin
	} else if fromFile != "" {
		mode = FilesFromFile
	} else if query != "" {
		mode = FilesFromQuery
	}

	// Create and configure FileSelector
	fileSelector := NewFileSelector().
		WithIgnorePatterns(ignorePatterns).
		WithQuery(query).
		WithSourceFile(fromFile)

	return mode, fileSelector, nil
}
