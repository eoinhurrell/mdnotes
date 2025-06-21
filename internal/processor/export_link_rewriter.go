package processor

import (
	"fmt"
	"strings"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// LinkRewriteStrategy represents different strategies for handling external links
type LinkRewriteStrategy string

const (
	RemoveStrategy LinkRewriteStrategy = "remove" // Convert external links to plain text
	URLStrategy    LinkRewriteStrategy = "url"    // Use frontmatter url field when available
)

// LinkRewriteResult contains the result of a link rewrite operation
type LinkRewriteResult struct {
	OriginalContent   string
	RewrittenContent  string
	ExternalLinksRemoved int
	ExternalLinksConverted int
	InternalLinksUpdated int
	ChangedLinks      []LinkChange
}

// LinkChange represents a single link change
type LinkChange struct {
	OriginalText string
	NewText      string
	LinkType     vault.LinkType
	Category     LinkCategory
	Position     vault.Position
	WasConverted bool // true if converted to URL, false if removed to plain text
}

// ExportLinkRewriter handles rewriting links based on export context
type ExportLinkRewriter struct {
	analyzer *ExportLinkAnalyzer
	strategy LinkRewriteStrategy
}

// NewExportLinkRewriter creates a new link rewriter
func NewExportLinkRewriter(analyzer *ExportLinkAnalyzer, strategy LinkRewriteStrategy) *ExportLinkRewriter {
	return &ExportLinkRewriter{
		analyzer: analyzer,
		strategy: strategy,
	}
}

// RewriteFileContent rewrites all links in a file's content based on the strategy
func (lr *ExportLinkRewriter) RewriteFileContent(file *vault.VaultFile) *LinkRewriteResult {
	result := &LinkRewriteResult{
		OriginalContent:  file.Body,
		RewrittenContent: file.Body,
		ChangedLinks:     make([]LinkChange, 0),
	}

	// Analyze links in the file
	analysis := lr.analyzer.AnalyzeFile(file)
	if len(analysis.Links) == 0 {
		return result
	}

	// Process links in reverse order to maintain position accuracy
	content := file.Body
	for i := len(analysis.Links) - 1; i >= 0; i-- {
		link := analysis.Links[i]
		change := lr.rewriteLink(link, file)
		if change != nil {
			// Apply the change to content
			before := content[:link.Link.Position.Start]
			after := content[link.Link.Position.End:]
			content = before + change.NewText + after

			// Update counters
			switch link.Category {
			case ExternalLink:
				if change.WasConverted {
					result.ExternalLinksConverted++
				} else {
					result.ExternalLinksRemoved++
				}
			case InternalLink:
				result.InternalLinksUpdated++
			}

			result.ChangedLinks = append(result.ChangedLinks, *change)
		}
	}

	result.RewrittenContent = content
	return result
}

// rewriteLink rewrites a single analyzed link based on the strategy
func (lr *ExportLinkRewriter) rewriteLink(analyzedLink AnalyzedLink, file *vault.VaultFile) *LinkChange {
	link := analyzedLink.Link
	originalText := lr.extractLinkText(file.Body, link)

	switch analyzedLink.Category {
	case ExternalLink:
		return lr.rewriteExternalLink(link, originalText, file)
	case InternalLink:
		return lr.rewriteInternalLink(link, originalText, file)
	case AssetLink:
		// For now, leave asset links unchanged
		// This will be handled in Phase 3
		return nil
	case URLLink:
		// External URLs are left unchanged
		return nil
	}

	return nil
}

// rewriteExternalLink handles external link rewriting based on strategy
func (lr *ExportLinkRewriter) rewriteExternalLink(link vault.Link, originalText string, file *vault.VaultFile) *LinkChange {
	switch lr.strategy {
	case RemoveStrategy:
		// Convert to plain text - use the display text if available, otherwise the target
		var plainText string
		if link.Text != "" && link.Text != link.Target {
			plainText = link.Text
		} else {
			plainText = link.Target
		}
		
		return &LinkChange{
			OriginalText: originalText,
			NewText:      plainText,
			LinkType:     link.Type,
			Category:     ExternalLink,
			Position:     link.Position,
			WasConverted: false,
		}

	case URLStrategy:
		// Try to find a URL in frontmatter and create a markdown link
		if url := lr.findURLInFrontmatter(link.Target, file); url != "" {
			var newText string
			displayText := link.Text
			if displayText == "" || displayText == link.Target {
				displayText = link.Target
			}
			newText = fmt.Sprintf("[%s](%s)", displayText, url)
			
			return &LinkChange{
				OriginalText: originalText,
				NewText:      newText,
				LinkType:     link.Type,
				Category:     ExternalLink,
				Position:     link.Position,
				WasConverted: true,
			}
		}
		
		// If no URL found, fall back to remove strategy
		// Convert to plain text - use the display text if available, otherwise the target
		var plainText string
		if link.Text != "" && link.Text != link.Target {
			plainText = link.Text
		} else {
			plainText = link.Target
		}
		
		return &LinkChange{
			OriginalText: originalText,
			NewText:      plainText,
			LinkType:     link.Type,
			Category:     ExternalLink,
			Position:     link.Position,
			WasConverted: false,
		}

	default:
		return nil
	}
}

// rewriteInternalLink handles internal link rewriting (path updates if needed)
func (lr *ExportLinkRewriter) rewriteInternalLink(link vault.Link, originalText string, file *vault.VaultFile) *LinkChange {
	// For now, internal links are preserved as-is
	// In the future, this could handle path updates if files are reorganized during export
	return nil
}

// findURLInFrontmatter looks for a URL field in frontmatter that matches the link target
func (lr *ExportLinkRewriter) findURLInFrontmatter(target string, file *vault.VaultFile) string {
	if file.Frontmatter == nil {
		return ""
	}

	// Common URL fields to check
	urlFields := []string{"url", "link", "source", "website"}
	
	for _, field := range urlFields {
		if value, exists := file.Frontmatter[field]; exists {
			if url, ok := value.(string); ok && strings.HasPrefix(url, "http") {
				return url
			}
		}
	}

	return ""
}

// extractLinkText extracts the original text of a link from content
func (lr *ExportLinkRewriter) extractLinkText(content string, link vault.Link) string {
	if link.Position.Start < 0 || link.Position.End > len(content) {
		return ""
	}
	return content[link.Position.Start:link.Position.End]
}

// GetRewriteStrategies returns all available rewrite strategies
func GetRewriteStrategies() []LinkRewriteStrategy {
	return []LinkRewriteStrategy{RemoveStrategy, URLStrategy}
}

// IsValidStrategy checks if a strategy is valid
func IsValidStrategy(strategy string) bool {
	for _, s := range GetRewriteStrategies() {
		if string(s) == strategy {
			return true
		}
	}
	return false
}