package processor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/eoinhurrell/mdnotes/internal/linkding"
	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// MockLinkdingClient is a mock implementation of LinkdingClient
type MockLinkdingClient struct {
	mock.Mock
}

func (m *MockLinkdingClient) CreateBookmark(ctx context.Context, req linkding.CreateBookmarkRequest) (*linkding.BookmarkResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*linkding.BookmarkResponse), args.Error(1)
}

func (m *MockLinkdingClient) GetBookmarks(ctx context.Context) (*linkding.BookmarkListResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*linkding.BookmarkListResponse), args.Error(1)
}

func (m *MockLinkdingClient) UpdateBookmark(ctx context.Context, id int, req linkding.UpdateBookmarkRequest) (*linkding.BookmarkResponse, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*linkding.BookmarkResponse), args.Error(1)
}

func (m *MockLinkdingClient) GetBookmark(ctx context.Context, id int) (*linkding.BookmarkResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*linkding.BookmarkResponse), args.Error(1)
}

func (m *MockLinkdingClient) DeleteBookmark(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockLinkdingClient) CheckBookmark(ctx context.Context, url string) (*linkding.CheckBookmarkResponse, error) {
	args := m.Called(ctx, url)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*linkding.CheckBookmarkResponse), args.Error(1)
}

func TestLinkdingSync_FindUnsyncedFiles(t *testing.T) {
	files := []*vault.VaultFile{
		{
			Path: "synced.md",
			Frontmatter: map[string]interface{}{
				"url":         "https://example.com",
				"linkding_id": 123,
			},
		},
		{
			Path: "unsynced.md",
			Frontmatter: map[string]interface{}{
				"url": "https://example2.com",
			},
		},
		{
			Path: "no-url.md",
			Frontmatter: map[string]interface{}{
				"title": "No URL here",
			},
		},
		{
			Path: "empty-url.md",
			Frontmatter: map[string]interface{}{
				"url": "",
			},
		},
	}

	sync := NewLinkdingSync(LinkdingSyncConfig{
		URLField: "url",
		IDField:  "linkding_id",
	})

	unsynced := sync.FindUnsyncedFiles(files)
	assert.Len(t, unsynced, 1)
	assert.Equal(t, "unsynced.md", unsynced[0].Path)
}

func TestLinkdingSync_SyncFile(t *testing.T) {
	mockClient := &MockLinkdingClient{}
	// Mock CheckBookmark to return no existing bookmark
	mockClient.On("CheckBookmark", mock.Anything, "https://example.com").Return(&linkding.CheckBookmarkResponse{
		Bookmark: nil,
	}, nil)
	
	mockClient.On("CreateBookmark", mock.Anything, mock.MatchedBy(func(req linkding.CreateBookmarkRequest) bool {
		return req.URL == "https://example.com" &&
			req.Title == "Example Article" &&
			len(req.Tags) == 2 &&
			req.Tags[0] == "tech" &&
			req.Tags[1] == "go"
	})).Return(&linkding.BookmarkResponse{ID: 456}, nil)

	sync := NewLinkdingSync(LinkdingSyncConfig{
		URLField:    "url",
		IDField:     "linkding_id",
		TitleField:  "title",
		TagsField:   "tags",
		SyncTitle:   true,
		SyncTags:    true,
	})
	sync.client = mockClient

	file := &vault.VaultFile{
		Path: "test.md",
		Frontmatter: map[string]interface{}{
			"url":   "https://example.com",
			"title": "Example Article",
			"tags":  []interface{}{"tech", "go"},
		},
	}

	err := sync.SyncFile(context.Background(), file)
	assert.NoError(t, err)
	assert.Equal(t, 456, file.Frontmatter["linkding_id"])
	mockClient.AssertExpectations(t)
}

func TestLinkdingSync_SyncFile_AlreadySynced(t *testing.T) {
	mockClient := &MockLinkdingClient{}
	// Mock GetBookmark to verify the existing ID is valid
	mockClient.On("GetBookmark", mock.Anything, 123).Return(&linkding.BookmarkResponse{
		ID:    123,
		URL:   "https://example.com",
		Title: "Existing Bookmark",
	}, nil)

	sync := NewLinkdingSync(LinkdingSyncConfig{
		URLField: "url",
		IDField:  "linkding_id",
	})
	sync.client = mockClient

	file := &vault.VaultFile{
		Path: "synced.md",
		Frontmatter: map[string]interface{}{
			"url":         "https://example.com",
			"linkding_id": 123,
		},
	}

	err := sync.SyncFile(context.Background(), file)
	assert.NoError(t, err)
	assert.Equal(t, 123, file.Frontmatter["linkding_id"])
	mockClient.AssertExpectations(t)
}

