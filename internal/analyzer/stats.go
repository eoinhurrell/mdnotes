package analyzer

import (
	"crypto/md5"
	"fmt"
	"regexp"
	"sort"
	"strconv"
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
	TotalFiles              int                       `json:"total_files"`
	FilesWithFrontmatter    int                       `json:"files_with_frontmatter"`
	FilesWithoutFrontmatter int                       `json:"files_without_frontmatter"`
	TotalSize               int64                     `json:"total_size"`
	AverageFileSize         float64                   `json:"average_file_size"`
	TotalLinks              int                       `json:"total_links"`
	TotalHeadings           int                       `json:"total_headings"`
	TagDistribution         map[string]int            `json:"tag_distribution"`
	FieldPresence           map[string]int            `json:"field_presence"`
	TypeDistribution        map[string]map[string]int `json:"type_distribution"`
	OrphanedFiles           []string                  `json:"orphaned_files"`
	DuplicateCount          int                       `json:"duplicate_count"`
	BrokenLinksCount        int                       `json:"broken_links_count"`
	LastModified            time.Time                 `json:"last_modified"`
	OldestFile              time.Time                 `json:"oldest_file"`
}

// Duplicate represents a set of duplicate values
type Duplicate struct {
	Field string      `json:"field"`
	Value interface{} `json:"value"`
	Files []string    `json:"files"`
	Count int         `json:"count"`
}

// ContentDuplicate represents files with duplicate content
type ContentDuplicate struct {
	Hash  string   `json:"hash"`
	Files []string `json:"files"`
	Count int      `json:"count"`
	Size  int      `json:"size"`
}

// SyncConflictFile represents a file that appears to be a sync conflict
type SyncConflictFile struct {
	OriginalFile string `json:"original_file"`
	ConflictFile string `json:"conflict_file"`
	ConflictType string `json:"conflict_type"`
	Pattern      string `json:"pattern"`
}

// ObsidianCopy represents an Obsidian duplicate file
type ObsidianCopy struct {
	OriginalFile string `json:"original_file"`
	CopyFile     string `json:"copy_file"`
	CopyNumber   int    `json:"copy_number"`
}

// DuplicateMatchType defines how to match duplicates
type DuplicateMatchType int

const (
	ExactMatch DuplicateMatchType = iota
	SimilarityMatch
)

