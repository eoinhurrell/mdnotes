package analyzer

import (
	"crypto/md5"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// Analyzer provides vault analysis capabilities
type Analyzer struct {
	linkParser LinkParser
}

// LinkParser interface for parsing links (to avoid circular imports)
type LinkParser interface {
	UpdateFile(file *vault.VaultFile)
}

// NewAnalyzer creates a new analyzer
func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

// SetLinkParser sets the link parser for the analyzer
func (a *Analyzer) SetLinkParser(parser LinkParser) {
	a.linkParser = parser
}

// VaultStats represents statistics about a vault
type VaultStats struct {
	TotalFiles              int                    `json:"total_files"`
	FilesWithFrontmatter    int                    `json:"files_with_frontmatter"`
	FilesWithoutFrontmatter int                    `json:"files_without_frontmatter"`
	TotalSize               int64                  `json:"total_size"`
	AverageFileSize         float64                `json:"average_file_size"`
	TotalLinks              int                    `json:"total_links"`
	TotalHeadings           int                    `json:"total_headings"`
	TagDistribution         map[string]int         `json:"tag_distribution"`
	FieldPresence           map[string]int         `json:"field_presence"`
	TypeDistribution        map[string]map[string]int `json:"type_distribution"`
	OrphanedFiles           []string               `json:"orphaned_files"`
	DuplicateCount          int                    `json:"duplicate_count"`
	BrokenLinksCount        int                    `json:"broken_links_count"`
	LastModified            time.Time              `json:"last_modified"`
	OldestFile              time.Time              `json:"oldest_file"`
}

// Duplicate represents a set of duplicate values
type Duplicate struct {
	Field string   `json:"field"`
	Value interface{} `json:"value"`
	Files []string `json:"files"`
	Count int      `json:"count"`
}

// ContentDuplicate represents files with duplicate content
type ContentDuplicate struct {
	Hash  string   `json:"hash"`
	Files []string `json:"files"`
	Count int      `json:"count"`
	Size  int      `json:"size"`
}

// DuplicateMatchType defines how to match duplicates
type DuplicateMatchType int

const (
	ExactMatch DuplicateMatchType = iota
	SimilarityMatch
)

// FieldAnalysis represents analysis of a specific field
type FieldAnalysis struct {
	FieldName         string                 `json:"field_name"`
	TotalFiles        int                    `json:"total_files"`
	MissingCount      int                    `json:"missing_count"`
	UniqueValues      int                    `json:"unique_values"`
	ValueDistribution map[interface{}]int    `json:"value_distribution"`
	TypeDistribution  map[string]int         `json:"type_distribution"`
	PredominantType   string                 `json:"predominant_type"`
	Examples          []interface{}          `json:"examples"`
}

// HealthScore represents the overall health of a vault
type HealthScore struct {
	Level       HealthLevel `json:"level"`
	Score       float64     `json:"score"`
	Issues      []string    `json:"issues"`
	Suggestions []string    `json:"suggestions"`
}

// HealthLevel represents different health levels
type HealthLevel string

const (
	Excellent HealthLevel = "excellent"
	Good      HealthLevel = "good"
	Fair      HealthLevel = "fair"
	Poor      HealthLevel = "poor"
	Critical  HealthLevel = "critical"
)

// GenerateStats generates comprehensive statistics for a vault
func (a *Analyzer) GenerateStats(files []*vault.VaultFile) VaultStats {
	stats := VaultStats{
		TotalFiles:       len(files),
		TagDistribution:  make(map[string]int),
		FieldPresence:    make(map[string]int),
		TypeDistribution: make(map[string]map[string]int),
	}

	if len(files) == 0 {
		return stats
	}

	var totalSize int64
	var lastModified, oldestFile time.Time
	firstFile := true

	for _, file := range files {
		// File size and dates
		fileSize := int64(len(file.Content))
		totalSize += fileSize

		if firstFile || file.Modified.After(lastModified) {
			lastModified = file.Modified
		}
		if firstFile || file.Modified.Before(oldestFile) {
			oldestFile = file.Modified
		}
		firstFile = false

		// Frontmatter analysis
		if len(file.Frontmatter) > 0 {
			stats.FilesWithFrontmatter++
			a.analyzeFrontmatter(file.Frontmatter, &stats)
		} else {
			stats.FilesWithoutFrontmatter++
		}

		// Parse links if parser is available
		if a.linkParser != nil {
			a.linkParser.UpdateFile(file)
		}

		// Count links and headings
		stats.TotalLinks += len(file.Links)
		stats.TotalHeadings += len(file.Headings)
	}

	stats.TotalSize = totalSize
	stats.AverageFileSize = float64(totalSize) / float64(len(files))
	stats.LastModified = lastModified
	stats.OldestFile = oldestFile

	// Find orphaned files
	orphaned := a.FindOrphanedFiles(files)
	for _, file := range orphaned {
		stats.OrphanedFiles = append(stats.OrphanedFiles, file.Path)
	}

	return stats
}

// analyzeFrontmatter analyzes frontmatter fields
func (a *Analyzer) analyzeFrontmatter(frontmatter map[string]interface{}, stats *VaultStats) {
	for field, value := range frontmatter {
		stats.FieldPresence[field]++

		// Analyze tags specially
		if field == "tags" {
			tags := a.extractTags(value)
			for _, tag := range tags {
				stats.TagDistribution[tag]++
			}
		}

		// Type distribution
		typeName := a.getTypeName(value)
		if stats.TypeDistribution[field] == nil {
			stats.TypeDistribution[field] = make(map[string]int)
		}
		stats.TypeDistribution[field][typeName]++
	}
}

// extractTags extracts tags from various formats
func (a *Analyzer) extractTags(value interface{}) []string {
	switch v := value.(type) {
	case []interface{}:
		var tags []string
		for _, item := range v {
			if str, ok := item.(string); ok {
				tags = append(tags, str)
			}
		}
		return tags
	case []string:
		return v
	case string:
		if strings.Contains(v, ",") {
			var tags []string
			for _, tag := range strings.Split(v, ",") {
				tags = append(tags, strings.TrimSpace(tag))
			}
			return tags
		}
		return []string{v}
	default:
		return []string{}
	}
}

// getTypeName returns the type name of a value
func (a *Analyzer) getTypeName(value interface{}) string {
	if value == nil {
		return "null"
	}

	switch value.(type) {
	case string:
		return "string"
	case int, int64, float64:
		return "number"
	case bool:
		return "boolean"
	case []interface{}, []string:
		return "array"
	case time.Time:
		return "date"
	default:
		return "object"
	}
}

// FindDuplicates finds duplicate values for a specific field
func (a *Analyzer) FindDuplicates(files []*vault.VaultFile, field string) []Duplicate {
	valueMap := make(map[interface{}][]string)
	originalValues := make(map[interface{}]interface{}) // Keep track of original values

	for _, file := range files {
		if value, exists := file.Frontmatter[field]; exists {
			// Normalize value for comparison
			normalized := a.normalizeValue(value)
			valueMap[normalized] = append(valueMap[normalized], file.Path)
			// Store the first original value we see for this normalized value
			if _, exists := originalValues[normalized]; !exists {
				originalValues[normalized] = value
			}
		}
	}

	var duplicates []Duplicate
	for normalizedValue, paths := range valueMap {
		if len(paths) > 1 {
			duplicates = append(duplicates, Duplicate{
				Field: field,
				Value: originalValues[normalizedValue], // Use original value for display
				Files: paths,
				Count: len(paths),
			})
		}
	}

	// Sort by count descending
	sort.Slice(duplicates, func(i, j int) bool {
		return duplicates[i].Count > duplicates[j].Count
	})

	return duplicates
}

// normalizeValue normalizes values for duplicate detection
func (a *Analyzer) normalizeValue(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(strings.ToLower(v))
	case []interface{}:
		var normalized []string
		for _, item := range v {
			if str, ok := item.(string); ok {
				normalized = append(normalized, strings.TrimSpace(strings.ToLower(str)))
			}
		}
		sort.Strings(normalized)
		return strings.Join(normalized, ",")
	default:
		return v
	}
}

