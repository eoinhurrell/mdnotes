package processor

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/eoinhurrell/mdnotes/internal/config"
)

// WatchProcessor monitors file system changes and executes configured actions
type WatchProcessor struct {
	config        *config.Config
	watcher       *fsnotify.Watcher
	eventChan     chan fsnotify.Event
	debounceMap   map[string]*time.Timer
	debounceMutex sync.Mutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewWatchProcessor creates a new watch processor
func NewWatchProcessor(cfg *config.Config) (*WatchProcessor, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating file watcher: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	wp := &WatchProcessor{
		config:      cfg,
		watcher:     watcher,
		eventChan:   make(chan fsnotify.Event, 100),
		debounceMap: make(map[string]*time.Timer),
		ctx:         ctx,
		cancel:      cancel,
	}

	return wp, nil
}

// Start begins watching configured paths
func (wp *WatchProcessor) Start() error {
	if !wp.config.Watch.Enabled {
		return fmt.Errorf("watch is not enabled in configuration")
	}

	// Add all configured paths to the watcher
	for _, rule := range wp.config.Watch.Rules {
		for _, path := range rule.Paths {
			if err := wp.addPath(path); err != nil {
				return fmt.Errorf("adding path %s: %w", path, err)
			}
		}
	}

	// Start the event processing goroutine
	go wp.processEvents()

	log.Printf("Watch processor started with %d rules", len(wp.config.Watch.Rules))
	return nil
}

// Stop stops the watch processor
func (wp *WatchProcessor) Stop() error {
	wp.cancel()
	return wp.watcher.Close()
}

// addPath adds a path to the watcher, handling both files and directories
func (wp *WatchProcessor) addPath(path string) error {
	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path does not exist: %w", err)
	}

	if info.IsDir() {
		// For directories, add recursively
		return filepath.Walk(path, func(walkPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() && !wp.shouldIgnore(walkPath) {
				return wp.watcher.Add(walkPath)
			}
			return nil
		})
	} else {
		// For files, add the parent directory
		return wp.watcher.Add(filepath.Dir(path))
	}
}

// shouldIgnore checks if a path should be ignored based on ignore patterns
func (wp *WatchProcessor) shouldIgnore(path string) bool {
	for _, pattern := range wp.config.Watch.IgnorePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
		if strings.Contains(path, strings.TrimSuffix(pattern, "/*")) {
			return true
		}
	}
	return false
}

// processEvents processes file system events with debouncing
func (wp *WatchProcessor) processEvents() {
	for {
		select {
		case <-wp.ctx.Done():
			return
		case event := <-wp.watcher.Events:
			if wp.shouldIgnore(event.Name) {
				continue
			}

			// Only process markdown files
			if !strings.HasSuffix(strings.ToLower(event.Name), ".md") {
				continue
			}

			wp.debounceEvent(event)

		case err := <-wp.watcher.Errors:
			log.Printf("Watch error: %v", err)
		}
	}
}

// debounceEvent debounces file system events to avoid processing rapid changes
func (wp *WatchProcessor) debounceEvent(event fsnotify.Event) {
	wp.debounceMutex.Lock()
	defer wp.debounceMutex.Unlock()

	// Cancel existing timer for this file
	if timer, exists := wp.debounceMap[event.Name]; exists {
		timer.Stop()
	}

	// Parse debounce timeout
	timeout, err := time.ParseDuration(wp.config.Watch.DebounceTimeout)
	if err != nil {
		timeout = 2 * time.Second // Default fallback
	}

	// Create new timer
	wp.debounceMap[event.Name] = time.AfterFunc(timeout, func() {
		wp.debounceMutex.Lock()
		delete(wp.debounceMap, event.Name)
		wp.debounceMutex.Unlock()

		wp.executeActions(event)
	})
}

// executeActions executes configured actions for a file system event
func (wp *WatchProcessor) executeActions(event fsnotify.Event) {
	eventType := wp.getEventType(event)

	for _, rule := range wp.config.Watch.Rules {
		if wp.matchesRule(event.Name, eventType, rule) {
			for _, action := range rule.Actions {
				if err := wp.executeAction(action, event.Name); err != nil {
					log.Printf("Error executing action '%s' for file '%s': %v", action, event.Name, err)
				}
			}
		}
	}
}

// getEventType converts fsnotify event to string
func (wp *WatchProcessor) getEventType(event fsnotify.Event) string {
	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		return "create"
	case event.Op&fsnotify.Write == fsnotify.Write:
		return "write"
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		return "remove"
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		return "rename"
	case event.Op&fsnotify.Chmod == fsnotify.Chmod:
		return "chmod"
	default:
		return "unknown"
	}
}

// matchesRule checks if an event matches a watch rule
func (wp *WatchProcessor) matchesRule(filePath, eventType string, rule config.WatchRule) bool {
	// Check if event type matches
	eventMatches := false
	for _, ruleEvent := range rule.Events {
		if ruleEvent == eventType {
			eventMatches = true
			break
		}
	}
	if !eventMatches {
		return false
	}

	// Check if path matches
	for _, rulePath := range rule.Paths {
		if wp.pathMatches(filePath, rulePath) {
			return true
		}
	}

	return false
}

// pathMatches checks if a file path matches a rule path pattern
func (wp *WatchProcessor) pathMatches(filePath, rulePath string) bool {
	// Convert to absolute paths for comparison
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		absFilePath = filePath
	}

	absRulePath, err := filepath.Abs(rulePath)
	if err != nil {
		absRulePath = rulePath
	}

	// Check if file is within the rule directory
	if strings.HasPrefix(absFilePath, absRulePath) {
		return true
	}

	// Check glob pattern match
	if matched, _ := filepath.Match(rulePath, filePath); matched {
		return true
	}

	return false
}

// executeAction executes a single action command
func (wp *WatchProcessor) executeAction(action, filePath string) error {
	// Replace {{file}} placeholder with actual file path
	action = strings.ReplaceAll(action, "{{file}}", filePath)
	action = strings.ReplaceAll(action, "{{dir}}", filepath.Dir(filePath))
	action = strings.ReplaceAll(action, "{{basename}}", filepath.Base(filePath))

	log.Printf("Executing action: %s", action)

	// Parse action command
	parts := strings.Fields(action)
	if len(parts) == 0 {
		return fmt.Errorf("empty action command")
	}

	// For now, we'll support basic mdnotes commands
	// In a full implementation, this could be extended to support shell commands
	if parts[0] == "mdnotes" {
		return wp.executeMdnotesCommand(parts[1:], filePath)
	}

	return fmt.Errorf("unsupported action command: %s", parts[0])
}

// executeMdnotesCommand executes an mdnotes command
func (wp *WatchProcessor) executeMdnotesCommand(args []string, filePath string) error {
	if len(args) == 0 {
		return fmt.Errorf("no mdnotes command specified")
	}

	log.Printf("Would execute mdnotes command: %v for file: %s", args, filePath)

	// In a full implementation, this would use the actual command processors
	// For now, we'll just log the action
	switch args[0] {
	case "frontmatter":
		if len(args) > 1 && args[1] == "ensure" {
			log.Printf("Would ensure frontmatter for: %s", filePath)
		}
	case "linkding":
		if len(args) > 1 && args[1] == "sync" {
			log.Printf("Would sync linkding for: %s", filePath)
		}
	default:
		log.Printf("Unsupported mdnotes command: %s", args[0])
	}

	return nil
}