// FieldAnalysis represents analysis of a specific field
type FieldAnalysis struct {
	FieldName         string              `json:"field_name"`
	TotalFiles        int                 `json:"total_files"`
	MissingCount      int                 `json:"missing_count"`
	UniqueValues      int                 `json:"unique_values"`
	ValueDistribution map[interface{}]int `json:"value_distribution"`
	TypeDistribution  map[string]int      `json:"type_distribution"`
	PredominantType   string              `json:"predominant_type"`
	Examples          []interface{}       `json:"examples"`
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

// LinkAnalysis represents comprehensive link structure analysis
type LinkAnalysis struct {
	TotalFiles             int                 `json:"total_files"`
	FilesWithOutboundLinks int                 `json:"files_with_outbound_links"`
	FilesWithInboundLinks  int                 `json:"files_with_inbound_links"`
	OrphanedFiles          []string            `json:"orphaned_files"`
	TotalLinks             int                 `json:"total_links"`
	BrokenLinks            int                 `json:"broken_links"`
	AvgOutboundLinks       float64             `json:"avg_outbound_links"`
	AvgInboundLinks        float64             `json:"avg_inbound_links"`
	MostConnectedFile      string              `json:"most_connected_file"`
	MaxConnections         int                 `json:"max_connections"`
	LinkDensity            float64             `json:"link_density"`
	LinkGraph              map[string][]string `json:"link_graph"`
	CentralFiles           []CentralFile       `json:"central_files"`
}

// CentralFile represents a file with its centrality score
type CentralFile struct {
	Path            string  `json:"path"`
	CentralityScore float64 `json:"centrality_score"`
}

// ContentAnalysis represents content quality analysis
type ContentAnalysis struct {
	OverallScore         float64            `json:"overall_score"`
	ScoreDistribution    map[string]int     `json:"score_distribution"`
	AvgContentLength     float64            `json:"avg_content_length"`
	AvgWordCount         float64            `json:"avg_word_count"`
	FilesWithFrontmatter int                `json:"files_with_frontmatter"`
	FilesWithHeadings    int                `json:"files_with_headings"`
	FilesWithLinks       int                `json:"files_with_links"`
	QualityIssues        []string           `json:"quality_issues"`
	Suggestions          []string           `json:"suggestions"`
	FileScores           []FileQualityScore `json:"file_scores"`
}

// FileQualityScore represents the quality score of an individual file
type FileQualityScore struct {
	Path             string  `json:"path"`
	Score            float64 `json:"score"`
	ReadabilityScore float64 `json:"readability_score"`
	LinkDensityScore float64 `json:"link_density_score"`
	CompletenessScore float64 `json:"completeness_score"`
	AtomicityScore   float64 `json:"atomicity_score"`
	RecencyScore     float64 `json:"recency_score"`
	SuggestedFixes   []string `json:"suggested_fixes"`
}

// TrendsAnalysis represents vault growth and trend analysis
type TrendsAnalysis struct {
	StartDate          time.Time           `json:"start_date"`
	EndDate            time.Time           `json:"end_date"`
	TotalDuration      string              `json:"total_duration"`
	TotalFilesCreated  int                 `json:"total_files_created"`
	PeakPeriod         string              `json:"peak_period"`
	PeakFiles          int                 `json:"peak_files"`
	Granularity        string              `json:"granularity"`
	AvgFilesPerPeriod  float64             `json:"avg_files_per_period"`
	GrowthRate         float64             `json:"growth_rate"`
	MostActiveDay      string              `json:"most_active_day"`
	MostActiveMonth    string              `json:"most_active_month"`
	WritingStreak      int                 `json:"writing_streak"`
	ActiveDays         int                 `json:"active_days"`
	TotalDays          int                 `json:"total_days"`
	ActivityPercentage float64             `json:"activity_percentage"`
	Timeline           []TimelinePoint     `json:"timeline"`
	TagTrends          map[string]TagTrend `json:"tag_trends"`
}

// TimelinePoint represents a point in the timeline
type TimelinePoint struct {
	Period string `json:"period"`
	Count  int    `json:"count"`
}

// TagTrend represents trending information for a tag
type TagTrend struct {
	Count      int     `json:"count"`
	GrowthRate float64 `json:"growth_rate"`
}

// InboxAnalysis represents analysis of INBOX sections that need processing
type InboxAnalysis struct {
	TotalSections  int           `json:"total_sections"`
	TotalItems     int           `json:"total_items"`
	TotalSize      int           `json:"total_size"`
	InboxSections  []InboxSection `json:"inbox_sections"`
	Summary        string        `json:"summary"`
}

// InboxSection represents a single INBOX section found in a file
type InboxSection struct {
	File              string   `json:"file"`
	Heading           string   `json:"heading"`
	LineNumber        int      `json:"line_number"`
	ItemCount         int      `json:"item_count"`
	ContentSize       int      `json:"content_size"`
	UrgencyLevel      string   `json:"urgency_level"`
	ActionSuggestions []string `json:"action_suggestions"`
	Content           string   `json:"content"`
}

// AnalyzeLinks performs comprehensive link structure analysis
func (a *Analyzer) AnalyzeLinks(files []*vault.VaultFile) LinkAnalysis {
	analysis := LinkAnalysis{
		TotalFiles:   len(files),
		LinkGraph:    make(map[string][]string),
		CentralFiles: []CentralFile{},
	}

	if len(files) == 0 {
		return analysis
	}

	// Build link graph and collect statistics
	inboundLinks := make(map[string][]string)
	outboundCounts := make(map[string]int)
	totalLinks := 0

	for _, file := range files {
		// Parse links if parser is available
		if a.linkParser != nil {
			a.linkParser.UpdateFile(file)
		}

		// Count outbound links
		if len(file.Links) > 0 {
			analysis.FilesWithOutboundLinks++
			outboundCounts[file.RelativePath] = len(file.Links)
			totalLinks += len(file.Links)

			// Build link graph
			for _, link := range file.Links {
				target := link.Target
				// Normalize target path
				if link.Type == vault.WikiLink && !strings.HasSuffix(target, ".md") {
					target = target + ".md"
				}

				analysis.LinkGraph[file.RelativePath] = append(analysis.LinkGraph[file.RelativePath], target)
				inboundLinks[target] = append(inboundLinks[target], file.RelativePath)
			}
		}
	}

	analysis.TotalLinks = totalLinks
	analysis.FilesWithInboundLinks = len(inboundLinks)

	// Calculate averages
	if len(files) > 0 {
		analysis.AvgOutboundLinks = float64(totalLinks) / float64(len(files))
		analysis.AvgInboundLinks = float64(analysis.FilesWithInboundLinks) / float64(len(files))
		analysis.LinkDensity = float64(totalLinks) / float64(len(files)*len(files))
	}

	// Find most connected file
	maxConnections := 0
	for file, count := range outboundCounts {
		inbound := len(inboundLinks[file])
		totalConnections := count + inbound
		if totalConnections > maxConnections {
			maxConnections = totalConnections
			analysis.MostConnectedFile = file
		}
	}
	analysis.MaxConnections = maxConnections

	// Find orphaned files
	orphaned := a.FindOrphanedFiles(files)
	for _, file := range orphaned {
		analysis.OrphanedFiles = append(analysis.OrphanedFiles, file.RelativePath)
	}

	// Calculate centrality scores
	analysis.CentralFiles = a.calculateCentralityScores(files, inboundLinks, outboundCounts)

	return analysis
}

// calculateCentralityScores calculates centrality scores for files
func (a *Analyzer) calculateCentralityScores(files []*vault.VaultFile, inboundLinks map[string][]string, outboundCounts map[string]int) []CentralFile {
	var centralFiles []CentralFile

	for _, file := range files {
		inbound := len(inboundLinks[file.RelativePath])
		outbound := outboundCounts[file.RelativePath]

		// Simple centrality score: weighted combination of inbound and outbound links
		score := float64(inbound)*0.7 + float64(outbound)*0.3

		if score > 0 {
			centralFiles = append(centralFiles, CentralFile{
				Path:            file.RelativePath,
				CentralityScore: score,
			})
		}
	}

	// Sort by centrality score descending
	sort.Slice(centralFiles, func(i, j int) bool {
		return centralFiles[i].CentralityScore > centralFiles[j].CentralityScore
	})

	return centralFiles
}

// AnalyzeContentQuality performs comprehensive content quality analysis
func (a *Analyzer) AnalyzeContentQuality(files []*vault.VaultFile) ContentAnalysis {
	analysis := ContentAnalysis{
		ScoreDistribution: make(map[string]int),
		QualityIssues:     []string{},
		Suggestions:       []string{},
		FileScores:        []FileQualityScore{},
	}

	if len(files) == 0 {
		return analysis
	}

	var totalContentLength, totalWordCount float64
	var totalScore float64

	// Initialize score distribution
	analysis.ScoreDistribution["excellent"] = 0
	analysis.ScoreDistribution["good"] = 0
	analysis.ScoreDistribution["fair"] = 0
	analysis.ScoreDistribution["poor"] = 0
	analysis.ScoreDistribution["critical"] = 0

	for _, file := range files {
		// Calculate file quality score with detailed breakdown
		overallScore := a.calculateFileQualityScore(file)
		
		// Calculate individual scores for detailed breakdown
		readabilityScore := a.calculateReadabilityScore(file)
		linkDensityScore := a.calculateLinkDensityScore(file)
		completenessScore := a.calculateCompletenessScore(file)
		atomicityScore := a.calculateAtomicityScore(file)
		recencyScore := a.calculateRecencyScore(file)
		
		// Generate suggested fixes
		suggestedFixes := a.generateFileQualityFixes(file, readabilityScore, linkDensityScore, completenessScore, atomicityScore, recencyScore)
		
		analysis.FileScores = append(analysis.FileScores, FileQualityScore{
			Path:             file.RelativePath,
			Score:            overallScore * 100, // Convert to 0-100 scale
			ReadabilityScore: readabilityScore,
			LinkDensityScore: linkDensityScore,
			CompletenessScore: completenessScore,
			AtomicityScore:   atomicityScore,
			RecencyScore:     recencyScore,
			SuggestedFixes:   suggestedFixes,
		})

		totalScore += overallScore

		// Categorize score
		switch {
		case overallScore >= 0.9:
			analysis.ScoreDistribution["excellent"]++
		case overallScore >= 0.75:
			analysis.ScoreDistribution["good"]++
		case overallScore >= 0.6:
			analysis.ScoreDistribution["fair"]++
		case overallScore >= 0.4:
			analysis.ScoreDistribution["poor"]++
		default:
			analysis.ScoreDistribution["critical"]++
		}

		// Content metrics
		contentLength := float64(len(file.Body))
		wordCount := float64(len(strings.Fields(file.Body)))
		totalContentLength += contentLength
		totalWordCount += wordCount

		// Count files with various features
		if len(file.Frontmatter) > 0 {
			analysis.FilesWithFrontmatter++
		}
		if len(file.Headings) > 0 {
			analysis.FilesWithHeadings++
		}
		if len(file.Links) > 0 {
			analysis.FilesWithLinks++
		}
	}

	// Calculate overall metrics
	analysis.OverallScore = (totalScore / float64(len(files))) * 100
	analysis.AvgContentLength = totalContentLength / float64(len(files))
	analysis.AvgWordCount = totalWordCount / float64(len(files))

	// Generate quality issues and suggestions
	analysis.QualityIssues, analysis.Suggestions = a.generateQualityInsights(analysis, len(files))

	// Sort file scores by score descending
	sort.Slice(analysis.FileScores, func(i, j int) bool {
		return analysis.FileScores[i].Score > analysis.FileScores[j].Score
	})

	return analysis
}

// calculateFileQualityScore calculates a Zettelkasten quality score for an individual file
func (a *Analyzer) calculateFileQualityScore(file *vault.VaultFile) float64 {
	// Calculate all five Zettelkasten quality criteria
	readability := a.calculateReadabilityScore(file)
	linkDensity := a.calculateLinkDensityScore(file)
	completeness := a.calculateCompletenessScore(file)
	atomicity := a.calculateAtomicityScore(file)
	recency := a.calculateRecencyScore(file)

	// Weighted average (equal weights for each criterion)
	totalScore := (readability + linkDensity + completeness + atomicity + recency) / 5.0

	return totalScore
}

// CalculateReadabilityScore calculates Flesch-Kincaid Reading Ease score (0.0-1.0)
func (a *Analyzer) CalculateReadabilityScore(file *vault.VaultFile) float64 {
	return a.calculateReadabilityScore(file)
}

// CalculateLinkDensityScore calculates outbound links per 100 words (0.0-1.0)
func (a *Analyzer) CalculateLinkDensityScore(file *vault.VaultFile) float64 {
	return a.calculateLinkDensityScore(file)
}

// CalculateCompletenessScore calculates completeness based on frontmatter and content (0.0-1.0)
func (a *Analyzer) CalculateCompletenessScore(file *vault.VaultFile) float64 {
	return a.calculateCompletenessScore(file)
}

// CalculateAtomicityScore calculates atomicity based on content length and focus (0.0-1.0)
func (a *Analyzer) CalculateAtomicityScore(file *vault.VaultFile) float64 {
	return a.calculateAtomicityScore(file)
}

// CalculateRecencyScore calculates recency based on modification time (0.0-1.0)
func (a *Analyzer) CalculateRecencyScore(file *vault.VaultFile) float64 {
	return a.calculateRecencyScore(file)
}

// calculateReadabilityScore calculates Flesch-Kincaid Reading Ease score (0.0-1.0)
func (a *Analyzer) calculateReadabilityScore(file *vault.VaultFile) float64 {
	if len(file.Body) == 0 {
		return 0.0
	}

	// Extract text for readability analysis
	text := a.extractReadableText(file.Body)
	if len(text) == 0 {
		return 0.0
	}

	// Calculate Flesch-Kincaid Reading Ease
	sentences := a.countSentences(text)
	words := len(strings.Fields(text))
	syllables := a.countSyllables(text)

	if sentences == 0 || words == 0 {
		return 0.0
	}

	// Flesch Reading Ease formula: 206.835 - (1.015 × ASL) - (84.6 × ASW)
	// ASL = Average Sentence Length = words/sentences
	// ASW = Average Syllables per Word = syllables/words
	asl := float64(words) / float64(sentences)
	asw := float64(syllables) / float64(words)
	
	fleschScore := 206.835 - (1.015 * asl) - (84.6 * asw)

	// Convert Flesch score (0-100) to 0-1 scale
	// Scores: 90-100=very easy, 80-89=easy, 70-79=fairly easy, 60-69=standard, 50-59=fairly difficult, 30-49=difficult, 0-29=very difficult
	normalizedScore := fleschScore / 100.0
	if normalizedScore > 1.0 {
		normalizedScore = 1.0
	}
	if normalizedScore < 0.0 {
		normalizedScore = 0.0
	}

	return normalizedScore
}

// calculateLinkDensityScore calculates outbound links per 100 words (0.0-1.0)
func (a *Analyzer) calculateLinkDensityScore(file *vault.VaultFile) float64 {
	wordCount := len(strings.Fields(file.Body))
	if wordCount == 0 {
		return 0.0
	}

	// Count outbound links
	linkCount := len(file.Links)
	linksPer100Words := float64(linkCount) / float64(wordCount) * 100.0

	// Optimal link density for Zettelkasten: 2-5 links per 100 words
	// Score peaks at 3-4 links per 100 words
	var score float64
	switch {
	case linksPer100Words >= 3.0 && linksPer100Words <= 4.0:
		score = 1.0 // Optimal
	case linksPer100Words >= 2.0 && linksPer100Words < 3.0:
		score = 0.8 + (linksPer100Words-2.0)*0.2 // Good
	case linksPer100Words > 4.0 && linksPer100Words <= 6.0:
		score = 1.0 - (linksPer100Words-4.0)*0.1 // Slightly too many
	case linksPer100Words > 6.0:
		score = 0.5 // Too many links, probably not focused
	case linksPer100Words >= 1.0:
		score = linksPer100Words * 0.4 // Some links but could be better
	default:
		score = 0.0 // No links - poor for Zettelkasten
	}

	if score > 1.0 {
		score = 1.0
	}
	return score
}

// calculateCompletenessScore checks for title, summary, and adequate word count (0.0-1.0)
func (a *Analyzer) calculateCompletenessScore(file *vault.VaultFile) float64 {
	score := 0.0

	// Title presence (40% weight)
	if title, hasTitle := file.Frontmatter["title"]; hasTitle {
		if titleStr, ok := title.(string); ok && len(strings.TrimSpace(titleStr)) > 0 {
			score += 0.4
		}
	}

	// Summary/description presence (30% weight) 
	summaryFields := []string{"summary", "description", "abstract", "excerpt"}
	for _, field := range summaryFields {
		if summary, hasSummary := file.Frontmatter[field]; hasSummary {
			if summaryStr, ok := summary.(string); ok && len(strings.TrimSpace(summaryStr)) > 0 {
				score += 0.3
				break
			}
		}
	}

	// Word count adequacy (30% weight)
	wordCount := len(strings.Fields(file.Body))
	switch {
	case wordCount >= 50:
		score += 0.3 // Good length
	case wordCount >= 30:
		score += 0.2 // Acceptable
	case wordCount >= 15:
		score += 0.1 // Minimal
	default:
		// Too short - no points
	}

	return score
}

// calculateAtomicityScore checks if note follows "one concept per note" principle (0.0-1.0)
func (a *Analyzer) calculateAtomicityScore(file *vault.VaultFile) float64 {
	score := 1.0 // Start with perfect score

	// Check word count - notes over 500 words may be too complex
	wordCount := len(strings.Fields(file.Body))
	if wordCount > 500 {
		// Gradually reduce score for longer notes
		penalty := float64(wordCount-500) / 1000.0 // Lose 0.1 for every 100 words over 500
		score -= penalty
	}

	// Check heading count - multiple h1/h2 headings suggest multiple concepts
	h1Count := 0
	h2Count := 0
	for _, heading := range file.Headings {
		if heading.Level == 1 {
			h1Count++
		} else if heading.Level == 2 {
			h2Count++
		}
	}

	// Penalize multiple top-level concepts
	if h1Count > 1 {
		score -= float64(h1Count-1) * 0.3 // Heavy penalty for multiple H1s
	}
	if h2Count > 3 {
		score -= float64(h2Count-3) * 0.1 // Lighter penalty for many H2s
	}

	// Check for topic coherence by examining repeated terms
	// This is a simple heuristic - more sophisticated NLP could be used
	if wordCount > 0 {
		topicCoherence := a.calculateTopicCoherence(file.Body)
		score = score * topicCoherence // Multiply by coherence factor
	}

	if score < 0.0 {
		score = 0.0
	}
	return score
}

// calculateRecencyScore penalizes old untouched notes (0.0-1.0)
func (a *Analyzer) calculateRecencyScore(file *vault.VaultFile) float64 {
	now := time.Now()
	daysSinceModified := now.Sub(file.Modified).Hours() / 24

	// Scoring based on how recently the file was modified
	switch {
	case daysSinceModified <= 7:
		return 1.0 // Perfect - modified within a week
	case daysSinceModified <= 30:
		return 0.9 // Excellent - modified within a month
	case daysSinceModified <= 90:
		return 0.7 // Good - modified within 3 months
	case daysSinceModified <= 150:
		return 0.5 // Fair - modified within 5 months
	case daysSinceModified <= 365:
		return 0.3 // Poor - modified within a year
	default:
		return 0.1 // Very old - over a year since modification
	}
}

// Helper functions for readability analysis

// extractReadableText removes markdown formatting for readability analysis
func (a *Analyzer) extractReadableText(markdown string) string {
	// Remove code blocks
	codeBlockRegex := regexp.MustCompile("```[\\s\\S]*?```")
	text := codeBlockRegex.ReplaceAllString(markdown, "")
	
	// Remove inline code
	inlineCodeRegex := regexp.MustCompile("`[^`]+`")
	text = inlineCodeRegex.ReplaceAllString(text, "")
	
	// Remove links but keep text
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	text = linkRegex.ReplaceAllString(text, "$1")
	
	// Remove wiki links but keep text
	wikiLinkRegex := regexp.MustCompile(`\[\[([^|\]]+)(\|[^\]]+)?\]\]`)
	text = wikiLinkRegex.ReplaceAllString(text, "$1")
	
	// Remove headings markers
	headingRegex := regexp.MustCompile(`^#+\s*`)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = headingRegex.ReplaceAllString(line, "")
	}
	text = strings.Join(lines, "\n")
	
	// Remove list markers
	listRegex := regexp.MustCompile(`^(\s*[-*+]\s*|\s*\d+\.\s*)`)
	lines = strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = listRegex.ReplaceAllString(line, "")
	}
	
	return strings.Join(lines, "\n")
}