func TestLinkdingSync_SyncFile_NoURL(t *testing.T) {
	mockClient := &MockLinkdingClient{}
	// Should not call CreateBookmark

	sync := NewLinkdingSync(LinkdingSyncConfig{
		URLField: "url",
		IDField:  "linkding_id",
	})
	sync.client = mockClient

	file := &vault.VaultFile{
		Path: "no-url.md",
		Frontmatter: map[string]interface{}{
			"title": "No URL here",
		},
	}

	err := sync.SyncFile(context.Background(), file)
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestLinkdingSync_UpdateExisting(t *testing.T) {
	mockClient := &MockLinkdingClient{}
	// Mock GetBookmark to verify bookmark exists
	mockClient.On("GetBookmark", mock.Anything, 123).Return(&linkding.BookmarkResponse{
		ID:    123,
		URL:   "https://example.com",
		Title: "Original Title",
	}, nil)
	
	mockClient.On("UpdateBookmark", mock.Anything, 123, mock.MatchedBy(func(req linkding.UpdateBookmarkRequest) bool {
		return req.Title == "Updated Title" &&
			len(req.Tags) == 1 &&
			req.Tags[0] == "updated"
	})).Return(&linkding.BookmarkResponse{
		ID:    123,
		URL:   "https://example.com",
		Title: "Updated Title",
		Tags:  []string{"updated"},
	}, nil)

	sync := NewLinkdingSync(LinkdingSyncConfig{
		URLField:    "url",
		IDField:     "linkding_id",
		TitleField:  "title",
		TagsField:   "tags",
		SyncTitle:   true,
		SyncTags:    true,
	})
	sync.client = mockClient

	file := &vault.VaultFile{
		Path: "existing.md",
		Frontmatter: map[string]interface{}{
			"url":         "https://example.com",
			"linkding_id": 123,
			"title":       "Updated Title",
			"tags":        []interface{}{"updated"},
		},
	}

	err := sync.UpdateExisting(context.Background(), file)
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestLinkdingSync_TypeConversion(t *testing.T) {
	sync := NewLinkdingSync(LinkdingSyncConfig{})

	tests := []struct {
		name     string
		input    interface{}
		expected []string
	}{
		{
			name:     "string slice",
			input:    []string{"tag1", "tag2"},
			expected: []string{"tag1", "tag2"},
		},
		{
			name:     "interface slice",
			input:    []interface{}{"tag1", "tag2"},
			expected: []string{"tag1", "tag2"},
		},
		{
			name:     "single string",
			input:    "single-tag",
			expected: []string{"single-tag"},
		},
		{
			name:     "comma-separated string",
			input:    "tag1, tag2, tag3",
			expected: []string{"tag1", "tag2", "tag3"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "nil",
			input:    nil,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sync.convertToStringSlice(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLinkdingSync_SyncBatch(t *testing.T) {
	mockClient := &MockLinkdingClient{}
	
	// Mock CheckBookmark calls - no existing bookmarks
	mockClient.On("CheckBookmark", mock.Anything, "https://example1.com").Return(&linkding.CheckBookmarkResponse{
		Bookmark: nil,
	}, nil)
	mockClient.On("CheckBookmark", mock.Anything, "https://example2.com").Return(&linkding.CheckBookmarkResponse{
		Bookmark: nil,
	}, nil)
	
	// Mock GetBookmark for file3 which already has linkding_id
	mockClient.On("GetBookmark", mock.Anything, 103).Return(&linkding.BookmarkResponse{
		ID:    103,
		URL:   "https://example3.com",
		Title: "Existing Bookmark",
	}, nil)
	
	// First file needs to be created
	mockClient.On("CreateBookmark", mock.Anything, mock.MatchedBy(func(req linkding.CreateBookmarkRequest) bool {
		return req.URL == "https://example1.com"
	})).Return(&linkding.BookmarkResponse{ID: 101}, nil)
	
	// Second file needs to be created
	mockClient.On("CreateBookmark", mock.Anything, mock.MatchedBy(func(req linkding.CreateBookmarkRequest) bool {
		return req.URL == "https://example2.com"
	})).Return(&linkding.BookmarkResponse{ID: 102}, nil)

	sync := NewLinkdingSync(LinkdingSyncConfig{
		URLField: "url",
		IDField:  "linkding_id",
	})
	sync.client = mockClient

	files := []*vault.VaultFile{
		{
			Path: "file1.md",
			Frontmatter: map[string]interface{}{
				"url": "https://example1.com",
			},
		},
		{
			Path: "file2.md",
			Frontmatter: map[string]interface{}{
				"url": "https://example2.com",
			},
		},
		{
			Path: "file3.md",
			Frontmatter: map[string]interface{}{
				"url":         "https://example3.com",
				"linkding_id": 103, // Already synced
			},
		},
	}

	results, err := sync.SyncBatch(context.Background(), files)
	assert.NoError(t, err)
	assert.Len(t, results, 3) // All 3 files were processed (2 created, 1 verified)

	// Check that IDs were set
	assert.Equal(t, 101, files[0].Frontmatter["linkding_id"])
	assert.Equal(t, 102, files[1].Frontmatter["linkding_id"])
	assert.Equal(t, 103, files[2].Frontmatter["linkding_id"]) // Unchanged

	mockClient.AssertExpectations(t)
}