// FindContentDuplicates finds files with duplicate content
func (a *Analyzer) FindContentDuplicates(files []*vault.VaultFile, matchType DuplicateMatchType) []ContentDuplicate {
	switch matchType {
	case ExactMatch:
		return a.findExactContentDuplicates(files)
	case SimilarityMatch:
		return a.findSimilarContentDuplicates(files)
	default:
		return []ContentDuplicate{}
	}
}

// findExactContentDuplicates finds files with identical content
func (a *Analyzer) findExactContentDuplicates(files []*vault.VaultFile) []ContentDuplicate {
	hashMap := make(map[string][]string)

	for _, file := range files {
		// Hash the body content (excluding frontmatter)
		hash := fmt.Sprintf("%x", md5.Sum([]byte(file.Body)))
		hashMap[hash] = append(hashMap[hash], file.Path)
	}

	var duplicates []ContentDuplicate
	for hash, paths := range hashMap {
		if len(paths) > 1 {
			duplicates = append(duplicates, ContentDuplicate{
				Hash:  hash,
				Files: paths,
				Count: len(paths),
				Size:  len(files[0].Body), // Approximate size
			})
		}
	}

	// Sort by count descending
	sort.Slice(duplicates, func(i, j int) bool {
		return duplicates[i].Count > duplicates[j].Count
	})

	return duplicates
}

