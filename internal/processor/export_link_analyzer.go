package processor

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// LinkCategory represents the category of a link for export purposes
type LinkCategory int

const (
	InternalLink LinkCategory = iota // Link target is included in the export
	ExternalLink                     // Link target is not included in the export
	AssetLink                        // Link points to an asset file (image, pdf, etc.)
	URLLink                          // Link is an external URL
)

// AnalyzedLink represents a link with export analysis information
type AnalyzedLink struct {
	Link     vault.Link
	Category LinkCategory
	Exists   bool // Whether the target file exists in the vault
	IsAsset  bool // Whether the target is an asset file
}

// LinkAnalysis contains the complete analysis of links in a file
type LinkAnalysis struct {
	File          *vault.VaultFile
	Links         []AnalyzedLink
	InternalCount int
	ExternalCount int
	AssetCount    int
	URLCount      int
}

// ExportLinkAnalyzer analyzes links in the context of an export operation
type ExportLinkAnalyzer struct {
	parser          *LinkParser
	exportedFiles   map[string]bool // Set of files being exported (relative paths)
	vaultFiles      map[string]bool // Set of all files in vault (relative paths)
	assetExtensions []string
}

// NewExportLinkAnalyzer creates a new export link analyzer
func NewExportLinkAnalyzer(exportedFiles []*vault.VaultFile, allVaultFiles []*vault.VaultFile) *ExportLinkAnalyzer {
	analyzer := &ExportLinkAnalyzer{
		parser:        NewLinkParser(),
		exportedFiles: make(map[string]bool),
		vaultFiles:    make(map[string]bool),
		assetExtensions: []string{
			".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg",
			".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
			".csv", ".txt", ".zip", ".mp3", ".mp4", ".mov", ".avi",
		},
	}

	// Build set of exported files
	for _, file := range exportedFiles {
		analyzer.exportedFiles[file.RelativePath] = true
	}

	// Build set of all vault files
	for _, file := range allVaultFiles {
		analyzer.vaultFiles[file.RelativePath] = true
	}

	return analyzer
}

// AnalyzeFile analyzes all links in a file and categorizes them for export
func (la *ExportLinkAnalyzer) AnalyzeFile(file *vault.VaultFile) *LinkAnalysis {
	analysis := &LinkAnalysis{
		File:  file,
		Links: make([]AnalyzedLink, 0),
	}

	// Extract ALL links from the file (including external URLs)
	links := la.extractAllLinks(file.Body)

	for _, link := range links {
		analyzed := la.analyzeLink(link, file)
		analysis.Links = append(analysis.Links, analyzed)

		// Update counters
		switch analyzed.Category {
		case InternalLink:
			analysis.InternalCount++
		case ExternalLink:
			analysis.ExternalCount++
		case AssetLink:
			analysis.AssetCount++
		case URLLink:
			analysis.URLCount++
		}
	}

	return analysis
}

// extractAllLinks extracts ALL links from content, including external URLs
// This is different from the standard LinkParser.Extract which filters out external links
func (la *ExportLinkAnalyzer) extractAllLinks(content string) []vault.Link {
	var links []vault.Link
	usedPositions := make(map[vault.Position]bool)

	// Process in order: embeds first (they contain [[ like wiki links), then wiki, then markdown
	embedLinks := la.extractType(content, vault.EmbedLink)
	for _, link := range embedLinks {
		links = append(links, link)
		usedPositions[link.Position] = true
	}

	wikiLinks := la.extractType(content, vault.WikiLink)
	for _, link := range wikiLinks {
		if !la.overlapsUsedPosition(link.Position, usedPositions) {
			links = append(links, link)
			usedPositions[link.Position] = true
		}
	}

	markdownLinks := la.extractType(content, vault.MarkdownLink)
	for _, link := range markdownLinks {
		if !la.overlapsUsedPosition(link.Position, usedPositions) {
			links = append(links, link)
			usedPositions[link.Position] = true
		}
	}

	// Sort links by position in document
	sort.Slice(links, func(i, j int) bool {
		return links[i].Position.Start < links[j].Position.Start
	})

	return links
}

// overlapsUsedPosition checks if a position overlaps with any used position
func (la *ExportLinkAnalyzer) overlapsUsedPosition(pos vault.Position, used map[vault.Position]bool) bool {
	for usedPos := range used {
		if (pos.Start >= usedPos.Start && pos.Start < usedPos.End) ||
			(pos.End > usedPos.Start && pos.End <= usedPos.End) ||
			(pos.Start <= usedPos.Start && pos.End >= usedPos.End) {
			return true
		}
	}
	return false
}

