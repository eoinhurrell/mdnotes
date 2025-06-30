package analyzer

import (
	"strings"
	"testing"
	"time"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

func TestCalculateReadabilityScore(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name     string
		content  string
		expected float64 // approximate expected value
		min      float64 // minimum acceptable value
		max      float64 // maximum acceptable value
	}{
		{
			name:     "Empty content",
			content:  "",
			expected: 0.0,
			min:      0.0,
			max:      0.0,
		},
		{
			name:     "Simple sentences",
			content:  "This is easy to read. Short sentences work well.",
			expected: 0.8,
			min:      0.5,
			max:      1.0,
		},
		{
			name:     "Complex content",
			content:  "The implementation of sophisticated algorithms requires comprehensive understanding of computational complexity theory and advanced mathematical concepts.",
			expected: 0.3,
			min:      0.0,
			max:      1.0,
		},
		{
			name:     "Medium complexity",
			content:  "Content quality analysis helps identify areas for improvement in Zettelkasten notes. This scoring system evaluates multiple factors.",
			expected: 0.6,
			min:      0.0,
			max:      1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := &vault.VaultFile{
				Body: tt.content,
			}
			score := analyzer.CalculateReadabilityScore(file)

			if score < tt.min || score > tt.max {
				t.Errorf("calculateReadabilityScore() = %f, want between %f and %f", score, tt.min, tt.max)
			}
		})
	}
}

func TestCalculateLinkDensityScore(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name      string
		content   string
		linkCount int
		expected  float64
	}{
		{
			name:      "No links",
			content:   "This content has no links at all. It should score poorly for Zettelkasten principles.",
			linkCount: 0,
			expected:  0.0,
		},
		{
			name:      "Optimal link density",
			content:   strings.Repeat("word ", 100), // exactly 100 words
			linkCount: 3,
			expected:  1.0, // 3 links per 100 words = exactly 3.0, optimal range (3.0-4.0)
		},
		{
			name:      "Too many links",
			content:   "This content has way too many links for good readability.",
			linkCount: 8,
			expected:  0.5,
		},
		{
			name:      "Good link density",
			content:   strings.Repeat("word ", 100), // exactly 100 words
			linkCount: 2,
			expected:  0.8, // 2 links per 100 words = exactly 2.0, score = 0.8 + (2.0-2.0)*0.2 = 0.8
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := &vault.VaultFile{
				Body:  tt.content,
				Links: make([]vault.Link, tt.linkCount),
			}
			score := analyzer.CalculateLinkDensityScore(file)

			// Allow some tolerance for floating point comparison
			tolerance := 0.2
			if score < tt.expected-tolerance || score > tt.expected+tolerance {
				t.Errorf("calculateLinkDensityScore() = %f, want approximately %f", score, tt.expected)
			}
		})
	}
}

func TestCalculateCompletenessScore(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name        string
		frontmatter map[string]interface{}
		content     string
		expected    float64
	}{
		{
			name:        "Complete note",
			frontmatter: map[string]interface{}{"title": "Test Note", "summary": "A test note"},
			content:     strings.Repeat("This is a complete note with good content length for analysis. ", 10), // ~100 words, gets 0.3 for word count
			expected:    1.0, // 0.4 (title) + 0.3 (summary) + 0.3 (>=50 words) = 1.0
		},
		{
			name:        "Missing title",
			frontmatter: map[string]interface{}{"summary": "A test note"},
			content:     strings.Repeat("This note is missing a title but has other elements and sufficient length. ", 7), // ~70 words
			expected:    0.6, // 0.0 (no title) + 0.3 (summary) + 0.3 (>=50 words) = 0.6
		},
		{
			name:        "Missing summary",
			frontmatter: map[string]interface{}{"title": "Test Note"},
			content:     strings.Repeat("This note has a title but no summary field and needs sufficient content length. ", 7), // ~70 words
			expected:    0.7, // 0.4 (title) + 0.0 (no summary) + 0.3 (>=50 words) = 0.7
		},
		{
			name:        "Too short",
			frontmatter: map[string]interface{}{"title": "Test", "summary": "Short"},
			content:     "Short.",
			expected:    0.7, // 0.4 for title + 0.3 for summary + 0.0 for short content
		},
		{
			name:        "Empty content",
			frontmatter: map[string]interface{}{},
			content:     "",
			expected:    0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := &vault.VaultFile{
				Body:        tt.content,
				Frontmatter: tt.frontmatter,
			}
			score := analyzer.CalculateCompletenessScore(file)

			tolerance := 0.1
			if score < tt.expected-tolerance || score > tt.expected+tolerance {
				t.Errorf("calculateCompletenessScore() = %f, want approximately %f", score, tt.expected)
			}
		})
	}
}

