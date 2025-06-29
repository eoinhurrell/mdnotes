package watch

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/eoinhurrell/mdnotes/internal/config"
	"github.com/eoinhurrell/mdnotes/internal/processor"
)

// Cmd represents the watch command
var Cmd = &cobra.Command{
	Use:   "watch",
	Short: "Monitor files and automatically execute mdnotes commands",
	Long: `Monitor specified directories for markdown file changes and automatically 
run configured mdnotes commands when files are created, modified, or deleted.

Watch rules are configured in the YAML configuration file:

watch:
  enabled: true
  debounce_timeout: "2s"
  rules:
    - name: "Auto-ensure frontmatter"
      paths: ["./notes/", "./inbox/"]
      events: ["create", "write"]
      actions: ["mdnotes frontmatter ensure {{file}}"]
    - name: "Sync with Linkding"
      paths: ["./notes/"]
      events: ["write"]
      actions: ["mdnotes linkding sync {{file}}"]

The watch command will run in the foreground by default. Use --daemon to run
in the background (requires external process management).`,
	Example: `  # Start watching with default config
  mdnotes watch

  # Start watching with specific config file
  mdnotes watch --config .obsidian-admin.yaml

  # Run in daemon mode (background)
  mdnotes watch --daemon`,
	RunE: runWatch,
}

var (
	configPath string
	daemon     bool
)

func init() {
	Cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to configuration file")
	Cmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "Run in daemon mode (background)")
}

func runWatch(cmd *cobra.Command, args []string) error {
	// Load configuration
	var cfg *config.Config
	var err error

	if configPath != "" {
		cfg, err = config.LoadConfigFromFile(configPath)
		if err != nil {
			return fmt.Errorf("loading config file: %w", err)
		}
	} else {
		// Try default config paths
		cfg, err = config.LoadConfigWithFallback(config.GetDefaultConfigPaths())
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Check if watch is enabled
	if !cfg.Watch.Enabled {
		return fmt.Errorf("watch is not enabled in configuration. Set 'watch.enabled: true' in your config file")
	}

	// Check if any rules are configured
	if len(cfg.Watch.Rules) == 0 {
		return fmt.Errorf("no watch rules configured. Add rules to the 'watch.rules' section in your config file")
	}

	fmt.Printf("Starting file watcher with %d rules...\n", len(cfg.Watch.Rules))
	
	// List configured rules
	for i, rule := range cfg.Watch.Rules {
		fmt.Printf("  Rule %d: %s\n", i+1, rule.Name)
		fmt.Printf("    Paths: %v\n", rule.Paths)
		fmt.Printf("    Events: %v\n", rule.Events)
		fmt.Printf("    Actions: %v\n", rule.Actions)
	}

	// Create and start watch processor
	watchProcessor, err := processor.NewWatchProcessor(cfg)
	if err != nil {
		return fmt.Errorf("creating watch processor: %w", err)
	}

	if err := watchProcessor.Start(); err != nil {
		return fmt.Errorf("starting watch processor: %w", err)
	}

	if daemon {
		fmt.Println("Running in daemon mode. Use Ctrl+C to stop.")
	} else {
		fmt.Println("Watching for file changes. Press Ctrl+C to stop.")
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for signal
	<-sigChan

	fmt.Println("\nShutting down watch processor...")
	if err := watchProcessor.Stop(); err != nil {
		log.Printf("Error stopping watch processor: %v", err)
	}

	fmt.Println("Watch processor stopped.")
	return nil
}