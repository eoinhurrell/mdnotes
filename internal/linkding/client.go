package linkding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

// Client represents a Linkding API client
type Client struct {
	baseURL     string
	apiToken    string
	httpClient  *http.Client
	rateLimiter *rate.Limiter
}

// ClientOption configures a Client
type ClientOption func(*Client)

// WithRateLimit sets the rate limit for API calls
func WithRateLimit(reqPerSec int) ClientOption {
	return func(c *Client) {
		c.rateLimiter = rate.NewLimiter(rate.Limit(reqPerSec), 1)
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = client
	}
}

// NewClient creates a new Linkding API client
func NewClient(baseURL, apiToken string, opts ...ClientOption) *Client {
	// Create a custom dialer that forces IPv4 only
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	// Create IPv4-only dial function
	ipv4OnlyDialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		// Force TCP4 instead of TCP to prevent IPv6
		if network == "tcp" {
			network = "tcp4"
		}
		return dialer.DialContext(ctx, network, addr)
	}

	// Create a custom HTTP client with robust networking configuration
	transport := &http.Transport{
		// Force IPv4-only connections
		DialContext:           ipv4OnlyDialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		// Disable HTTP/2 server push
		DisableCompression: false,
	}

	httpClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}

	client := &Client{
		baseURL:     baseURL,
		apiToken:    apiToken,
		httpClient:  httpClient,
		rateLimiter: rate.NewLimiter(rate.Limit(5), 2), // More conservative: 5 req/sec with burst of 2
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// isRetryableError checks if an error is worth retrying
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for network errors that are typically transient
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}

	// Check for specific connection errors
	errStr := err.Error()
	retryableErrors := []string{
		"connection refused",
		"connection reset",
		"connection timeout",
		"dial tcp",
		"no such host",
		"network is unreachable",
		"i/o timeout",
		"invalid argument", // IPv6 link-local issues
	}

	for _, retryable := range retryableErrors {
		if strings.Contains(errStr, retryable) {
			return true
		}
	}

	return false
}

// doRequestWithRetry executes an HTTP request with retry logic
func (c *Client) doRequestWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	const maxRetries = 3
	const baseDelay = 1 * time.Second

	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Wait for rate limit
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}

		// Clone request for retry (body needs to be reset)
		var reqClone *http.Request
		if req.Body != nil {
			// Read body to buffer for retries
			body := &bytes.Buffer{}
			if _, err := body.ReadFrom(req.Body); err != nil {
				return nil, fmt.Errorf("reading request body: %w", err)
			}
			_ = req.Body.Close()

			// Create clone with fresh body
			reqClone = req.Clone(ctx)
			reqClone.Body = &nopCloser{bytes.NewReader(body.Bytes())}
			req.Body = &nopCloser{bytes.NewReader(body.Bytes())}
		} else {
			reqClone = req.Clone(ctx)
		}

		resp, err := c.httpClient.Do(reqClone)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// Don't retry if error is not retryable or if this is the last attempt
		if !isRetryableError(err) || attempt == maxRetries {
			break
		}

		// Calculate delay with exponential backoff
		delay := time.Duration(attempt+1) * baseDelay

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", maxRetries+1, lastErr)
}

// nopCloser is a helper to create io.ReadCloser from io.Reader
type nopCloser struct {
	*bytes.Reader
}

func (n *nopCloser) Close() error { return nil }

// CreateBookmarkRequest represents a request to create a bookmark
type CreateBookmarkRequest struct {
	URL         string   `json:"url"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Notes       string   `json:"notes,omitempty"`
	Tags        []string `json:"tag_names,omitempty"`
	IsArchived  bool     `json:"is_archived,omitempty"`
	Unread      bool     `json:"unread,omitempty"`
	Shared      bool     `json:"shared,omitempty"`
}

// UpdateBookmarkRequest represents a request to update a bookmark
type UpdateBookmarkRequest struct {
	URL         string   `json:"url,omitempty"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Notes       string   `json:"notes,omitempty"`
	Tags        []string `json:"tag_names,omitempty"`
	IsArchived  bool     `json:"is_archived,omitempty"`
	Unread      bool     `json:"unread,omitempty"`
	Shared      bool     `json:"shared,omitempty"`
}

