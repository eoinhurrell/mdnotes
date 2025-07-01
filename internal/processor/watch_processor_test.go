package processor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eoinhurrell/mdnotes/internal/config"
)

func TestNewWatchProcessor(t *testing.T) {
	cfg := &config.Config{
		Watch: config.WatchConfig{
			Enabled:         true,
			DebounceTimeout: "2s",
		},
	}

	wp, err := NewWatchProcessor(cfg)
	require.NoError(t, err)
	require.NotNil(t, wp)

	defer wp.Stop()

	assert.Equal(t, cfg, wp.config)
	assert.NotNil(t, wp.watcher)
	assert.NotNil(t, wp.eventChan)
	assert.NotNil(t, wp.debounceMap)
}

func TestShouldIgnore(t *testing.T) {
	cfg := &config.Config{
		Watch: config.WatchConfig{
			IgnorePatterns: []string{
				".obsidian/*",
				".git/*",
				"*.tmp",
				"*.bak",
				".DS_Store",
			},
		},
	}

	wp, err := NewWatchProcessor(cfg)
	require.NoError(t, err)
	defer wp.Stop()

	tests := []struct {
		path     string
		expected bool
	}{
		{".obsidian/config.json", true},
		{".git/HEAD", true},
		{"temp.tmp", true},
		{"backup.bak", true},
		{".DS_Store", true},
		{"normal.md", false},
		{"subfolder/note.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := wp.shouldIgnore(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEventType(t *testing.T) {
	cfg := &config.Config{}
	wp, err := NewWatchProcessor(cfg)
	require.NoError(t, err)
	defer wp.Stop()

	tests := []struct {
		event    fsnotify.Event
		expected string
	}{
		{fsnotify.Event{Op: fsnotify.Create}, "create"},
		{fsnotify.Event{Op: fsnotify.Write}, "write"},
		{fsnotify.Event{Op: fsnotify.Remove}, "remove"},
		{fsnotify.Event{Op: fsnotify.Rename}, "rename"},
		{fsnotify.Event{Op: fsnotify.Chmod}, "chmod"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := wp.getEventType(tt.event)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMatchesRule(t *testing.T) {
	cfg := &config.Config{}
	wp, err := NewWatchProcessor(cfg)
	require.NoError(t, err)
	defer wp.Stop()

	rule := config.WatchRule{
		Name:    "test-rule",
		Paths:   []string{"/test/path", "*.md"},
		Events:  []string{"create", "write"},
		Actions: []string{"test-action"},
	}

	tests := []struct {
		name      string
		filePath  string
		eventType string
		expected  bool
	}{
		{"matching path and event", "/test/path/file.md", "create", true},
		{"matching event, wrong path", "/other/path/file.md", "create", false},
		{"matching path, wrong event", "/test/path/file.md", "remove", false},
		{"glob pattern match", "note.md", "write", true},
		{"no match", "other.txt", "delete", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wp.matchesRule(tt.filePath, tt.eventType, rule)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPathMatches(t *testing.T) {
	cfg := &config.Config{}
	wp, err := NewWatchProcessor(cfg)
	require.NoError(t, err)
	defer wp.Stop()

	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "watch-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.md")
	f, err := os.Create(testFile)
	require.NoError(t, err)
	f.Close()

	tests := []struct {
		name     string
		filePath string
		rulePath string
		expected bool
	}{
		{"exact match", testFile, testFile, true},
		{"directory match", testFile, tempDir, true},
		{"glob match", "test.md", "*.md", true},
		{"no match", testFile, "/other/path", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wp.pathMatches(tt.filePath, tt.rulePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExecuteAction(t *testing.T) {
	cfg := &config.Config{}
	wp, err := NewWatchProcessor(cfg)
	require.NoError(t, err)
	defer wp.Stop()

	tests := []struct {
		name     string
		action   string
		filePath string
		wantErr  bool
	}{
		{
			name:     "valid mdnotes command",
			action:   "mdnotes frontmatter ensure {{file}}",
			filePath: "/test/file.md",
			wantErr:  false,
		},
		{
			name:     "unsupported command",
			action:   "invalid command",
			filePath: "/test/file.md",
			wantErr:  true,
		},
		{
			name:     "empty action",
			action:   "",
			filePath: "/test/file.md",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := wp.executeAction(tt.action, tt.filePath)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStartStop(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "watch-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		Watch: config.WatchConfig{
			Enabled:         true,
			DebounceTimeout: "100ms",
			Rules: []config.WatchRule{
				{
					Name:    "test-rule",
					Paths:   []string{tempDir},
					Events:  []string{"create", "write"},
					Actions: []string{"mdnotes frontmatter ensure {{file}}"},
				},
			},
			IgnorePatterns: []string{".tmp"},
		},
	}

	wp, err := NewWatchProcessor(cfg)
	require.NoError(t, err)

	// Test starting
	err = wp.Start()
	require.NoError(t, err)

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Test stopping
	err = wp.Stop()
	require.NoError(t, err)
}

func TestWatchDisabled(t *testing.T) {
	cfg := &config.Config{
		Watch: config.WatchConfig{
			Enabled: false,
		},
	}

	wp, err := NewWatchProcessor(cfg)
	require.NoError(t, err)
	defer wp.Stop()

	err = wp.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "watch is not enabled")
}

func TestDebounceEvent(t *testing.T) {
	cfg := &config.Config{
		Watch: config.WatchConfig{
			DebounceTimeout: "50ms",
		},
	}

	wp, err := NewWatchProcessor(cfg)
	require.NoError(t, err)
	defer wp.Stop()

	event := fsnotify.Event{
		Name: "test.md",
		Op:   fsnotify.Write,
	}

	// Call debounceEvent
	wp.debounceEvent(event)

	// Check that timer was created
	wp.debounceMutex.Lock()
	_, exists := wp.debounceMap["test.md"]
	wp.debounceMutex.Unlock()

	assert.True(t, exists, "Timer should be created for debouncing")

	// Wait for debounce timeout and a bit more
	time.Sleep(100 * time.Millisecond)

	// Check that timer was cleaned up
	wp.debounceMutex.Lock()
	_, exists = wp.debounceMap["test.md"]
	wp.debounceMutex.Unlock()

	assert.False(t, exists, "Timer should be cleaned up after debounce timeout")
}
