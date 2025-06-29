package linkding

import (
	"context"
	"fmt"
	"os"

	"github.com/eoinhurrell/mdnotes/cmd/root"
	"github.com/eoinhurrell/mdnotes/internal/config"
	"github.com/eoinhurrell/mdnotes/internal/linkding"
	"github.com/eoinhurrell/mdnotes/internal/processor"
	"github.com/eoinhurrell/mdnotes/internal/selector"
	"github.com/eoinhurrell/mdnotes/internal/vault"
	"github.com/spf13/cobra"
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
	cmd.AddCommand(newGetCommand())

	return cmd
}

func newSyncCommand() *cobra.Command {
	var (
		urlField         string
		titleField       string
		tagsField        string
		syncTitle        bool
		syncTags         bool
		skipVerification bool
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

			// Get flags from persistent flags
			dryRun, _ := cmd.Root().PersistentFlags().GetBool("dry-run")
			verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
			quiet, _ := cmd.Root().PersistentFlags().GetBool("quiet")

			// Override verbose if quiet is specified
			if quiet {
				verbose = false
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

			// Get file selection configuration from global flags
			mode, fileSelector, err := root.GetGlobalSelectionConfig(cmd)
			if err != nil {
				return fmt.Errorf("getting file selection config: %w", err)
			}
			
			// Merge config ignore patterns with global ignore patterns if needed
			if len(fileSelector.IgnorePatterns) == 0 {
				fileSelector = fileSelector.WithIgnorePatterns(cfg.Vault.IgnorePatterns)
			}
			
			// Select files using unified architecture
			selection, err := fileSelector.SelectFiles(vaultPath, mode)
			if err != nil {
				return fmt.Errorf("selecting files: %w", err)
			}
			
			// Print selection summary if verbose
			if verbose {
				fmt.Printf("%s\n", selection.GetSelectionSummary())
			}
			
			// Print parse errors if any
			if len(selection.ParseErrors) > 0 && verbose {
				selection.PrintParseErrors()
			}
			
			files := selection.Files

			// Create sync configuration
			syncConfig := processor.LinkdingSyncConfig{
				URLField:         urlField,
				IDField:          "linkding_id", // Default ID field
				TitleField:       titleField,
				TagsField:        tagsField,
				SyncTitle:        syncTitle || cfg.Linkding.SyncTitle,
				SyncTags:         syncTags || cfg.Linkding.SyncTags,
				DryRun:           dryRun,
				SkipVerification: skipVerification,
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
				if !quiet {
					fmt.Println("No files with URLs found.")
				}
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
				fmt.Printf("Dry run: analyzing what would be synced...\n\n")

				// Show what would be done for each file
				for _, file := range syncableFiles {
					url := file.Frontmatter[syncConfig.URLField]

					// Check if file already has linkding_id
					if linkdingID, exists := file.Frontmatter[syncConfig.IDField]; exists {
						if id, ok := linkdingID.(int); ok && id > 0 {
							fmt.Printf("Would verify: %s - Bookmark ID %d\n", file.RelativePath, id)
						} else if f, ok := linkdingID.(float64); ok && f > 0 {
							fmt.Printf("Would verify: %s - Bookmark ID %.0f\n", file.RelativePath, f)
						} else {
							fmt.Printf("Would create: %s - New bookmark for %v\n", file.RelativePath, url)
						}
					} else {
						fmt.Printf("Would create: %s - New bookmark for %v\n", file.RelativePath, url)
					}
				}

				fmt.Printf("\nDry run completed. Would process %d files with URLs.\n", len(syncableFiles))
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

			if !quiet {
				fmt.Printf("\nSync completed: %d created, %d verified, %d updated, %d skipped, %d errors\n", created, verified, updated, skipped, errors)
			}

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
	cmd.Flags().BoolVar(&skipVerification, "skip-verification", false, "Only sync new items, skip verification of existing bookmarks")

	return cmd
}

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list [vault-path]",
		Aliases: []string{"l"},
		Short:   "List vault files with URLs",
		Long:    `List vault files that contain URLs and their sync status with Linkding`,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vaultPath := "."
			if len(args) > 0 {
				vaultPath = args[0]
			}

			// Get flags from persistent flags
			quiet, _ := cmd.Root().PersistentFlags().GetBool("quiet")
			verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")

			// Load configuration
			cfg, err := loadConfig(cmd)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			// Get file selection configuration from global flags
			mode, fileSelector, err := root.GetGlobalSelectionConfig(cmd)
			if err != nil {
				return fmt.Errorf("getting file selection config: %w", err)
			}
			
			// Merge config ignore patterns with global ignore patterns if needed
			if len(fileSelector.IgnorePatterns) == 0 {
				fileSelector = fileSelector.WithIgnorePatterns(cfg.Vault.IgnorePatterns)
			}
			
			// Select files using unified architecture
			selection, err := fileSelector.SelectFiles(vaultPath, mode)
			if err != nil {
				return fmt.Errorf("selecting files: %w", err)
			}
			
			// Print selection summary if verbose
			if verbose {
				fmt.Printf("%s\n", selection.GetSelectionSummary())
			}
			
			// Print parse errors if any
			if len(selection.ParseErrors) > 0 && verbose {
				selection.PrintParseErrors()
			}
			
			files := selection.Files

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
				if !quiet {
					fmt.Println("No files with URLs found.")
				}
				return nil
			}

			if !quiet {
				fmt.Printf("Found %d files with URLs:\n\n", len(urlFiles))
				fmt.Printf("%-50s %-10s %s\n", "File", "Status", "URL")
				fmt.Printf("%-50s %-10s %s\n", "----", "------", "---")
			}

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

				if !quiet {
					fmt.Printf("%-50s %-10s %s\n", fileName, status, urlDisplay)
				}
			}

			// Summary
			unsyncedCount := len(syncProcessor.FindUnsyncedFiles(urlFiles))
			syncedCount := len(urlFiles) - unsyncedCount
			if !quiet {
				fmt.Printf("\nSummary: %d synced, %d unsynced\n", syncedCount, unsyncedCount)
			}

			return nil
		},
	}

	return cmd
}