// countSentences counts sentences in text
func (a *Analyzer) countSentences(text string) int {
	// Simple sentence counting based on sentence-ending punctuation
	sentenceRegex := regexp.MustCompile(`[.!?]+`)
	matches := sentenceRegex.FindAllString(text, -1)
	count := len(matches)
	
	// Ensure at least 1 sentence if there's text
	if count == 0 && len(strings.TrimSpace(text)) > 0 {
		count = 1
	}
	
	return count
}

// countSyllables estimates syllable count for words
func (a *Analyzer) countSyllables(text string) int {
	words := strings.Fields(strings.ToLower(text))
	totalSyllables := 0
	
	for _, word := range words {
		syllables := a.estimateSyllables(word)
		totalSyllables += syllables
	}
	
	return totalSyllables
}

// estimateSyllables estimates syllables in a single word using simple heuristics
func (a *Analyzer) estimateSyllables(word string) int {
	if len(word) == 0 {
		return 0
	}
	
	// Remove punctuation
	wordRegex := regexp.MustCompile(`[^a-z]`)
	cleanWord := wordRegex.ReplaceAllString(word, "")
	
	if len(cleanWord) == 0 {
		return 1
	}
	
	// Count vowel groups
	vowelRegex := regexp.MustCompile(`[aeiouy]+`)
	vowelGroups := vowelRegex.FindAllString(cleanWord, -1)
	syllables := len(vowelGroups)
	
	// Adjust for silent 'e' at the end
	if strings.HasSuffix(cleanWord, "e") && syllables > 1 {
		syllables--
	}
	
	// Ensure at least 1 syllable
	if syllables == 0 {
		syllables = 1
	}
	
	return syllables
}

