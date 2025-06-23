package vault

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
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
	Path                string
	RelativePath        string
	Content             []byte
	Frontmatter         map[string]interface{}
	frontmatterOrder    []string // Preserve original field order
	originalFrontmatter string   // Store original frontmatter text for reference
	Body                string
	Links               []Link
	Headings            []Heading
	Modified            time.Time
}

// LinkType represents the type of markdown link
type LinkType int

const (
	WikiLink LinkType = iota
	MarkdownLink
	EmbedLink
)

// Link represents a link in markdown content with comprehensive metadata
type Link struct {
	Type     LinkType
	Target   string   // The target file/path without fragments
	Text     string   // Display text or alias
	Fragment string   // Fragment identifier (#heading or #^blockid)
	Alias    string   // Explicit alias for wiki links
	Encoding string   // Original encoding style (url, angle, none)
	RawText  string   // Original link text as found in document
	Position Position
}

// HasFragment returns true if the link has a fragment identifier
func (l Link) HasFragment() bool {
	return l.Fragment != ""
}

// IsHeadingFragment returns true if the fragment is a heading reference
func (l Link) IsHeadingFragment() bool {
	return l.Fragment != "" && !strings.HasPrefix(l.Fragment, "^")
}

// IsBlockFragment returns true if the fragment is a block reference
func (l Link) IsBlockFragment() bool {
	return l.Fragment != "" && strings.HasPrefix(l.Fragment, "^")
}

// FullTarget returns the target with fragment if present
func (l Link) FullTarget() string {
	if l.Fragment == "" {
		return l.Target
	}
	return l.Target + "#" + l.Fragment
}

// ShouldUpdate determines if this link should be updated for a file move
// This method should only return true if the link actually points to the file being moved
func (l Link) ShouldUpdate(oldPath, newPath string) bool {
	// Get the base target (without fragment) for comparison
	linkTarget := l.Target
	if l.Fragment != "" {
		// If link has fragment, we need to compare against the target without fragment
		linkTarget = l.Target // Target already excludes fragment in our parsing
	}
	
	// Remove extensions for comparison
	oldBase := strings.TrimSuffix(oldPath, ".md")
	targetBase := strings.TrimSuffix(linkTarget, ".md")
	
	// For exact path matching, check both original and URL-decoded versions
	if linkTarget == oldPath || targetBase == oldBase {
		return true
	}
	
	// Also check URL-decoded version of the link target against the old path
	if decodedTarget, err := url.QueryUnescape(linkTarget); err == nil {
		decodedBase := strings.TrimSuffix(decodedTarget, ".md")
		if decodedTarget == oldPath || decodedBase == oldBase {
			return true
		}
	}
	
	// Check if oldPath URL-encoded matches the link target
	encodedOldPath := obsidianURLEncode(oldPath)
	encodedOldBase := strings.TrimSuffix(encodedOldPath, ".md")
	if linkTarget == encodedOldPath || targetBase == encodedOldBase {
		return true
	}
	
	// For wiki links, allow basename-only matches but only exact matches
	if l.Type == WikiLink {
		oldBasename := filepath.Base(oldBase)
		targetBasename := filepath.Base(targetBase)
		
		// Exact basename match (case-sensitive for file system accuracy)
		if targetBasename == oldBasename || linkTarget == filepath.Base(oldPath) {
			return true
		}
		
		// Also check URL-decoded basename matching
		if decodedTarget, err := url.QueryUnescape(linkTarget); err == nil {
			decodedBasename := filepath.Base(strings.TrimSuffix(decodedTarget, ".md"))
			if decodedBasename == oldBasename {
				return true
			}
		}
	}
	
	return false
}

// GenerateUpdatedLink creates the new link text for a moved file
func (l Link) GenerateUpdatedLink(newPath string) string {
	newTarget := newPath
	
	switch l.Type {
	case WikiLink:
		// Remove .md extension for wiki links
		newTarget = strings.TrimSuffix(newPath, ".md")
		
		// Add fragment if present
		if l.Fragment != "" {
			newTarget += "#" + l.Fragment
		}
		
		// Check if we need an alias
		if l.Alias != "" {
			return "[[" + newTarget + "|" + l.Alias + "]]"
		} else {
			return "[[" + newTarget + "]]"
		}
		
	case MarkdownLink:
		// Apply encoding to the path part only, then add fragment
		encodedPath := newTarget
		if l.Encoding == "url" || needsURLEncoding(newTarget) {
			encodedPath = obsidianURLEncode(newTarget)
		}
		
		// Add fragment after encoding (encode fragment if it contains special characters)
		if l.Fragment != "" {
			if needsURLEncoding(l.Fragment) {
				encodedFragment := obsidianURLEncode(l.Fragment)
				encodedPath += "#" + encodedFragment
			} else {
				encodedPath += "#" + l.Fragment
			}
		}
		
		// Apply angle bracket wrapping if needed
		if l.Encoding == "angle" {
			encodedPath = "<" + encodedPath + ">"
		}
		
		return "[" + l.Text + "](" + encodedPath + ")"
		
	case EmbedLink:
		// Remove .md extension for embed links
		newTarget = strings.TrimSuffix(newPath, ".md")
		
		// Add fragment if present
		if l.Fragment != "" {
			newTarget += "#" + l.Fragment
		}
		
		return "![[" + newTarget + "]]"
		
	default:
		return l.RawText
	}
}

// Helper functions for encoding
func needsURLEncoding(path string) bool {
	return strings.ContainsAny(path, " '\"()[]{}#%&+,;=?@<>|\\:*")
}

func obsidianURLEncode(path string) string {
	result := strings.ReplaceAll(path, " ", "%20")
	result = strings.ReplaceAll(result, "'", "%27")
	result = strings.ReplaceAll(result, "\"", "%22")
	result = strings.ReplaceAll(result, "(", "%28")
	result = strings.ReplaceAll(result, ")", "%29")
	result = strings.ReplaceAll(result, "[", "%5B")
	result = strings.ReplaceAll(result, "]", "%5D")
	result = strings.ReplaceAll(result, "{", "%7B")
	result = strings.ReplaceAll(result, "}", "%7D")
	result = strings.ReplaceAll(result, "#", "%23")
	result = strings.ReplaceAll(result, "&", "%26")
	result = strings.ReplaceAll(result, "+", "%2B")
	result = strings.ReplaceAll(result, ",", "%2C")
	result = strings.ReplaceAll(result, ";", "%3B")
	result = strings.ReplaceAll(result, "=", "%3D")
	result = strings.ReplaceAll(result, "?", "%3F")
	result = strings.ReplaceAll(result, "@", "%40")
	result = strings.ReplaceAll(result, "<", "%3C")
	result = strings.ReplaceAll(result, ">", "%3E")
	result = strings.ReplaceAll(result, "|", "%7C")
	result = strings.ReplaceAll(result, "\\", "%5C")
	result = strings.ReplaceAll(result, ":", "%3A")
	result = strings.ReplaceAll(result, "*", "%2A")
	return result
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

// LoadVaultFile loads a single vault file from a path
func LoadVaultFile(path string) (*VaultFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("getting file info: %w", err)
	}

	vf := &VaultFile{
		Path:         path,
		RelativePath: filepath.Base(path),
		Modified:     fileInfo.ModTime(),
		Frontmatter:  make(map[string]interface{}),
	}

	if err := vf.Parse(content); err != nil {
		return nil, fmt.Errorf("parsing file: %w", err)
	}

	return vf, nil
}
