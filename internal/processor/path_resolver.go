package processor

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// PathResolver handles vault-relative path resolution with disambiguation
type PathResolver struct {
	vaultRoot string
}

// NewPathResolver creates a new path resolver
func NewPathResolver(vaultRoot string) *PathResolver {
	return &PathResolver{
		vaultRoot: vaultRoot,
	}
}

// ResolveTarget converts a link target to an absolute vault path with disambiguation
func (pr *PathResolver) ResolveTarget(linkTarget string, contextFile string) (string, error) {
	// URL decode the target first
	decodedTarget := linkTarget
	if strings.Contains(linkTarget, "%") {
		if decoded, err := url.QueryUnescape(linkTarget); err == nil {
			decodedTarget = decoded
		}
	}

	// Remove fragment for path resolution
	targetPath := decodedTarget
	if idx := strings.Index(decodedTarget, "#"); idx != -1 {
		targetPath = decodedTarget[:idx]
	}

	// Convert to absolute path relative to vault root
	var absPath string
	if filepath.IsAbs(targetPath) {
		absPath = targetPath
	} else {
		// All links in Obsidian are relative to vault root
		absPath = filepath.Join(pr.vaultRoot, targetPath)
	}

	return filepath.Clean(absPath), nil
}

// ShouldUpdateLink determines if a link should be updated for a file move with priority logic
func (pr *PathResolver) ShouldUpdateLink(link vault.Link, oldPath, newPath string) bool {
	// Get relative paths for comparison
	oldRel, err := filepath.Rel(pr.vaultRoot, oldPath)
	if err != nil {
		return false
	}

	// Use the Link.ShouldUpdate method which handles all the logic
	return link.ShouldUpdate(oldRel, newPath)
}

// GenerateNewTarget creates the updated link target for a moved file
func (pr *PathResolver) GenerateNewTarget(link vault.Link, oldPath, newPath string) (string, error) {
	// Get relative paths
	newRel, err := filepath.Rel(pr.vaultRoot, newPath)
	if err != nil {
		return "", fmt.Errorf("getting relative path for new target: %w", err)
	}

	// Use the Link.GenerateUpdatedLink method
	return link.GenerateUpdatedLink(newRel), nil
}

// MatchPriority represents the priority of a link match
type MatchPriority int

const (
	NoMatch MatchPriority = iota
	BaseNameMatch
	FullPathMatch
)

// AnalyzeLinkMatch analyzes how well a link matches a file path and returns priority
func (pr *PathResolver) AnalyzeLinkMatch(link vault.Link, filePath string) MatchPriority {
	// Get relative path
	fileRel, err := filepath.Rel(pr.vaultRoot, filePath)
	if err != nil {
		return NoMatch
	}

	// Normalize target for comparison
	target := link.Target
	if strings.Contains(target, "%") {
		if decoded, err := url.QueryUnescape(target); err == nil {
			target = decoded
		}
	}

	// Remove fragment for matching
	if idx := strings.Index(target, "#"); idx != -1 {
		target = target[:idx]
	}

	// Remove extensions for comparison
	targetBase := strings.TrimSuffix(target, ".md")
	fileBase := strings.TrimSuffix(fileRel, ".md")

	// Check for exact path match (highest priority)
	if targetBase == fileBase || target == fileRel {
		return FullPathMatch
	}

	// For wiki links, check basename match (lower priority)
	if link.Type == vault.WikiLink {
		fileBasename := filepath.Base(fileBase)
		targetBasename := filepath.Base(targetBase)

		if targetBasename == fileBasename || target == filepath.Base(fileRel) {
			return BaseNameMatch
		}
	}

	return NoMatch
}

// DisambiguationResult represents the result of disambiguation analysis
type DisambiguationResult struct {
	HasAmbiguity bool
	Matches      []DisambiguationMatch
}