// calculateTopicCoherence estimates how focused the content is on a single topic
func (a *Analyzer) calculateTopicCoherence(text string) float64 {
	words := strings.Fields(strings.ToLower(text))
	if len(words) < 10 {
		return 1.0 // Short text is assumed coherent
	}
	
	// Count word frequencies
	wordFreq := make(map[string]int)
	for _, word := range words {
		// Skip very short words and common words
		if len(word) >= 4 && !a.isCommonWord(word) {
			wordFreq[word]++
		}
	}
	
	if len(wordFreq) == 0 {
		return 0.5 // Neutral if no significant words
	}
	
	// Calculate the proportion of total content covered by top words
	totalSignificantWords := 0
	for _, count := range wordFreq {
		totalSignificantWords += count
	}
	
	// Find top 5 most frequent words
	type wordCount struct {
		word  string
		count int
	}
	
	var wordCounts []wordCount
	for word, count := range wordFreq {
		wordCounts = append(wordCounts, wordCount{word, count})
	}
	
	sort.Slice(wordCounts, func(i, j int) bool {
		return wordCounts[i].count > wordCounts[j].count
	})
	
	// Calculate coherence based on how much content is covered by top words
	topWordsCount := 0
	maxWords := 5
	if len(wordCounts) < maxWords {
		maxWords = len(wordCounts)
	}
	
	for i := 0; i < maxWords; i++ {
		topWordsCount += wordCounts[i].count
	}
	
	coherence := float64(topWordsCount) / float64(totalSignificantWords)
	
	// Scale coherence to be more reasonable (0.5 to 1.0 range typically)
	coherence = 0.5 + (coherence * 0.5)
	if coherence > 1.0 {
		coherence = 1.0
	}
	
	return coherence
}

