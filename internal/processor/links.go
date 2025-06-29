package processor

import (
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// Import types from vault package
type LinkType = vault.LinkType
type Link = vault.Link
type Position = vault.Position

const (
	WikiLink     = vault.WikiLink
	MarkdownLink = vault.MarkdownLink
	EmbedLink    = vault.EmbedLink
)

// LinkParser handles parsing links from markdown content
type LinkParser struct {
	patterns map[LinkType]*regexp.Regexp
}

// NewLinkParser creates a new link parser with comprehensive patterns
func NewLinkParser() *LinkParser {
	return &LinkParser{
		patterns: map[LinkType]*regexp.Regexp{
			// Wiki links: [[target]] or [[target|alias]] with fragment support
			// Supports: [[file]], [[file#heading]], [[file#^blockid]], [[file|alias]], [[file#heading|alias]]
			WikiLink: regexp.MustCompile(`\[\[([^|\]#]+(?:#[^|\]]+)?(?:\[[^\]]*\][^|\]#]*)*?)(?:\|([^\]]+(?:\[[^\]]*\][^\]]*)*?))?\]\]`),
			// Markdown links: [text](target) with angle brackets and fragments
			// Supports: [text](file.md), [text](file.md#heading), [text](<file with spaces.md>)
			// Use simple pattern first, then handle balanced parentheses manually
			MarkdownLink: regexp.MustCompile(`\[([^\]]*)\]\(`),
			// Embed links: ![[target]] with fragment support
			// Supports: ![[file]], ![[file#heading]], ![[file#^blockid]]
			EmbedLink: regexp.MustCompile(`!\[\[([^\]#]+(?:#[^\]]+)?(?:\[[^\]]*\][^\]#]*)*?)\]\]`),
		},
	}
}

// Extract finds all links in the given content
func (p *LinkParser) Extract(content string) []Link {
	var links []Link
	usedPositions := make(map[Position]bool)

	// Process in order: embeds first (they contain [[ like wiki links), then wiki, then markdown
	embedLinks := p.extractType(content, EmbedLink)
	for _, link := range embedLinks {
		if link.Type == MarkdownLink && !p.IsInternalLink(link.Target) {
			continue
		}
		links = append(links, link)
		usedPositions[link.Position] = true
	}

	wikiLinks := p.extractType(content, WikiLink)
	for _, link := range wikiLinks {
		if !p.overlapsUsedPosition(link.Position, usedPositions) && p.IsInternalLink(link.Target) {
			links = append(links, link)
			usedPositions[link.Position] = true
		}
	}

	markdownLinks := p.extractType(content, MarkdownLink)
	for _, link := range markdownLinks {
		if link.Type == MarkdownLink && !p.IsInternalLink(link.Target) {
			continue
		}
		if !p.overlapsUsedPosition(link.Position, usedPositions) {
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
func (p *LinkParser) overlapsUsedPosition(pos Position, used map[Position]bool) bool {
	for usedPos := range used {
		if (pos.Start >= usedPos.Start && pos.Start < usedPos.End) ||
			(pos.End > usedPos.Start && pos.End <= usedPos.End) ||
			(pos.Start <= usedPos.Start && pos.End >= usedPos.End) {
			return true
		}
	}
	return false
}

// extractType extracts links of a specific type
func (p *LinkParser) extractType(content string, linkType LinkType) []Link {
	var links []Link
	pattern := p.patterns[linkType]

	matches := pattern.FindAllStringSubmatchIndex(content, -1)
	for _, match := range matches {
		link := Link{
			Type: linkType,
			Position: Position{
				Start: match[0],
				End:   match[1],
			},
			RawText: content[match[0]:match[1]],
		}

		// Extract groups from the match
		groups := pattern.FindStringSubmatch(content[match[0]:match[1]])
		if len(groups) < 2 {
			continue
		}

		switch linkType {
		case WikiLink:
			// Parse target with potential fragment
			fullTarget := groups[1]
			link.Target, link.Fragment = p.parseTargetAndFragment(fullTarget)

			// Parse alias
			if len(groups) > 2 && groups[2] != "" {
				link.Alias = groups[2]
				link.Text = groups[2]
			} else {
				link.Text = link.Target
				if link.Fragment != "" {
					link.Text = fullTarget // Include fragment in display
				}
			}

		case MarkdownLink:
			link.Text = groups[1]

			// For markdown links, we need to manually find the balanced closing parenthesis
			// since our pattern only matches up to the opening parenthesis
			linkStart := match[0]
			afterOpenParen := linkStart + len(groups[0])

			target, endPos := p.findBalancedTarget(content, afterOpenParen)
			if target == "" {
				continue // Skip malformed links
			}

			// Update the link position to include the full link
			link.Position.End = endPos + 1 // +1 for the closing )
			link.RawText = content[linkStart:link.Position.End]

			// Handle angle bracket encoding
			if strings.HasPrefix(target, "<") && strings.HasSuffix(target, ">") {
				target = target[1 : len(target)-1] // Remove < >
				link.Encoding = "angle"
			} else {
				// Detect URL encoding
				if strings.Contains(target, "%") {
					link.Encoding = "url"
				} else {
					link.Encoding = "none"
				}
			}

			// Parse target and fragment
			link.Target, link.Fragment = p.parseTargetAndFragment(target)

		case EmbedLink:
			// Parse target with potential fragment
			fullTarget := groups[1]
			link.Target, link.Fragment = p.parseTargetAndFragment(fullTarget)
			link.Text = fullTarget // Embeds don't have separate display text
		}

		links = append(links, link)
	}

	return links
}

// findBalancedTarget finds the target string within balanced parentheses
func (p *LinkParser) findBalancedTarget(content string, start int) (target string, endPos int) {
	depth := 0
	i := start

	for i < len(content) {
		switch content[i] {
		case '(':
			depth++
		case ')':
			if depth == 0 {
				// Found the closing parenthesis for our link
				return content[start:i], i
			}
			depth--
		}
		i++
	}

	// No balanced closing parenthesis found
	return "", -1
}

// parseTargetAndFragment separates target from fragment (#heading or #^blockid)
func (p *LinkParser) parseTargetAndFragment(fullTarget string) (target, fragment string) {
	// URL decode first to handle encoded fragments
	decodedTarget := fullTarget
	if strings.Contains(fullTarget, "%") {
		if decoded, err := url.QueryUnescape(fullTarget); err == nil {
			decodedTarget = decoded
		} else {
			// Log warning but continue with original target for robustness
			// In a full implementation, we might use a proper logger here
			decodedTarget = fullTarget
		}
	}

	// Find fragment separator
	if idx := strings.Index(decodedTarget, "#"); idx != -1 {
		target = decodedTarget[:idx]
		fragment = decodedTarget[idx+1:]
	} else {
		target = decodedTarget
	}

	return target, fragment
}

// IsInternalLink checks if a link target is internal (not external URL)
func (p *LinkParser) IsInternalLink(target string) bool {
	// Check for common external schemes
	external := []string{"http://", "https://", "ftp://", "mailto:", "tel:", "www."}
	lowerTarget := strings.ToLower(target)
	for _, scheme := range external {
		if strings.HasPrefix(lowerTarget, scheme) {
			return false
		}
	}

	// Check for domain-like patterns (for wiki links that might reference external)
	// This is a heuristic - if it looks like a domain, treat as external
	if strings.Contains(target, ".com") || strings.Contains(target, ".org") ||
		strings.Contains(target, ".net") || strings.Contains(target, ".edu") {
		// But allow if it looks like a filename with extension
		if strings.HasSuffix(target, ".md") || strings.HasSuffix(target, ".png") ||
			strings.HasSuffix(target, ".jpg") || strings.HasSuffix(target, ".pdf") {
			return true
		}
		return false
	}

	return true
}

// UpdateFile parses links from file content and updates the Links field
func (p *LinkParser) UpdateFile(file *vault.VaultFile) {
	file.Links = p.Extract(file.Body)
}
