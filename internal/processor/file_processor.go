package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// FileProcessor provides a reusable interface for processing vault files
type FileProcessor struct {
	DryRun         bool
	Verbose        bool
	IgnorePatterns []string
	
	// Callbacks
	ProcessFile   func(file *vault.VaultFile) (modified bool, err error)
	OnFileProcessed func(file *vault.VaultFile, modified bool)
	OnProgress    func(current, total int, filename string)
}

// ProcessResult contains the results of a file processing operation
type ProcessResult struct {
	TotalFiles     int
	ProcessedFiles int
	Errors         []error
}

// ProcessPath processes files at the given path (file or directory)
func (fp *FileProcessor) ProcessPath(path string) (*ProcessResult, error) {
	// Load files
	files, err := fp.loadFiles(path)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		if fp.Verbose {
			fmt.Println("No markdown files found")
		}
		return &ProcessResult{TotalFiles: 0, ProcessedFiles: 0}, nil
	}

	// Process files
	result := &ProcessResult{
		TotalFiles: len(files),
		Errors:     []error{},
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

// loadFiles loads files from the given path (file or directory)
func (fp *FileProcessor) loadFiles(path string) ([]*vault.VaultFile, error) {
	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("path error: %w", err)
	}

	var files []*vault.VaultFile

	if info.IsDir() {
		// Scan directory
		scanner := vault.NewScanner(vault.WithIgnorePatterns(fp.IgnorePatterns))
		files, err = scanner.Walk(path)
		if err != nil {
			return nil, fmt.Errorf("scanning directory: %w", err)
		}
	} else {
		// Single file
		if !strings.HasSuffix(path, ".md") {
			return nil, fmt.Errorf("file must have .md extension")
		}
		
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading file: %w", err)
		}

		// Get file info for modification time
		fileInfo, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("getting file info: %w", err)
		}

		vf := &vault.VaultFile{
			Path:         path,
			RelativePath: filepath.Base(path),
			Modified:     fileInfo.ModTime(),
		}
		if err := vf.Parse(content); err != nil {
			return nil, fmt.Errorf("parsing file: %w", err)
		}
		files = []*vault.VaultFile{vf}
	}

	return files, nil
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
	if len(result.Errors) > 0 {
		for _, err := range result.Errors {
			fmt.Printf("✗ %v\n", err)
		}
	}

	if fp.DryRun {
		fmt.Printf("\nDry run completed. Would modify %d files.\n", result.ProcessedFiles)
	} else {
		fmt.Printf("\nCompleted. Modified %d files.\n", result.ProcessedFiles)
	}
}