// isCommonWord checks if a word is a common English word that shouldn't count for topic coherence
func (a *Analyzer) isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"that": true, "with": true, "have": true, "this": true, "will": true,
		"your": true, "from": true, "they": true, "know": true, "want": true,
		"been": true, "good": true, "much": true, "some": true, "time": true,
		"very": true, "when": true, "come": true, "here": true, "just": true,
		"like": true, "long": true, "make": true, "many": true, "over": true,
		"such": true, "take": true, "than": true, "them": true, "well": true,
		"were": true, "also": true, "back": true, "call": true, "came": true,
		"each": true, "find": true, "give": true, "hand": true, "high": true,
		"keep": true, "last": true, "left": true, "life": true, "live": true,
		"look": true, "made": true, "most": true, "move": true, "must": true,
		"name": true, "need": true, "next": true, "open": true, "part": true,
		"play": true, "said": true, "same": true, "seem": true, "show": true,
		"side": true, "tell": true, "turn": true, "used": true, "ways": true,
		"week": true, "went": true, "what": true, "work": true, "year": true, 
		"years": true, "about": true, "after": true, "again": true, "before": true, 
		"being": true, "could": true, "every": true, "first": true, "found": true, 
		"great": true, "group": true, "might": true, "never": true, "often": true, 
		"other": true, "place": true, "right": true, "should": true, "small": true, 
		"still": true, "their": true, "there": true, "these": true, "think": true, 
		"three": true, "through": true, "under": true, "until": true, "water": true, 
		"where": true, "which": true, "while": true, "world": true, "would": true, 
		"write": true, "young": true,
	}
	
	return commonWords[word]
}

// generateFileQualityFixes generates specific improvement suggestions for a file
func (a *Analyzer) generateFileQualityFixes(file *vault.VaultFile, readability, linkDensity, completeness, atomicity, recency float64) []string {
	var fixes []string
	
	// Readability fixes
	if readability < 0.4 {
		fixes = append(fixes, "Simplify sentence structure for better readability")
		fixes = append(fixes, "Use shorter sentences and common vocabulary")
	}
	
	// Link density fixes
	if linkDensity < 0.3 {
		fixes = append(fixes, "Add more links to related concepts (aim for 2-4 links per 100 words)")
	} else if linkDensity < 0.6 {
		fixes = append(fixes, "Consider adding a few more relevant links")
	}
	
	// Completeness fixes
	if completeness < 0.7 {
		if _, hasTitle := file.Frontmatter["title"]; !hasTitle {
			fixes = append(fixes, "Add a descriptive title in frontmatter")
		}
		
		summaryFields := []string{"summary", "description", "abstract", "excerpt"}
		hasSummary := false
		for _, field := range summaryFields {
			if _, exists := file.Frontmatter[field]; exists {
				hasSummary = true
				break
			}
		}
		if !hasSummary {
			fixes = append(fixes, "Add a summary or description in frontmatter")
		}
		
		wordCount := len(strings.Fields(file.Body))
		if wordCount < 50 {
			fixes = append(fixes, "Expand content - add more detail and context")
		}
	}
	
	// Atomicity fixes
	if atomicity < 0.6 {
		wordCount := len(strings.Fields(file.Body))
		if wordCount > 500 {
			fixes = append(fixes, "Consider breaking this into smaller, more focused notes")
		}
		
		h1Count := 0
		for _, heading := range file.Headings {
			if heading.Level == 1 {
				h1Count++
			}
		}
		if h1Count > 1 {
			fixes = append(fixes, "Split multiple main topics into separate notes")
		}
	}
	
	// Recency fixes
	if recency < 0.5 {
		fixes = append(fixes, "Review and update this note - it hasn't been modified recently")
		fixes = append(fixes, "Add current date to track when content was last reviewed")
	}
	
	// General suggestions based on overall quality
	if len(fixes) == 0 {
		fixes = append(fixes, "This note has good quality - consider linking it to related concepts")
	}
	
	return fixes
}

