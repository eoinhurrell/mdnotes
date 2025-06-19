package linkding

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/eoinhurrell/mdnotes/internal/config"
	"github.com/eoinhurrell/mdnotes/internal/linkding"
	"github.com/eoinhurrell/mdnotes/internal/processor"
	"github.com/eoinhurrell/mdnotes/internal/vault"
)

// NewLinkdingCommand creates the linkding command
func NewLinkdingCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "linkding",
		Aliases: []string{"ld"},
		Short:   "Sync URLs with Linkding bookmarks",
		Long:    `Synchronize URLs from vault files with your Linkding bookmark manager`,
	}

	// Add subcommands
	cmd.AddCommand(newSyncCommand())
	cmd.AddCommand(newListCommand())

	return cmd
}

func newSyncCommand() *cobra.Command {
	var (
		urlField    string
		titleField  string
		tagsField   string
		syncTitle   bool
		syncTags    bool
		dryRun      bool
		verbose     bool
	)

	cmd := &cobra.Command{
		Use:     "sync [vault-path]",
		Aliases: []string{"s"},
		Short:   "Sync vault URLs to Linkding bookmarks",
		Long: `Sync URLs from vault files to Linkding bookmarks.
Files with 'url' frontmatter field will be synced to Linkding.
The Linkding ID will be stored in the 'linkding_id' field.

Configuration:
  Linkding API URL and token should be configured in .obsidian-admin.yaml:
  
  linkding:
    api_url: "${LINKDING_URL}"
    api_token: "${LINKDING_TOKEN}"
    sync_title: true
    sync_tags: true`,
		Args: cobra.MaximumNArgs(1),
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

			// Validate Linkding configuration
			if cfg.Linkding.APIURL == "" {
				return fmt.Errorf("linkding.api_url not configured")
			}
			if cfg.Linkding.APIToken == "" {
				return fmt.Errorf("linkding.api_token not configured")
			}

			// Create Linkding client
			client := linkding.NewClient(cfg.Linkding.APIURL, cfg.Linkding.APIToken)

			// Scan vault files
			scanner := vault.NewScanner(vault.WithIgnorePatterns(cfg.Vault.IgnorePatterns))
			files, err := scanner.Walk(vaultPath)
			if err != nil {
				return fmt.Errorf("scanning vault: %w", err)
			}

			// Create sync configuration
			syncConfig := processor.LinkdingSyncConfig{
				URLField:    urlField,
				TitleField:  titleField,
				TagsField:   tagsField,
				SyncTitle:   syncTitle || cfg.Linkding.SyncTitle,
				SyncTags:    syncTags || cfg.Linkding.SyncTags,
				DryRun:      dryRun,
			}
			
			// Add progress callback for verbose mode
			if verbose {
				syncConfig.ProgressCallback = func(result processor.SyncResult) {
					switch result.Action {
					case "created":
						fmt.Printf("✓ %s: Created bookmark ID %d\n", result.File.RelativePath, result.BookmarkID)
					case "verified":
						fmt.Printf("✓ %s: Verified bookmark ID %d\n", result.File.RelativePath, result.BookmarkID)
					case "updated":
						fmt.Printf("✓ %s: Updated to bookmark ID %d\n", result.File.RelativePath, result.BookmarkID)
					case "skipped":
						fmt.Printf("- %s: Skipped (no URL)\n", result.File.RelativePath)
					case "error":
						fmt.Printf("✗ %s: Error - %v\n", result.File.RelativePath, result.Error)
					}
				}
			}

			// Create sync processor
			syncProcessor := processor.NewLinkdingSync(syncConfig)
			syncProcessor.SetClient(client)

			// Find files to sync (all files with URLs)
			syncableFiles := syncProcessor.FindAllSyncableFiles(files)
			if len(syncableFiles) == 0 {
				fmt.Println("No files with URLs found.")
				return nil
			}

			if verbose {
				fmt.Printf("Found %d files with URLs to process:\n", len(syncableFiles))
				for _, file := range syncableFiles {
					url := file.Frontmatter[syncConfig.URLField]
					status := "unsynced"
					if linkdingID, exists := file.Frontmatter[syncConfig.IDField]; exists {
						if id, ok := linkdingID.(int); ok && id > 0 {
							status = fmt.Sprintf("synced #%d", id)
						} else if f, ok := linkdingID.(float64); ok && f > 0 {
							status = fmt.Sprintf("synced #%.0f", f)
						}
					}
					fmt.Printf("  %s (%s): %v\n", file.RelativePath, status, url)
				}
				fmt.Println()
			}

			if dryRun {
				fmt.Printf("Would process %d files with URLs (dry run)\n", len(syncableFiles))
				return nil
			}

			// Perform sync
			ctx := context.Background()
			results, err := syncProcessor.SyncBatch(ctx, syncableFiles)
			if err != nil {
				return fmt.Errorf("syncing files: %w", err)
			}

			// Report results summary
			created := 0
			verified := 0
			updated := 0
			skipped := 0
			errors := 0
			for _, result := range results {
				switch result.Action {
				case "created":
					created++
				case "verified":
					verified++
				case "updated":
					updated++
				case "skipped":
					skipped++
				case "error":
					errors++
					// Errors are always shown, even in non-verbose mode
					if !verbose {
						fmt.Printf("✗ %s: Error - %v\n", result.File.RelativePath, result.Error)
					}
				}
			}

			fmt.Printf("\nSync completed: %d created, %d verified, %d updated, %d skipped, %d errors\n", created, verified, updated, skipped, errors)

			// Save files with updated Linkding IDs
			if created > 0 || updated > 0 {
				for _, result := range results {
					if result.Action == "created" || result.Action == "updated" {
						content, err := result.File.Serialize()
						if err != nil {
							fmt.Printf("Warning: Failed to serialize %s: %v\n", result.File.RelativePath, err)
							continue
						}
						if err := os.WriteFile(result.File.Path, content, 0644); err != nil {
							fmt.Printf("Warning: Failed to save %s: %v\n", result.File.RelativePath, err)
						}
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&urlField, "url-field", "url", "Frontmatter field containing the URL")
	cmd.Flags().StringVar(&titleField, "title-field", "title", "Frontmatter field containing the title")
	cmd.Flags().StringVar(&tagsField, "tags-field", "tags", "Frontmatter field containing tags")
	cmd.Flags().BoolVar(&syncTitle, "sync-title", false, "Sync title to Linkding")
	cmd.Flags().BoolVar(&syncTags, "sync-tags", false, "Sync tags to Linkding")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview sync without making changes")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Verbose output")

	return cmd
}

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list [vault-path]",
		Aliases: []string{"l"},
		Short:   "List vault files with URLs",
		Long:  `List vault files that contain URLs and their sync status with Linkding`,
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

			// Scan vault files
			scanner := vault.NewScanner(vault.WithIgnorePatterns(cfg.Vault.IgnorePatterns))
			files, err := scanner.Walk(vaultPath)
			if err != nil {
				return fmt.Errorf("scanning vault: %w", err)
			}

			// Create sync processor to analyze files
			syncConfig := processor.LinkdingSyncConfig{}
			syncProcessor := processor.NewLinkdingSync(syncConfig)

			// Find files with URLs
			var urlFiles []*vault.VaultFile
			for _, file := range files {
				if url, exists := file.Frontmatter["url"]; exists {
					if urlStr, ok := url.(string); ok && urlStr != "" {
						urlFiles = append(urlFiles, file)
					}
				}
			}

			if len(urlFiles) == 0 {
				fmt.Println("No files with URLs found.")
				return nil
			}

			fmt.Printf("Found %d files with URLs:\n\n", len(urlFiles))
			fmt.Printf("%-50s %-10s %s\n", "File", "Status", "URL")
			fmt.Printf("%-50s %-10s %s\n", "----", "------", "---")

			for _, file := range urlFiles {
				url := file.Frontmatter["url"].(string)
				status := "unsynced"
				
				if linkdingID, exists := file.Frontmatter["linkding_id"]; exists {
					if id, ok := linkdingID.(int); ok && id > 0 {
						status = fmt.Sprintf("synced #%d", id)
					} else if f, ok := linkdingID.(float64); ok && f > 0 {
						status = fmt.Sprintf("synced #%.0f", f)
					}
				}

				fileName := file.RelativePath
				if len(fileName) > 47 {
					fileName = fileName[:44] + "..."
				}

				urlDisplay := url
				if len(urlDisplay) > 60 {
					urlDisplay = urlDisplay[:57] + "..."
				}

				fmt.Printf("%-50s %-10s %s\n", fileName, status, urlDisplay)
			}

			// Summary
			unsyncedCount := len(syncProcessor.FindUnsyncedFiles(urlFiles))
			syncedCount := len(urlFiles) - unsyncedCount
			fmt.Printf("\nSummary: %d synced, %d unsynced\n", syncedCount, unsyncedCount)

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