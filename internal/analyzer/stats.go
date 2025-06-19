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
	Path  string  `json:"path"`
	Score float64 `json:"score"`
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
		// Calculate file quality score
		score := a.calculateFileQualityScore(file)
		analysis.FileScores = append(analysis.FileScores, FileQualityScore{
			Path:  file.RelativePath,
			Score: score * 100, // Convert to 0-100 scale
		})

		totalScore += score

		// Categorize score
		switch {
		case score >= 0.9:
			analysis.ScoreDistribution["excellent"]++
		case score >= 0.75:
			analysis.ScoreDistribution["good"]++
		case score >= 0.6:
			analysis.ScoreDistribution["fair"]++
		case score >= 0.4:
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

// calculateFileQualityScore calculates a quality score for an individual file
func (a *Analyzer) calculateFileQualityScore(file *vault.VaultFile) float64 {
	score := 0.0
	maxScore := 0.0

	// Frontmatter presence (20% weight)
	maxScore += 0.2
	if len(file.Frontmatter) > 0 {
		score += 0.2

		// Bonus for essential fields
		if _, hasTitle := file.Frontmatter["title"]; hasTitle {
			score += 0.05
		}
		if _, hasTags := file.Frontmatter["tags"]; hasTags {
			score += 0.05
		}
	}

	// Content length (25% weight)
	maxScore += 0.25
	wordCount := len(strings.Fields(file.Body))
	switch {
	case wordCount >= 500:
		score += 0.25
	case wordCount >= 200:
		score += 0.20
	case wordCount >= 100:
		score += 0.15
	case wordCount >= 50:
		score += 0.10
	case wordCount > 0:
		score += 0.05
	}

	// Structure - headings (20% weight)
	maxScore += 0.2
	if len(file.Headings) > 0 {
		score += 0.15
		// Bonus for proper heading hierarchy
		if len(file.Headings) >= 2 {
			score += 0.05
		}
	}

	// Links and connectivity (20% weight)
	maxScore += 0.2
	if len(file.Links) > 0 {
		score += 0.15
		// Bonus for multiple links
		if len(file.Links) >= 3 {
			score += 0.05
		}
	}

	// Content quality indicators (15% weight)
	maxScore += 0.15
	if len(file.Body) > 0 {
		// Check for code blocks, lists, etc.
		if strings.Contains(file.Body, "```") {
			score += 0.05
		}
		if strings.Contains(file.Body, "- ") || strings.Contains(file.Body, "* ") {
			score += 0.05
		}
		if strings.Count(file.Body, "\n") >= 10 { // Multi-paragraph content
			score += 0.05
		}
	}

	// Normalize score to 0-1 range
	if maxScore > 0 {
		return score / maxScore
	}
	return 0
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