func newGetCommand() *cobra.Command {
	var (
		maxSize uint64
		timeout string
		tmpDir  string
	)

	cmd := &cobra.Command{
		Use:     "get <path-to-note.md>",
		Aliases: []string{"g"},
		Short:   "Get HTML snapshot or live content from a note's Linkding bookmark",
		Long: `Get HTML snapshot or live content from a note's Linkding bookmark.

Given a note with a 'linkding_id' frontmatter field, this command will:
1. Query the Linkding API for any HTML "snapshot" assets
2. If found, download the latest snapshot, extract text, and print to stdout
3. If no snapshots exist, fetch the live URL from frontmatter and extract text

The note must have either:
- linkding_id: 123 (for snapshot retrieval)
- url: "https://example.com" (for live fallback)

Configuration:
  Linkding API URL and token should be configured in .obsidian-admin.yaml:
  
  linkding:
    api_url: "${LINKDING_URL}"
    api_token: "${LINKDING_TOKEN}"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			notePath := args[0]
			
			// Get flags from persistent flags
			dryRun, _ := cmd.Root().PersistentFlags().GetBool("dry-run")
			verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
			quiet, _ := cmd.Root().PersistentFlags().GetBool("quiet")

			// Override verbose if quiet is specified
			if quiet {
				verbose = false
			}

			if dryRun {
				if !quiet {
					fmt.Printf("Dry run: Would process note %s\n", notePath)
				}
				
				// Parse note to extract linkding_id and url
				vaultFile, err := vault.LoadVaultFile(notePath)
				if err != nil {
					return fmt.Errorf("loading note: %w", err)
				}

				linkdingID, hasID := vaultFile.Frontmatter["linkding_id"]
				url, hasURL := vaultFile.Frontmatter["url"]

				if !hasID && !hasURL {
					return fmt.Errorf("note must have either 'linkding_id' or 'url' frontmatter field")
				}

				if hasID {
					fmt.Printf("Found linkding_id: %v\n", linkdingID)
					fmt.Println("Would query Linkding API for snapshot assets")
				}
				
				if hasURL {
					fmt.Printf("Found fallback url: %v\n", url)
					fmt.Println("Would use as fallback if no snapshots available")
				}

				return nil
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

			// Parse note to extract linkding_id and url
			vaultFile, err := vault.LoadVaultFile(notePath)
			if err != nil {
				return fmt.Errorf("loading note: %w", err)
			}

			linkdingID, hasID := vaultFile.Frontmatter["linkding_id"]
			url, hasURL := vaultFile.Frontmatter["url"]

			if !hasID && !hasURL {
				return fmt.Errorf("note must have either 'linkding_id' or 'url' frontmatter field")
			}

			// Create Linkding client
			client := linkding.NewClient(cfg.Linkding.APIURL, cfg.Linkding.APIToken)

			// Create get processor
			getProcessor := processor.NewLinkdingGet(processor.LinkdingGetConfig{
				MaxSize: maxSize,
				Timeout: timeout,
				TmpDir:  tmpDir,
				Verbose: verbose,
			})
			getProcessor.SetClient(client)

			// Process the note
			ctx := context.Background()
			text, err := getProcessor.GetContent(ctx, linkdingID, url)
			if err != nil {
				return fmt.Errorf("getting content: %w", err)
			}

			// Print the extracted text to stdout
			fmt.Print(text)
			
			return nil
		},
	}

	cmd.Flags().Uint64Var(&maxSize, "max-size", 1000000, "Maximum bytes to fetch from live URL")
	cmd.Flags().StringVar(&timeout, "timeout", "10s", "Request timeout")
	cmd.Flags().StringVar(&tmpDir, "tmp-dir", "", "Where to store downloaded asset (default: OS temp)")

	return cmd
}

func loadConfig(cmd *cobra.Command) (*config.Config, error) {
	configPath, _ := cmd.Flags().GetString("config")

	if configPath != "" {
		return config.LoadConfigFromFile(configPath)
	}

	return config.LoadConfigWithFallback(config.GetDefaultConfigPaths())
}
