package security

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/eoinhurrell/mdnotes/internal/errors"
)

// PathSanitizer provides secure path handling functionality
type PathSanitizer struct {
	allowedRoots []string
	maxDepth     int
}

// NewPathSanitizer creates a new path sanitizer
func NewPathSanitizer(allowedRoots []string, maxDepth int) *PathSanitizer {
	if maxDepth <= 0 {
		maxDepth = 32 // Default maximum depth
	}

	return &PathSanitizer{
		allowedRoots: allowedRoots,
		maxDepth:     maxDepth,
	}
}

// SanitizePath safely cleans and validates a file path
func (ps *PathSanitizer) SanitizePath(inputPath string) (string, error) {
	// Clean the path first
	cleaned := filepath.Clean(inputPath)

	// Convert to absolute path
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", errors.NewErrorBuilder().
			WithOperation("path sanitization").
			WithError(fmt.Errorf("failed to resolve absolute path: %w", err)).
			WithCode(errors.ErrCodePathInvalid).
			WithSuggestion("Ensure the path is valid and accessible").
			Build()
	}

	// Check for path traversal attempts
	if err := ps.checkTraversal(abs); err != nil {
		return "", err
	}

	// Check depth limits
	if err := ps.checkDepth(abs); err != nil {
		return "", err
	}

	// Check against allowed roots if specified
	if len(ps.allowedRoots) > 0 {
		if err := ps.checkAllowedRoots(abs); err != nil {
			return "", err
		}
	}

	return abs, nil
}

