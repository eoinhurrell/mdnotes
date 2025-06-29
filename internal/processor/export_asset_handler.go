package processor

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// AssetDiscoveryResult contains information about discovered assets
type AssetDiscoveryResult struct {
	AssetFiles    map[string]string // asset path -> source file that references it
	MissingAssets []string          // asset paths that don't exist
	TotalAssets   int               // total number of asset references found
}

// AssetProcessingResult contains the results of asset processing
type AssetProcessingResult struct {
	AssetsCopied  int
	AssetsMissing int
	CopiedAssets  []string
	MissingAssets []string
}

// ExportAssetHandler handles asset discovery and copying during export
type ExportAssetHandler struct {
	vaultPath           string
	outputPath          string
	verbose             bool
	supportedExtensions []string
}

// NewExportAssetHandler creates a new asset handler
func NewExportAssetHandler(vaultPath, outputPath string, verbose bool) *ExportAssetHandler {
	return &ExportAssetHandler{
		vaultPath:  vaultPath,
		outputPath: outputPath,
		verbose:    verbose,
		supportedExtensions: []string{
			".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".bmp", ".tiff",
			".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
			".csv", ".txt", ".md", ".zip", ".tar", ".gz",
			".mp3", ".mp4", ".mov", ".avi", ".wmv", ".flv", ".webm",
		},
	}
}

// DiscoverAssets finds all asset files referenced by the exported files
func (ah *ExportAssetHandler) DiscoverAssets(files []*vault.VaultFile) *AssetDiscoveryResult {
	result := &AssetDiscoveryResult{
		AssetFiles:    make(map[string]string),
		MissingAssets: make([]string, 0),
	}

	// Use a basic link parser to find all links, then filter for assets
	parser := NewLinkParser()

	for _, file := range files {
		// Extract all links from the file
		links := parser.Extract(file.Body)

		for _, link := range links {
			// Try to resolve as asset path
			assetPath := ah.resolveAssetPath(link.Target, file.RelativePath)

			if assetPath != "" {
				result.TotalAssets++

				// Check if asset exists
				fullAssetPath := filepath.Join(ah.vaultPath, assetPath)
				if _, err := os.Stat(fullAssetPath); err == nil {
					result.AssetFiles[assetPath] = file.RelativePath
				} else {
					result.MissingAssets = append(result.MissingAssets, assetPath)
				}
			}
		}
	}

	return result
}

// ProcessAssets copies discovered assets to the output directory
func (ah *ExportAssetHandler) ProcessAssets(discovery *AssetDiscoveryResult) *AssetProcessingResult {
	result := &AssetProcessingResult{
		CopiedAssets:  make([]string, 0),
		MissingAssets: discovery.MissingAssets,
		AssetsMissing: len(discovery.MissingAssets),
	}

	// Copy each asset file
	for assetPath, sourceFile := range discovery.AssetFiles {
		srcPath := filepath.Join(ah.vaultPath, assetPath)
		dstPath := filepath.Join(ah.outputPath, assetPath)

		err := ah.copyAssetFile(srcPath, dstPath)
		if err != nil {
			if ah.verbose {
				fmt.Printf("Warning: Failed to copy asset %s (referenced in %s): %v\n",
					assetPath, sourceFile, err)
			}
			result.MissingAssets = append(result.MissingAssets, assetPath)
			result.AssetsMissing++
		} else {
			result.CopiedAssets = append(result.CopiedAssets, assetPath)
			result.AssetsCopied++

			if ah.verbose {
				fmt.Printf("Copied asset: %s\n", assetPath)
			}
		}
	}

	return result
}

// copyAssetFile copies a single asset file from source to destination
func (ah *ExportAssetHandler) copyAssetFile(srcPath, dstPath string) error {
	// Create destination directory
	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("creating asset directory %s: %w", dstDir, err)
	}

	// Open source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("opening source asset %s: %w", srcPath, err)
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("creating destination asset %s: %w", dstPath, err)
	}
	defer dstFile.Close()

	// Copy content
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("copying asset content: %w", err)
	}

	// Copy file mode
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("getting source file info: %w", err)
	}

	return os.Chmod(dstPath, srcInfo.Mode())
}

// resolveAssetPath resolves an asset link target to a vault-relative path
func (ah *ExportAssetHandler) resolveAssetPath(target, sourceRelativePath string) string {
	// Clean the target path
	target = strings.TrimSpace(target)

	// Remove any fragment identifiers (#section)
	if hashIndex := strings.Index(target, "#"); hashIndex != -1 {
		target = target[:hashIndex]
	}

	// Skip if it's not a supported asset extension
	ext := strings.ToLower(filepath.Ext(target))
	supported := false
	for _, supportedExt := range ah.supportedExtensions {
		if ext == supportedExt {
			supported = true
			break
		}
	}
	if !supported {
		return ""
	}

	// Handle absolute paths from vault root
	if filepath.IsAbs(target) || strings.HasPrefix(target, "/") {
		return strings.TrimPrefix(target, "/")
	}

	// Handle relative paths
	if strings.Contains(target, "/") {
		// Try relative to vault root first
		if ah.assetExists(target) {
			return target
		}

		// Try relative to source file directory
		sourceDir := filepath.Dir(sourceRelativePath)
		relativePath := filepath.Join(sourceDir, target)
		// Clean the path to handle ../ properly
		cleanRelativePath := filepath.Clean(relativePath)
		if ah.assetExists(cleanRelativePath) {
			return cleanRelativePath
		}

		// Return vault root attempt as fallback
		return target
	}

	// Just a filename - try in source directory first, then vault root
	sourceDir := filepath.Dir(sourceRelativePath)
	sameDirPath := filepath.Join(sourceDir, target)
	if ah.assetExists(sameDirPath) {
		return sameDirPath
	}

	if ah.assetExists(target) {
		return target
	}

	// Return same directory attempt as fallback
	return sameDirPath
}

// assetExists checks if an asset file exists in the vault
func (ah *ExportAssetHandler) assetExists(assetPath string) bool {
	fullPath := filepath.Join(ah.vaultPath, assetPath)
	_, err := os.Stat(fullPath)
	return err == nil
}