// BookmarkResponse represents a bookmark from the API
type BookmarkResponse struct {
	ID           int      `json:"id"`
	URL          string   `json:"url"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Notes        string   `json:"notes"`
	Tags         []string `json:"tag_names"`
	IsArchived   bool     `json:"is_archived"`
	Unread       bool     `json:"unread"`
	Shared       bool     `json:"shared"`
	DateAdded    string   `json:"date_added"`
	DateModified string   `json:"date_modified"`
}

// BookmarkListResponse represents a list of bookmarks from the API
type BookmarkListResponse struct {
	Count    int                `json:"count"`
	Next     *string            `json:"next"`
	Previous *string            `json:"previous"`
	Results  []BookmarkResponse `json:"results"`
}

// CreateBookmark creates a new bookmark
func (c *Client) CreateBookmark(ctx context.Context, req CreateBookmarkRequest) (*BookmarkResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.baseURL+"/api/bookmarks/", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.doRequestWithRetry(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := c.checkResponse(resp); err != nil {
		return nil, err
	}

	var bookmark BookmarkResponse
	if err := json.NewDecoder(resp.Body).Decode(&bookmark); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &bookmark, nil
}

// GetBookmarks retrieves bookmarks from the API
func (c *Client) GetBookmarks(ctx context.Context) (*BookmarkListResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET",
		c.baseURL+"/api/bookmarks/", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.doRequestWithRetry(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := c.checkResponse(resp); err != nil {
		return nil, err
	}

	var bookmarks BookmarkListResponse
	if err := json.NewDecoder(resp.Body).Decode(&bookmarks); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &bookmarks, nil
}

// UpdateBookmark updates an existing bookmark
func (c *Client) UpdateBookmark(ctx context.Context, id int, req UpdateBookmarkRequest) (*BookmarkResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "PATCH",
		c.baseURL+"/api/bookmarks/"+strconv.Itoa(id)+"/", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.doRequestWithRetry(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := c.checkResponse(resp); err != nil {
		return nil, err
	}

	var bookmark BookmarkResponse
	if err := json.NewDecoder(resp.Body).Decode(&bookmark); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &bookmark, nil
}

// GetBookmark retrieves a specific bookmark by ID
func (c *Client) GetBookmark(ctx context.Context, id int) (*BookmarkResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET",
		c.baseURL+"/api/bookmarks/"+strconv.Itoa(id)+"/", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.doRequestWithRetry(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := c.checkResponse(resp); err != nil {
		return nil, err
	}

	var bookmark BookmarkResponse
	if err := json.NewDecoder(resp.Body).Decode(&bookmark); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &bookmark, nil
}

// DeleteBookmark deletes a bookmark by ID
func (c *Client) DeleteBookmark(ctx context.Context, id int) error {
	httpReq, err := http.NewRequestWithContext(ctx, "DELETE",
		c.baseURL+"/api/bookmarks/"+strconv.Itoa(id)+"/", nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.doRequestWithRetry(ctx, httpReq)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := c.checkResponse(resp); err != nil {
		return err
	}

	return nil
}

// setHeaders sets common headers for API requests
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Token "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}

// checkResponse checks the HTTP response for errors
func (c *Client) checkResponse(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
		return nil
	case http.StatusBadRequest:
		return fmt.Errorf("validation error: bad request")
	case http.StatusUnauthorized:
		return fmt.Errorf("authentication error: invalid API token")
	case http.StatusForbidden:
		return fmt.Errorf("authorization error: insufficient permissions")
	case http.StatusNotFound:
		return fmt.Errorf("bookmark not found")
	case http.StatusTooManyRequests:
		return fmt.Errorf("rate limited: too many requests")
	case http.StatusInternalServerError:
		return fmt.Errorf("server error: internal server error")
	default:
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

// CheckBookmarkRequest represents metadata for checking a bookmark
type CheckBookmarkResponse struct {
	Bookmark *BookmarkResponse `json:"bookmark"`
	Metadata struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	} `json:"metadata"`
	AutoTags []string `json:"auto_tags"`
}

// Asset represents a Linkding bookmark asset
type Asset struct {
	ID          int    `json:"id"`
	AssetType   string `json:"asset_type"`
	ContentType string `json:"content_type"`
	DisplayName string `json:"display_name"`
	FileSize    int64  `json:"file_size"`
	Status      string `json:"status"`
	DateCreated string `json:"date_created"`
	File        string `json:"file"`
}

// AssetListResponse represents a list of assets from the API
type AssetListResponse struct {
	Count    int     `json:"count"`
	Next     *string `json:"next"`
	Previous *string `json:"previous"`
	Results  []Asset `json:"results"`
}

// CheckBookmark checks if a URL is already bookmarked
func (c *Client) CheckBookmark(ctx context.Context, url string) (*CheckBookmarkResponse, error) {
	// Create request with URL parameter
	httpReq, err := http.NewRequestWithContext(ctx, "GET",
		c.baseURL+"/api/bookmarks/check/", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Add URL as query parameter
	q := httpReq.URL.Query()
	q.Add("url", url)
	httpReq.URL.RawQuery = q.Encode()

	c.setHeaders(httpReq)

	resp, err := c.doRequestWithRetry(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Handle response - for check endpoint, 404 means URL not bookmarked
	switch resp.StatusCode {
	case http.StatusOK:
		// Bookmark exists or no bookmark found - decode response
		var checkResp CheckBookmarkResponse
		if err := json.NewDecoder(resp.Body).Decode(&checkResp); err != nil {
			return nil, fmt.Errorf("decoding response: %w", err)
		}
		return &checkResp, nil
	case http.StatusNotFound:
		// URL not bookmarked - return empty response
		return &CheckBookmarkResponse{Bookmark: nil}, nil
	default:
		// Use standard error handling for other status codes
		return nil, c.checkResponse(resp)
	}
}

// ListAssets retrieves assets for a specific bookmark
func (c *Client) ListAssets(ctx context.Context, bookmarkID int) ([]Asset, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET",
		c.baseURL+"/api/bookmarks/"+strconv.Itoa(bookmarkID)+"/assets/", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.doRequestWithRetry(ctx, httpReq)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := c.checkResponse(resp); err != nil {
		return nil, err
	}

	var assetList AssetListResponse
	if err := json.NewDecoder(resp.Body).Decode(&assetList); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return assetList.Results, nil
}

// DownloadAsset downloads a specific asset and writes it to the destination path
func (c *Client) DownloadAsset(ctx context.Context, bookmarkID, assetID int, destPath string) error {
	httpReq, err := http.NewRequestWithContext(ctx, "GET",
		c.baseURL+"/api/bookmarks/"+strconv.Itoa(bookmarkID)+"/assets/"+strconv.Itoa(assetID)+"/download/", nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.doRequestWithRetry(ctx, httpReq)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := c.checkResponse(resp); err != nil {
		return err
	}

	// Write response body to file
	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("creating destination file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("copying response to file: %w", err)
	}

	return nil
}
