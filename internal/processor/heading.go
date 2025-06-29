package processor

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// Heading represents a markdown heading
type Heading struct {
	Level int    // Heading level (1-6)
	Text  string // Heading text
	Line  int    // Line number
}

// HeadingIssue represents a problem with heading structure
type HeadingIssue struct {
	Type     string // Issue type (multiple_h1, h1_title_mismatch, skipped_level, missing_h1)
	Line     int    // Line number
	Expected string // Expected value
	Actual   string // Actual value
}

// HeadingAnalysis contains the results of heading analysis
type HeadingAnalysis struct {
	Issues []HeadingIssue
}

// HeadingRules defines rules for fixing headings
type HeadingRules struct {
	EnsureH1Title bool // Ensure first content line is H1 matching title
	SingleH1      bool // Only one H1 allowed
	FixSequence   bool // Fix skipped heading levels
	MinLevel      int  // Minimum heading level after H1
}

// CleanRules defines rules for cleaning headings for Obsidian compatibility
type CleanRules struct {
	SquareBrackets bool // Convert [X] to <X> in headings
	LinkHeaders    bool // Convert headings containing links to list items
}

// CleanStats tracks the number of transformations applied
type CleanStats struct {
	SquareBracketsFixed  int // Number of square bracket replacements
	LinkHeadersConverted int // Number of link headers converted to list items
}

// HeadingProcessor handles heading analysis and fixes
type HeadingProcessor struct {
	headingPattern       *regexp.Regexp
	squareBracketPattern *regexp.Regexp
	wikiLinkPattern      *regexp.Regexp
	mdLinkPattern        *regexp.Regexp
}

// NewHeadingProcessor creates a new heading processor
func NewHeadingProcessor() *HeadingProcessor {
	return &HeadingProcessor{
		headingPattern:       regexp.MustCompile(`^(#{1,6})\s+(.+)$`),
		squareBracketPattern: regexp.MustCompile(`\[([^\]]*)\]`),
		wikiLinkPattern:      regexp.MustCompile(`\[\[([^\]]+)\]\]`),
		mdLinkPattern:        regexp.MustCompile(`\[([^\]]*)\]\(([^\)]+)\)`),
	}
}

// Analyze examines heading structure and reports issues
func (p *HeadingProcessor) Analyze(file *vault.VaultFile) HeadingAnalysis {
	headings := p.ExtractHeadings(file.Body)
	analysis := HeadingAnalysis{}

	// Check for multiple H1s
	h1Count := 0
	var firstH1 *Heading
	for i, h := range headings {
		if h.Level == 1 {
			h1Count++
			if h1Count == 1 {
				firstH1 = &headings[i]
			} else {
				analysis.Issues = append(analysis.Issues, HeadingIssue{
					Type: "multiple_h1",
					Line: h.Line,
				})
			}
		}
	}

	// Check H1 matches title
	if title, ok := file.Frontmatter["title"].(string); ok {
		if h1Count == 0 {
			analysis.Issues = append(analysis.Issues, HeadingIssue{
				Type:     "missing_h1",
				Expected: title,
			})
		} else if firstH1 != nil && firstH1.Text != title {
			analysis.Issues = append(analysis.Issues, HeadingIssue{
				Type:     "h1_title_mismatch",
				Expected: title,
				Actual:   firstH1.Text,
			})
		}
	}

	// Check for skipped heading levels
	expectedLevel := 2 // After H1, expect H2
	for _, h := range headings {
		if h.Level == 1 {
			expectedLevel = 2
			continue
		}

		if h.Level > expectedLevel {
			analysis.Issues = append(analysis.Issues, HeadingIssue{
				Type:     "skipped_level",
				Line:     h.Line,
				Expected: p.levelToString(expectedLevel),
				Actual:   p.levelToString(h.Level),
			})
		}

		expectedLevel = h.Level + 1
		if expectedLevel > 6 {
			expectedLevel = 6
		}
	}

	return analysis
}

