package processor

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// ProgressReporter defines the interface for progress reporting
type ProgressReporter interface {
	Start(total int)
	Update(current int, message string)
	Finish()
	SetWriter(w io.Writer)
}

// TerminalProgress implements progress reporting for terminal output
type TerminalProgress struct {
	total     int
	current   int
	startTime time.Time
	writer    io.Writer
	width     int
	lastLine  string
}

// NewTerminalProgress creates a new terminal progress reporter
func NewTerminalProgress() *TerminalProgress {
	return &TerminalProgress{
		writer: os.Stdout,
		width:  50, // Default progress bar width
	}
}

// Start initializes the progress reporter
func (tp *TerminalProgress) Start(total int) {
	tp.total = total
	tp.current = 0
	tp.startTime = time.Now()
	tp.render("Starting...")
}

// Update updates the progress with current count and message
func (tp *TerminalProgress) Update(current int, message string) {
	tp.current = current
	tp.render(message)
}

// Finish completes the progress reporting
func (tp *TerminalProgress) Finish() {
	tp.current = tp.total
	elapsed := time.Since(tp.startTime)
	tp.render(fmt.Sprintf("Completed in %s", elapsed.Round(time.Millisecond)))
	fmt.Fprintln(tp.writer) // Add final newline
}

// SetWriter sets the output writer
func (tp *TerminalProgress) SetWriter(w io.Writer) {
	tp.writer = w
}

// render draws the progress bar
func (tp *TerminalProgress) render(message string) {
	if tp.total == 0 {
		return
	}

	percentage := float64(tp.current) / float64(tp.total)
	filled := int(float64(tp.width) * percentage)
	
	// Create progress bar
	bar := strings.Repeat("█", filled) + strings.Repeat("░", tp.width-filled)
	
	// Calculate ETA
	eta := ""
	if tp.current > 0 {
		elapsed := time.Since(tp.startTime)
		rate := float64(tp.current) / elapsed.Seconds()
		remaining := tp.total - tp.current
		if rate > 0 {
			etaSeconds := float64(remaining) / rate
			eta = fmt.Sprintf(" ETA: %s", time.Duration(etaSeconds*float64(time.Second)).Round(time.Second))
		}
	}

	// Build the line
	line := fmt.Sprintf("\r[%s] %d/%d (%.1f%%)%s - %s",
		bar, tp.current, tp.total, percentage*100, eta, message)

	// Clear previous line if it was longer
	if len(tp.lastLine) > len(line) {
		fmt.Fprint(tp.writer, "\r"+strings.Repeat(" ", len(tp.lastLine))+"\r")
	}

	fmt.Fprint(tp.writer, line)
	tp.lastLine = line
}

// SilentProgress implements a no-op progress reporter
type SilentProgress struct{}

// NewSilentProgress creates a new silent progress reporter
func NewSilentProgress() *SilentProgress {
	return &SilentProgress{}
}

// Start does nothing for silent progress
func (sp *SilentProgress) Start(total int) {}

// Update does nothing for silent progress
func (sp *SilentProgress) Update(current int, message string) {}

// Finish does nothing for silent progress
func (sp *SilentProgress) Finish() {}

// SetWriter does nothing for silent progress
func (sp *SilentProgress) SetWriter(w io.Writer) {}

// JSONProgress implements JSON-based progress reporting
type JSONProgress struct {
	writer    io.Writer
	startTime time.Time
	total     int
}

// ProgressEvent represents a progress event in JSON format
type ProgressEvent struct {
	Type       string    `json:"type"`
	Timestamp  time.Time `json:"timestamp"`
	Current    int       `json:"current"`
	Total      int       `json:"total"`
	Percentage float64   `json:"percentage"`
	Message    string    `json:"message"`
	Elapsed    string    `json:"elapsed,omitempty"`
}

// NewJSONProgress creates a new JSON progress reporter
func NewJSONProgress() *JSONProgress {
	return &JSONProgress{
		writer: os.Stdout,
	}
}

// Start initializes JSON progress reporting
func (jp *JSONProgress) Start(total int) {
	jp.total = total
	jp.startTime = time.Now()
	jp.emit(ProgressEvent{
		Type:      "start",
		Timestamp: jp.startTime,
		Total:     total,
		Message:   "Starting operation",
	})
}

// Update emits a progress update event
func (jp *JSONProgress) Update(current int, message string) {
	percentage := float64(current) / float64(jp.total) * 100
	jp.emit(ProgressEvent{
		Type:       "progress",
		Timestamp:  time.Now(),
		Current:    current,
		Total:      jp.total,
		Percentage: percentage,
		Message:    message,
		Elapsed:    time.Since(jp.startTime).String(),
	})
}

// Finish emits the completion event
func (jp *JSONProgress) Finish() {
	elapsed := time.Since(jp.startTime)
	jp.emit(ProgressEvent{
		Type:       "complete",
		Timestamp:  time.Now(),
		Current:    jp.total,
		Total:      jp.total,
		Percentage: 100.0,
		Message:    "Operation completed",
		Elapsed:    elapsed.String(),
	})
}

// SetWriter sets the output writer
func (jp *JSONProgress) SetWriter(w io.Writer) {
	jp.writer = w
}

// emit writes a progress event as JSON
func (jp *JSONProgress) emit(event ProgressEvent) {
	// In a real implementation, this would use json.Marshal
	// For simplicity, we'll format manually
	fmt.Fprintf(jp.writer, `{"type":"%s","timestamp":"%s","current":%d,"total":%d,"percentage":%.1f,"message":"%s"}%s`,
		event.Type, event.Timestamp.Format(time.RFC3339),
		event.Current, event.Total, event.Percentage, event.Message, "\n")
}

// ProgressOptions configures progress reporting
type ProgressOptions struct {
	Type   string // "terminal", "json", "silent"
	Writer io.Writer
	Width  int // For terminal progress bar
}

// NewProgressReporter creates a progress reporter based on options
func NewProgressReporter(opts ProgressOptions) ProgressReporter {
	switch opts.Type {
	case "json":
		reporter := NewJSONProgress()
		if opts.Writer != nil {
			reporter.SetWriter(opts.Writer)
		}
		return reporter
	case "silent":
		return NewSilentProgress()
	default: // "terminal" or empty
		reporter := NewTerminalProgress()
		if opts.Writer != nil {
			reporter.SetWriter(opts.Writer)
		}
		if opts.Width > 0 {
			reporter.width = opts.Width
		}
		return reporter
	}
}