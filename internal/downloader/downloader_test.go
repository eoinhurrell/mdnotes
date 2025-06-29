package downloader

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/eoinhurrell/mdnotes/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a test downloader
func createTestDownloader(t *testing.T) (*Downloader, string) {
	tmpDir, err := os.MkdirTemp("", "downloader-test-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	cfg := config.DownloadConfig{
		AttachmentsDir: tmpDir,
		Timeout:        "30s",
		UserAgent:      "mdnotes-test",
		MaxFileSize:    10 * 1024 * 1024, // 10MB
	}

	downloader, err := NewDownloader(cfg)
	require.NoError(t, err)

	return downloader, tmpDir
}

func TestNewDownloader(t *testing.T) {
	downloader, _ := createTestDownloader(t)
	assert.NotNil(t, downloader)
}

func TestNewDownloader_InvalidTimeout(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "downloader-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cfg := config.DownloadConfig{
		AttachmentsDir: tmpDir,
		Timeout:        "invalid-timeout",
		UserAgent:      "mdnotes-test",
		MaxFileSize:    10 * 1024 * 1024,
	}

	_, err = NewDownloader(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid timeout duration")
}

func TestDownloadResource_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test content"))
	}))
	defer server.Close()

	downloader, tmpDir := createTestDownloader(t)
	ctx := context.Background()

	result, err := downloader.DownloadResource(ctx, server.URL, "test-file", "test-attr")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.LocalPath)
	assert.Equal(t, server.URL, result.OriginalURL)
	assert.False(t, result.Skipped)

	// Verify file exists and has correct content
	content, err := os.ReadFile(result.LocalPath)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(content))

	// Verify file is in attachments directory
	assert.Contains(t, result.LocalPath, tmpDir)
}

func TestDownloadResource_InvalidURL(t *testing.T) {
	downloader, _ := createTestDownloader(t)
	ctx := context.Background()

	_, err := downloader.DownloadResource(ctx, "invalid-url", "test", "attr")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid URL")
}

func TestDownloadResource_HTTPError(t *testing.T) {
	// Create test server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	downloader, _ := createTestDownloader(t)
	ctx := context.Background()

	_, err := downloader.DownloadResource(ctx, server.URL, "test", "attr")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestDownloadResource_UnsupportedScheme(t *testing.T) {
	downloader, _ := createTestDownloader(t)
	ctx := context.Background()

	_, err := downloader.DownloadResource(ctx, "ftp://example.com/file", "test", "attr")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported URL scheme")
}

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://example.com", true},
		{"http://example.com/file.pdf", true},
		{"ftp://example.com", false},
		{"invalid-url", false},
		{"", false},
		{"mailto:test@example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := IsValidURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateWikiLink(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"attachments/image.png", "![[image.png]]"},
		{"docs/document.pdf", "![[document.pdf]]"},
		{"/full/path/file.txt", "![[file.txt]]"},
		{"file-without-extension", "![[file-without-extension]]"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := GenerateWikiLink(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDownloadResource_ContentTypes(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		content     []byte
	}{
		{
			name:        "Plain text",
			contentType: "text/plain",
			content:     []byte("plain text content"),
		},
		{
			name:        "JSON",
			contentType: "application/json",
			content:     []byte(`{"key": "value"}`),
		},
		{
			name:        "PNG image",
			contentType: "image/png",
			content:     []byte("fake png content"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				w.WriteHeader(http.StatusOK)
				w.Write(tt.content)
			}))
			defer server.Close()

			downloader, _ := createTestDownloader(t)
			ctx := context.Background()

			result, err := downloader.DownloadResource(ctx, server.URL, "test-file", "test-attr")
			require.NoError(t, err)

			// Verify content
			content, err := os.ReadFile(result.LocalPath)
			require.NoError(t, err)
			assert.Equal(t, tt.content, content)
		})
	}
}

// Benchmark test
func BenchmarkDownloadResource(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("benchmark content"))
	}))
	defer server.Close()

	downloader, _ := createTestDownloader(&testing.T{})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		downloader.DownloadResource(ctx, server.URL, "bench-file", "attr")
	}
}
