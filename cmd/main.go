package main

import (
	"fmt"
	"os"

	"github.com/eoinhurrell/mdnotes/cmd/root"
)

// Build-time variables set by goreleaser
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

func main() {
	// Set version information
	rootCmd := root.NewRootCommand()
	rootCmd.Version = buildVersion()

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func buildVersion() string {
	if version == "dev" {
		return "dev (built from source)"
	}

	return fmt.Sprintf("%s\ncommit: %s\nbuilt at: %s\nbuilt by: %s", version, commit, date, builtBy)
}
