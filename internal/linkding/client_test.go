package linkding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_CreateBookmark(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/bookmarks/", r.URL.Path)
		assert.Equal(t, "Token test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req CreateBookmarkRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		assert.Equal(t, "https://example.com", req.URL)
		assert.Equal(t, "Example", req.Title)

		resp := BookmarkResponse{
			ID:    123,
			URL:   req.URL,
			Title: req.Title,
			Tags:  req.Tags,
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	bookmark, err := client.CreateBookmark(context.Background(), CreateBookmarkRequest{
		URL:   "https://example.com",
		Title: "Example",
		Tags:  []string{"test"},
	})

	assert.NoError(t, err)
	assert.Equal(t, 123, bookmark.ID)
	assert.Equal(t, "https://example.com", bookmark.URL)
	assert.Equal(t, "Example", bookmark.Title)
}

func TestClient_GetBookmarks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/bookmarks/", r.URL.Path)
		assert.Equal(t, "Token test-token", r.Header.Get("Authorization"))

		resp := BookmarkListResponse{
			Count: 2,
			Results: []BookmarkResponse{
				{ID: 1, URL: "https://example1.com", Title: "Example 1"},
				{ID: 2, URL: "https://example2.com", Title: "Example 2"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	bookmarks, err := client.GetBookmarks(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, 2, bookmarks.Count)
	assert.Len(t, bookmarks.Results, 2)
	assert.Equal(t, "Example 1", bookmarks.Results[0].Title)
}

func TestClient_UpdateBookmark(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		assert.Equal(t, "/api/bookmarks/123/", r.URL.Path)

		var req UpdateBookmarkRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		resp := BookmarkResponse{
			ID:    123,
			URL:   "https://example.com",
			Title: req.Title,
			Tags:  req.Tags,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	bookmark, err := client.UpdateBookmark(context.Background(), 123, UpdateBookmarkRequest{
		Title: "Updated Title",
		Tags:  []string{"updated"},
	})

	assert.NoError(t, err)
	assert.Equal(t, 123, bookmark.ID)
	assert.Equal(t, "Updated Title", bookmark.Title)
}

func TestClient_RateLimiting(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount > 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(BookmarkListResponse{})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token", WithRateLimit(2)) // 2 req/sec

	start := time.Now()

	// Make 4 requests
	for i := 0; i < 4; i++ {
		_, err := client.GetBookmarks(context.Background())
		if i < 2 {
			assert.NoError(t, err)
		}
	}

	elapsed := time.Since(start)
	// Should take at least 1 second due to rate limiting
	assert.Greater(t, elapsed, 500*time.Millisecond)
}

func TestClient_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectedError string
	}{
		{
			name:          "404 not found",
			statusCode:    http.StatusNotFound,
			expectedError: "bookmark not found",
		},
		{
			name:          "400 bad request",
			statusCode:    http.StatusBadRequest,
			responseBody:  `{"url": ["This field is required."]}`,
			expectedError: "validation error",
		},
		{
			name:          "500 server error",
			statusCode:    http.StatusInternalServerError,
			expectedError: "server error",
		},
		{
			name:          "429 rate limited",
			statusCode:    http.StatusTooManyRequests,
			expectedError: "rate limited",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.responseBody != "" {
					w.Write([]byte(tt.responseBody))
				}
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-token")
			_, err := client.CreateBookmark(context.Background(), CreateBookmarkRequest{
				URL:   "https://example.com",
				Title: "Test",
			})

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.GetBookmarks(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestClient_ListAssets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/bookmarks/123/assets/", r.URL.Path)
		assert.Equal(t, "Token test-token", r.Header.Get("Authorization"))

		resp := AssetListResponse{
			Count: 2,
			Results: []Asset{
				{
					ID:          1,
					AssetType:   "snapshot",
					ContentType: "text/html",
					DisplayName: "Snapshot 1",
					FileSize:    1024,
					Status:      "complete",
					DateCreated: "2023-01-01T10:00:00Z",
					File:        "/path/to/file1.html",
				},
				{
					ID:          2,
					AssetType:   "snapshot",
					ContentType: "text/html",
					DisplayName: "Snapshot 2",
					FileSize:    2048,
					Status:      "complete",
					DateCreated: "2023-01-02T10:00:00Z",
					File:        "/path/to/file2.html",
				},
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	assets, err := client.ListAssets(context.Background(), 123)

	assert.NoError(t, err)
	assert.Len(t, assets, 2)
	assert.Equal(t, 1, assets[0].ID)
	assert.Equal(t, "snapshot", assets[0].AssetType)
	assert.Equal(t, "complete", assets[0].Status)
	assert.Equal(t, int64(1024), assets[0].FileSize)
	assert.Equal(t, 2, assets[1].ID)
	assert.Equal(t, "2023-01-02T10:00:00Z", assets[1].DateCreated)
}

func TestClient_ListAssets_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := AssetListResponse{
			Count:   0,
			Results: []Asset{},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	assets, err := client.ListAssets(context.Background(), 123)

	assert.NoError(t, err)
	assert.Len(t, assets, 0)
}

func TestClient_ListAssets_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	assets, err := client.ListAssets(context.Background(), 999)

	assert.Error(t, err)
	assert.Nil(t, assets)
	assert.Contains(t, err.Error(), "bookmark not found")
}

func TestClient_DownloadAsset(t *testing.T) {
	testContent := "<html><body><h1>Test HTML</h1><p>This is test content.</p></body></html>"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/bookmarks/123/assets/456/download/", r.URL.Path)
		assert.Equal(t, "Token test-token", r.Header.Get("Authorization"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testContent))
	}))
	defer server.Close()

	// Create temporary file for download
	tmpFile, err := os.CreateTemp("", "test-download-*.html")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	client := NewClient(server.URL, "test-token")
	err = client.DownloadAsset(context.Background(), 123, 456, tmpFile.Name())

	assert.NoError(t, err)

	// Verify file content
	content, err := os.ReadFile(tmpFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

func TestClient_DownloadAsset_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tmpFile, err := os.CreateTemp("", "test-download-*.html")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	client := NewClient(server.URL, "test-token")
	err = client.DownloadAsset(context.Background(), 123, 456, tmpFile.Name())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bookmark not found")
}

func TestClient_DownloadAsset_InvalidPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test content"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	err := client.DownloadAsset(context.Background(), 123, 456, "/invalid/path/file.html")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "creating destination file")
}