// findSimilarContentDuplicates finds files with similar content (basic implementation)
func (a *Analyzer) findSimilarContentDuplicates(files []*vault.VaultFile) []ContentDuplicate {
	// This is a simplified similarity check based on common words
	// A more sophisticated implementation would use algorithms like Jaccard similarity
	
	var duplicates []ContentDuplicate
	
	for i, file1 := range files {
		var similarFiles []string
		similarFiles = append(similarFiles, file1.Path)
		
		for j, file2 := range files {
			if i >= j {
				continue
			}
			
			similarity := a.calculateSimilarity(file1.Body, file2.Body)
			if similarity > 0.8 { // 80% similarity threshold
				similarFiles = append(similarFiles, file2.Path)
			}
		}
		
		if len(similarFiles) > 1 {
			duplicates = append(duplicates, ContentDuplicate{
				Hash:  fmt.Sprintf("similar_%d", i),
				Files: similarFiles,
				Count: len(similarFiles),
				Size:  len(file1.Body),
			})
		}
	}
	
	return duplicates
}

// calculateSimilarity calculates basic similarity between two texts
func (a *Analyzer) calculateSimilarity(text1, text2 string) float64 {
	words1 := strings.Fields(strings.ToLower(text1))
	words2 := strings.Fields(strings.ToLower(text2))
	
	if len(words1) == 0 && len(words2) == 0 {
		return 1.0
	}
	
	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}
	
	// Simple Jaccard similarity
	set1 := make(map[string]bool)
	for _, word := range words1 {
		set1[word] = true
	}
	
	set2 := make(map[string]bool)
	for _, word := range words2 {
		set2[word] = true
	}
	
	intersection := 0
	for word := range set1 {
		if set2[word] {
			intersection++
		}
	}
	
	union := len(set1) + len(set2) - intersection
	return float64(intersection) / float64(union)
}

