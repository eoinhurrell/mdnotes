package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eoinhurrell/mdnotes/internal/vault"
	"github.com/eoinhurrell/mdnotes/pkg/template"
)

// ConflictResolution defines how to handle naming conflicts
type ConflictResolution int

const (
	ConflictSkip ConflictResolution = iota
	ConflictNumber
	ConflictOverwrite
)

// FileMove represents a file move operation
type FileMove struct {
	From string
	To   string
}

// Organizer handles file organization and renaming
type Organizer struct {
	templateEngine *template.Engine
}

// NewOrganizer creates a new file organizer
func NewOrganizer() *Organizer {
	return &Organizer{
		templateEngine: template.NewEngine(),
	}
}

// GenerateFilename creates a filename using a template pattern
func (o *Organizer) GenerateFilename(pattern string, file *vault.VaultFile) string {
	return o.templateEngine.Process(pattern, file)
}

// GenerateDirectoryPath creates a directory path using a template pattern
func (o *Organizer) GenerateDirectoryPath(pattern string, file *vault.VaultFile) string {
	path := o.templateEngine.Process(pattern, file)
	// Clean the path to handle any template artifacts
	return filepath.Clean(path)
}

// RenameFile renames a file according to the pattern and conflict resolution
func (o *Organizer) RenameFile(file *vault.VaultFile, pattern, baseDir string, onConflict ConflictResolution) (string, error) {
	// Generate new filename
	newFilename := o.GenerateFilename(pattern, file)
	if newFilename == "" {
		return "", fmt.Errorf("pattern resulted in empty filename")
	}

	// Clean the filename to remove any path separators
	newFilename = filepath.Base(newFilename)

	// Build target path
	targetPath := filepath.Join(baseDir, newFilename)

	// Handle conflicts
	finalPath := o.handleConflict(targetPath, onConflict)
	if finalPath == "" {
		// Skip was chosen and file exists
		return file.Path, nil
	}

	// Ensure target directory exists
	targetDir := filepath.Dir(finalPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("creating directory %s: %w", targetDir, err)
	}

	// Don't rename if source and target are the same
	if file.Path == finalPath {
		return file.Path, nil
	}

	// Perform the rename
	if err := os.Rename(file.Path, finalPath); err != nil {
		return "", fmt.Errorf("renaming %s to %s: %w", file.Path, finalPath, err)
	}

	return finalPath, nil
}

// MoveFile moves a file to a new directory structure
func (o *Organizer) MoveFile(file *vault.VaultFile, dirPattern, filenamePattern, baseDir string, onConflict ConflictResolution) (string, error) {
	// Generate directory path
	dirPath := ""
	if dirPattern != "" {
		dirPath = o.GenerateDirectoryPath(dirPattern, file)
	}

	// Generate filename
	var filename string
	if filenamePattern != "" {
		filename = o.GenerateFilename(filenamePattern, file)
		filename = filepath.Base(filename) // Remove any path components
	} else {
		filename = filepath.Base(file.Path) // Use original filename
	}

	// Build target path
	var targetPath string
	if dirPath != "" {
		targetPath = filepath.Join(baseDir, dirPath, filename)
	} else {
		targetPath = filepath.Join(baseDir, filename)
	}

	// Handle conflicts
	finalPath := o.handleConflict(targetPath, onConflict)
	if finalPath == "" {
		// Skip was chosen and file exists
		return file.Path, nil
	}

	// Ensure target directory exists
	targetDir := filepath.Dir(finalPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("creating directory %s: %w", targetDir, err)
	}

	// Don't move if source and target are the same
	if file.Path == finalPath {
		return file.Path, nil
	}

	// Perform the move
	if err := os.Rename(file.Path, finalPath); err != nil {
		return "", fmt.Errorf("moving %s to %s: %w", file.Path, finalPath, err)
	}

	return finalPath, nil
}

// handleConflict resolves naming conflicts
func (o *Organizer) handleConflict(targetPath string, resolution ConflictResolution) string {
	// Check if file already exists
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		// No conflict
		return targetPath
	}

	switch resolution {
	case ConflictSkip:
		return ""

	case ConflictOverwrite:
		return targetPath

	case ConflictNumber:
		// Add number suffix
		dir := filepath.Dir(targetPath)
		name := strings.TrimSuffix(filepath.Base(targetPath), filepath.Ext(targetPath))
		ext := filepath.Ext(targetPath)

		for i := 1; i <= 999; i++ {
			numberedPath := filepath.Join(dir, fmt.Sprintf("%s-%d%s", name, i, ext))
			if _, err := os.Stat(numberedPath); os.IsNotExist(err) {
				return numberedPath
			}
		}

		// If we can't find a number that works, fall back to timestamp
		return filepath.Join(dir, fmt.Sprintf("%s-%d%s", name, len(targetPath), ext))

	default:
		return targetPath
	}
}

// OrganizeByRule organizes files according to a rule
func (o *Organizer) OrganizeByRule(files []*vault.VaultFile, rule OrganizationRule, baseDir string) ([]FileMove, error) {
	var moves []FileMove

	for _, file := range files {
		// Check if rule applies to this file
		if rule.Condition != "" && !o.evaluateCondition(file, rule.Condition) {
			continue
		}

		// Determine new path
		newPath, err := o.MoveFile(file, rule.DirectoryPattern, rule.FilenamePattern, baseDir, rule.OnConflict)
		if err != nil {
			return moves, fmt.Errorf("organizing %s: %w", file.Path, err)
		}

		if newPath != file.Path {
			moves = append(moves, FileMove{
				From: file.Path,
				To:   newPath,
			})

			// Update file path for link updates
			file.Path = newPath
			// Update relative path
			if relPath, err := filepath.Rel(baseDir, newPath); err == nil {
				file.RelativePath = relPath
			}
		}
	}

	return moves, nil
}

// OrganizationRule defines how to organize files
type OrganizationRule struct {
	Name             string
	Condition        string // Field-based condition (e.g., "type=note")
	DirectoryPattern string // Template for directory structure
	FilenamePattern  string // Template for filename (optional)
	OnConflict       ConflictResolution
}

// evaluateCondition checks if a condition applies to a file
func (o *Organizer) evaluateCondition(file *vault.VaultFile, condition string) bool {
	// Simple condition evaluation: field=value
	parts := strings.SplitN(condition, "=", 2)
	if len(parts) != 2 {
		return true // Invalid condition, apply to all
	}

	field := strings.TrimSpace(parts[0])
	expectedValue := strings.TrimSpace(parts[1])

	value, exists := file.GetField(field)
	if !exists {
		return false
	}

	valueStr := fmt.Sprintf("%v", value)
	return valueStr == expectedValue
}
