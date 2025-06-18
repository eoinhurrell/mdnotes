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
	timeout, err := time.ParseDuration(cfg.Timeout)
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

	return &Downloader{
		client:      client,
		config:      cfg,
		userAgent:   cfg.UserAgent,
		maxFileSize: cfg.MaxFileSize,
	}, nil
}

// DownloadResult contains information about a downloaded file
type DownloadResult struct {
	LocalPath    string
	OriginalURL  string
	ContentType  string
	Size         int64
	Extension    string
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
	defer resp.Body.Close()

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
		os.Remove(localPath)
		return nil, fmt.Errorf("copying file content: %w", err)
	}

	// Check if we exceeded the size limit
	if bytesWritten > d.maxFileSize {
		os.Remove(localPath)
		return nil, fmt.Errorf("file too large: %d bytes (max: %d)", bytesWritten, d.maxFileSize)
	}

	return &DownloadResult{
		LocalPath:   localPath,
		OriginalURL: urlStr,
		ContentType: resp.Header.Get("Content-Type"),
		Size:        bytesWritten,
		Extension:   extension,
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
			// Get extension from MIME type
			if exts, err := mime.ExtensionsByType(mediaType); err == nil && len(exts) > 0 {
				return exts[0] // Return the first (usually most common) extension
			}
		}
	}

	// Fall back to URL path extension
	parsedURL, err := url.Parse(urlStr)
	if err == nil {
		ext := filepath.Ext(parsedURL.Path)
		if ext != "" {
			return ext
		}
	}

	// Default extensions for common types
	if strings.Contains(contentType, "image/") {
		if strings.Contains(contentType, "jpeg") || strings.Contains(contentType, "jpg") {
			return ".jpg"
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
	// Convert to forward slashes for consistency
	linkPath := filepath.ToSlash(localPath)
	
	// Remove leading ./ if present
	linkPath = strings.TrimPrefix(linkPath, "./")
	
	return fmt.Sprintf("[[%s]]", linkPath)
}