// Fix applies heading rules to fix issues
func (p *HeadingProcessor) Fix(file *vault.VaultFile, rules HeadingRules) error {
	body := file.Body

	if rules.EnsureH1Title {
		if title, ok := file.Frontmatter["title"].(string); ok {
			body = p.ensureH1Title(body, title)
		}
	}

	if rules.SingleH1 {
		body = p.convertExtraH1s(body)
	}

	if rules.FixSequence {
		body = p.fixHeadingSequence(body)
	}

	file.Body = body
	return nil
}

// Clean applies cleaning rules to headings for Obsidian compatibility
func (p *HeadingProcessor) Clean(file *vault.VaultFile, rules CleanRules) (CleanStats, error) {
	stats := CleanStats{}
	body := file.Body

	// Process link headers first, since square bracket replacement might interfere with link detection
	if rules.LinkHeaders {
		newBody, count := p.convertLinkHeaders(body)
		body = newBody
		stats.LinkHeadersConverted = count
	}

	if rules.SquareBrackets {
		newBody, count := p.replaceSquareBrackets(body)
		body = newBody
		stats.SquareBracketsFixed = count
	}

	file.Body = body
	return stats, nil
}

// ExtractHeadings parses headings from markdown content
func (p *HeadingProcessor) ExtractHeadings(content string) []Heading {
	var headings []Heading
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		matches := p.headingPattern.FindStringSubmatch(strings.TrimSpace(line))
		if len(matches) == 3 {
			headings = append(headings, Heading{
				Level: len(matches[1]),
				Text:  strings.TrimSpace(matches[2]),
				Line:  i + 1,
			})
		}
	}

	return headings
}

// ensureH1Title ensures the first content line is H1 matching title
func (p *HeadingProcessor) ensureH1Title(body, title string) string {
	lines := strings.Split(body, "\n")

	// Find first non-empty line
	firstContentIndex := -1
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			firstContentIndex = i
			break
		}
	}

	if firstContentIndex == -1 {
		// No content, prepend H1
		return "# " + title + "\n\n" + body
	}

	// Check if first line is already correct H1
	firstLine := strings.TrimSpace(lines[firstContentIndex])
	if firstLine == "# "+title {
		return body
	}

	// Check if first line is an H1 (correct or incorrect)
	if strings.HasPrefix(firstLine, "# ") {
		// Replace existing H1
		lines[firstContentIndex] = "# " + title
	} else {
		// Insert H1 before first content
		newLines := make([]string, 0, len(lines)+2)
		newLines = append(newLines, lines[:firstContentIndex]...)
		newLines = append(newLines, "# "+title)
		newLines = append(newLines, "")
		newLines = append(newLines, lines[firstContentIndex:]...)
		lines = newLines
	}

	return strings.Join(lines, "\n")
}

// convertExtraH1s converts additional H1s to H2s
func (p *HeadingProcessor) convertExtraH1s(body string) string {
	lines := strings.Split(body, "\n")
	h1Count := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") && !strings.HasPrefix(trimmed, "## ") {
			h1Count++
			if h1Count > 1 {
				// Convert to H2
				lines[i] = strings.Replace(line, "# ", "## ", 1)
			}
		}
	}

	return strings.Join(lines, "\n")
}

// fixHeadingSequence adjusts heading levels to avoid skipping
func (p *HeadingProcessor) fixHeadingSequence(body string) string {
	lines := strings.Split(body, "\n")
	expectedLevel := 2 // After H1, expect H2

	for i, line := range lines {
		matches := p.headingPattern.FindStringSubmatch(strings.TrimSpace(line))
		if len(matches) == 3 {
			currentLevel := len(matches[1])

			if currentLevel == 1 {
				expectedLevel = 2
				continue
			}

			if currentLevel > expectedLevel {
				// Adjust to expected level
				newHashes := strings.Repeat("#", expectedLevel)
				lines[i] = strings.Replace(line, matches[1], newHashes, 1)
				currentLevel = expectedLevel
			}

			expectedLevel = currentLevel + 1
			if expectedLevel > 6 {
				expectedLevel = 6
			}
		}
	}

	return strings.Join(lines, "\n")
}

