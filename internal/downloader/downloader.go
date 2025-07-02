package downloader

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eoinhurrell/mdnotes/internal/config"
)

// Downloader handles downloading web resources
type Downloader struct {
	client      *http.Client
	config      config.DownloadConfig
	userAgent   string
	maxFileSize int64
}

// NewDownloader creates a new downloader with the given configuration
func NewDownloader(cfg config.DownloadConfig) (*Downloader, error) {
	// Use default timeout if empty
	timeoutStr := cfg.Timeout
	if timeoutStr == "" {
		timeoutStr = "30s"
	}

	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout duration: %w", err)
	}

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			IdleConnTimeout:       30 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	// Apply defaults for empty values
	userAgent := cfg.UserAgent
	if userAgent == "" {
		userAgent = "mdnotes/1.0"
	}

	maxFileSize := cfg.MaxFileSize
	if maxFileSize == 0 {
		maxFileSize = 10 * 1024 * 1024 // 10MB
	}

	attachmentsDir := cfg.AttachmentsDir
	if attachmentsDir == "" {
		attachmentsDir = "./resources/attachments"
	}

	// Update config with defaults
	finalConfig := cfg
	finalConfig.Timeout = timeoutStr
	finalConfig.UserAgent = userAgent
	finalConfig.MaxFileSize = maxFileSize
	finalConfig.AttachmentsDir = attachmentsDir

	return &Downloader{
		client:      client,
		config:      finalConfig,
		userAgent:   userAgent,
		maxFileSize: maxFileSize,
	}, nil
}

// DownloadResult contains information about a downloaded file
type DownloadResult struct {
	LocalPath   string
	OriginalURL string
	ContentType string
	Size        int64
	Extension   string
	Skipped     bool // Indicates file already existed and was skipped
}

// DownloadResource downloads a resource from a URL to a local file
func (d *Downloader) DownloadResource(ctx context.Context, urlStr, baseFilename, attributeName string) (*DownloadResult, error) {
	// Parse and validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme: %s", parsedURL.Scheme)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", d.userAgent)

	// Make the request
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading resource: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Check content length if provided
	if resp.ContentLength > 0 && resp.ContentLength > d.maxFileSize {
		return nil, fmt.Errorf("file too large: %d bytes (max: %d)", resp.ContentLength, d.maxFileSize)
	}

	// Determine file extension from content type or URL
	extension := d.determineExtension(resp, urlStr)

	// Generate local filename
	filename := fmt.Sprintf("%s-%s%s", baseFilename, attributeName, extension)
	localPath := filepath.Join(d.config.AttachmentsDir, filename)

	// Ensure directory exists
	if err := os.MkdirAll(d.config.AttachmentsDir, 0755); err != nil {
		return nil, fmt.Errorf("creating attachments directory: %w", err)
	}

	// Check if file already exists
	if stat, err := os.Stat(localPath); err == nil {
		return &DownloadResult{
			LocalPath:   localPath,
			OriginalURL: urlStr,
			ContentType: resp.Header.Get("Content-Type"),
			Size:        stat.Size(), // Use existing file size
			Extension:   extension,
			Skipped:     true, // Mark as skipped
		}, nil // Not an error, just skipped
	}

	// Create local file
	file, err := os.Create(localPath)
	if err != nil {
		return nil, fmt.Errorf("creating local file: %w", err)
	}
	defer file.Close()

	// Copy with size limit
	limitedReader := io.LimitReader(resp.Body, d.maxFileSize+1)
	bytesWritten, err := io.Copy(file, limitedReader)
	if err != nil {
		// Clean up partial file on error
		_ = os.Remove(localPath)
		return nil, fmt.Errorf("copying file content: %w", err)
	}

	// Check if we exceeded the size limit
	if bytesWritten > d.maxFileSize {
		_ = os.Remove(localPath)
		return nil, fmt.Errorf("file too large: %d bytes (max: %d)", bytesWritten, d.maxFileSize)
	}

	return &DownloadResult{
		LocalPath:   localPath,
		OriginalURL: urlStr,
		ContentType: resp.Header.Get("Content-Type"),
		Size:        bytesWritten,
		Extension:   extension,
		Skipped:     false, // Actually downloaded
	}, nil
}

