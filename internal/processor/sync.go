package processor

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// FrontmatterSync handles synchronization of frontmatter fields with file system data
type FrontmatterSync struct{}

// NewFrontmatterSync creates a new frontmatter sync processor
func NewFrontmatterSync() *FrontmatterSync {
	return &FrontmatterSync{}
}

// SyncField synchronizes a field based on the specified source
// Returns true if the field was modified
func (fs *FrontmatterSync) SyncField(file *vault.VaultFile, field, source string) bool {
	// Don't overwrite existing fields unless they're empty
	if existingValue, exists := file.GetField(field); exists && existingValue != nil && existingValue != "" {
		return false
	}

	sourceType, config := fs.parseSource(source)
	var value interface{}

	switch sourceType {
	case "file-mtime":
		value = file.Modified.Format("2006-01-02")
	case "file-mtime-iso":
		value = file.Modified.Format("2006-01-02T15:04:05Z")
	case "filename":
		if config != "" && strings.HasPrefix(config, "pattern:") {
			pattern := strings.TrimPrefix(config, "pattern:")
			value = fs.extractFromFilename(file.Path, pattern)
		} else {
			// Default: filename without extension
			filename := filepath.Base(file.Path)
			value = strings.TrimSuffix(filename, filepath.Ext(filename))
		}
	case "path":
		if config == "dir" {
			value = fs.getDirectoryFromPath(file.RelativePath)
		} else {
			value = file.RelativePath
		}
	default:
		return false
	}

	// Only set if we got a non-empty value
	if value != nil && value != "" {
		file.SetField(field, value)
		return true
	}

	return false
}

// parseSource parses a source specification into type and configuration
// Examples: "file-mtime", "filename:pattern:^(\d{8})", "path:dir"
func (fs *FrontmatterSync) parseSource(source string) (string, string) {
	parts := strings.SplitN(source, ":", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

// extractFromFilename extracts a value from filename using a regex pattern
func (fs *FrontmatterSync) extractFromFilename(path, pattern string) string {
	filename := filepath.Base(path)
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))

	re, err := regexp.Compile(pattern)
	if err != nil {
		return ""
	}

	matches := re.FindStringSubmatch(filename)
	if len(matches) < 2 {
		return ""
	}

	return matches[1]
}

// getDirectoryFromPath returns the immediate parent directory name
func (fs *FrontmatterSync) getDirectoryFromPath(relativePath string) string {
	dir := filepath.Dir(relativePath)
	if dir == "." || dir == "/" {
		return ""
	}
	return filepath.Base(dir)
}