// levelToString converts heading level to string representation
func (p *HeadingProcessor) levelToString(level int) string {
	switch level {
	case 1:
		return "H1"
	case 2:
		return "H2"
	case 3:
		return "H3"
	case 4:
		return "H4"
	case 5:
		return "H5"
	case 6:
		return "H6"
	default:
		return "H?"
	}
}

// replaceSquareBrackets converts [X] to <X> in headings, but ignores wiki links [[]] and markdown links []()
func (p *HeadingProcessor) replaceSquareBrackets(body string) (string, int) {
	lines := strings.Split(body, "\n")
	count := 0

	for i, line := range lines {
		// Check if this line is a heading
		if matches := p.headingPattern.FindStringSubmatch(strings.TrimSpace(line)); len(matches) == 3 {
			headingText := matches[2]

			// First, protect wiki links and markdown links by replacing them with placeholders
			wikiLinkMatches := p.wikiLinkPattern.FindAllString(headingText, -1)
			mdLinkMatches := p.mdLinkPattern.FindAllString(headingText, -1)

			protectedText := headingText
			placeholders := []string{}

			// Replace wiki links with placeholders
			for j, match := range wikiLinkMatches {
				placeholder := "WIKILINK_PLACEHOLDER_" + fmt.Sprintf("%d", j)
				protectedText = strings.Replace(protectedText, match, placeholder, 1)
				placeholders = append(placeholders, match)
			}

			// Replace markdown links with placeholders
			for j, match := range mdLinkMatches {
				placeholder := "MDLINK_PLACEHOLDER_" + fmt.Sprintf("%d", j)
				protectedText = strings.Replace(protectedText, match, placeholder, 1)
				placeholders = append(placeholders, match)
			}

			// Now replace square brackets in the protected text
			newText := p.squareBracketPattern.ReplaceAllStringFunc(protectedText, func(match string) string {
				content := match[1 : len(match)-1] // Remove [ and ]
				count++
				return "<" + content + ">"
			})

			// Restore the protected links
			for j, match := range wikiLinkMatches {
				placeholder := "WIKILINK_PLACEHOLDER_" + fmt.Sprintf("%d", j)
				newText = strings.Replace(newText, placeholder, match, 1)
			}

			for j, match := range mdLinkMatches {
				placeholder := "MDLINK_PLACEHOLDER_" + fmt.Sprintf("%d", j)
				newText = strings.Replace(newText, placeholder, match, 1)
			}

			if newText != headingText {
				// Reconstruct the line with the new heading text
				lines[i] = strings.Replace(line, headingText, newText, 1)
			}
		}
	}

	return strings.Join(lines, "\n"), count
}

// convertLinkHeaders converts headings containing links to list items
func (p *HeadingProcessor) convertLinkHeaders(body string) (string, int) {
	lines := strings.Split(body, "\n")
	count := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this line is a heading
		if matches := p.headingPattern.FindStringSubmatch(trimmed); len(matches) == 3 {
			headingText := matches[2]

			// Check if heading contains wiki links or markdown links
			hasWikiLink := p.wikiLinkPattern.MatchString(headingText)
			hasMdLink := p.mdLinkPattern.MatchString(headingText)

			if hasWikiLink || hasMdLink {
				// Extract the original indentation (everything before the trimmed content)
				originalIndent := line[:len(line)-len(trimmed)]
				// Convert heading to list item
				lines[i] = originalIndent + "- " + headingText
				count++
			}
		}
	}

	return strings.Join(lines, "\n"), count
}