// determineExtension determines the file extension from HTTP response or URL
func (d *Downloader) determineExtension(resp *http.Response, urlStr string) string {
	// First try from Content-Type header
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" {
		// Remove charset and other parameters
		mediaType, _, err := mime.ParseMediaType(contentType)
		if err == nil {
			// Use custom mapping for Obsidian-compatible extensions
			if ext := d.getObsidianCompatibleExtension(mediaType); ext != "" {
				return ext
			}
		}
	}

	// Fall back to URL path extension, but fix known problematic extensions
	parsedURL, err := url.Parse(urlStr)
	if err == nil {
		ext := filepath.Ext(parsedURL.Path)
		if ext != "" {
			return d.normalizeExtensionForObsidian(ext)
		}
	}

	// Default extensions for common types
	if strings.Contains(contentType, "image/") {
		if strings.Contains(contentType, "jpeg") || strings.Contains(contentType, "jpg") {
			return ".jpeg"
		} else if strings.Contains(contentType, "png") {
			return ".png"
		} else if strings.Contains(contentType, "gif") {
			return ".gif"
		} else if strings.Contains(contentType, "webp") {
			return ".webp"
		}
	}

	// If all else fails, use .bin
	return ".bin"
}

// getObsidianCompatibleExtension returns Obsidian-compatible extensions for common MIME types
func (d *Downloader) getObsidianCompatibleExtension(mediaType string) string {
	// Map of MIME types to Obsidian-compatible extensions
	extensionMap := map[string]string{
		// Image formats - use extensions that Obsidian recognizes
		"image/jpeg":    ".jpeg",
		"image/jpg":     ".jpeg",
		"image/png":     ".png",
		"image/gif":     ".gif",
		"image/webp":    ".webp",
		"image/svg+xml": ".svg",
		"image/bmp":     ".bmp",
		"image/tiff":    ".tiff",
		"image/x-icon":  ".ico",

		// Document formats
		"application/pdf": ".pdf",
		"text/plain":      ".txt",
		"text/markdown":   ".md",
		"text/html":       ".html",

		// Audio formats
		"audio/mpeg": ".mp3",
		"audio/wav":  ".wav",
		"audio/ogg":  ".ogg",
		"audio/mp4":  ".m4a",

		// Video formats
		"video/mp4":  ".mp4",
		"video/webm": ".webm",
		"video/ogg":  ".ogv",
	}

	return extensionMap[mediaType]
}

// normalizeExtensionForObsidian fixes problematic file extensions for Obsidian compatibility
func (d *Downloader) normalizeExtensionForObsidian(ext string) string {
	// Convert to lowercase for consistent comparison
	lowerExt := strings.ToLower(ext)

	// Map of problematic extensions to Obsidian-compatible ones
	normalizeMap := map[string]string{
		".jpe": ".jpeg", // Fix the main issue - jpe is not recognized by Obsidian
		".jpg": ".jpeg", // Prefer .jpeg over .jpg for consistency
		".tif": ".tiff", // Prefer full extension name
		".htm": ".html", // Prefer full extension name
	}

	if normalized := normalizeMap[lowerExt]; normalized != "" {
		return normalized
	}

	return ext // Return original if no normalization needed
}

// IsValidURL checks if a string looks like a downloadable HTTP/HTTPS URL
func IsValidURL(str string) bool {
	if str == "" {
		return false
	}

	parsedURL, err := url.Parse(str)
	if err != nil {
		return false
	}

	return parsedURL.Scheme == "http" || parsedURL.Scheme == "https"
}

// GenerateWikiLink creates a wiki link for the downloaded file
func GenerateWikiLink(localPath string) string {
	// Extract just the filename from the path
	filename := filepath.Base(localPath)

	// Return as embed link format (with !)
	return fmt.Sprintf("![[%s]]", filename)
}
