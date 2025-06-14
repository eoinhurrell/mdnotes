package processor

import (
	"context"
	"fmt"
	"strings"

	"github.com/eoinhurrell/mdnotes/internal/linkding"
	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// LinkdingClient interface for dependency injection and testing
type LinkdingClient interface {
	CreateBookmark(ctx context.Context, req linkding.CreateBookmarkRequest) (*linkding.BookmarkResponse, error)
	GetBookmarks(ctx context.Context) (*linkding.BookmarkListResponse, error)
	UpdateBookmark(ctx context.Context, id int, req linkding.UpdateBookmarkRequest) (*linkding.BookmarkResponse, error)
	GetBookmark(ctx context.Context, id int) (*linkding.BookmarkResponse, error)
	DeleteBookmark(ctx context.Context, id int) error
}

// LinkdingSyncConfig configures the Linkding synchronization
type LinkdingSyncConfig struct {
	URLField         string // Frontmatter field containing the URL
	IDField          string // Frontmatter field to store Linkding ID
	TitleField       string // Frontmatter field containing the title
	TagsField        string // Frontmatter field containing tags
	DescriptionField string // Frontmatter field containing description
	NotesField       string // Frontmatter field containing notes
	SyncTitle        bool   // Whether to sync title to Linkding
	SyncTags         bool   // Whether to sync tags to Linkding
	SyncDescription  bool   // Whether to sync description to Linkding
	SyncNotes        bool   // Whether to sync notes to Linkding
	DryRun           bool   // Whether to perform a dry run
}

// LinkdingSync handles synchronization between vault files and Linkding
type LinkdingSync struct {
	config LinkdingSyncConfig
	client LinkdingClient
}

// SyncResult represents the result of a sync operation
type SyncResult struct {
	File      *vault.VaultFile
	Action    string // "created", "updated", "skipped", "error"
	BookmarkID int
	Error     error
}

// NewLinkdingSync creates a new Linkding sync processor
func NewLinkdingSync(config LinkdingSyncConfig) *LinkdingSync {
	// Set default field names if not provided
	if config.URLField == "" {
		config.URLField = "url"
	}
	if config.IDField == "" {
		config.IDField = "linkding_id"
	}
	if config.TitleField == "" {
		config.TitleField = "title"
	}
	if config.TagsField == "" {
		config.TagsField = "tags"
	}
	if config.DescriptionField == "" {
		config.DescriptionField = "description"
	}
	if config.NotesField == "" {
		config.NotesField = "notes"
	}

	return &LinkdingSync{
		config: config,
	}
}

// SetClient sets the Linkding client (for dependency injection)
func (ls *LinkdingSync) SetClient(client LinkdingClient) {
	ls.client = client
}

// FindUnsyncedFiles returns files that have URLs but no Linkding IDs
func (ls *LinkdingSync) FindUnsyncedFiles(files []*vault.VaultFile) []*vault.VaultFile {
	var unsynced []*vault.VaultFile

	for _, file := range files {
		if ls.hasURL(file) && !ls.hasLinkdingID(file) {
			unsynced = append(unsynced, file)
		}
	}

	return unsynced
}

// SyncFile synchronizes a single file with Linkding
func (ls *LinkdingSync) SyncFile(ctx context.Context, file *vault.VaultFile) error {
	// Skip if no URL
	if !ls.hasURL(file) {
		return nil
	}

	// Skip if already synced and not updating
	if ls.hasLinkdingID(file) {
		return nil
	}

	if ls.config.DryRun {
		return nil
	}

	// Create bookmark request
	req := ls.buildCreateRequest(file)

	// Create bookmark in Linkding
	bookmark, err := ls.client.CreateBookmark(ctx, req)
	if err != nil {
		return fmt.Errorf("creating bookmark: %w", err)
	}

	// Store the Linkding ID in frontmatter
	file.Frontmatter[ls.config.IDField] = bookmark.ID

	return nil
}

// UpdateExisting updates an existing bookmark in Linkding
func (ls *LinkdingSync) UpdateExisting(ctx context.Context, file *vault.VaultFile) error {
	if !ls.hasLinkdingID(file) {
		return fmt.Errorf("file has no Linkding ID")
	}

	if ls.config.DryRun {
		return nil
	}

	linkdingID, ok := file.Frontmatter[ls.config.IDField].(int)
	if !ok {
		// Try converting from float64 (JSON numbers)
		if f, ok := file.Frontmatter[ls.config.IDField].(float64); ok {
			linkdingID = int(f)
		} else {
			return fmt.Errorf("invalid Linkding ID type")
		}
	}

	// Build update request
	req := ls.buildUpdateRequest(file)

	// Update bookmark in Linkding
	_, err := ls.client.UpdateBookmark(ctx, linkdingID, req)
	if err != nil {
		return fmt.Errorf("updating bookmark %d: %w", linkdingID, err)
	}

	return nil
}

// SyncBatch synchronizes multiple files with Linkding
func (ls *LinkdingSync) SyncBatch(ctx context.Context, files []*vault.VaultFile) ([]SyncResult, error) {
	var results []SyncResult

	for _, file := range files {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		result := SyncResult{File: file}

		if !ls.hasURL(file) {
			result.Action = "skipped"
			results = append(results, result)
			continue
		}

		if ls.hasLinkdingID(file) {
			result.Action = "skipped"
			if id, ok := file.Frontmatter[ls.config.IDField].(int); ok {
				result.BookmarkID = id
			}
			results = append(results, result)
			continue
		}

		err := ls.SyncFile(ctx, file)
		if err != nil {
			result.Action = "error"
			result.Error = err
		} else {
			result.Action = "created"
			if id, ok := file.Frontmatter[ls.config.IDField].(int); ok {
				result.BookmarkID = id
			}
		}

		results = append(results, result)
	}

	return results, nil
}

// hasURL checks if the file has a valid URL
func (ls *LinkdingSync) hasURL(file *vault.VaultFile) bool {
	url, exists := file.Frontmatter[ls.config.URLField]
	if !exists {
		return false
	}

	urlStr, ok := url.(string)
	return ok && strings.TrimSpace(urlStr) != ""
}

// hasLinkdingID checks if the file has a Linkding ID
func (ls *LinkdingSync) hasLinkdingID(file *vault.VaultFile) bool {
	id, exists := file.Frontmatter[ls.config.IDField]
	if !exists {
		return false
	}

	switch v := id.(type) {
	case int:
		return v > 0
	case float64:
		return v > 0
	default:
		return false
	}
}

// buildCreateRequest builds a bookmark creation request from a file
func (ls *LinkdingSync) buildCreateRequest(file *vault.VaultFile) linkding.CreateBookmarkRequest {
	req := linkding.CreateBookmarkRequest{
		URL: file.Frontmatter[ls.config.URLField].(string),
	}

	if ls.config.SyncTitle {
		if title, ok := file.Frontmatter[ls.config.TitleField].(string); ok {
			req.Title = title
		}
	}

	if ls.config.SyncTags {
		if tags := ls.getTags(file); len(tags) > 0 {
			req.Tags = tags
		}
	}

	if ls.config.SyncDescription {
		if desc, ok := file.Frontmatter[ls.config.DescriptionField].(string); ok {
			req.Description = desc
		}
	}

	if ls.config.SyncNotes {
		if notes, ok := file.Frontmatter[ls.config.NotesField].(string); ok {
			req.Notes = notes
		}
	}

	return req
}

// buildUpdateRequest builds a bookmark update request from a file
func (ls *LinkdingSync) buildUpdateRequest(file *vault.VaultFile) linkding.UpdateBookmarkRequest {
	req := linkding.UpdateBookmarkRequest{}

	if ls.config.SyncTitle {
		if title, ok := file.Frontmatter[ls.config.TitleField].(string); ok {
			req.Title = title
		}
	}

	if ls.config.SyncTags {
		if tags := ls.getTags(file); len(tags) > 0 {
			req.Tags = tags
		}
	}

	if ls.config.SyncDescription {
		if desc, ok := file.Frontmatter[ls.config.DescriptionField].(string); ok {
			req.Description = desc
		}
	}

	if ls.config.SyncNotes {
		if notes, ok := file.Frontmatter[ls.config.NotesField].(string); ok {
			req.Notes = notes
		}
	}

	return req
}

// getTags extracts tags from the file frontmatter
func (ls *LinkdingSync) getTags(file *vault.VaultFile) []string {
	tagsValue, exists := file.Frontmatter[ls.config.TagsField]
	if !exists {
		return []string{}
	}

	return ls.convertToStringSlice(tagsValue)
}

// convertToStringSlice converts various tag formats to string slice
func (ls *LinkdingSync) convertToStringSlice(value interface{}) []string {
	if value == nil {
		return []string{}
	}

	switch v := value.(type) {
	case []string:
		// Filter out empty strings
		var result []string
		for _, tag := range v {
			if strings.TrimSpace(tag) != "" {
				result = append(result, strings.TrimSpace(tag))
			}
		}
		return result

	case []interface{}:
		var result []string
		for _, item := range v {
			if str, ok := item.(string); ok && strings.TrimSpace(str) != "" {
				result = append(result, strings.TrimSpace(str))
			}
		}
		return result

	case string:
		str := strings.TrimSpace(v)
		if str == "" {
			return []string{}
		}
		
		// Handle comma-separated tags
		if strings.Contains(str, ",") {
			var result []string
			for _, tag := range strings.Split(str, ",") {
				if trimmed := strings.TrimSpace(tag); trimmed != "" {
					result = append(result, trimmed)
				}
			}
			return result
		}
		
		return []string{str}

	default:
		return []string{}
	}
}