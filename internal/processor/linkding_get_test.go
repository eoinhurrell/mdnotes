package processor

import (
	"context"
	"os"
	"testing"

	"github.com/eoinhurrell/mdnotes/internal/linkding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPickLatestSnapshot(t *testing.T) {
	tests := []struct {
		name     string
		assets   []linkding.Asset
		expected *linkding.Asset
		wantErr  bool
	}{
		{
			name:    "no assets",
			assets:  []linkding.Asset{},
			wantErr: true,
		},
		{
			name: "no snapshots",
			assets: []linkding.Asset{
				{AssetType: "other", Status: "complete"},
			},
			wantErr: true,
		},
		{
			name: "no complete snapshots",
			assets: []linkding.Asset{
				{AssetType: "snapshot", Status: "pending"},
				{AssetType: "snapshot", Status: "failed"},
			},
			wantErr: true,
		},
		{
			name: "single complete snapshot",
			assets: []linkding.Asset{
				{ID: 1, AssetType: "snapshot", Status: "complete", DateCreated: "2023-01-01T10:00:00Z"},
			},
			expected: &linkding.Asset{ID: 1, AssetType: "snapshot", Status: "complete", DateCreated: "2023-01-01T10:00:00Z"},
		},
		{
			name: "multiple snapshots - picks latest",
			assets: []linkding.Asset{
				{ID: 1, AssetType: "snapshot", Status: "complete", DateCreated: "2023-01-01T10:00:00Z"},
				{ID: 2, AssetType: "snapshot", Status: "complete", DateCreated: "2023-01-02T10:00:00Z"},
				{ID: 3, AssetType: "snapshot", Status: "pending", DateCreated: "2023-01-03T10:00:00Z"},
			},
			expected: &linkding.Asset{ID: 2, AssetType: "snapshot", Status: "complete", DateCreated: "2023-01-02T10:00:00Z"},
		},
		{
			name: "mixed asset types",
			assets: []linkding.Asset{
				{ID: 1, AssetType: "other", Status: "complete", DateCreated: "2023-01-01T10:00:00Z"},
				{ID: 2, AssetType: "snapshot", Status: "complete", DateCreated: "2023-01-02T10:00:00Z"},
				{ID: 3, AssetType: "snapshot", Status: "failed", DateCreated: "2023-01-03T10:00:00Z"},
			},
			expected: &linkding.Asset{ID: 2, AssetType: "snapshot", Status: "complete", DateCreated: "2023-01-02T10:00:00Z"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PickLatestSnapshot(tt.assets)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.ID, result.ID)
				assert.Equal(t, tt.expected.AssetType, result.AssetType)
				assert.Equal(t, tt.expected.Status, result.Status)
				assert.Equal(t, tt.expected.DateCreated, result.DateCreated)
			}
		})
	}
}

func TestExtractTextFromHTML(t *testing.T) {
	tests := []struct {
		name        string
		htmlContent string
		expected    string
	}{
		{
			name:        "simple html",
			htmlContent: `<html><body><p>Hello <b>World</b></p></body></html>`,
			expected:    "Hello World",
		},
		{
			name: "html with script and style",
			htmlContent: `<html>
				<head><style>body { color: red; }</style></head>
				<body>
					<script>console.log('hidden');</script>
					<p>Visible text</p>
				</body>
			</html>`,
			expected: "Visible text",
		},
		{
			name: "html with block elements",
			htmlContent: `<html><body>
				<h1>Title</h1>
				<div>Content</div>
				<p>Paragraph</p>
			</body></html>`,
			expected: "Title Content Paragraph",
		},
		{
			name:        "empty html",
			htmlContent: `<html><body></body></html>`,
			expected:    "",
		},
		{
			name: "text with newlines and whitespace",
			htmlContent: `<html><body>
				<p>  Line 1  </p>
				<br>
				<p>Line 2</p>
			</body></html>`,
			expected: "Line 1 Line 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary HTML file
			tmpFile, err := os.CreateTemp("", "test-*.html")
			require.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			// Write HTML content
			_, err = tmpFile.WriteString(tt.htmlContent)
			require.NoError(t, err)
			tmpFile.Close()

			// Extract text
			result, err := ExtractTextFromHTML(tmpFile.Name())
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLinkdingGetProcessor_convertToInt(t *testing.T) {
	processor := &LinkdingGetProcessor{}

	tests := []struct {
		name     string
		value    interface{}
		expected int
		wantErr  bool
	}{
		{"int", 123, 123, false},
		{"int64", int64(456), 456, false},
		{"float64", float64(789), 789, false},
		{"string valid", "321", 321, false},
		{"string invalid", "abc", 0, true},
		{"nil", nil, 0, true},
		{"bool", true, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.convertToInt(tt.value)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestNewLinkdingGet(t *testing.T) {
	config := LinkdingGetConfig{
		MaxSize: 1000000,
		Timeout: "10s",
		TmpDir:  "/tmp",
		Verbose: true,
	}

	processor := NewLinkdingGet(config)

	assert.NotNil(t, processor)
	assert.Equal(t, config.MaxSize, processor.config.MaxSize)
	assert.Equal(t, config.Timeout, processor.config.Timeout)
	assert.Equal(t, config.TmpDir, processor.config.TmpDir)
	assert.Equal(t, config.Verbose, processor.config.Verbose)
	assert.Nil(t, processor.client)
}

func TestLinkdingGetProcessor_SetClient(t *testing.T) {
	processor := NewLinkdingGet(LinkdingGetConfig{})
	client := &linkding.Client{} // Minimal client for testing

	processor.SetClient(client)

	assert.Equal(t, client, processor.client)
}

func TestExtractTextFromHTML_FileNotFound(t *testing.T) {
	result, err := ExtractTextFromHTML("/nonexistent/file.html")
	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "opening HTML file")
}

func TestPickLatestSnapshot_DateParsing(t *testing.T) {
	// Test with various date formats and edge cases
	assets := []linkding.Asset{
		{ID: 1, AssetType: "snapshot", Status: "complete", DateCreated: "2023-01-01T10:00:00Z"},
		{ID: 2, AssetType: "snapshot", Status: "complete", DateCreated: "2023-01-02T10:00:00Z"},
	}

	result, err := PickLatestSnapshot(assets)
	assert.NoError(t, err)

	// Should pick ID 2 since it has the newer date
	assert.Equal(t, 2, result.ID)
}

func TestLinkdingGetProcessor_GetContent_NoValidInput(t *testing.T) {
	processor := NewLinkdingGet(LinkdingGetConfig{})
	ctx := context.Background()

	// Test with nil values
	result, err := processor.GetContent(ctx, nil, nil)
	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "no valid linkding_id or url available")

	// Test with empty string URL
	result, err = processor.GetContent(ctx, nil, "")
	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "no valid linkding_id or url available")

	// Test with invalid linkding_id
	result, err = processor.GetContent(ctx, "invalid", nil)
	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "no valid linkding_id or url available")
}
