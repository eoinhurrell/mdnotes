package vault

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Date represents a date that serializes as YYYY-MM-DD without quotes in YAML
type Date struct {
	time.Time
}

// MarshalYAML implements yaml.Marshaler to output dates without quotes
func (d Date) MarshalYAML() (interface{}, error) {
	// Check if time component is not midnight (00:00:00)
	hour, min, sec := d.Time.Clock()
	if hour != 0 || min != 0 || sec != 0 {
		// Has meaningful time component, format as datetime
		node := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: d.Time.Format("2006-01-02 15:04:05"),
			Tag:   "!!timestamp",
		}
		return node, nil
	}
	
	// No time component, format as date only
	node := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: d.Time.Format("2006-01-02"),
		Tag:   "!!timestamp",
	}
	return node, nil
}

// String returns the date as YYYY-MM-DD
func (d Date) String() string {
	return d.Time.Format("2006-01-02")
}

// VaultFile represents a markdown file in an Obsidian vault
type VaultFile struct {
	Path              string
	RelativePath      string
	Content           []byte
	Frontmatter       map[string]interface{}
	frontmatterOrder  []string // Preserve original field order
	originalFrontmatter string // Store original frontmatter text for reference
	Body              string
	Links             []Link
	Headings          []Heading
	Modified          time.Time
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

	// Parse YAML frontmatter while preserving order
	if strings.TrimSpace(frontmatterContent) != "" {
		vf.originalFrontmatter = frontmatterContent
		
		// Parse YAML to extract key order
		vf.frontmatterOrder = extractFieldOrder(frontmatterContent)
		
		// Parse YAML content
		if err := yaml.Unmarshal([]byte(frontmatterContent), &vf.Frontmatter); err != nil {
			return fmt.Errorf("parsing frontmatter: %w", err)
		}
		
		// Convert time.Time values to Date type for most fields
		// Keep datetime fields as time.Time for full timestamp serialization
		vf.normalizeFieldTypes()
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

// Serialize converts the VaultFile back to markdown content preserving field order
func (vf *VaultFile) Serialize() ([]byte, error) {
	var buf bytes.Buffer

	// Write frontmatter if it exists and is not empty
	if len(vf.Frontmatter) > 0 {
		buf.WriteString("---\n")
		
		// Serialize frontmatter preserving order
		frontmatterYAML, err := vf.serializeFrontmatterWithOrder()
		if err != nil {
			return nil, fmt.Errorf("marshaling frontmatter: %w", err)
		}
		
		buf.WriteString(frontmatterYAML)
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

// SetField sets a frontmatter field value while preserving order
func (vf *VaultFile) SetField(key string, value interface{}) {
	if vf.Frontmatter == nil {
		vf.Frontmatter = make(map[string]interface{})
	}
	
	// Don't add new fields to frontmatterOrder here - let serializeFrontmatterWithOrder handle them
	// The frontmatterOrder should only contain the original fields to preserve their order
	
	vf.Frontmatter[key] = value
}


// extractFieldOrder extracts the order of fields from the original YAML content
func extractFieldOrder(yamlContent string) []string {
	var order []string
	lines := strings.Split(yamlContent, "\n")
	
	// Regex to match YAML keys (including those with spaces)
	// Matches: key:, "key with spaces":, 'key with spaces':, or unquoted keys with spaces
	keyRegex := regexp.MustCompile(`^(\s*)(?:"([^"]+)"|'([^']+)'|([a-zA-Z_][a-zA-Z0-9_\s]*?))\s*:`)
	
	for _, line := range lines {
		if matches := keyRegex.FindStringSubmatch(line); len(matches) > 1 {
			// Extract key from the appropriate capture group
			var key string
			if matches[2] != "" {
				key = matches[2] // Quoted with double quotes
			} else if matches[3] != "" {
				key = matches[3] // Quoted with single quotes
			} else if matches[4] != "" {
				key = strings.TrimSpace(matches[4]) // Unquoted (may have spaces)
			}
			
			if key != "" {
				// Only add if not already in order (handles multi-line values)
				found := false
				for _, existing := range order {
					if existing == key {
						found = true
						break
					}
				}
				if !found {
					order = append(order, key)
				}
			}
		}
	}
	
	return order
}

// serializeFrontmatterWithOrder serializes frontmatter while preserving field order
func (vf *VaultFile) serializeFrontmatterWithOrder() (string, error) {
	if len(vf.Frontmatter) == 0 {
		return "", nil
	}
	
	var lines []string
	processedKeys := make(map[string]bool)
	
	// First, write fields in their original order
	for _, key := range vf.frontmatterOrder {
		if value, exists := vf.Frontmatter[key]; exists {
			yamlLine, err := formatYAMLField(key, value)
			if err != nil {
				return "", fmt.Errorf("formatting field %s: %w", key, err)
			}
			lines = append(lines, yamlLine)
			processedKeys[key] = true
		}
	}
	
	// Then, add any new fields that weren't in the original order
	var newKeys []string
	for key := range vf.Frontmatter {
		if !processedKeys[key] {
			newKeys = append(newKeys, key)
		}
	}
	
	// Sort new keys for consistent output
	sort.Strings(newKeys)
	
	for _, key := range newKeys {
		yamlLine, err := formatYAMLField(key, vf.Frontmatter[key])
		if err != nil {
			return "", fmt.Errorf("formatting field %s: %w", key, err)
		}
		lines = append(lines, yamlLine)
	}
	
	return strings.Join(lines, "\n") + "\n", nil
}

// formatYAMLField formats a single YAML field properly
func formatYAMLField(key string, value interface{}) (string, error) {
	// Create a temporary map with just this field
	tempMap := map[string]interface{}{key: value}
	
	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(tempMap)
	if err != nil {
		return "", err
	}
	
	// Return the line without the trailing newline
	yamlStr := strings.TrimSpace(string(yamlBytes))
	return yamlStr, nil
}

// normalizeFieldTypes converts time.Time values to Date type
// Date type will automatically format as YYYY-MM-DD or YYYY-MM-DD HH:mm:ss based on time component
func (vf *VaultFile) normalizeFieldTypes() {
	for field, value := range vf.Frontmatter {
		if timeValue, ok := value.(time.Time); ok {
			// Convert all time.Time values to our Date type for smart formatting
			vf.Frontmatter[field] = Date{Time: timeValue}
		}
	}
}