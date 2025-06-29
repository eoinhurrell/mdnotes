package processor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewExportProgressReporter(t *testing.T) {
	tests := []struct {
		name    string
		quiet   bool
		verbose bool
	}{
		{
			name:    "Default settings",
			quiet:   false,
			verbose: false,
		},
		{
			name:    "Quiet mode",
			quiet:   true,
			verbose: false,
		},
		{
			name:    "Verbose mode",
			quiet:   false,
			verbose: true,
		},
		{
			name:    "Quiet overrides verbose",
			quiet:   true,
			verbose: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter := NewExportProgressReporter(tt.quiet, tt.verbose)

			assert.Equal(t, tt.quiet, reporter.quiet)
			assert.Equal(t, tt.verbose, reporter.verbose)
			assert.NotNil(t, reporter.reporter)
		})
	}
}

func TestExportProgressReporter_StartPhase(t *testing.T) {
	reporter := NewExportProgressReporter(false, false)

	// Should not panic
	reporter.StartPhase(10, "Starting test phase")

	// Test with different scenarios
	reporter.StartPhase(0, "Phase with no total")
	reporter.StartPhase(100, "")
}

func TestExportProgressReporter_UpdatePhase(t *testing.T) {
	reporter := NewExportProgressReporter(false, true) // verbose mode

	reporter.StartPhase(10, "Starting test")

	// Should not panic
	reporter.UpdatePhase(5, "Halfway through")
	reporter.UpdatePhase(10, "Complete")

	// Test edge cases
	reporter.UpdatePhase(0, "")
	reporter.UpdatePhase(-1, "Negative progress")
	reporter.UpdatePhase(15, "Beyond total")
}

func TestExportProgressReporter_FinishPhase(t *testing.T) {
	reporter := NewExportProgressReporter(false, false)

	// Should not panic
	reporter.FinishPhase("Phase completed successfully")
	reporter.FinishPhase("")

	// Test quiet mode
	quietReporter := NewExportProgressReporter(true, false)
	quietReporter.FinishPhase("Should be quiet")
}

func TestExportProgressReporter_Integration(t *testing.T) {
	// Test the complete flow of progress reporting
	reporter := NewExportProgressReporter(false, true)

	// Start phase
	reporter.StartPhase(3, "🔍 Starting file processing")

	// Simulate processing files
	reporter.UpdatePhase(1, "Processing file 1")
	reporter.UpdatePhase(2, "Processing file 2")
	reporter.UpdatePhase(3, "Processing file 3")

	// Finish phase
	reporter.FinishPhase("✅ Processing completed successfully")
}

func TestExportProgressReporter_QuietMode(t *testing.T) {
	// Test that quiet mode properly suppresses output
	reporter := NewExportProgressReporter(true, true) // quiet overrides verbose

	// These should not produce output but also should not panic
	reporter.StartPhase(10, "Should be quiet")
	reporter.UpdatePhase(5, "Should be quiet")
	reporter.FinishPhase("Should be quiet")
}

func TestExportProgressReporter_VerboseMode(t *testing.T) {
	// Test verbose mode behavior
	reporter := NewExportProgressReporter(false, true)

	reporter.StartPhase(5, "Verbose processing")

	// Verbose mode should pass messages to underlying reporter
	reporter.UpdatePhase(1, "Detailed message 1")
	reporter.UpdatePhase(2, "Detailed message 2")

	reporter.FinishPhase("Verbose completion")
}

func TestExportProgressReporter_NonVerboseMode(t *testing.T) {
	// Test non-verbose mode behavior
	reporter := NewExportProgressReporter(false, false)

	reporter.StartPhase(5, "Non-verbose processing")

	// Non-verbose mode should pass empty messages to underlying reporter
	reporter.UpdatePhase(1, "This message should be ignored")
	reporter.UpdatePhase(2, "Another ignored message")

	reporter.FinishPhase("Non-verbose completion")
}