// generateQualityInsights generates quality issues and suggestions
func (a *Analyzer) generateQualityInsights(analysis ContentAnalysis, totalFiles int) ([]string, []string) {
	var issues, suggestions []string

	// Check for common quality issues
	if analysis.FilesWithFrontmatter < totalFiles/2 {
		issues = append(issues, fmt.Sprintf("%.0f%% of files lack frontmatter", float64(totalFiles-analysis.FilesWithFrontmatter)/float64(totalFiles)*100))
		suggestions = append(suggestions, "Add frontmatter to files using 'mdnotes frontmatter ensure'")
	}

	if analysis.FilesWithHeadings < totalFiles/3 {
		issues = append(issues, fmt.Sprintf("%.0f%% of files lack heading structure", float64(totalFiles-analysis.FilesWithHeadings)/float64(totalFiles)*100))
		suggestions = append(suggestions, "Add headings to improve content structure")
	}

	if analysis.FilesWithLinks < totalFiles/4 {
		issues = append(issues, "Low interconnectivity between files")
		suggestions = append(suggestions, "Add more links between related content")
	}

	if analysis.AvgWordCount < 100 {
		issues = append(issues, "Many files have very short content")
		suggestions = append(suggestions, "Consider expanding content or combining related short files")
	}

	// Score-based insights
	criticalFiles := analysis.ScoreDistribution["critical"] + analysis.ScoreDistribution["poor"]
	if criticalFiles > totalFiles/4 {
		issues = append(issues, fmt.Sprintf("%d files have poor quality scores", criticalFiles))
		suggestions = append(suggestions, "Focus on improving content structure and completeness")
	}

	return issues, suggestions
}

// AnalyzeTrends performs vault growth and trend analysis
func (a *Analyzer) AnalyzeTrends(files []*vault.VaultFile, timespan, granularity string) TrendsAnalysis {
	analysis := TrendsAnalysis{
		Granularity: granularity,
		Timeline:    []TimelinePoint{},
		TagTrends:   make(map[string]TagTrend),
	}

	if len(files) == 0 {
		return analysis
	}

	// Parse timespan and find date range
	endDate := time.Now()
	startDate := a.parseTimespan(timespan, endDate)

	analysis.StartDate = startDate
	analysis.EndDate = endDate
	analysis.TotalDuration = endDate.Sub(startDate).String()

	// Filter files within timespan and collect data
	var filesInRange []*vault.VaultFile
	dayActivity := make(map[string]int)
	monthActivity := make(map[string]int)
	periodActivity := make(map[string]int)
	tagFrequency := make(map[string]int)

	for _, file := range files {
		if file.Modified.After(startDate) && file.Modified.Before(endDate) {
			filesInRange = append(filesInRange, file)

			// Track daily activity
			dayKey := file.Modified.Format("2006-01-02")
			dayActivity[dayKey]++

			// Track monthly activity
			monthKey := file.Modified.Format("2006-01")
			monthActivity[monthKey]++

			// Track period activity based on granularity
			periodKey := a.formatPeriod(file.Modified, granularity)
			periodActivity[periodKey]++

			// Track tag trends
			if tags, exists := file.Frontmatter["tags"]; exists {
				extractedTags := a.extractTags(tags)
				for _, tag := range extractedTags {
					tagFrequency[tag]++
				}
			}
		}
	}

	analysis.TotalFilesCreated = len(filesInRange)

	// Calculate growth metrics
	totalDays := int(endDate.Sub(startDate).Hours() / 24)
	analysis.TotalDays = totalDays
	analysis.ActiveDays = len(dayActivity)
	if totalDays > 0 {
		analysis.ActivityPercentage = float64(analysis.ActiveDays) / float64(totalDays) * 100
	}

	// Find peak period and most active periods
	maxFiles := 0
	for period, count := range periodActivity {
		if count > maxFiles {
			maxFiles = count
			analysis.PeakPeriod = period
		}
	}
	analysis.PeakFiles = maxFiles

	// Calculate averages
	periods := len(periodActivity)
	if periods > 0 {
		analysis.AvgFilesPerPeriod = float64(analysis.TotalFilesCreated) / float64(periods)
		analysis.GrowthRate = (float64(analysis.TotalFilesCreated) / float64(periods)) * 100 // Simplified growth rate
	}

	// Find most active day and month
	analysis.MostActiveDay = a.findMostActive(dayActivity)
	analysis.MostActiveMonth = a.findMostActive(monthActivity)

	// Calculate writing streak
	analysis.WritingStreak = a.calculateWritingStreak(dayActivity, endDate)

	// Build timeline
	analysis.Timeline = a.buildTimeline(periodActivity, granularity)

	// Build tag trends
	for tag, count := range tagFrequency {
		analysis.TagTrends[tag] = TagTrend{
			Count:      count,
			GrowthRate: float64(count) / float64(analysis.TotalFilesCreated) * 100,
		}
	}

	return analysis
}

// Helper methods for trend analysis

func (a *Analyzer) parseTimespan(timespan string, endDate time.Time) time.Time {
	switch timespan {
	case "1w":
		return endDate.AddDate(0, 0, -7)
	case "1m":
		return endDate.AddDate(0, -1, 0)
	case "3m":
		return endDate.AddDate(0, -3, 0)
	case "6m":
		return endDate.AddDate(0, -6, 0)
	case "1y":
		return endDate.AddDate(-1, 0, 0)
	case "all":
		return time.Time{} // Beginning of time
	default:
		return endDate.AddDate(-1, 0, 0) // Default to 1 year
	}
}

func (a *Analyzer) formatPeriod(date time.Time, granularity string) string {
	switch granularity {
	case "day":
		return date.Format("2006-01-02")
	case "week":
		year, week := date.ISOWeek()
		return fmt.Sprintf("%d-W%02d", year, week)
	case "month":
		return date.Format("2006-01")
	case "quarter":
		quarter := (date.Month()-1)/3 + 1
		return fmt.Sprintf("%d-Q%d", date.Year(), quarter)
	default:
		return date.Format("2006-01") // Default to month
	}
}

func (a *Analyzer) findMostActive(activity map[string]int) string {
	maxCount := 0
	mostActive := ""
	for period, count := range activity {
		if count > maxCount {
			maxCount = count
			mostActive = period
		}
	}
	return mostActive
}

