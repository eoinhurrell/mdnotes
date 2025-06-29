package processor

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// FilenameNormalizationOptions contains options for filename normalization
type FilenameNormalizationOptions struct {
	Slugify bool // Convert filenames to URL-safe slugs
	Flatten bool // Put all files in single directory
}

// FilenameNormalizationResult contains the results of filename normalization
type FilenameNormalizationResult struct {
	FileMap      map[string]string   // original path -> new path
	RenamedFiles int                 // number of files that were renamed
	Collisions   map[string][]string // filename -> list of original paths that collided
}

// FilenameMapping represents a file path mapping
type FilenameMapping struct {
	OriginalPath string
	NewPath      string
	Renamed      bool
}

// ExportFilenameNormalizer handles filename normalization for export
type ExportFilenameNormalizer struct {
	options       FilenameNormalizationOptions
	verbose       bool
	usedFilenames map[string]int // filename -> count (for collision handling)
}

// NewExportFilenameNormalizer creates a new filename normalizer
func NewExportFilenameNormalizer(options FilenameNormalizationOptions, verbose bool) *ExportFilenameNormalizer {
	return &ExportFilenameNormalizer{
		options:       options,
		verbose:       verbose,
		usedFilenames: make(map[string]int),
	}
}

// NormalizeFilenames processes a list of files and returns normalized filename mappings
func (fn *ExportFilenameNormalizer) NormalizeFilenames(files []*vault.VaultFile) *FilenameNormalizationResult {
	result := &FilenameNormalizationResult{
		FileMap:    make(map[string]string),
		Collisions: make(map[string][]string),
	}

	for _, file := range files {
		newPath := fn.normalizeFilePath(file.RelativePath)

		if newPath != file.RelativePath {
			result.RenamedFiles++
		}

		result.FileMap[file.RelativePath] = newPath

		// Track collisions
		if newPath != file.RelativePath {
			filename := filepath.Base(newPath)
			if _, exists := result.Collisions[filename]; !exists {
				result.Collisions[filename] = make([]string, 0)
			}
			result.Collisions[filename] = append(result.Collisions[filename], file.RelativePath)
		}

		if fn.verbose && newPath != file.RelativePath {
			fmt.Printf("Normalized: %s -> %s\n", file.RelativePath, newPath)
		}
	}

	return result
}

// normalizeFilePath normalizes a single file path according to options
func (fn *ExportFilenameNormalizer) normalizeFilePath(originalPath string) string {
	dir := filepath.Dir(originalPath)
	filename := filepath.Base(originalPath)
	nameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))
	ext := filepath.Ext(filename)

	// Apply slugification if requested
	if fn.options.Slugify {
		nameWithoutExt = fn.slugify(nameWithoutExt)
	}

	// Apply flattening if requested
	if fn.options.Flatten {
		dir = "." // Put everything in root directory
	}

	// Reconstruct filename
	newFilename := nameWithoutExt + ext

	// Handle collisions by adding numbers
	newFilename = fn.handleCollisions(newFilename)

	// Construct final path
	if dir == "." {
		return newFilename
	}
	return filepath.Join(dir, newFilename)
}

// slugify converts a string to a URL-safe slug
func (fn *ExportFilenameNormalizer) slugify(input string) string {
	// Convert to lowercase
	slug := strings.ToLower(input)

	// Replace spaces and underscores with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	// Remove or replace special characters
	// Keep alphanumeric characters, hyphens, and dots
	var result strings.Builder
	for _, r := range slug {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '.' {
			result.WriteRune(r)
		} else if unicode.IsSpace(r) {
			result.WriteRune('-')
		}
		// Skip all other characters
	}
	slug = result.String()

	// Clean up multiple consecutive hyphens
	slug = regexp.MustCompile(`-+`).ReplaceAllString(slug, "-")

	// Remove leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	// Ensure the slug is not empty
	if slug == "" {
		slug = "untitled"
	}

	return slug
}

// handleCollisions handles filename collisions by adding numbers
func (fn *ExportFilenameNormalizer) handleCollisions(filename string) string {
	// Check if this filename has been used
	count, exists := fn.usedFilenames[filename]
	if !exists {
		// First time using this filename
		fn.usedFilenames[filename] = 1
		return filename
	}

	// Collision detected, generate a unique filename
	nameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))
	ext := filepath.Ext(filename)

	for {
		count++
		newFilename := fmt.Sprintf("%s-%d%s", nameWithoutExt, count, ext)

		if _, exists := fn.usedFilenames[newFilename]; !exists {
			fn.usedFilenames[filename] = count
			fn.usedFilenames[newFilename] = 1
			return newFilename
		}
	}
}

