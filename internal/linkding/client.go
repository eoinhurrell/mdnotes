package linkding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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
	client := &Client{
		baseURL:    baseURL,
		apiToken:   apiToken,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		rateLimiter: rate.NewLimiter(rate.Limit(10), 1), // Default: 10 req/sec
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

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
	ID          int      `json:"id"`
	URL         string   `json:"url"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Notes       string   `json:"notes"`
	Tags        []string `json:"tag_names"`
	IsArchived  bool     `json:"is_archived"`
	Unread      bool     `json:"unread"`
	Shared      bool     `json:"shared"`
	DateAdded   string   `json:"date_added"`
	DateModified string  `json:"date_modified"`
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
	// Wait for rate limit
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

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

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

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
	// Wait for rate limit
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET",
		c.baseURL+"/api/bookmarks/", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

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
	// Wait for rate limit
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

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

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

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
	// Wait for rate limit
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET",
		c.baseURL+"/api/bookmarks/"+strconv.Itoa(id)+"/", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

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
	// Wait for rate limit
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "DELETE",
		c.baseURL+"/api/bookmarks/"+strconv.Itoa(id)+"/", nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

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