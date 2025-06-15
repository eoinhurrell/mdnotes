package batch

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/eoinhurrell/mdnotes/internal/config"
	"github.com/eoinhurrell/mdnotes/internal/processor"
	"github.com/eoinhurrell/mdnotes/internal/safety"
	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// NewBatchCommand creates the batch command
func NewBatchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batch",
		Short: "Execute batch operations",
		Long:  `Execute multiple operations in batch with transaction support and progress reporting`,
	}

	// Add subcommands
	cmd.AddCommand(newExecuteCommand())
	cmd.AddCommand(newValidateCommand())

	return cmd
}

func newExecuteCommand() *cobra.Command {
	var (
		configFile     string
		progressMode   string
		createBackup   bool
		stopOnError    bool
		maxWorkers     int
	)

	cmd := &cobra.Command{
		Use:   "execute [vault-path]",
		Short: "Execute batch operations",
		Long:  `Execute a series of operations defined in a configuration file`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vaultPath := "."
			if len(args) > 0 {
				vaultPath = args[0]
			}

			// Load configuration
			cfg, err := loadConfig(cmd)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			// Override batch config with command line flags
			if cmd.Flags().Changed("backup") {
				cfg.Batch.CreateBackup = createBackup
			}
			if cmd.Flags().Changed("stop-on-error") {
				cfg.Batch.StopOnError = stopOnError
			}
			if cmd.Flags().Changed("workers") {
				cfg.Batch.MaxWorkers = maxWorkers
			}

			// Set up context for cancellation
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Handle interruption signals
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigChan
				fmt.Println("\nReceived interrupt signal, stopping operations...")
				cancel()
			}()

			// Scan vault files
			scanner := vault.NewScanner(vault.WithIgnorePatterns(cfg.Vault.IgnorePatterns))
			files, err := scanner.Walk(vaultPath)
			if err != nil {
				return fmt.Errorf("scanning vault: %w", err)
			}

			// Initialize batch processor
			batchProcessor := processor.NewBatchProcessor()

			// Set up progress reporter
			var reporter processor.ProgressReporter
			switch progressMode {
			case "json":
				reporter = processor.NewJSONProgress()
				reporter.SetWriter(os.Stdout)
			case "silent":
				reporter = processor.NewSilentProgress()
			default:
				reporter = processor.NewTerminalProgress()
			}

			// Set up safety features
			isDryRun, _ := cmd.Flags().GetBool("dry-run")
			var dryRunRecorder *safety.DryRunRecorder
			if isDryRun {
				dryRunRecorder = safety.NewDryRunRecorder()
			}

			// Execute batch operations
			vaultObj := &processor.Vault{Files: files, Path: vaultPath}
			batchConfig := processor.BatchConfig{
				Operations:   buildOperationsFromConfig(cfg),
				StopOnError:  cfg.Batch.StopOnError,
				CreateBackup: cfg.Batch.CreateBackup,
				DryRun:       isDryRun,
				MaxWorkers:   cfg.Batch.MaxWorkers,
			}
			
			result, err := batchProcessor.Execute(ctx, vaultObj, batchConfig)

			if err != nil {
				return fmt.Errorf("executing batch operations: %w", err)
			}

			// Show dry run results
			if isDryRun && dryRunRecorder != nil {
				fmt.Println("\n" + dryRunRecorder.GenerateReport())
			}

			// Show final results
			if !isDryRun {
				fmt.Printf("\nBatch execution completed:\n")
				fmt.Printf("  Total operations: %d\n", len(result))
				successful := 0
				failed := 0
				for _, op := range result {
					if op.Success {
						successful++
					} else {
						failed++
					}
				}
				fmt.Printf("  Successful: %d operations\n", successful)
				fmt.Printf("  Failed: %d operations\n", failed)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&configFile, "config", "", "Batch configuration file")
	cmd.Flags().StringVar(&progressMode, "progress", "terminal", "Progress reporting mode (terminal, json, silent)")
	cmd.Flags().BoolVar(&createBackup, "backup", true, "Create backup before operations")
	cmd.Flags().BoolVar(&stopOnError, "stop-on-error", false, "Stop on first error")
	cmd.Flags().IntVar(&maxWorkers, "workers", 4, "Maximum number of worker goroutines")

	return cmd
}

func newValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate [config-file]",
		Short: "Validate batch configuration",
		Long:  `Validate a batch configuration file without executing operations`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile := "batch-config.yaml"
			if len(args) > 0 {
				configFile = args[0]
			}

			// Load and validate configuration
			cfg, err := config.LoadConfigFromFile(configFile)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("validating config: %w", err)
			}

			fmt.Printf("Configuration file '%s' is valid.\n", configFile)
			fmt.Printf("Would execute %d operation types.\n", countOperationTypes(cfg))

			return nil
		},
	}

	return cmd
}

func loadConfig(cmd *cobra.Command) (*config.Config, error) {
	configPath, _ := cmd.Flags().GetString("config")
	
	if configPath != "" {
		return config.LoadConfigFromFile(configPath)
	}
	
	return config.LoadConfigWithFallback(config.GetDefaultConfigPaths())
}

func buildOperationsFromConfig(cfg *config.Config) []processor.Operation {
	var operations []processor.Operation

	// Add frontmatter operations if configured
	if len(cfg.Frontmatter.RequiredFields) > 0 {
		operations = append(operations, processor.Operation{
			Name:    "Ensure frontmatter fields",
			Command: "frontmatter.ensure",
			Parameters: map[string]interface{}{
				"fields": cfg.Frontmatter.RequiredFields,
			},
		})
	}

	if len(cfg.Frontmatter.TypeRules.Fields) > 0 {
		operations = append(operations, processor.Operation{
			Name:    "Cast frontmatter types",
			Command: "frontmatter.cast",
			Parameters: map[string]interface{}{
				"type_rules": cfg.Frontmatter.TypeRules.Fields,
			},
		})
	}

	// Add validation operation
	operations = append(operations, processor.Operation{
		Name:    "Validate frontmatter",
		Command: "frontmatter.validate",
		Parameters: map[string]interface{}{
			"required_fields": cfg.Frontmatter.RequiredFields,
			"type_rules":      cfg.Frontmatter.TypeRules.Fields,
		},
	})

	return operations
}

func countOperationTypes(cfg *config.Config) int {
	count := 0
	
	if len(cfg.Frontmatter.RequiredFields) > 0 {
		count++
	}
	if len(cfg.Frontmatter.TypeRules.Fields) > 0 {
		count++
	}
	
	// Always include validation
	count++
	
	return count
}