package vault

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// VaultFile represents a markdown file in an Obsidian vault
type VaultFile struct {
	Path         string
	RelativePath string
	Content      []byte
	Frontmatter  map[string]interface{}
	Body         string
	Links        []Link
	Headings     []Heading
	Modified     time.Time
}

// LinkType represents the type of markdown link
type LinkType int

const (
	WikiLink LinkType = iota
	MarkdownLink
	EmbedLink
)

// Link represents a link in markdown content
type Link struct {
	Type     LinkType
	Target   string
	Text     string
	Position Position
}

// Position represents a position in text
type Position struct {
	Start int
	End   int
}

// Heading represents a markdown heading
type Heading struct {
	Level int
	Text  string
	Line  int
}

// Parse extracts frontmatter and body from markdown content
func (vf *VaultFile) Parse(content []byte) error {
	vf.Content = content
	vf.Frontmatter = make(map[string]interface{})

	// Check for frontmatter
	if !bytes.HasPrefix(content, []byte("---\n")) && !bytes.HasPrefix(content, []byte("---\r\n")) {
		// No frontmatter, entire content is body
		vf.Body = string(content)
		return nil
	}

	// Find the closing --- delimiter
	contentStr := string(content)
	lines := strings.Split(contentStr, "\n")
	
	if len(lines) < 2 {
		vf.Body = contentStr
		return nil
	}

	// Find closing delimiter
	var endIndex int = -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIndex = i
			break
		}
	}

	if endIndex == -1 {
		// No closing delimiter found, treat as regular content
		vf.Body = contentStr
		return nil
	}

	// Extract frontmatter content (between the --- delimiters)
	frontmatterLines := lines[1:endIndex]
	frontmatterContent := strings.Join(frontmatterLines, "\n")

	// Parse YAML frontmatter
	if strings.TrimSpace(frontmatterContent) != "" {
		if err := yaml.Unmarshal([]byte(frontmatterContent), &vf.Frontmatter); err != nil {
			return fmt.Errorf("parsing frontmatter: %w", err)
		}
	}

	// Extract body (everything after closing ---)
	if endIndex+1 < len(lines) {
		bodyLines := lines[endIndex+1:]
		// Remove leading empty line if present
		if len(bodyLines) > 0 && strings.TrimSpace(bodyLines[0]) == "" {
			bodyLines = bodyLines[1:]
		}
		vf.Body = strings.Join(bodyLines, "\n")
	}

	return nil
}

// Serialize converts the VaultFile back to markdown content
func (vf *VaultFile) Serialize() ([]byte, error) {
	var buf bytes.Buffer

	// Write frontmatter if it exists and is not empty
	if len(vf.Frontmatter) > 0 {
		buf.WriteString("---\n")
		
		// Serialize frontmatter as YAML
		yamlData, err := yaml.Marshal(vf.Frontmatter)
		if err != nil {
			return nil, fmt.Errorf("marshaling frontmatter: %w", err)
		}
		
		buf.Write(yamlData)
		buf.WriteString("---\n")
		
		// Add blank line after frontmatter if body exists
		if vf.Body != "" {
			buf.WriteString("\n")
		}
	}

	// Write body
	buf.WriteString(vf.Body)

	return buf.Bytes(), nil
}

// HasFrontmatter returns true if the file has frontmatter
func (vf *VaultFile) HasFrontmatter() bool {
	return len(vf.Frontmatter) > 0
}

// GetField returns a frontmatter field value
func (vf *VaultFile) GetField(key string) (interface{}, bool) {
	value, exists := vf.Frontmatter[key]
	return value, exists
}

// SetField sets a frontmatter field value
func (vf *VaultFile) SetField(key string, value interface{}) {
	if vf.Frontmatter == nil {
		vf.Frontmatter = make(map[string]interface{})
	}
	vf.Frontmatter[key] = value
}