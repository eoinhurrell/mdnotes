package vault

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Scanner walks directories and finds markdown files
type Scanner struct {
	ignorePatterns   []string
	continueOnErrors bool
	parseErrors      []ParseError
}

// ParseError represents a file parsing error
type ParseError struct {
	Path  string
	Error error
}

// ScannerOption configures a Scanner
type ScannerOption func(*Scanner)

// WithIgnorePatterns sets ignore patterns for the scanner
func WithIgnorePatterns(patterns []string) ScannerOption {
	return func(s *Scanner) {
		s.ignorePatterns = patterns
	}
}

// WithContinueOnErrors configures the scanner to continue on parsing errors
func WithContinueOnErrors() ScannerOption {
	return func(s *Scanner) {
		s.continueOnErrors = true
	}
}

// NewScanner creates a new scanner with optional configuration
func NewScanner(opts ...ScannerOption) *Scanner {
	s := &Scanner{
		ignorePatterns: []string{},
		parseErrors:    []ParseError{},
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// GetParseErrors returns any parsing errors encountered during scanning
func (s *Scanner) GetParseErrors() []ParseError {
	return s.parseErrors
}

// Walk scans a directory tree and returns all markdown files
func (s *Scanner) Walk(root string) ([]*VaultFile, error) {
	var files []*VaultFile

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Get relative path from root
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		// Check if path should be ignored
		if s.shouldIgnore(relPath) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process markdown files
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		// Load the file
		vf, err := s.loadFile(path, relPath)
		if err != nil {
			if s.continueOnErrors {
				// Store the error and continue
				s.parseErrors = append(s.parseErrors, ParseError{
					Path:  relPath,
					Error: err,
				})
				return nil
			}
			return fmt.Errorf("loading %s: %w", path, err)
		}

		files = append(files, vf)
		return nil
	})

	return files, err
}

// shouldIgnore checks if a path matches any ignore pattern
func (s *Scanner) shouldIgnore(path string) bool {
	for _, pattern := range s.ignorePatterns {
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}

		// Check if any parent directory matches the pattern
		// This handles patterns like ".obsidian/*"
		if strings.Contains(pattern, "/*") {
			prefix := strings.TrimSuffix(pattern, "/*")
			if strings.HasPrefix(path, prefix+"/") || path == prefix {
				return true
			}
		}
	}
	return false
}

// loadFile reads and parses a markdown file
func (s *Scanner) loadFile(path, relPath string) (*VaultFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Get file info for modification time
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	vf := &VaultFile{
		Path:         path,
		RelativePath: relPath,
		Modified:     info.ModTime(),
	}

	if err := vf.Parse(content); err != nil {
		return nil, err
	}

	return vf, nil
}
