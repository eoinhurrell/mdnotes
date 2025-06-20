package root

import (
	"os"
	"strings"

	"github.com/eoinhurrell/mdnotes/cmd/analyze"
	"github.com/eoinhurrell/mdnotes/cmd/frontmatter"
	"github.com/eoinhurrell/mdnotes/cmd/headings"
	"github.com/eoinhurrell/mdnotes/cmd/linkding"
	"github.com/eoinhurrell/mdnotes/cmd/links"
	"github.com/eoinhurrell/mdnotes/cmd/profile"
	"github.com/eoinhurrell/mdnotes/cmd/rename"
	"github.com/eoinhurrell/mdnotes/cmd/watch"
	"github.com/spf13/cobra"
)

// NewRootCommand creates the root command for mdnotes
func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mdnotes",
		Short: "A CLI tool for managing Obsidian markdown notes",
		Long: `mdnotes is a command-line tool designed to automate and standardize 
administrative tasks for Obsidian vaults. It provides powerful operations 
for managing frontmatter, headings, links, and file organization.`,
		Version: "1.0.0",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// Add global flags
	cmd.PersistentFlags().Bool("dry-run", false, "Preview changes without applying them; shows exactly what would be changed")
	cmd.PersistentFlags().Bool("verbose", false, "Detailed output; prints filepath of every file examined and actions taken")
	cmd.PersistentFlags().Bool("quiet", false, "Suppress all output except errors and final summary; overrides --verbose")
	cmd.PersistentFlags().String("config", "", "Config file (default: .obsidian-admin.yaml)")

	// Add subcommands
	cmd.AddCommand(analyze.NewAnalyzeCommand())
	cmd.AddCommand(frontmatter.NewFrontmatterCommand())
	cmd.AddCommand(headings.NewHeadingsCommand())
	cmd.AddCommand(links.NewLinksCommand())
	cmd.AddCommand(linkding.NewLinkdingCommand())
	cmd.AddCommand(profile.NewProfileCommand())
	cmd.AddCommand(rename.NewRenameCommand())
	cmd.AddCommand(watch.Cmd)

	// Add ultra-short global shortcuts for most common commands
	cmd.AddCommand(newEnsureShortcut())
	cmd.AddCommand(newSetShortcut())
	cmd.AddCommand(newQueryShortcut())
	cmd.AddCommand(newFixShortcut())
	cmd.AddCommand(newCheckShortcut())

	// Add completion command for generating shell completions
	cmd.AddCommand(newCompletionCommand())

	// Set up custom completions for all commands and flags
	setupCustomCompletions(cmd)

	return cmd
}