func TestCalculateAtomicityScore(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name     string
		content  string
		headings []vault.Heading
		expected float64
		min      float64
		max      float64
	}{
		{
			name:     "Perfect atomic note",
			content:  "This is a focused note about a single concept. It's the right length and covers one topic well with focused content and coherent discussion of the main topic.",
			headings: []vault.Heading{{Level: 1, Text: "Main Topic"}},
			expected: 1.0,
			min:      0.70, // Topic coherence reduces the score to ~0.71
			max:      1.0,
		},
		{
			name:     "Too long note",
			content:  generateLongContent(600), // Over 500 words
			headings: []vault.Heading{{Level: 1, Text: "Topic"}},
			expected: 0.9,
			min:      0.65, // Lower due to word penalty and topic coherence
			max:      0.85,
		},
		{
			name:     "Multiple H1 headings",
			content:  "This note covers multiple main topics which violates atomicity.",
			headings: []vault.Heading{{Level: 1, Text: "Topic 1"}, {Level: 1, Text: "Topic 2"}},
			expected: 0.7,
			min:      0.5,
			max:      0.8,
		},
		{
			name:     "Too many H2 headings",
			content:  "This note has too many subsections.",
			headings: []vault.Heading{{Level: 2, Text: "Sub1"}, {Level: 2, Text: "Sub2"}, {Level: 2, Text: "Sub3"}, {Level: 2, Text: "Sub4"}, {Level: 2, Text: "Sub5"}},
			expected: 0.8,
			min:      0.6,
			max:      0.9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := &vault.VaultFile{
				Body:     tt.content,
				Headings: tt.headings,
			}
			score := analyzer.CalculateAtomicityScore(file)

			if score < tt.min || score > tt.max {
				t.Errorf("calculateAtomicityScore() = %f, want between %f and %f", score, tt.min, tt.max)
			}
		})
	}
}

func TestCalculateRecencyScore(t *testing.T) {
	analyzer := NewAnalyzer()
	now := time.Now()

	tests := []struct {
		name     string
		modified time.Time
		expected float64
	}{
		{
			name:     "Very recent",
			modified: now.AddDate(0, 0, -3), // 3 days ago
			expected: 1.0,
		},
		{
			name:     "Recent",
			modified: now.AddDate(0, 0, -15), // 15 days ago
			expected: 0.9,
		},
		{
			name:     "Old",
			modified: now.AddDate(0, -6, 0), // 6 months ago
			expected: 0.3, // According to algorithm: > 365 days = 0.1, <=365 = 0.3. 6 months = ~180 days, so 0.3
		},
		{
			name:     "Very old",
			modified: now.AddDate(-2, 0, 0), // 2 years ago
			expected: 0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := &vault.VaultFile{
				Modified: tt.modified,
			}
			score := analyzer.CalculateRecencyScore(file)

			if score != tt.expected {
				t.Errorf("calculateRecencyScore() = %f, want %f", score, tt.expected)
			}
		})
	}
}

