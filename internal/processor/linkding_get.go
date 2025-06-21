package processor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/eoinhurrell/mdnotes/internal/linkding"
	"golang.org/x/net/html"
)

// LinkdingGetConfig configures the Linkding get processor
type LinkdingGetConfig struct {
	MaxSize uint64 // Maximum bytes to fetch from live URL
	Timeout string // Request timeout
	TmpDir  string // Temporary directory for downloads
	Verbose bool   // Verbose output
}

// LinkdingGetProcessor handles retrieving content from Linkding snapshots or live URLs
type LinkdingGetProcessor struct {
	config LinkdingGetConfig
	client *linkding.Client
}

// NewLinkdingGet creates a new Linkding get processor
func NewLinkdingGet(config LinkdingGetConfig) *LinkdingGetProcessor {
	return &LinkdingGetProcessor{
		config: config,
	}
}

// SetClient sets the Linkding API client
func (p *LinkdingGetProcessor) SetClient(client *linkding.Client) {
	p.client = client
}

// GetContent retrieves content from either a snapshot or live URL
func (p *LinkdingGetProcessor) GetContent(ctx context.Context, linkdingID interface{}, fallbackURL interface{}) (string, error) {
	// Try to get content from snapshot first if linkding_id is provided
	if linkdingID != nil {
		bookmarkID, err := p.convertToInt(linkdingID)
		if err == nil && bookmarkID > 0 {
			if p.config.Verbose {
				fmt.Fprintf(os.Stderr, "Querying Linkding API for bookmark %d assets...\n", bookmarkID)
			}

			text, err := p.getContentFromSnapshot(ctx, bookmarkID)
			if err == nil {
				if p.config.Verbose {
					fmt.Fprintf(os.Stderr, "Successfully retrieved content from snapshot\n")
				}
				return text, nil
			}

			if p.config.Verbose {
				fmt.Fprintf(os.Stderr, "Snapshot retrieval failed: %v\n", err)
				fmt.Fprintf(os.Stderr, "Falling back to live URL...\n")
			}
		}
	}

	// Fall back to live URL if snapshot is not available
	if fallbackURL != nil {
		if urlStr, ok := fallbackURL.(string); ok && urlStr != "" {
			if p.config.Verbose {
				fmt.Fprintf(os.Stderr, "Fetching content from live URL: %s\n", urlStr)
			}

			text, err := p.getContentFromLiveURL(ctx, urlStr)
			if err != nil {
				return "", fmt.Errorf("failed to get content from live URL: %w", err)
			}

			if p.config.Verbose {
				fmt.Fprintf(os.Stderr, "Successfully retrieved content from live URL\n")
			}
			return text, nil
		}
	}

	return "", fmt.Errorf("no valid linkding_id or url available")
}

// getContentFromSnapshot retrieves and extracts text from a Linkding snapshot
func (p *LinkdingGetProcessor) getContentFromSnapshot(ctx context.Context, bookmarkID int) (string, error) {
	// List assets for the bookmark
	assets, err := p.client.ListAssets(ctx, bookmarkID)
	if err != nil {
		return "", fmt.Errorf("listing assets: %w", err)
	}

	// Pick the latest snapshot
	snapshot, err := PickLatestSnapshot(assets)
	if err != nil {
		return "", fmt.Errorf("selecting snapshot: %w", err)
	}

	// Create temporary file for download
	tmpDir := p.config.TmpDir
	if tmpDir == "" {
		tmpDir = os.TempDir()
	}

	tmpFile, err := os.CreateTemp(tmpDir, "mdnotes-snapshot-*.html")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name()) // Clean up
	tmpFile.Close()                 // Close file so download can write to it

	// Download the snapshot
	if p.config.Verbose {
		fmt.Fprintf(os.Stderr, "Downloading snapshot asset %d to %s...\n", snapshot.ID, tmpFile.Name())
	}

	if err := p.client.DownloadAsset(ctx, bookmarkID, snapshot.ID, tmpFile.Name()); err != nil {
		return "", fmt.Errorf("downloading asset: %w", err)
	}

	// Extract text from HTML
	text, err := ExtractTextFromHTML(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("extracting text from HTML: %w", err)
	}

	return text, nil
}