// newCompletionCommand creates the completion command
func newCompletionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate completion script",
		Long: `To load completions:

Bash:

  $ source <(mdnotes completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ mdnotes completion bash > /etc/bash_completion.d/mdnotes
  # macOS:
  $ mdnotes completion bash > /usr/local/etc/bash_completion.d/mdnotes

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ mdnotes completion zsh > "${fpath[1]}/_mdnotes"

  # You will need to start a new shell for this setup to take effect.

fish:

  $ mdnotes completion fish | source

  # To load completions for each session, execute once:
  $ mdnotes completion fish > ~/.config/fish/completions/mdnotes.fish

PowerShell:

  PS> mdnotes completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> mdnotes completion powershell > mdnotes.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
		},
	}

	return cmd
}

// setupCustomCompletions adds custom completion functions for file paths and other common arguments
func setupCustomCompletions(cmd *cobra.Command) {
	// Set completion for config files globally
	cmd.RegisterFlagCompletionFunc("config", CompleteConfigFiles)

	// Add completion for commands that need path arguments
	for _, subCmd := range cmd.Commands() {
		switch subCmd.Name() {
		case "frontmatter", "headings", "links", "analyze":
			// These commands take vault/directory paths
			subCmd.ValidArgsFunction = CompleteDirs

		case "rename":
			// Rename takes a source file as first argument
			subCmd.ValidArgsFunction = CompleteMarkdownFiles

		case "linkding":
			// Linkding takes vault paths
			subCmd.ValidArgsFunction = CompleteDirs

		case "e", "s", "f", "c":
			// Global shortcuts for path-based commands
			subCmd.ValidArgsFunction = CompleteDirs
		}

		// Add completion for common flags across commands
		subCmd.RegisterFlagCompletionFunc("config", CompleteConfigFiles)
		subCmd.RegisterFlagCompletionFunc("ignore", CompleteIgnorePatterns)
		subCmd.RegisterFlagCompletionFunc("format", CompleteOutputFormats)
		
		// Add completion for global shortcuts
		if subCmd.Name() == "e" {
			// Global ensure shortcut
			subCmd.RegisterFlagCompletionFunc("field", CompleteFrontmatterFields)
			subCmd.RegisterFlagCompletionFunc("type", CompleteFieldTypesWithFormat)
			subCmd.RegisterFlagCompletionFunc("default", CompleteDefaultValues)
		} else if subCmd.Name() == "s" {
			// Global set shortcut
			subCmd.RegisterFlagCompletionFunc("field", CompleteFrontmatterFields)
			subCmd.RegisterFlagCompletionFunc("type", CompleteFieldTypesWithFormat)
			subCmd.RegisterFlagCompletionFunc("value", CompleteDefaultValues)
		}

		// Add specific completions for different command types
		switch subCmd.Name() {
		case "frontmatter":
			setupFrontmatterCompletions(subCmd)
		case "rename":
			setupRenameCompletions(subCmd)
		case "analyze":
			setupAnalyzeCompletions(subCmd)
		case "links":
			setupLinksCompletions(subCmd)
		case "linkding":
			setupLinkdingCompletions(subCmd)
		}
	}
}

// CompleteDirs provides directory completion
func CompleteDirs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveFilterDirs
}

// CompleteMarkdownFiles provides markdown file completion
func CompleteMarkdownFiles(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"md", "markdown"}, cobra.ShellCompDirectiveFilterFileExt
}

// CompleteConfigFiles provides config file completion
func CompleteConfigFiles(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"yaml", "yml"}, cobra.ShellCompDirectiveFilterFileExt
}

// CompleteIgnorePatterns provides completion for ignore patterns
func CompleteIgnorePatterns(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	patterns := []string{
		".obsidian/*",
		"*.tmp",
		"*.bak",
		".DS_Store",
		"*.swp",
		"*.swo",
		"node_modules/*",
		".git/*",
	}
	return patterns, cobra.ShellCompDirectiveNoFileComp
}

// setupFrontmatterCompletions sets up completion for frontmatter subcommands
func setupFrontmatterCompletions(cmd *cobra.Command) {
	for _, subCmd := range cmd.Commands() {
		switch subCmd.Name() {
		case "ensure", "set", "cast", "sync", "check", "download", "query":
			// All frontmatter subcommands take paths
			subCmd.ValidArgsFunction = CompleteDirs
		}

		// Field completions for commands that work with fields
		switch subCmd.Name() {
		case "ensure", "set":
			subCmd.RegisterFlagCompletionFunc("field", CompleteFrontmatterFields)
			subCmd.RegisterFlagCompletionFunc("type", CompleteFieldTypesWithFormat)
			subCmd.RegisterFlagCompletionFunc("default", CompleteDefaultValues)
		case "cast":
			subCmd.RegisterFlagCompletionFunc("field", CompleteFrontmatterFields)
			subCmd.RegisterFlagCompletionFunc("type", CompleteFieldTypesWithFormat)
		case "sync":
			subCmd.RegisterFlagCompletionFunc("field", CompleteFrontmatterFields)
			subCmd.RegisterFlagCompletionFunc("source", CompleteSyncSources)
		case "check":
			subCmd.RegisterFlagCompletionFunc("required", CompleteFrontmatterFields)
			subCmd.RegisterFlagCompletionFunc("type", CompleteFieldTypesWithFormat)
		case "query":
			subCmd.RegisterFlagCompletionFunc("missing", CompleteFrontmatterFields)
			subCmd.RegisterFlagCompletionFunc("duplicates", CompleteFrontmatterFields)
			subCmd.RegisterFlagCompletionFunc("field", CompleteFrontmatterFields)
			subCmd.RegisterFlagCompletionFunc("filter", CompleteQueryFilters)
		case "download":
			subCmd.RegisterFlagCompletionFunc("field", CompleteCommonFields)
		}
	}
}

// setupRenameCompletions sets up completion for rename command
func setupRenameCompletions(cmd *cobra.Command) {
	// First argument is source file (markdown)
	// Second argument should allow any filename/path
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			// First argument: source file
			return []string{"md"}, cobra.ShellCompDirectiveFilterFileExt
		}
		// Second argument: new name/path (no specific completion)
		return nil, cobra.ShellCompDirectiveDefault
	}

	cmd.RegisterFlagCompletionFunc("vault", CompleteDirs)
}

// CompleteCommonFields provides completion for common frontmatter fields
func CompleteCommonFields(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	fields := []string{
		"cover",
		"image",
		"avatar",
		"thumbnail",
		"icon",
		"banner",
		"photo",
		"picture",
		"attachment",
		"document",
		"file",
		"url",
		"link",
		"resource",
	}
	return fields, cobra.ShellCompDirectiveNoFileComp
}

// CompleteOutputFormats provides completion for output format flags
func CompleteOutputFormats(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	formats := []string{"text", "json", "csv", "yaml", "table"}
	return formats, cobra.ShellCompDirectiveNoFileComp
}

// CompleteFrontmatterFields provides completion for standard frontmatter fields
func CompleteFrontmatterFields(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	fields := []string{
		"title",
		"tags",
		"created",
		"modified",
		"priority",
		"status",
		"published",
		"category",
		"author",
		"description",
		"url",
		"type",
		"id",
	}
	return fields, cobra.ShellCompDirectiveNoFileComp
}

// CompleteFieldTypes provides completion for frontmatter field types
func CompleteFieldTypes(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	types := []string{
		"string",
		"number",
		"boolean",
		"array",
		"date",
		"null",
	}
	return types, cobra.ShellCompDirectiveNoFileComp
}

// CompleteFieldTypesWithFormat provides completion for field types in field:type format
func CompleteFieldTypesWithFormat(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Support both formats: "field:type" and just "type" for single field commands
	if strings.Contains(toComplete, ":") {
		// User is typing field:type format, complete the type part
		parts := strings.Split(toComplete, ":")
		if len(parts) == 2 {
			prefix := parts[0] + ":"
			types := []string{
				prefix + "string",
				prefix + "number", 
				prefix + "boolean",
				prefix + "array",
				prefix + "date",
				prefix + "null",
			}
			return types, cobra.ShellCompDirectiveNoFileComp
		}
	}
	
	// Provide both standalone types and common field:type combinations
	completions := []string{}
	
	// Standalone types (for single field commands)
	types := []string{"string", "number", "boolean", "array", "date", "null"}
	completions = append(completions, types...)
	
	// Common field:type combinations
	fields := []string{"title", "tags", "created", "modified", "priority", "status"}
	for _, field := range fields {
		for _, fieldType := range types {
			completions = append(completions, field+":"+fieldType)
		}
	}
	
	return completions, cobra.ShellCompDirectiveNoFileComp
}

// CompleteDefaultValues provides completion for default values
func CompleteDefaultValues(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	defaults := []string{
		"\"\"",          // empty string
		"null",          // null value
		"[]",            // empty array
		"[\"tag1\"]",    // single item array
		"0",             // number
		"false",         // boolean
		"{{current_date}}", // template variable
		"{{filename}}",     // template variable
		"{{uuid}}",         // template variable
	}
	return defaults, cobra.ShellCompDirectiveNoFileComp
}

// CompleteSyncSources provides completion for sync sources
func CompleteSyncSources(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	sources := []string{
		"file-mtime",
		"file-ctime", 
		"file-atime",
		"filename:pattern:regex",
		"path:dir",
		"path:parent",
		"content:first-line",
	}
	return sources, cobra.ShellCompDirectiveNoFileComp
}

// CompleteQueryFilters provides completion for query filters
func CompleteQueryFilters(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	filters := []string{
		"title:exists",
		"title:missing", 
		"tags:exists",
		"tags:missing",
		"created:exists",
		"created:missing",
		"modified:exists",
		"modified:missing",
		"status:active",
		"status:draft",
		"type:note",
		"type:daily",
	}
	return filters, cobra.ShellCompDirectiveNoFileComp
}

// CompleteLinkFormats provides completion for link format flags
func CompleteLinkFormats(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	formats := []string{"wiki", "markdown"}
	return formats, cobra.ShellCompDirectiveNoFileComp
}

// CompleteDuplicateTypes provides completion for duplicate analysis types
func CompleteDuplicateTypes(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	types := []string{"all", "obsidian", "sync-conflicts", "content"}
	return types, cobra.ShellCompDirectiveNoFileComp
}

// CompleteTimeSpans provides completion for time span analysis
func CompleteTimeSpans(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	spans := []string{"1w", "1m", "3m", "6m", "1y", "all"}
	return spans, cobra.ShellCompDirectiveNoFileComp
}

// CompleteGranularities provides completion for time granularity
func CompleteGranularities(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	granularities := []string{"day", "week", "month", "quarter"}
	return granularities, cobra.ShellCompDirectiveNoFileComp
}

// CompleteOutputFiles provides completion for output file names
func CompleteOutputFiles(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	extensions := []string{"txt", "json", "csv", "yaml", "md"}
	return extensions, cobra.ShellCompDirectiveFilterFileExt
}

// CompleteDepthValues provides completion for depth values
func CompleteDepthValues(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	depths := []string{"1", "2", "3", "4", "5", "10", "unlimited"}
	return depths, cobra.ShellCompDirectiveNoFileComp
}

// CompleteMinConnectionValues provides completion for minimum connection values
func CompleteMinConnectionValues(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	values := []string{"1", "2", "3", "5", "10", "20"}
	return values, cobra.ShellCompDirectiveNoFileComp
}

// CompleteQualityScores provides completion for quality score values
func CompleteQualityScores(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	scores := []string{"0.0", "0.1", "0.2", "0.3", "0.4", "0.5", "0.6", "0.7", "0.8", "0.9", "1.0"}
	return scores, cobra.ShellCompDirectiveNoFileComp
}

// setupAnalyzeCompletions sets up completion for analyze subcommands
func setupAnalyzeCompletions(cmd *cobra.Command) {
	for _, subCmd := range cmd.Commands() {
		subCmd.ValidArgsFunction = CompleteDirs
		
		// All analyze commands have format flag
		subCmd.RegisterFlagCompletionFunc("format", CompleteOutputFormats)
		subCmd.RegisterFlagCompletionFunc("output", CompleteOutputFiles)
		
		switch subCmd.Name() {
		case "duplicates":
			subCmd.RegisterFlagCompletionFunc("type", CompleteDuplicateTypes)
		case "trends":
			subCmd.RegisterFlagCompletionFunc("timespan", CompleteTimeSpans)
			subCmd.RegisterFlagCompletionFunc("granularity", CompleteGranularities)
		case "links":
			// Add depth and min-connections completions for links command
			subCmd.RegisterFlagCompletionFunc("depth", CompleteDepthValues)
			subCmd.RegisterFlagCompletionFunc("min-connections", CompleteMinConnectionValues)
		case "quality":
			// Add min-score completion for quality command
			subCmd.RegisterFlagCompletionFunc("min-score", CompleteQualityScores)
		}
	}
}

// setupLinksCompletions sets up completion for links subcommands
func setupLinksCompletions(cmd *cobra.Command) {
	for _, subCmd := range cmd.Commands() {
		subCmd.ValidArgsFunction = CompleteDirs
		
		if subCmd.Name() == "convert" {
			subCmd.RegisterFlagCompletionFunc("from", CompleteLinkFormats)
			subCmd.RegisterFlagCompletionFunc("to", CompleteLinkFormats)
		}
	}
}

// setupLinkdingCompletions sets up completion for linkding subcommands
func setupLinkdingCompletions(cmd *cobra.Command) {
	for _, subCmd := range cmd.Commands() {
		subCmd.ValidArgsFunction = CompleteDirs
		
		if subCmd.Name() == "sync" {
			subCmd.RegisterFlagCompletionFunc("url-field", CompleteFrontmatterFields)
			subCmd.RegisterFlagCompletionFunc("title-field", CompleteFrontmatterFields)
			subCmd.RegisterFlagCompletionFunc("tags-field", CompleteFrontmatterFields)
		}
	}
}

// Ultra-short global shortcuts for most common commands

// newEnsureShortcut creates a global shortcut for frontmatter ensure
func newEnsureShortcut() *cobra.Command {
	ensureCmd := frontmatter.NewFrontmatterCommand()
	for _, subCmd := range ensureCmd.Commands() {
		if subCmd.Name() == "ensure" {
			// Create a new command that mimics the ensure subcommand
			cmd := &cobra.Command{
				Use:    "e [path]",
				Short:  "Shortcut for: frontmatter ensure",
				Long:   "Global shortcut for 'mdnotes frontmatter ensure'. " + subCmd.Long,
				Args:   subCmd.Args,
				RunE:   subCmd.RunE,
				Hidden: false,
			}
			// Copy flags from the original ensure command
			cmd.Flags().AddFlagSet(subCmd.Flags())
			return cmd
		}
	}
	return nil
}

// newSetShortcut creates a global shortcut for frontmatter set
func newSetShortcut() *cobra.Command {
	frontmatterCmd := frontmatter.NewFrontmatterCommand()
	for _, subCmd := range frontmatterCmd.Commands() {
		if subCmd.Name() == "set" {
			cmd := &cobra.Command{
				Use:    "s [path]",
				Short:  "Shortcut for: frontmatter set",
				Long:   "Global shortcut for 'mdnotes frontmatter set'. " + subCmd.Long,
				Args:   subCmd.Args,
				RunE:   subCmd.RunE,
				Hidden: false,
			}
			cmd.Flags().AddFlagSet(subCmd.Flags())
			return cmd
		}
	}
	return nil
}

// newFixShortcut creates a global shortcut for headings fix
func newFixShortcut() *cobra.Command {
	headingsCmd := headings.NewHeadingsCommand()
	for _, subCmd := range headingsCmd.Commands() {
		if subCmd.Name() == "fix" {
			cmd := &cobra.Command{
				Use:    "f [path]",
				Short:  "Shortcut for: headings fix",
				Long:   "Global shortcut for 'mdnotes headings fix'. " + subCmd.Long,
				Args:   subCmd.Args,
				RunE:   subCmd.RunE,
				Hidden: false,
			}
			cmd.Flags().AddFlagSet(subCmd.Flags())
			return cmd
		}
	}
	return nil
}

// newCheckShortcut creates a global shortcut for links check
func newCheckShortcut() *cobra.Command {
	linksCmd := links.NewLinksCommand()
	for _, subCmd := range linksCmd.Commands() {
		if subCmd.Name() == "check" {
			cmd := &cobra.Command{
				Use:    "c [path]",
				Short:  "Shortcut for: links check",
				Long:   "Global shortcut for 'mdnotes links check'. " + subCmd.Long,
				Args:   subCmd.Args,
				RunE:   subCmd.RunE,
				Hidden: false,
			}
			cmd.Flags().AddFlagSet(subCmd.Flags())
			return cmd
		}
	}
	return nil
}

// newQueryShortcut creates a global shortcut for frontmatter query
func newQueryShortcut() *cobra.Command {
	frontmatterCmd := frontmatter.NewFrontmatterCommand()
	for _, subCmd := range frontmatterCmd.Commands() {
		if subCmd.Name() == "query" {
			cmd := &cobra.Command{
				Use:    "q [path]",
				Short:  "Shortcut for: frontmatter query",
				Long:   "Global shortcut for 'mdnotes frontmatter query'. " + subCmd.Long,
				Args:   subCmd.Args,
				RunE:   subCmd.RunE,
				Hidden: false,
			}
			cmd.Flags().AddFlagSet(subCmd.Flags())
			return cmd
		}
	}
	return nil
}
