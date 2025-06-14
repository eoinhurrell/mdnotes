package processor

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTerminalProgress(t *testing.T) {
	var buf bytes.Buffer
	progress := NewTerminalProgress()
	progress.SetWriter(&buf)
	progress.width = 20 // Smaller width for testing

	// Test start
	progress.Start(10)
	output := buf.String()
	assert.Contains(t, output, "[")
	assert.Contains(t, output, "0/10")
	assert.Contains(t, output, "Starting...")

	// Test update
	buf.Reset()
	progress.Update(5, "Processing file 5")
	output = buf.String()
	assert.Contains(t, output, "5/10")
	assert.Contains(t, output, "50.0%")
	assert.Contains(t, output, "Processing file 5")

	// Test finish
	buf.Reset()
	progress.Finish()
	output = buf.String()
	assert.Contains(t, output, "10/10")
	assert.Contains(t, output, "100.0%")
	assert.Contains(t, output, "Completed")
}

func TestTerminalProgress_ProgressBar(t *testing.T) {
	var buf bytes.Buffer
	progress := NewTerminalProgress()
	progress.SetWriter(&buf)
	progress.width = 10

	progress.Start(10)
	buf.Reset()

	// Test 50% progress
	progress.Update(5, "Half done")
	output := buf.String()
	
	// Should have some filled characters and some empty
	assert.Contains(t, output, "█")
	assert.Contains(t, output, "░")
}

func TestSilentProgress(t *testing.T) {
	var buf bytes.Buffer
	progress := NewSilentProgress()
	progress.SetWriter(&buf)

	// All operations should produce no output
	progress.Start(10)
	progress.Update(5, "test")
	progress.Finish()

	assert.Empty(t, buf.String())
}

func TestJSONProgress(t *testing.T) {
	var buf bytes.Buffer
	progress := NewJSONProgress()
	progress.SetWriter(&buf)

	progress.Start(5)
	progress.Update(2, "Processing item 2")
	progress.Finish()

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	
	// Should have 3 lines (start, progress, complete)
	assert.Len(t, lines, 3)

	// Check start event
	assert.Contains(t, lines[0], `"type":"start"`)
	assert.Contains(t, lines[0], `"total":5`)

	// Check progress event
	assert.Contains(t, lines[1], `"type":"progress"`)
	assert.Contains(t, lines[1], `"current":2`)
	assert.Contains(t, lines[1], `"percentage":40.0`)
	assert.Contains(t, lines[1], `"Processing item 2"`)

	// Check complete event
	assert.Contains(t, lines[2], `"type":"complete"`)
	assert.Contains(t, lines[2], `"percentage":100.0`)
}

func TestNewProgressReporter(t *testing.T) {
	tests := []struct {
		name     string
		opts     ProgressOptions
		expected string
	}{
		{
			name: "terminal reporter",
			opts: ProgressOptions{Type: "terminal"},
			expected: "*processor.TerminalProgress",
		},
		{
			name: "json reporter",
			opts: ProgressOptions{Type: "json"},
			expected: "*processor.JSONProgress",
		},
		{
			name: "silent reporter",
			opts: ProgressOptions{Type: "silent"},
			expected: "*processor.SilentProgress",
		},
		{
			name: "default to terminal",
			opts: ProgressOptions{Type: ""},
			expected: "*processor.TerminalProgress",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter := NewProgressReporter(tt.opts)
			typeName := fmt.Sprintf("%T", reporter)
			assert.Contains(t, typeName, tt.expected)
		})
	}
}

func TestTerminalProgress_ETA(t *testing.T) {
	var buf bytes.Buffer
	progress := NewTerminalProgress()
	progress.SetWriter(&buf)
	progress.width = 10

	progress.Start(10)
	
	// Simulate some time passing
	progress.startTime = time.Now().Add(-2 * time.Second)
	
	buf.Reset()
	progress.Update(2, "Test")
	output := buf.String()
	
	// Should contain ETA
	assert.Contains(t, output, "ETA:")
}

func TestTerminalProgress_ZeroTotal(t *testing.T) {
	var buf bytes.Buffer
	progress := NewTerminalProgress()
	progress.SetWriter(&buf)

	// Should handle zero total gracefully
	progress.Start(0)
	progress.Update(0, "Test")
	progress.Finish()

	// Should not panic and produce some output
	assert.NotEmpty(t, buf.String())
}

func TestProgressOptions_CustomWriter(t *testing.T) {
	var buf bytes.Buffer
	
	opts := ProgressOptions{
		Type:   "terminal",
		Writer: &buf,
		Width:  15,
	}
	
	reporter := NewProgressReporter(opts)
	reporter.Start(5)
	reporter.Update(1, "Test")
	
	assert.NotEmpty(t, buf.String())
	assert.Contains(t, buf.String(), "1/5")
}

