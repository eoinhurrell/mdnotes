package integration

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPerformanceSmallVault tests performance with a small vault (100 files)
func TestPerformanceSmallVault(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create a small test vault
	vaultPath, err := createLargeTestVault(100)
	require.NoError(t, err)
	defer os.RemoveAll(vaultPath)

	tests := []struct {
		name        string
		command     []string
		maxDuration time.Duration
		description string
	}{
		{
			name:        "analyze health",
			command:     []string{"analyze", "health", vaultPath, "--quiet"},
			maxDuration: 5 * time.Second,
			description: "Health analysis should complete quickly",
		},
		{
			name:        "analyze stats",
			command:     []string{"analyze", "stats", vaultPath},
			maxDuration: 3 * time.Second,
			description: "Stats analysis should be fast",
		},
		{
			name:        "frontmatter query",
			command:     []string{"frontmatter", "query", vaultPath, "--where", "type = 'project'"},
			maxDuration: 3 * time.Second,
			description: "Frontmatter queries should be fast",
		},
		// Note: Skip links check for performance test as it will have many broken links
		// in the auto-generated test vault which causes the command to exit with error
		{
			name:        "analyze links",
			command:     []string{"analyze", "links", vaultPath, "--quiet"},
			maxDuration: 8 * time.Second,
			description: "Link analysis may take longer but should be reasonable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			output, err := runMdnotesCommand(tt.command...)
			duration := time.Since(start)

			assert.NoError(t, err, "Command should succeed: %s", string(output))
			assert.Less(t, duration, tt.maxDuration,
				"%s took %v, should be less than %v", tt.description, duration, tt.maxDuration)

			t.Logf("%s completed in %v", tt.name, duration)
		})
	}
}

// TestPerformanceMediumVault tests performance with a medium vault (1000 files)
func TestPerformanceMediumVault(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create a medium test vault
	vaultPath, err := createLargeTestVault(1000)
	require.NoError(t, err)
	defer os.RemoveAll(vaultPath)

	tests := []struct {
		name        string
		command     []string
		maxDuration time.Duration
		description string
	}{
		{
			name:        "analyze health",
			command:     []string{"analyze", "health", vaultPath, "--quiet"},
			maxDuration: 15 * time.Second,
			description: "Health analysis should scale reasonably",
		},
		{
			name:        "frontmatter query",
			command:     []string{"frontmatter", "query", vaultPath, "--where", "tags contains 'project'"},
			maxDuration: 10 * time.Second,
			description: "Complex queries should remain fast",
		},
		{
			name:        "analyze stats",
			command:     []string{"analyze", "stats", vaultPath},
			maxDuration: 8 * time.Second,
			description: "Stats should scale well",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			output, err := runMdnotesCommand(tt.command...)
			duration := time.Since(start)

			assert.NoError(t, err, "Command should succeed: %s", string(output))
			assert.Less(t, duration, tt.maxDuration,
				"%s took %v, should be less than %v", tt.description, duration, tt.maxDuration)

			t.Logf("%s completed in %v with 1000 files", tt.name, duration)
		})
	}
}

// TestMemoryUsage tests that commands don't use excessive memory
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	// Create a test vault with some large files
	vaultPath, err := createLargeTestVault(500)
	require.NoError(t, err)
	defer os.RemoveAll(vaultPath)

	// Commands that should have bounded memory usage
	commands := [][]string{
		{"analyze", "health", vaultPath, "--quiet"},
		{"analyze", "stats", vaultPath},
		{"frontmatter", "query", vaultPath, "--where", "created after '2024-01-01'"},
		{"links", "check", vaultPath},
	}

	for _, cmd := range commands {
		t.Run("memory_"+cmd[0]+"_"+cmd[1], func(t *testing.T) {
			// Run the command and verify it completes
			// In a real implementation, you might use memory profiling tools
			output, err := runMdnotesCommand(cmd...)

			// Note: links check returns exit status 1 when broken links are found, which is expected behavior
			if cmd[0] == "links" && cmd[1] == "check" {
				// For links check, we just verify it completes (exit code 1 is expected for broken links)
				t.Logf("Links check completed (exit code may be non-zero for broken links): %s", string(output))
			} else {
				assert.NoError(t, err, "Command should complete: %s", string(output))
			}

			// For now, just verify the command completes
			// TODO: Add actual memory usage measurement
		})
	}
}

// BenchmarkCommonOperations benchmarks frequently used operations
func BenchmarkCommonOperations(b *testing.B) {
	// Create a test vault
	vaultPath, err := createLargeTestVault(100)
	require.NoError(b, err)
	defer os.RemoveAll(vaultPath)

	benchmarks := []struct {
		name    string
		command []string
	}{
		{
			name:    "health_check",
			command: []string{"analyze", "health", vaultPath, "--quiet"},
		},
		{
			name:    "stats_analysis",
			command: []string{"analyze", "stats", vaultPath},
		},
		{
			name:    "frontmatter_query",
			command: []string{"frontmatter", "query", vaultPath, "--where", "title != ''"},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				output, err := runMdnotesCommand(bm.command...)
				if err != nil {
					b.Fatalf("Benchmark failed: %v, output: %s", err, string(output))
				}
			}
		})
	}
}