// DisambiguationMatch represents a potential file match for a link
type DisambiguationMatch struct {
	FilePath string
	Priority MatchPriority
}

// FindAllMatches finds all potential matches for a link across the vault
func (pr *PathResolver) FindAllMatches(link vault.Link, vaultFiles []*vault.VaultFile) DisambiguationResult {
	var matches []DisambiguationMatch

	for _, file := range vaultFiles {
		priority := pr.AnalyzeLinkMatch(link, file.Path)
		if priority != NoMatch {
			matches = append(matches, DisambiguationMatch{
				FilePath: file.Path,
				Priority: priority,
			})
		}
	}

	return DisambiguationResult{
		HasAmbiguity: len(matches) > 1,
		Matches:      matches,
	}
}

// ResolveBestMatch returns the best match for a link, handling ambiguity
func (pr *PathResolver) ResolveBestMatch(link vault.Link, vaultFiles []*vault.VaultFile) (string, error) {
	result := pr.FindAllMatches(link, vaultFiles)

	if len(result.Matches) == 0 {
		return "", fmt.Errorf("no matches found for link target: %s", link.Target)
	}

	if len(result.Matches) == 1 {
		return result.Matches[0].FilePath, nil
	}

	// Multiple matches - prioritize by match type
	var fullPathMatches []DisambiguationMatch
	var baseNameMatches []DisambiguationMatch

	for _, match := range result.Matches {
		switch match.Priority {
		case FullPathMatch:
			fullPathMatches = append(fullPathMatches, match)
		case BaseNameMatch:
			baseNameMatches = append(baseNameMatches, match)
		}
	}

	// Prefer full path matches
	if len(fullPathMatches) == 1 {
		return fullPathMatches[0].FilePath, nil
	} else if len(fullPathMatches) > 1 {
		return "", fmt.Errorf("ambiguous full path matches for %s: %v", link.Target, fullPathMatches)
	}

	// Fall back to basename matches
	if len(baseNameMatches) == 1 {
		return baseNameMatches[0].FilePath, nil
	} else if len(baseNameMatches) > 1 {
		return "", fmt.Errorf("ambiguous basename matches for %s: %v", link.Target, baseNameMatches)
	}

	return "", fmt.Errorf("no resolvable matches for link target: %s", link.Target)
}

// NormalizePath normalizes a path for consistent comparison
func (pr *PathResolver) NormalizePath(path string) string {
	// Convert to slash separators for cross-platform consistency
	normalized := strings.ReplaceAll(path, "\\", "/")

	// Make relative to vault root if absolute
	if filepath.IsAbs(normalized) {
		vaultRootNormalized := strings.ReplaceAll(pr.vaultRoot, "\\", "/")
		if rel, err := filepath.Rel(vaultRootNormalized, normalized); err == nil {
			normalized = rel
		}
	}

	return normalized
}

// IsVaultRelative checks if a path is relative to the vault root
func (pr *PathResolver) IsVaultRelative(path string) bool {
	if filepath.IsAbs(path) {
		// Check if the absolute path is within the vault
		rel, err := filepath.Rel(pr.vaultRoot, path)
		return err == nil && !strings.HasPrefix(rel, "..")
	}

	// Relative paths are assumed to be vault-relative
	return true
}

// GetVaultRelativePath returns the vault-relative path for a file
func (pr *PathResolver) GetVaultRelativePath(absolutePath string) (string, error) {
	rel, err := filepath.Rel(pr.vaultRoot, absolutePath)
	if err != nil {
		return "", fmt.Errorf("getting vault-relative path: %w", err)
	}

	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path is outside vault: %s", absolutePath)
	}

	return filepath.ToSlash(rel), nil
}

// ResolveAbsolutePath converts a vault-relative path to absolute
func (pr *PathResolver) ResolveAbsolutePath(vaultRelativePath string) string {
	return filepath.Join(pr.vaultRoot, filepath.FromSlash(vaultRelativePath))
}