// UpdateFileLinks updates links in file content to point to new filenames
func (fn *ExportFilenameNormalizer) UpdateFileLinks(file *vault.VaultFile, fileMap map[string]string) string {
	content := file.Body
	parser := NewLinkParser()

	// Extract all links
	links := parser.Extract(content)

	// Process links in reverse order to maintain position accuracy
	for i := len(links) - 1; i >= 0; i-- {
		link := links[i]

		// Resolve the link target to see if it needs updating
		resolvedPath := fn.resolveLinkTarget(link.Target, file.RelativePath)

		if newPath, exists := fileMap[resolvedPath]; exists && newPath != resolvedPath {
			// Update the link to point to the new filename
			newTarget := fn.calculateNewLinkTarget(newPath, fileMap[file.RelativePath])
			newLinkText := fn.createUpdatedLink(link, newTarget)

			// Replace the link in content
			start := link.Position.Start
			end := link.Position.End
			content = content[:start] + newLinkText + content[end:]
		}
	}

	return content
}

// resolveLinkTarget resolves a link target to a file path (similar to backlinks resolution)
func (fn *ExportFilenameNormalizer) resolveLinkTarget(target, sourceRelativePath string) string {
	// Clean the target path
	target = strings.TrimSpace(target)

	// Remove any fragment identifiers (#section)
	if hashIndex := strings.Index(target, "#"); hashIndex != -1 {
		target = target[:hashIndex]
	}

	// Skip external URLs
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		return ""
	}

	// Handle absolute paths from vault root
	if filepath.IsAbs(target) || strings.HasPrefix(target, "/") {
		cleanTarget := strings.TrimPrefix(target, "/")
		// Add .md extension if it doesn't have an extension
		if filepath.Ext(cleanTarget) == "" {
			cleanTarget += ".md"
		}
		return cleanTarget
	}

	// Handle relative paths
	if strings.Contains(target, "/") {
		// For wiki links without extension, try adding .md
		targetWithExt := target
		if filepath.Ext(target) == "" {
			targetWithExt = target + ".md"
		}

		// Try relative to source file directory
		sourceDir := filepath.Dir(sourceRelativePath)
		relativePath := filepath.Join(sourceDir, targetWithExt)
		cleanRelativePath := filepath.Clean(relativePath)
		return cleanRelativePath
	}

	// Just a filename - assume same directory first
	targetWithExt := target
	if filepath.Ext(target) == "" {
		targetWithExt = target + ".md"
	}

	sourceDir := filepath.Dir(sourceRelativePath)
	return filepath.Join(sourceDir, targetWithExt)
}

// calculateNewLinkTarget calculates the new link target based on normalized paths
func (fn *ExportFilenameNormalizer) calculateNewLinkTarget(targetNewPath, sourceNewPath string) string {
	if fn.options.Flatten {
		// When flattening, all files are in the same directory
		return filepath.Base(targetNewPath)
	}

	// Calculate relative path from source to target
	sourceDir := filepath.Dir(sourceNewPath)

	if sourceDir == "." {
		// Source is in root, target path is relative to root
		return targetNewPath
	}

	// Calculate relative path
	relPath, err := filepath.Rel(sourceDir, targetNewPath)
	if err != nil {
		// Fallback to absolute path
		return targetNewPath
	}

	return relPath
}

// createUpdatedLink creates the updated link text with new target
func (fn *ExportFilenameNormalizer) createUpdatedLink(link vault.Link, newTarget string) string {
	switch link.Type {
	case vault.WikiLink:
		if link.Text != "" && link.Text != link.Target {
			// Wiki link with custom text: [[target|text]]
			return fmt.Sprintf("[[%s|%s]]", newTarget, link.Text)
		}
		// Simple wiki link: [[target]]
		return fmt.Sprintf("[[%s]]", newTarget)

	case vault.MarkdownLink:
		// Markdown link: [text](target)
		return fmt.Sprintf("[%s](%s)", link.Text, newTarget)

	case vault.EmbedLink:
		// Embed link: ![[target]]
		return fmt.Sprintf("![[%s]]", newTarget)

	default:
		// Fallback - shouldn't happen
		return fmt.Sprintf("[[%s]]", newTarget)
	}
}
