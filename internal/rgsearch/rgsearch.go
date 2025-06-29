package rgsearch

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// SearchResult represents a single search result from ripgrep
type SearchResult struct {
	Type string           `json:"type"`
	Data SearchResultData `json:"data"`
}

// SearchResultData contains the actual search result data
type SearchResultData struct {
	Path           PathData   `json:"path"`
	Lines          LinesData  `json:"lines"`
	LineNumber     *int       `json:"line_number,omitempty"`
	AbsoluteOffset *int       `json:"absolute_offset,omitempty"`
	Submatches     []Submatch `json:"submatches,omitempty"`
}

// PathData contains file path information
type PathData struct {
	Text string `json:"text"`
}

// LinesData contains line information
type LinesData struct {
	Text string `json:"text"`
}

// Submatch contains match information
type Submatch struct {
	Match MatchData `json:"match"`
	Start int       `json:"start"`
	End   int       `json:"end"`
}

// MatchData contains the matched text
type MatchData struct {
	Text string `json:"text"`
}

// SearchOptions configures ripgrep search behavior
type SearchOptions struct {
	Pattern         string
	Path            string
	CaseSensitive   bool
	WordBoundary    bool
	FixedStrings    bool
	Regex           bool
	IncludePatterns []string
	ExcludePatterns []string
	MaxDepth        int
	FollowSymlinks  bool
	SearchZip       bool
	Multiline       bool
	ContextBefore   int
	ContextAfter    int
	MaxMatches      int
	MaxFileMatches  int
	Timeout         time.Duration
	AdditionalArgs  []string
}

// DefaultSearchOptions returns sensible defaults for ripgrep search
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		CaseSensitive:  false,
		WordBoundary:   false,
		FixedStrings:   false,
		Regex:          true,
		MaxDepth:       -1,
		FollowSymlinks: false,
		SearchZip:      false,
		Multiline:      false,
		ContextBefore:  0,
		ContextAfter:   0,
		MaxMatches:     1000,
		MaxFileMatches: 100,
		Timeout:        30 * time.Second,
	}
}

// Searcher wraps ripgrep functionality
type Searcher struct {
	rgPath    string
	available bool
}

// NewSearcher creates a new ripgrep searcher
func NewSearcher() *Searcher {
	searcher := &Searcher{}
	searcher.rgPath, searcher.available = searcher.findRipgrep()
	return searcher
}

// IsAvailable returns true if ripgrep is available on the system
func (s *Searcher) IsAvailable() bool {
	return s.available
}

// GetVersion returns the version of ripgrep
func (s *Searcher) GetVersion() (string, error) {
	if !s.available {
		return "", fmt.Errorf("ripgrep not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, s.rgPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting ripgrep version: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0]), nil
	}

	return "unknown", nil
}

// Search performs a search using ripgrep with JSON output
func (s *Searcher) Search(ctx context.Context, options SearchOptions) ([]SearchResult, error) {
	if !s.available {
		return nil, fmt.Errorf("ripgrep not available")
	}

	if options.Pattern == "" {
		return nil, fmt.Errorf("search pattern is required")
	}

	args := s.buildArgs(options)

	// Create command with context
	cmd := exec.CommandContext(ctx, s.rgPath, args...)

	// Set up output parsing
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting ripgrep command: %w", err)
	}

	// Parse JSON output
	var results []SearchResult
	scanner := bufio.NewScanner(stdout)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var result SearchResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			// Skip malformed JSON lines
			continue
		}

		// Only collect match results
		if result.Type == "match" {
			results = append(results, result)
		}
	}

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		// Exit code 1 means no matches found, which is not an error
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return results, nil
		}
		return nil, fmt.Errorf("ripgrep command failed: %w", err)
	}

	return results, nil
}

// SearchFiles returns just the file paths that contain matches
func (s *Searcher) SearchFiles(ctx context.Context, options SearchOptions) ([]string, error) {
	results, err := s.Search(ctx, options)
	if err != nil {
		return nil, err
	}

	// Extract unique file paths
	pathMap := make(map[string]bool)
	for _, result := range results {
		pathMap[result.Data.Path.Text] = true
	}

	paths := make([]string, 0, len(pathMap))
	for path := range pathMap {
		paths = append(paths, path)
	}

	return paths, nil
}

