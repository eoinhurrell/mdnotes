package processor

import (
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

// NewLinkParser creates a new link parser
func NewLinkParser() *LinkParser {
	return &LinkParser{
		patterns: map[LinkType]*regexp.Regexp{
			// Wiki links: [[target]] or [[target|alias]]
			// Allow nested brackets in the target/alias parts
			WikiLink:     regexp.MustCompile(`\[\[([^|\]]+(?:\[[^\]]*\][^|\]]*)*?)(?:\|([^\]]+(?:\[[^\]]*\][^\]]*)*?))?\]\]`),
			// Markdown links: [text](target)
			MarkdownLink: regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`),
			// Embed links: ![[target]]
			EmbedLink:    regexp.MustCompile(`!\[\[([^\]]+(?:\[[^\]]*\][^\]]*)*?)\]\]`),
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
		if !p.overlapsUsedPosition(link.Position, usedPositions) {
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
		}

		// Extract groups from the match
		groups := pattern.FindStringSubmatch(content[match[0]:match[1]])
		if len(groups) < 2 {
			continue
		}

		switch linkType {
		case WikiLink:
			link.Target = groups[1]
			if len(groups) > 2 && groups[2] != "" {
				link.Text = groups[2]
			} else {
				link.Text = groups[1]
			}
		case MarkdownLink:
			link.Text = groups[1]
			link.Target = groups[2]
		case EmbedLink:
			link.Target = groups[1]
		}

		links = append(links, link)
	}

	return links
}

// IsInternalLink checks if a link target is internal (not external URL)
func (p *LinkParser) IsInternalLink(target string) bool {
	// Check for common external schemes
	external := []string{"http://", "https://", "ftp://", "mailto:", "tel:"}
	for _, scheme := range external {
		if strings.HasPrefix(target, scheme) {
			return false
		}
	}
	return true
}

// UpdateFile parses links from file content and updates the Links field
func (p *LinkParser) UpdateFile(file *vault.VaultFile) {
	file.Links = p.Extract(file.Body)
}