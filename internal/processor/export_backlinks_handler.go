package processor

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// BacklinksDiscoveryResult contains information about discovered backlinks
type BacklinksDiscoveryResult struct {
	BacklinkFiles   []*vault.VaultFile // files that link to exported files
	BacklinkMap     map[string][]string // target file -> list of files that link to it
	TotalBacklinks  int                 // total number of backlink relationships found
	ProcessedFiles  map[string]bool     // files already processed to prevent cycles
}

// ExportBacklinksHandler handles backlink discovery for export
type ExportBacklinksHandler struct {
	allVaultFiles []*vault.VaultFile
	verbose       bool
	maxDepth      int // maximum depth to prevent runaway recursion
}

// NewExportBacklinksHandler creates a new backlinks handler
func NewExportBacklinksHandler(allVaultFiles []*vault.VaultFile, verbose bool) *ExportBacklinksHandler {
	return &ExportBacklinksHandler{
		allVaultFiles: allVaultFiles,
		verbose:       verbose,
		maxDepth:      10, // reasonable limit to prevent infinite recursion
	}
}

// DiscoverBacklinks finds all files that link to the exported files (recursively)
func (bh *ExportBacklinksHandler) DiscoverBacklinks(ctx context.Context, exportedFiles []*vault.VaultFile) *BacklinksDiscoveryResult {
	result := &BacklinksDiscoveryResult{
		BacklinkFiles:  make([]*vault.VaultFile, 0),
		BacklinkMap:    make(map[string][]string),
		ProcessedFiles: make(map[string]bool),
	}

	// Create a set of exported files for quick lookup
	exportedPaths := make(map[string]bool)
	for _, file := range exportedFiles {
		exportedPaths[file.RelativePath] = true
		result.ProcessedFiles[file.RelativePath] = true
	}

	// Start with the initially exported files and expand recursively
	filesToProcess := make([]*vault.VaultFile, len(exportedFiles))
	copy(filesToProcess, exportedFiles)

	depth := 0
	for len(filesToProcess) > 0 && depth < bh.maxDepth {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return result
		default:
		}

		currentBatch := filesToProcess
		filesToProcess = make([]*vault.VaultFile, 0)

		backlinksFound := bh.findBacklinksToFiles(currentBatch, result.ProcessedFiles)
		
		// Add newly found backlinks to result
		for _, backlinkFile := range backlinksFound {
			if !result.ProcessedFiles[backlinkFile.RelativePath] {
				result.BacklinkFiles = append(result.BacklinkFiles, backlinkFile)
				result.ProcessedFiles[backlinkFile.RelativePath] = true
				filesToProcess = append(filesToProcess, backlinkFile)
				
				if bh.verbose {
					fmt.Printf("Found backlink: %s\n", backlinkFile.RelativePath)
				}
			}
		}

		depth++
	}

	if depth >= bh.maxDepth && bh.verbose {
		fmt.Printf("Warning: Reached maximum backlink depth (%d), stopping recursion\n", bh.maxDepth)
	}

	result.TotalBacklinks = len(result.BacklinkFiles)
	return result
}

// findBacklinksToFiles finds all files that link to any of the target files
func (bh *ExportBacklinksHandler) findBacklinksToFiles(targetFiles []*vault.VaultFile, processedFiles map[string]bool) []*vault.VaultFile {
	var backlinks []*vault.VaultFile
	
	// Create set of target file paths for quick lookup
	targetPaths := make(map[string]bool)
	for _, file := range targetFiles {
		targetPaths[file.RelativePath] = true
	}

	// Scan all vault files for links to target files
	parser := NewLinkParser()
	
	for _, candidateFile := range bh.allVaultFiles {
		// Skip files that are already processed
		if processedFiles[candidateFile.RelativePath] {
			continue
		}

		// Parse links in this file
		links := parser.Extract(candidateFile.Body)
		
		// Check if any links point to target files
		hasBacklink := false
		for _, link := range links {
			resolvedPath := bh.resolveLinkPath(link.Target, candidateFile.RelativePath)
			if targetPaths[resolvedPath] {
				hasBacklink = true
				break
			}
		}

		if hasBacklink {
			backlinks = append(backlinks, candidateFile)
		}
	}

	return backlinks
}

// resolveLinkPath resolves a link target to a file path (similar to asset resolution but for markdown files)
func (bh *ExportBacklinksHandler) resolveLinkPath(target, sourceRelativePath string) string {
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
		
		// Try relative to vault root first
		if bh.fileExists(targetWithExt) {
			return targetWithExt
		}
		
		// Try relative to source file directory
		sourceDir := filepath.Dir(sourceRelativePath)
		relativePath := filepath.Join(sourceDir, targetWithExt)
		cleanRelativePath := filepath.Clean(relativePath)
		if bh.fileExists(cleanRelativePath) {
			return cleanRelativePath
		}
		
		// Return vault root attempt as fallback
		return targetWithExt
	}
	
	// Just a filename - search across vault
	targetWithExt := target
	if filepath.Ext(target) == "" {
		targetWithExt = target + ".md"
	}
	
	// Try same directory first
	sourceDir := filepath.Dir(sourceRelativePath)
	sameDirPath := filepath.Join(sourceDir, targetWithExt)
	if bh.fileExists(sameDirPath) {
		return sameDirPath
	}
	
	// Try vault root
	if bh.fileExists(targetWithExt) {
		return targetWithExt
	}
	
	// Search across entire vault for exact filename matches
	for _, vaultFile := range bh.allVaultFiles {
		if filepath.Base(vaultFile.RelativePath) == targetWithExt {
			return vaultFile.RelativePath
		}
	}
	
	// Return same directory attempt as fallback
	return sameDirPath
}

// fileExists checks if a file exists in the vault files list
func (bh *ExportBacklinksHandler) fileExists(filePath string) bool {
	for _, file := range bh.allVaultFiles {
		if file.RelativePath == filePath {
			return true
		}
	}
	return false
}