// SearchInFiles searches for patterns within specific files
func (s *Searcher) SearchInFiles(ctx context.Context, pattern string, files []string, options SearchOptions) ([]SearchResult, error) {
	if len(files) == 0 {
		return nil, nil
	}

	// Override path and pattern
	options.Pattern = pattern
	options.Path = "" // Will be handled by file list

	args := s.buildArgs(options)

	// Add file list
	args = append(args, files...)

	cmd := exec.CommandContext(ctx, s.rgPath, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting ripgrep command: %w", err)
	}

	var results []SearchResult
	scanner := bufio.NewScanner(stdout)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var result SearchResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			continue
		}

		if result.Type == "match" {
			results = append(results, result)
		}
	}

	if err := cmd.Wait(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return results, nil
		}
		return nil, fmt.Errorf("ripgrep command failed: %w", err)
	}

	return results, nil
}

// CountMatches returns the number of matches for a pattern
func (s *Searcher) CountMatches(ctx context.Context, options SearchOptions) (int, error) {
	if !s.available {
		return 0, fmt.Errorf("ripgrep not available")
	}

	args := s.buildArgs(options)
	args = append(args, "--count")

	cmd := exec.CommandContext(ctx, s.rgPath, args...)
	output, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return 0, nil
		}
		return 0, fmt.Errorf("ripgrep count command failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	count := 0
	for _, line := range lines {
		if line != "" {
			count++
		}
	}

	return count, nil
}

// findRipgrep locates the ripgrep binary
func (s *Searcher) findRipgrep() (string, bool) {
	// Try common names
	names := []string{"rg", "ripgrep"}

	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			return path, true
		}
	}

	// Try common installation locations
	commonPaths := []string{
		"/usr/local/bin/rg",
		"/usr/bin/rg",
		"/opt/homebrew/bin/rg",
		filepath.Join(os.Getenv("HOME"), ".cargo/bin/rg"),
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}

	return "", false
}

// buildArgs constructs ripgrep command line arguments
func (s *Searcher) buildArgs(options SearchOptions) []string {
	args := []string{"--json"}

	// Pattern type
	if options.FixedStrings {
		args = append(args, "--fixed-strings")
	}
	// Note: --regex is the default for ripgrep, so we don't need to specify it
	// Using --regexp incorrectly treats the pattern as a file path

	// Case sensitivity
	if options.CaseSensitive {
		args = append(args, "--case-sensitive")
	} else {
		args = append(args, "--ignore-case")
	}

	// Word boundary
	if options.WordBoundary {
		args = append(args, "--word-regexp")
	}

	// Multiline
	if options.Multiline {
		args = append(args, "--multiline")
	}

	// Context
	if options.ContextBefore > 0 {
		args = append(args, fmt.Sprintf("--before-context=%d", options.ContextBefore))
	}
	if options.ContextAfter > 0 {
		args = append(args, fmt.Sprintf("--after-context=%d", options.ContextAfter))
	}

	// Max matches
	if options.MaxMatches > 0 {
		args = append(args, fmt.Sprintf("--max-count=%d", options.MaxMatches))
	}

	// Max depth
	if options.MaxDepth > 0 {
		args = append(args, fmt.Sprintf("--max-depth=%d", options.MaxDepth))
	}

	// Follow symlinks
	if options.FollowSymlinks {
		args = append(args, "--follow")
	}

	// Search zip files
	if options.SearchZip {
		args = append(args, "--search-zip")
	}

	// Include patterns
	for _, pattern := range options.IncludePatterns {
		args = append(args, "--glob", pattern)
	}

	// Exclude patterns
	for _, pattern := range options.ExcludePatterns {
		args = append(args, "--glob", "!"+pattern)
	}

	// Additional args
	args = append(args, options.AdditionalArgs...)

	// Pattern
	args = append(args, options.Pattern)

	// Path (if specified)
	if options.Path != "" {
		args = append(args, options.Path)
	}

	return args
}

// ValidatePattern checks if a regex pattern is valid
func ValidatePattern(pattern string) error {
	_, err := regexp.Compile(pattern)
	return err
}

// EscapePattern escapes special regex characters in a string
func EscapePattern(s string) string {
	return regexp.QuoteMeta(s)
}
