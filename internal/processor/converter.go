package processor

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// LinkFormat represents the format of links
type LinkFormat int

const (
	WikiFormat LinkFormat = iota
	MarkdownFormat
)

// LinkConverter handles conversion between link formats
type LinkConverter struct {
	parser *LinkParser
}

// NewLinkConverter creates a new link converter
func NewLinkConverter() *LinkConverter {
	return &LinkConverter{
		parser: NewLinkParser(),
	}
}

// Convert transforms links in content from one format to another
func (c *LinkConverter) Convert(content string, from, to LinkFormat) string {
	if from == to {
		return content
	}

	links := c.parser.Extract(content)

	// Filter links that match the source format
	var targetLinks []Link
	for _, link := range links {
		if c.linkMatchesFormat(link, from) {
			targetLinks = append(targetLinks, link)
		}
	}

	if len(targetLinks) == 0 {
		return content
	}

	// Sort links by position (reverse order to avoid position shifts)
	sort.Slice(targetLinks, func(i, j int) bool {
		return targetLinks[i].Position.Start > targetLinks[j].Position.Start
	})

	result := content
	for _, link := range targetLinks {
		// Skip external links for markdown format
		if link.Type == MarkdownLink && !c.parser.IsInternalLink(link.Target) {
			continue
		}

		newLink := c.formatLink(link, to)
		oldLink := content[link.Position.Start:link.Position.End]
		result = strings.Replace(result, oldLink, newLink, 1)
	}

	return result
}

// linkMatchesFormat checks if a link matches the specified format
func (c *LinkConverter) linkMatchesFormat(link Link, format LinkFormat) bool {
	switch format {
	case WikiFormat:
		return link.Type == WikiLink
	case MarkdownFormat:
		return link.Type == MarkdownLink
	default:
		return false
	}
}

// formatLink converts a link to the specified format
func (c *LinkConverter) formatLink(link Link, format LinkFormat) string {
	switch format {
	case MarkdownFormat:
		return c.toMarkdown(link)
	case WikiFormat:
		return c.toWiki(link)
	default:
		return ""
	}
}

// toMarkdown converts a link to markdown format
func (c *LinkConverter) toMarkdown(link Link) string {
	switch link.Type {
	case WikiLink:
		target := link.Target
		text := link.Text

		// Add .md extension if not present and not already has an extension
		if !strings.HasSuffix(target, ".md") && !strings.Contains(filepath.Base(target), ".") {
			target += ".md"
		}

		// Escape spaces and special characters in path
		target = c.escapePath(target)

		return "[" + text + "](" + target + ")"

	case EmbedLink:
		// Embeds stay in wiki format
		return "![[" + link.Target + "]]"

	default:
		// Already markdown or unknown
		return "[" + link.Text + "](" + link.Target + ")"
	}
}

// toWiki converts a link to wiki format
func (c *LinkConverter) toWiki(link Link) string {
	switch link.Type {
	case MarkdownLink:
		target := c.normalizePath(link.Target)
		text := link.Text

		// If text is empty or same as target, use simple format
		if text == "" || text == target || text == link.Target {
			return "[[" + target + "]]"
		}

		return "[[" + target + "|" + text + "]]"

	case EmbedLink:
		// Embeds stay the same
		return "![[" + link.Target + "]]"

	default:
		// Already wiki or unknown
		if link.Text == link.Target {
			return "[[" + link.Target + "]]"
		}
		return "[[" + link.Target + "|" + link.Text + "]]"
	}
}

// normalizePath removes .md extension from path
func (c *LinkConverter) normalizePath(path string) string {
	if strings.HasSuffix(path, ".md") {
		return strings.TrimSuffix(path, ".md")
	}
	return path
}

// escapePath URL-escapes spaces in path for markdown links
func (c *LinkConverter) escapePath(path string) string {
	// Only escape spaces, keep other characters readable
	return strings.ReplaceAll(path, " ", "%20")
}

// ConvertFile converts all links in a file from one format to another
func (c *LinkConverter) ConvertFile(file *vault.VaultFile, from, to LinkFormat) bool {
	originalBody := file.Body
	file.Body = c.Convert(file.Body, from, to)

	// Update the parsed links
	c.parser.UpdateFile(file)

	return file.Body != originalBody
}