func TestExtractReadableText(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name     string
		markdown string
		expected string
	}{
		{
			name:     "Remove code blocks",
			markdown: "Text before\n```\ncode here\n```\nText after",
			expected: "Text before\n\nText after",
		},
		{
			name:     "Remove inline code",
			markdown: "Text with `inline code` here",
			expected: "Text with  here",
		},
		{
			name:     "Remove markdown links",
			markdown: "Check out [this link](http://example.com) for more info",
			expected: "Check out this link for more info",
		},
		{
			name:     "Remove wiki links",
			markdown: "See [[Other Note|alias]] and [[Simple Link]] for details",
			expected: "See Other Note and Simple Link for details",
		},
		{
			name:     "Remove headings",
			markdown: "# Main Heading\n## Sub Heading\nContent here",
			expected: "Main Heading\nSub Heading\nContent here",
		},
		{
			name:     "Remove list markers",
			markdown: "- Item 1\n* Item 2\n1. Numbered item",
			expected: "Item 1\nItem 2\nNumbered item",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.extractReadableText(tt.markdown)
			if result != tt.expected {
				t.Errorf("extractReadableText() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCountSentences(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "Multiple sentences",
			text:     "This is sentence one. This is sentence two! Is this sentence three?",
			expected: 3,
		},
		{
			name:     "Single sentence",
			text:     "Just one sentence here.",
			expected: 1,
		},
		{
			name:     "No punctuation",
			text:     "Text without sentence ending punctuation",
			expected: 1,
		},
		{
			name:     "Empty text",
			text:     "",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.countSentences(tt.text)
			if result != tt.expected {
				t.Errorf("countSentences() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestEstimateSyllables(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name     string
		word     string
		expected int
	}{
		{
			name:     "Simple word",
			word:     "cat",
			expected: 1,
		},
		{
			name:     "Two syllables",
			word:     "happy",
			expected: 2,
		},
		{
			name:     "Silent e",
			word:     "make",
			expected: 1,
		},
		{
			name:     "Complex word",
			word:     "beautiful",
			expected: 3,
		},
		{
			name:     "Empty word",
			word:     "",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.estimateSyllables(tt.word)
			if result != tt.expected {
				t.Errorf("estimateSyllables() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestGenerateFileQualityFixes(t *testing.T) {
	analyzer := NewAnalyzer()

	// Test with a file that needs all types of fixes
	file := &vault.VaultFile{
		Body:        "Short content.",
		Frontmatter: map[string]interface{}{},
		Headings:    []vault.Heading{{Level: 1, Text: "Topic 1"}, {Level: 1, Text: "Topic 2"}},
	}

	fixes := analyzer.generateFileQualityFixes(file, 0.2, 0.1, 0.3, 0.4, 0.3)

	if len(fixes) == 0 {
		t.Error("Expected some quality fixes to be generated")
	}

	// Check that different types of fixes are included
	hasReadabilityFix := false
	hasLinkFix := false
	hasCompletenessFix := false
	hasAtomicityFix := false
	hasRecencyFix := false

	for _, fix := range fixes {
		fixLower := strings.ToLower(fix)
		if strings.Contains(fixLower, "readability") || strings.Contains(fixLower, "sentence") {
			hasReadabilityFix = true
		}
		if strings.Contains(fixLower, "links") {
			hasLinkFix = true
		}
		if strings.Contains(fixLower, "title") || strings.Contains(fixLower, "summary") || strings.Contains(fixLower, "content") {
			hasCompletenessFix = true
		}
		if strings.Contains(fixLower, "break") || strings.Contains(fixLower, "split") {
			hasAtomicityFix = true
		}
		if strings.Contains(fixLower, "update") || strings.Contains(fixLower, "recent") {
			hasRecencyFix = true
		}
	}

	if !hasReadabilityFix {
		t.Error("Expected readability fix suggestion")
	}
	if !hasLinkFix {
		t.Error("Expected link density fix suggestion")
	}
	if !hasCompletenessFix {
		t.Error("Expected completeness fix suggestion")
	}
	if !hasAtomicityFix {
		t.Error("Expected atomicity fix suggestion")
	}
	if !hasRecencyFix {
		t.Error("Expected recency fix suggestion")
	}
}

// Helper function to generate long content for testing
func generateLongContent(wordCount int) string {
	words := []string{"test", "content", "analysis", "quality", "score", "evaluation", "measurement", "assessment", "review", "examination"}
	var content []string

	for i := 0; i < wordCount; i++ {
		content = append(content, words[i%len(words)])
	}

	return strings.Join(content, " ")
}