// getContentFromLiveURL fetches and extracts text from a live URL
func (p *LinkdingGetProcessor) getContentFromLiveURL(ctx context.Context, url string) (string, error) {
	// Parse timeout
	timeout := 10 * time.Second
	if p.config.Timeout != "" {
		if parsed, err := time.ParseDuration(p.config.Timeout); err == nil {
			timeout = parsed
		}
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: timeout,
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	// Set user agent
	req.Header.Set("User-Agent", "mdnotes/1.0")

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching URL: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Check content length if provided
	if resp.ContentLength > 0 && uint64(resp.ContentLength) > p.config.MaxSize {
		return "", fmt.Errorf("content size %d bytes exceeds limit %d bytes", resp.ContentLength, p.config.MaxSize)
	}

	// Read response body with size limit
	limitedReader := io.LimitReader(resp.Body, int64(p.config.MaxSize))
	
	// Create temporary file
	tmpDir := p.config.TmpDir
	if tmpDir == "" {
		tmpDir = os.TempDir()
	}

	tmpFile, err := os.CreateTemp(tmpDir, "mdnotes-live-*.html")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name()) // Clean up
	defer tmpFile.Close()

	// Copy content to temp file
	if _, err := io.Copy(tmpFile, limitedReader); err != nil {
		return "", fmt.Errorf("copying response to temp file: %w", err)
	}

	// Close file before reading
	tmpFile.Close()

	// Extract text from HTML
	text, err := ExtractTextFromHTML(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("extracting text from HTML: %w", err)
	}

	return text, nil
}

// PickLatestSnapshot selects the most recent complete snapshot from a list of assets
func PickLatestSnapshot(assets []linkding.Asset) (*linkding.Asset, error) {
	// Filter for complete snapshots
	var snapshots []linkding.Asset
	for _, asset := range assets {
		if asset.AssetType == "snapshot" && asset.Status == "complete" {
			snapshots = append(snapshots, asset)
		}
	}

	if len(snapshots) == 0 {
		return nil, fmt.Errorf("no complete snapshots found")
	}

	// Sort by date created (newest first)
	sort.Slice(snapshots, func(i, j int) bool {
		// Parse ISO dates and compare
		timeI, errI := time.Parse(time.RFC3339, snapshots[i].DateCreated)
		timeJ, errJ := time.Parse(time.RFC3339, snapshots[j].DateCreated)
		
		if errI != nil || errJ != nil {
			// Fall back to string comparison if parsing fails
			return snapshots[i].DateCreated > snapshots[j].DateCreated
		}
		
		return timeI.After(timeJ)
	})

	return &snapshots[0], nil
}

// ExtractTextFromHTML extracts clean text from an HTML file
func ExtractTextFromHTML(htmlPath string) (string, error) {
	file, err := os.Open(htmlPath)
	if err != nil {
		return "", fmt.Errorf("opening HTML file: %w", err)
	}
	defer file.Close()

	tokenizer := html.NewTokenizer(file)
	var result strings.Builder
	var inScript, inStyle bool

	for {
		tokenType := tokenizer.Next()
		
		switch tokenType {
		case html.ErrorToken:
			// End of document - normalize whitespace before returning
			text := strings.TrimSpace(result.String())
			// Replace multiple spaces with single space
			text = strings.Join(strings.Fields(text), " ")
			return text, nil
			
		case html.StartTagToken, html.EndTagToken:
			token := tokenizer.Token()
			tagName := strings.ToLower(token.Data)
			
			if tokenType == html.StartTagToken {
				if tagName == "script" || tagName == "style" {
					if tagName == "script" {
						inScript = true
					} else {
						inStyle = true
					}
				}
			} else { // EndTagToken
				if tagName == "script" {
					inScript = false
				} else if tagName == "style" {
					inStyle = false
				}
			}
			
			// Add space after block elements for readability
			if tokenType == html.EndTagToken {
				blockElements := []string{"p", "div", "br", "h1", "h2", "h3", "h4", "h5", "h6", "li", "tr"}
				for _, blockEl := range blockElements {
					if tagName == blockEl {
						result.WriteString(" ")
						break
					}
				}
			}
			
		case html.TextToken:
			if !inScript && !inStyle {
				text := strings.TrimSpace(tokenizer.Token().Data)
				if text != "" {
					result.WriteString(text)
					result.WriteString(" ")
				}
			}
		}
	}
}

// convertToInt converts various numeric types to int
func (p *LinkdingGetProcessor) convertToInt(value interface{}) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("cannot convert %T to int", value)
	}
}