func (a *Analyzer) calculateWritingStreak(dayActivity map[string]int, endDate time.Time) int {
	streak := 0
	currentDate := endDate

	for i := 0; i < 365; i++ { // Check up to 365 days back
		dayKey := currentDate.Format("2006-01-02")
		if dayActivity[dayKey] > 0 {
			streak++
		} else {
			break
		}
		currentDate = currentDate.AddDate(0, 0, -1)
	}

	return streak
}

func (a *Analyzer) buildTimeline(periodActivity map[string]int, granularity string) []TimelinePoint {
	var timeline []TimelinePoint

	for period, count := range periodActivity {
		timeline = append(timeline, TimelinePoint{
			Period: period,
			Count:  count,
		})
	}

	// Sort timeline by period
	sort.Slice(timeline, func(i, j int) bool {
		return timeline[i].Period > timeline[j].Period // Most recent first
	})

	return timeline
}

// FindObsidianCopies finds Obsidian duplicate files (with ' 1', ' 2', etc. suffixes)
func (a *Analyzer) FindObsidianCopies(files []*vault.VaultFile) []ObsidianCopy {
	var copies []ObsidianCopy
	baseFiles := make(map[string]*vault.VaultFile)

	// First pass: identify base files and potential copies
	for _, file := range files {
		filename := strings.TrimSuffix(file.RelativePath, ".md")
		
		// Check if this is a copy (ends with ' 1', ' 2', etc.)
		re := regexp.MustCompile(`^(.+) (\d+)$`)
		matches := re.FindStringSubmatch(filename)
		
		if len(matches) == 3 {
			// This is a copy
			baseFilename := matches[1]
			copyNumber, _ := strconv.Atoi(matches[2])
			originalPath := baseFilename + ".md"
			
			// Find the original file
			for _, originalFile := range files {
				if originalFile.RelativePath == originalPath {
					copies = append(copies, ObsidianCopy{
						OriginalFile: originalFile.RelativePath,
						CopyFile:     file.RelativePath,
						CopyNumber:   copyNumber,
					})
					break
				}
			}
		} else {
			// This is a potential base file
			baseFiles[filename] = file
		}
	}

	// Sort by copy number
	sort.Slice(copies, func(i, j int) bool {
		if copies[i].OriginalFile == copies[j].OriginalFile {
			return copies[i].CopyNumber < copies[j].CopyNumber
		}
		return copies[i].OriginalFile < copies[j].OriginalFile
	})

	return copies
}

// FindSyncConflictFiles finds files that appear to be sync conflicts
func (a *Analyzer) FindSyncConflictFiles(files []*vault.VaultFile) []SyncConflictFile {
	var conflicts []SyncConflictFile
	
	// Patterns for different sync conflict types
	patterns := []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{"syncthing", regexp.MustCompile(`^(.+)\.sync-conflict-\d{8}-\d{6}-[A-Z0-9]{8}\.md$`)},
		{"dropbox", regexp.MustCompile(`^(.+) \(.*'s conflicted copy \d{4}-\d{2}-\d{2}\)\.md$`)},
		{"onedrive", regexp.MustCompile(`^(.+)-[^-]+-OneDrive\.md$`)},
		{"google-drive", regexp.MustCompile(`^(.+) \(\d+\)\.md$`)},
		{"icloud", regexp.MustCompile(`^(.+) \d+\.md$`)},
	}

	for _, file := range files {
		for _, pattern := range patterns {
			matches := pattern.pattern.FindStringSubmatch(file.RelativePath)
			if len(matches) >= 2 {
				originalPath := matches[1] + ".md"
				
				// Check if original file exists
				for _, originalFile := range files {
					if originalFile.RelativePath == originalPath {
						conflicts = append(conflicts, SyncConflictFile{
							OriginalFile: originalFile.RelativePath,
							ConflictFile: file.RelativePath,
							ConflictType: pattern.name,
							Pattern:      pattern.pattern.String(),
						})
						break
					}
				}
				break // Found a match, no need to check other patterns
			}
		}
	}

	// Sort by original file name
	sort.Slice(conflicts, func(i, j int) bool {
		return conflicts[i].OriginalFile < conflicts[j].OriginalFile
	})

	return conflicts
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

// AnalyzeInbox analyzes INBOX sections and pending content that needs processing
func (a *Analyzer) AnalyzeInbox(files []*vault.VaultFile, inboxHeadings []string, sortBy string, minItems int) *InboxAnalysis {
	analysis := &InboxAnalysis{
		InboxSections: []InboxSection{},
	}

	// If no inbox headings provided, use default
	if len(inboxHeadings) == 0 {
		inboxHeadings = []string{"INBOX"}
	}

	// Build patterns for specified inbox headings
	var inboxPatterns []*regexp.Regexp
	for _, heading := range inboxHeadings {
		// Escape special regex characters and create pattern
		escaped := regexp.QuoteMeta(heading)
		pattern := fmt.Sprintf(`(?i)^#+ ?%s(\s|$)`, escaped)
		inboxPatterns = append(inboxPatterns, regexp.MustCompile(pattern))
	}

	totalItems := 0
	totalSize := 0

	for _, file := range files {
		sections := a.findInboxSections(file, inboxPatterns, minItems)
		for _, section := range sections {
			totalItems += section.ItemCount
			totalSize += section.ContentSize
			analysis.InboxSections = append(analysis.InboxSections, section)
		}
	}

	// Sort sections based on the sortBy parameter
	a.sortInboxSections(analysis.InboxSections, sortBy)

	analysis.TotalSections = len(analysis.InboxSections)
	analysis.TotalItems = totalItems
	analysis.TotalSize = totalSize

	// Generate summary
	if len(analysis.InboxSections) == 0 {
		analysis.Summary = "No INBOX sections found - vault appears well-organized!"
	} else {
		analysis.Summary = fmt.Sprintf("Found %d INBOX sections with %d items (%d chars) requiring attention",
			analysis.TotalSections, analysis.TotalItems, analysis.TotalSize)
	}

	return analysis
}