// extractType extracts links of a specific type using patterns from the LinkParser
func (la *ExportLinkAnalyzer) extractType(content string, linkType vault.LinkType) []vault.Link {
	var links []vault.Link

	// Define the same patterns as LinkParser
	var pattern *regexp.Regexp
	switch linkType {
	case vault.WikiLink:
		// Wiki links: [[target]] or [[target|alias]]
		pattern = regexp.MustCompile(`\[\[([^|\]]+(?:\[[^\]]*\][^|\]]*)*?)(?:\|([^\]]+(?:\[[^\]]*\][^\]]*)*?))?\]\]`)
	case vault.MarkdownLink:
		// Markdown links: [text](target)
		pattern = regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`)
	case vault.EmbedLink:
		// Embed links: ![[target]]
		pattern = regexp.MustCompile(`!\[\[([^\]]+(?:\[[^\]]*\][^\]]*)*?)\]\]`)
	default:
		return links
	}

	matches := pattern.FindAllStringSubmatchIndex(content, -1)
	for _, match := range matches {
		link := vault.Link{
			Type: linkType,
			Position: vault.Position{
				Start: match[0],
				End:   match[1],
			},
		}

		// Extract groups from the match
		groups := pattern.FindStringSubmatch(content[match[0]:match[1]])
		if len(groups) < 2 {
			continue
		}

		switch linkType {
		case vault.WikiLink:
			link.Target = groups[1]
			if len(groups) > 2 && groups[2] != "" {
				link.Text = groups[2]
			} else {
				link.Text = groups[1]
			}
		case vault.MarkdownLink:
			link.Text = groups[1]
			link.Target = groups[2]
		case vault.EmbedLink:
			link.Target = groups[1]
		}

		links = append(links, link)
	}

	return links
}

// analyzeLink analyzes a single link and determines its category
func (la *ExportLinkAnalyzer) analyzeLink(link vault.Link, sourceFile *vault.VaultFile) AnalyzedLink {
	analyzed := AnalyzedLink{
		Link: link,
	}

	// Check if it's an external URL first
	if !la.parser.IsInternalLink(link.Target) {
		analyzed.Category = URLLink
		return analyzed
	}

	// Resolve the target path relative to the source file's directory
	targetPath := la.resolveTargetPath(link.Target, sourceFile.RelativePath)

	// Check if target exists in vault
	analyzed.Exists = la.vaultFiles[targetPath]

	// Check if it's an asset file
	analyzed.IsAsset = la.isAssetFile(targetPath)
	if analyzed.IsAsset {
		analyzed.Category = AssetLink
		return analyzed
	}

	// Check if target is included in export
	if la.exportedFiles[targetPath] {
		analyzed.Category = InternalLink
	} else {
		analyzed.Category = ExternalLink
	}

	return analyzed
}

// resolveTargetPath resolves a link target to an absolute path within the vault
func (la *ExportLinkAnalyzer) resolveTargetPath(target, sourceRelativePath string) string {
	// Clean the target path
	target = strings.TrimSpace(target)

	// Handle absolute paths from vault root
	if filepath.IsAbs(target) || strings.HasPrefix(target, "/") {
		cleanTarget := strings.TrimPrefix(target, "/")
		// Add .md extension if it doesn't have an extension
		if filepath.Ext(cleanTarget) == "" {
			cleanTarget += ".md"
		}
		return cleanTarget
	}

	// For relative paths with directories (like "assets/image.png" or "folder/note2"),
	// try both relative to source file and relative to vault root
	if strings.Contains(target, "/") {
		// For wiki links without extension, try adding .md
		targetWithExt := target
		if filepath.Ext(target) == "" {
			targetWithExt = target + ".md"
		}

		// First try relative to vault root (this is common for assets and organized notes)
		if la.vaultFiles[targetWithExt] {
			return targetWithExt
		}
		if la.vaultFiles[target] {
			return target
		}

		// Then try relative to source file directory
		sourceDir := filepath.Dir(sourceRelativePath)
		relativePath := filepath.Join(sourceDir, targetWithExt)
		if la.vaultFiles[relativePath] {
			return relativePath
		}

		// Return the vault root attempt as fallback
		return targetWithExt
	}

	// Just a filename without directories
	targetWithExt := target
	// Only add .md extension if it doesn't already have an extension
	if filepath.Ext(target) == "" {
		targetWithExt = target + ".md"
	}

	// First try in the same directory as source file
	sourceDir := filepath.Dir(sourceRelativePath)
	sameDirPath := filepath.Join(sourceDir, targetWithExt)
	if la.vaultFiles[sameDirPath] {
		return sameDirPath
	}

	// If not found and it's just a filename, try vault root
	if la.vaultFiles[targetWithExt] {
		return targetWithExt
	}

	// Search across the entire vault for exact filename matches
	for vaultFile := range la.vaultFiles {
		if filepath.Base(vaultFile) == targetWithExt {
			return vaultFile
		}
	}

	// Return the same directory attempt as fallback
	return sameDirPath
}

// isAssetFile checks if a file path represents an asset (non-markdown) file
func (la *ExportLinkAnalyzer) isAssetFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, assetExt := range la.assetExtensions {
		if ext == assetExt {
			return true
		}
	}
	return false
}

// GetLinksByCategory returns links filtered by category
func (la *LinkAnalysis) GetLinksByCategory(category LinkCategory) []AnalyzedLink {
	var filtered []AnalyzedLink
	for _, link := range la.Links {
		if link.Category == category {
			filtered = append(filtered, link)
		}
	}
	return filtered
}

// HasExternalLinks returns true if the file has any external links that need processing
func (la *LinkAnalysis) HasExternalLinks() bool {
	return la.ExternalCount > 0
}

// HasAssets returns true if the file has any asset links
func (la *LinkAnalysis) HasAssets() bool {
	return la.AssetCount > 0
}

// Summary returns a summary string of the link analysis
func (la *LinkAnalysis) Summary() string {
	total := len(la.Links)
	if total == 0 {
		return "No links found"
	}

	parts := make([]string, 0, 4)
	if la.InternalCount > 0 {
		parts = append(parts, fmt.Sprintf("%d internal", la.InternalCount))
	}
	if la.ExternalCount > 0 {
		parts = append(parts, fmt.Sprintf("%d external", la.ExternalCount))
	}
	if la.AssetCount > 0 {
		parts = append(parts, fmt.Sprintf("%d assets", la.AssetCount))
	}
	if la.URLCount > 0 {
		parts = append(parts, fmt.Sprintf("%d URLs", la.URLCount))
	}

	return fmt.Sprintf("%d links (%s)", total, strings.Join(parts, ", "))
}
