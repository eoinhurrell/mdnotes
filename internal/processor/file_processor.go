package processor

import (
	"fmt"
	"os"

	"github.com/eoinhurrell/mdnotes/internal/selector"
	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// FileProcessor provides a reusable interface for processing vault files
type FileProcessor struct {
	DryRun         bool
	Verbose        bool
	Quiet          bool
	IgnorePatterns []string
	QueryFilter    string              // Query to filter files
	SelectionMode  selector.SelectionMode // How to select files
	SourceFile     string              // For FilesFromFile mode

	// Callbacks
	ProcessFile     func(file *vault.VaultFile) (modified bool, err error)
	OnFileProcessed func(file *vault.VaultFile, modified bool)
	OnProgress      func(current, total int, filename string)
}

// ProcessResult contains the results of a file processing operation
type ProcessResult struct {
	TotalFiles     int
	ProcessedFiles int
	Errors         []error
	Selection      *selector.SelectionResult // Information about file selection
}

// ProcessPath processes files at the given path using the configured selection mode
func (fp *FileProcessor) ProcessPath(path string) (*ProcessResult, error) {
	// Create file selector with processor settings
	fileSelector := selector.NewFileSelector().
		WithIgnorePatterns(fp.IgnorePatterns).
		WithQuery(fp.QueryFilter).
		WithSourceFile(fp.SourceFile)

	// Determine selection mode (default to AutoDetect)
	mode := fp.SelectionMode
	if mode == 0 {
		mode = selector.AutoDetect
	}

	// Select files
	selection, err := fileSelector.SelectFiles(path, mode)
	if err != nil {
		return nil, err
	}

	// Print parse errors if in verbose mode
	if fp.Verbose && len(selection.ParseErrors) > 0 {
		selection.PrintParseErrors()
	}

	files := selection.Files
	if len(files) == 0 {
		if fp.Verbose {
			fmt.Printf("No markdown files selected from %s\n", selection.Source)
		}
		return &ProcessResult{
			TotalFiles: 0, 
			ProcessedFiles: 0,
			Selection: selection,
		}, nil
	}

	if fp.Verbose {
		fmt.Printf("%s\n", selection.GetSelectionSummary())
	}

	// Process files
	result := &ProcessResult{
		TotalFiles: len(files),
		Errors:     []error{},
		Selection:  selection,
	}

	for i, file := range files {
		// Progress callback
		if fp.OnProgress != nil {
			fp.OnProgress(i+1, len(files), file.RelativePath)
		}

		// Process the file
		modified, err := fp.ProcessFile(file)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", file.RelativePath, err))
			continue
		}

		if modified {
			result.ProcessedFiles++

			// Write file back if not dry run
			if !fp.DryRun {
				if err := fp.writeFile(file); err != nil {
					result.Errors = append(result.Errors, fmt.Errorf("writing %s: %w", file.Path, err))
					continue
				}
			}
		}

		// File processed callback
		if fp.OnFileProcessed != nil {
			fp.OnFileProcessed(file, modified)
		}
	}

	return result, nil
}


// writeFile writes a vault file back to disk, preserving frontmatter order
func (fp *FileProcessor) writeFile(file *vault.VaultFile) error {
	content, err := file.Serialize()
	if err != nil {
		return fmt.Errorf("serializing: %w", err)
	}

	if err := os.WriteFile(file.Path, content, 0644); err != nil {
		return fmt.Errorf("writing: %w", err)
	}

	return nil
}

// PrintSummary prints a standardized summary of the processing results
func (fp *FileProcessor) PrintSummary(result *ProcessResult) {
	// Always show errors, even in quiet mode
	if len(result.Errors) > 0 {
		for _, err := range result.Errors {
			fmt.Printf("✗ %v\n", err)
		}
	}

	// Show summary unless quiet mode is enabled
	if !fp.Quiet {
		if fp.DryRun {
			fmt.Printf("\nDry run completed. Would modify %d files.\n", result.ProcessedFiles)
		} else {
			fmt.Printf("\nCompleted. Modified %d files.\n", result.ProcessedFiles)
		}
	}
}