// findInboxSections finds INBOX-like sections within a file
func (a *Analyzer) findInboxSections(file *vault.VaultFile, patterns []*regexp.Regexp, minItems int) []InboxSection {
	var sections []InboxSection
	
	lines := strings.Split(file.Body, "\n")
	var currentSection *InboxSection
	var sectionContent strings.Builder

	for lineNum, line := range lines {
		// Check if this line is an INBOX heading
		isInboxHeading := false
		for _, pattern := range patterns {
			if pattern.MatchString(line) {
				// Finish previous section if exists
				if currentSection != nil {
					content := sectionContent.String()
					itemCount := a.countItems(content)
					if itemCount >= minItems {
						currentSection.Content = content
						currentSection.ItemCount = itemCount
						currentSection.ContentSize = len(content)
						currentSection.UrgencyLevel = a.assessUrgency(content, currentSection.Heading)
						currentSection.ActionSuggestions = a.generateActionSuggestions(content, itemCount)
						sections = append(sections, *currentSection)
					}
				}

				// Start new section
				currentSection = &InboxSection{
					File:       file.Path,
					Heading:    strings.TrimSpace(line),
					LineNumber: lineNum + 1,
				}
				sectionContent.Reset()
				isInboxHeading = true
				break
			}
		}

		// If we're in an INBOX section, collect content until next heading
		if currentSection != nil && !isInboxHeading {
			// Stop if we hit another heading (not INBOX)
			if strings.HasPrefix(strings.TrimSpace(line), "#") {
				content := sectionContent.String()
				itemCount := a.countItems(content)
				if itemCount >= minItems {
					currentSection.Content = content
					currentSection.ItemCount = itemCount
					currentSection.ContentSize = len(content)
					currentSection.UrgencyLevel = a.assessUrgency(content, currentSection.Heading)
					currentSection.ActionSuggestions = a.generateActionSuggestions(content, itemCount)
					sections = append(sections, *currentSection)
				}
				currentSection = nil
			} else {
				sectionContent.WriteString(line + "\n")
			}
		}
	}

	// Handle last section if exists
	if currentSection != nil {
		content := sectionContent.String()
		itemCount := a.countItems(content)
		if itemCount >= minItems {
			currentSection.Content = content
			currentSection.ItemCount = itemCount
			currentSection.ContentSize = len(content)
			currentSection.UrgencyLevel = a.assessUrgency(content, currentSection.Heading)
			currentSection.ActionSuggestions = a.generateActionSuggestions(content, itemCount)
			sections = append(sections, *currentSection)
		}
	}

	return sections
}

// countItems counts the number of actionable items in the content
func (a *Analyzer) countItems(content string) int {
	lines := strings.Split(content, "\n")
	itemCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Count bullet points, numbered lists, checkboxes, etc.
		if strings.HasPrefix(trimmed, "-") ||
			strings.HasPrefix(trimmed, "*") ||
			strings.HasPrefix(trimmed, "+") ||
			strings.HasPrefix(trimmed, "- [ ]") ||
			strings.HasPrefix(trimmed, "- [x]") ||
			strings.HasPrefix(trimmed, "* [ ]") ||
			strings.HasPrefix(trimmed, "* [x]") ||
			regexp.MustCompile(`^\d+\.`).MatchString(trimmed) {
			if len(trimmed) > 3 { // Avoid counting empty bullets
				itemCount++
			}
		}
	}

	// If no structured items found, count non-empty lines as potential items
	if itemCount == 0 {
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				itemCount++
			}
		}
	}

	return itemCount
}

// assessUrgency determines the urgency level based on content and heading
func (a *Analyzer) assessUrgency(content, heading string) string {
	lowerContent := strings.ToLower(content)
	lowerHeading := strings.ToLower(heading)

	// High urgency indicators
	urgentKeywords := []string{"urgent", "asap", "deadline", "emergency", "critical", "priority", "due"}
	for _, keyword := range urgentKeywords {
		if strings.Contains(lowerContent, keyword) || strings.Contains(lowerHeading, keyword) {
			return "High"
		}
	}

	// Medium urgency indicators
	mediumKeywords := []string{"todo", "pending", "waiting", "review", "process"}
	for _, keyword := range mediumKeywords {
		if strings.Contains(lowerHeading, keyword) {
			return "Medium"
		}
	}

	// Check for dates that might indicate urgency
	datePattern := regexp.MustCompile(`\d{4}-\d{2}-\d{2}|\d{1,2}/\d{1,2}/\d{4}`)
	if datePattern.MatchString(content) {
		return "Medium"
	}

	return "Low"
}

// generateActionSuggestions provides actionable suggestions based on content analysis
func (a *Analyzer) generateActionSuggestions(content string, itemCount int) []string {
	var suggestions []string
	lowerContent := strings.ToLower(content)

	if itemCount > 10 {
		suggestions = append(suggestions, "Break down into smaller tasks")
	}

	if itemCount > 5 {
		suggestions = append(suggestions, "Prioritize by urgency")
	}

	if strings.Contains(lowerContent, "link") || strings.Contains(lowerContent, "url") {
		suggestions = append(suggestions, "Process links with linkding sync")
	}

	if strings.Contains(lowerContent, "note") || strings.Contains(lowerContent, "idea") {
		suggestions = append(suggestions, "Convert to permanent notes")
	}

	if strings.Contains(lowerContent, "book") || strings.Contains(lowerContent, "article") {
		suggestions = append(suggestions, "Add to reading list")
	}

	if len(suggestions) == 0 {
		suggestions = append(suggestions, "Review and organize content")
	}

	return suggestions
}

// sortInboxSections sorts inbox sections based on the specified criteria
func (a *Analyzer) sortInboxSections(sections []InboxSection, sortBy string) {
	switch sortBy {
	case "size":
		sort.Slice(sections, func(i, j int) bool {
			return sections[i].ContentSize > sections[j].ContentSize
		})
	case "count":
		sort.Slice(sections, func(i, j int) bool {
			return sections[i].ItemCount > sections[j].ItemCount
		})
	case "urgency":
		sort.Slice(sections, func(i, j int) bool {
			urgencyOrder := map[string]int{"High": 3, "Medium": 2, "Low": 1}
			return urgencyOrder[sections[i].UrgencyLevel] > urgencyOrder[sections[j].UrgencyLevel]
		})
	default: // Default to size
		sort.Slice(sections, func(i, j int) bool {
			return sections[i].ContentSize > sections[j].ContentSize
		})
	}
}