// AnalyzeField performs detailed analysis of a specific field
func (a *Analyzer) AnalyzeField(files []*vault.VaultFile, fieldName string) FieldAnalysis {
	analysis := FieldAnalysis{
		FieldName:         fieldName,
		ValueDistribution: make(map[interface{}]int),
		TypeDistribution:  make(map[string]int),
	}

	var examples []interface{}
	seenValues := make(map[interface{}]bool)
	filesWithField := 0

	for _, file := range files {
		if value, exists := file.Frontmatter[fieldName]; exists {
			filesWithField++

			// Count value occurrences - need to handle unhashable types
			var valueKey interface{}
			switch v := value.(type) {
			case []interface{}:
				// Convert to string for map key
				valueKey = fmt.Sprintf("%v", v)
			default:
				valueKey = value
			}
			analysis.ValueDistribution[valueKey]++

			// Count type occurrences
			typeName := a.getTypeName(value)
			analysis.TypeDistribution[typeName]++

			// Collect examples
			if !seenValues[valueKey] && len(examples) < 5 {
				examples = append(examples, value)
				seenValues[valueKey] = true
			}
		} else {
			analysis.MissingCount++
		}
	}

	analysis.TotalFiles = filesWithField

	analysis.UniqueValues = len(analysis.ValueDistribution)
	analysis.Examples = examples

	// Find predominant type
	maxCount := 0
	for typeName, count := range analysis.TypeDistribution {
		if count > maxCount {
			maxCount = count
			analysis.PredominantType = typeName
		}
	}

	return analysis
}

// FindOrphanedFiles finds files that are not linked by any other files
func (a *Analyzer) FindOrphanedFiles(files []*vault.VaultFile) []*vault.VaultFile {
	// Track which files are referenced by others
	referenced := make(map[string]bool)

	for _, file := range files {
		for _, link := range file.Links {
			target := link.Target
			
			// Normalize target for comparison
			if link.Type == vault.WikiLink {
				// Wiki links can point to files with or without .md extension
				if !strings.HasSuffix(target, ".md") {
					target = target + ".md"
				}
			}
			
			// Don't count self-references
			if target != file.Path {
				referenced[target] = true
			}
		}
	}

	// Find orphaned files (files not referenced by any other file)
	var orphaned []*vault.VaultFile
	for _, file := range files {
		if !referenced[file.Path] {
			orphaned = append(orphaned, file)
		}
	}

	return orphaned
}

// GetHealthScore calculates an overall health score for the vault
func (a *Analyzer) GetHealthScore(stats VaultStats) HealthScore {
	score := 100.0
	var issues []string
	var suggestions []string

	// Penalize missing frontmatter
	if stats.FilesWithoutFrontmatter > 0 {
		penalty := float64(stats.FilesWithoutFrontmatter) / float64(stats.TotalFiles) * 30
		score -= penalty
		issues = append(issues, fmt.Sprintf("%d files missing frontmatter", stats.FilesWithoutFrontmatter))
		suggestions = append(suggestions, "Add frontmatter to files using 'mdnotes frontmatter ensure'")
	}

	// Penalize orphaned files (but only if there are multiple files)
	if len(stats.OrphanedFiles) > 0 && stats.TotalFiles > 1 {
		penalty := float64(len(stats.OrphanedFiles)) / float64(stats.TotalFiles) * 20
		score -= penalty
		issues = append(issues, fmt.Sprintf("%d orphaned files", len(stats.OrphanedFiles)))
		suggestions = append(suggestions, "Review orphaned files and add links to integrate them")
	}

	// Penalize broken links
	if stats.BrokenLinksCount > 0 {
		penalty := float64(stats.BrokenLinksCount) / float64(stats.TotalLinks) * 25
		score -= penalty
		issues = append(issues, fmt.Sprintf("%d broken links", stats.BrokenLinksCount))
		suggestions = append(suggestions, "Fix broken links using 'mdnotes links check'")
	}

	// Penalize duplicates
	if stats.DuplicateCount > 0 {
		penalty := float64(stats.DuplicateCount) * 5
		score -= penalty
		issues = append(issues, fmt.Sprintf("%d duplicate entries", stats.DuplicateCount))
		suggestions = append(suggestions, "Review and resolve duplicate content")
	}

	// Ensure score doesn't go below 0
	if score < 0 {
		score = 0
	}

	// Determine health level
	var level HealthLevel
	switch {
	case score >= 90:
		level = Excellent
	case score >= 75:
		level = Good
	case score >= 60:
		level = Fair
	case score >= 40:
		level = Poor
	default:
		level = Critical
	}

	return HealthScore{
		Level:       level,
		Score:       score,
		Issues:      issues,
		Suggestions: suggestions,
	}
}