// SanitizeFilename sanitizes a filename for safe usage
func SanitizeFilename(filename string) string {
	// Remove or replace dangerous characters
	dangerous := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f\x7f]`)
	sanitized := dangerous.ReplaceAllString(filename, "_")

	// Remove leading/trailing dots and spaces
	sanitized = strings.Trim(sanitized, ". ")

	// Remove multiple consecutive underscores and clean up
	underscoreRegex := regexp.MustCompile(`_+`)
	sanitized = underscoreRegex.ReplaceAllString(sanitized, "_")
	sanitized = strings.Trim(sanitized, "_")

	// Check for reserved names (Windows)
	reserved := []string{"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"}

	upper := strings.ToUpper(sanitized)
	for _, res := range reserved {
		if upper == res || strings.HasPrefix(upper, res+".") {
			sanitized = "_" + sanitized
			break
		}
	}

	// Ensure filename is not empty
	if sanitized == "" {
		sanitized = "untitled"
	}

	// Limit length (most filesystems support 255 bytes)
	if len(sanitized) > 255 {
		ext := filepath.Ext(sanitized)
		base := sanitized[:255-len(ext)]
		sanitized = base + ext
	}

	return sanitized
}

// ValidatePath checks if a path is safe to use
func (ps *PathSanitizer) ValidatePath(path string) error {
	_, err := ps.SanitizePath(path)
	return err
}

// IsWithinAllowedRoots checks if a path is within allowed root directories
func (ps *PathSanitizer) IsWithinAllowedRoots(path string) bool {
	if len(ps.allowedRoots) == 0 {
		return true // No restrictions
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	for _, root := range ps.allowedRoots {
		rootAbs, err := filepath.Abs(root)
		if err != nil {
			continue
		}

		rel, err := filepath.Rel(rootAbs, abs)
		if err != nil {
			continue
		}

		// Check if the path is within this root (doesn't start with ..)
		if !strings.HasPrefix(rel, "..") {
			return true
		}
	}

	return false
}

// checkTraversal checks for path traversal attempts
func (ps *PathSanitizer) checkTraversal(path string) error {
	// Check for obvious traversal patterns
	if strings.Contains(path, "..") {
		return errors.NewErrorBuilder().
			WithOperation("path traversal check").
			WithError(fmt.Errorf("path contains traversal sequence: %s", path)).
			WithCode(errors.ErrCodePathInvalid).
			WithSuggestion("Remove '..' sequences from the path").
			Build()
	}

	// Check for encoded traversal attempts
	encoded := []string{
		"%2e%2e",     // ..
		"%2e%2e%2f",  // ../
		"%2e%2e%5c",  // ..\
		"%252e%252e", // double-encoded ..
		"..%2f",      // ../
		"..%5c",      // ..\
	}

	lowerPath := strings.ToLower(path)
	for _, pattern := range encoded {
		if strings.Contains(lowerPath, pattern) {
			return errors.NewErrorBuilder().
				WithOperation("path traversal check").
				WithError(fmt.Errorf("path contains encoded traversal sequence: %s", path)).
				WithCode(errors.ErrCodePathInvalid).
				WithSuggestion("Remove encoded traversal sequences from the path").
				Build()
		}
	}

	return nil
}

// checkDepth checks if path exceeds maximum depth
func (ps *PathSanitizer) checkDepth(path string) error {
	depth := strings.Count(path, string(os.PathSeparator))
	if depth > ps.maxDepth {
		return errors.NewErrorBuilder().
			WithOperation("path depth check").
			WithError(fmt.Errorf("path depth %d exceeds maximum %d", depth, ps.maxDepth)).
			WithCode(errors.ErrCodePathInvalid).
			WithSuggestion(fmt.Sprintf("Reduce path depth to %d or less", ps.maxDepth)).
			Build()
	}

	return nil
}

// checkAllowedRoots checks if path is within allowed root directories
func (ps *PathSanitizer) checkAllowedRoots(path string) error {
	if !ps.IsWithinAllowedRoots(path) {
		return errors.NewErrorBuilder().
			WithOperation("allowed roots check").
			WithError(fmt.Errorf("path is outside allowed root directories: %s", path)).
			WithCode(errors.ErrCodePathInvalid).
			WithSuggestion("Ensure the path is within allowed directories").
			Build()
	}

	return nil
}

// SecureJoin safely joins path components
func SecureJoin(base string, elem ...string) (string, error) {
	ps := NewPathSanitizer([]string{base}, 32)

	// Start with base path
	result := base

	// Join each element
	for _, e := range elem {
		// Sanitize each element first
		sanitized := SanitizeFilename(e)

		// Join with base
		candidate := filepath.Join(result, sanitized)

		// Validate the result
		if err := ps.ValidatePath(candidate); err != nil {
			return "", err
		}

		result = candidate
	}

	return result, nil
}

// IsHiddenFile checks if a file or directory is hidden
func IsHiddenFile(name string) bool {
	base := filepath.Base(name)

	// Unix-style hidden files (start with .)
	if strings.HasPrefix(base, ".") {
		return true
	}

	// Windows-style hidden files (check attributes on Windows)
	// This is a simplified check - would need platform-specific code for full Windows support
	return false
}

// IsSafeExtension checks if a file extension is considered safe
func IsSafeExtension(filename string, allowedExtensions []string) bool {
	if len(allowedExtensions) == 0 {
		return true // No restrictions
	}

	ext := strings.ToLower(filepath.Ext(filename))

	for _, allowed := range allowedExtensions {
		if strings.ToLower(allowed) == ext {
			return true
		}
	}

	return false
}

// DangerousExtensions returns a list of potentially dangerous file extensions
func DangerousExtensions() []string {
	return []string{
		// Executable files
		".exe", ".bat", ".cmd", ".com", ".scr", ".msi", ".app",

		// Script files
		".js", ".vbs", ".ps1", ".sh", ".py", ".pl", ".rb",

		// Document macros
		".docm", ".xlsm", ".pptm",

		// Archives (can contain executables)
		".zip", ".rar", ".7z", ".tar", ".gz",

		// System files
		".dll", ".sys", ".drv",
	}
}

// IsDangerousExtension checks if a file extension is potentially dangerous
func IsDangerousExtension(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	dangerous := DangerousExtensions()

	for _, danger := range dangerous {
		if ext == danger {
			return true
		}
	}

	return false
}

// ValidateMarkdownPath ensures a path is safe for markdown files
func ValidateMarkdownPath(path string) error {
	ps := NewPathSanitizer(nil, 32)

	// Basic path validation
	if err := ps.ValidatePath(path); err != nil {
		return err
	}

	// Check extension
	if !IsSafeExtension(path, []string{".md", ".markdown", ".mdown", ".mkd"}) {
		return errors.NewErrorBuilder().
			WithOperation("markdown path validation").
			WithError(fmt.Errorf("file must have a markdown extension: %s", path)).
			WithCode(errors.ErrCodeFormatUnsupported).
			WithSuggestion("Use .md, .markdown, .mdown, or .mkd extension").
			Build()
	}

	// Check for dangerous patterns in content
	if IsDangerousExtension(path) {
		return errors.NewErrorBuilder().
			WithOperation("security validation").
			WithError(fmt.Errorf("potentially dangerous file extension: %s", path)).
			WithCode(errors.ErrCodePathInvalid).
			WithSuggestion("Avoid using executable file extensions").
			Build()
	}

	return nil
}

// CreateSecureDir creates a directory with secure permissions
func CreateSecureDir(path string) error {
	ps := NewPathSanitizer(nil, 32)

	// Validate path first
	sanitized, err := ps.SanitizePath(path)
	if err != nil {
		return err
	}

	// Create directory with restrictive permissions (rwx for owner only)
	if err := os.MkdirAll(sanitized, 0700); err != nil {
		return errors.NewErrorBuilder().
			WithOperation("directory creation").
			WithError(fmt.Errorf("failed to create directory: %w", err)).
			WithCode(errors.ErrCodeFilePermission).
			WithSuggestion("Check parent directory permissions and disk space").
			Build()
	}

	return nil
}

// WriteSecureFile writes content to a file with secure permissions
func WriteSecureFile(path string, content []byte) error {
	ps := NewPathSanitizer(nil, 32)

	// Validate path first
	sanitized, err := ps.SanitizePath(path)
	if err != nil {
		return err
	}

	// Ensure parent directory exists
	dir := filepath.Dir(sanitized)
	if err := CreateSecureDir(dir); err != nil {
		return err
	}

	// Write file with restrictive permissions (rw for owner only)
	if err := os.WriteFile(sanitized, content, 0600); err != nil {
		return errors.NewErrorBuilder().
			WithOperation("secure file write").
			WithError(fmt.Errorf("failed to write file: %w", err)).
			WithCode(errors.ErrCodeFilePermission).
			WithSuggestion("Check directory permissions and disk space").
			Build()
	}

	return nil
}
