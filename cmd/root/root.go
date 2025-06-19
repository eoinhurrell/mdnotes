package root

import (
	"fmt"
	"os"

	"github.com/eoinhurrell/mdnotes/cmd/analyze"
	"github.com/eoinhurrell/mdnotes/cmd/frontmatter"
	"github.com/eoinhurrell/mdnotes/cmd/headings"
	"github.com/eoinhurrell/mdnotes/cmd/linkding"
	"github.com/eoinhurrell/mdnotes/cmd/links"
	"github.com/eoinhurrell/mdnotes/cmd/profile"
	"github.com/eoinhurrell/mdnotes/cmd/rename"
	"github.com/spf13/cobra"
)

// NewRootCommand creates the root command for mdnotes
func NewRootCommand() *cobra.Command {
	var zshCompletion bool

	cmd := &cobra.Command{
		Use:   "mdnotes",
		Short: "A CLI tool for managing Obsidian markdown notes",
		Long: `mdnotes is a command-line tool designed to automate and standardize 
administrative tasks for Obsidian vaults. It provides powerful operations 
for managing frontmatter, headings, links, and file organization.`,
		Version: "1.0.0",
		Run: func(cmd *cobra.Command, args []string) {
			if zshCompletion {
				err := cmd.Root().GenZshCompletion(os.Stdout)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error generating zsh completion: %v\n", err)
					os.Exit(1)
				}
				return
			}
			cmd.Help()
		},
	}

	// Add global flags
	cmd.PersistentFlags().Bool("dry-run", false, "Preview changes without applying them; shows exactly what would be changed")
	cmd.PersistentFlags().Bool("verbose", false, "Detailed output; prints filepath of every file examined and actions taken")
	cmd.PersistentFlags().Bool("quiet", false, "Suppress all output except errors and final summary; overrides --verbose")
	cmd.PersistentFlags().String("config", "", "Config file (default: .obsidian-admin.yaml)")

	// Add completion flag
	cmd.Flags().BoolVar(&zshCompletion, "zsh-completion", false, "Generate zsh completion script")

	// Add subcommands
	cmd.AddCommand(analyze.NewAnalyzeCommand())
	cmd.AddCommand(frontmatter.NewFrontmatterCommand())
	cmd.AddCommand(headings.NewHeadingsCommand())
	cmd.AddCommand(links.NewLinksCommand())
	cmd.AddCommand(linkding.NewLinkdingCommand())
	cmd.AddCommand(profile.NewProfileCommand())
	cmd.AddCommand(rename.NewRenameCommand())

	// Add ultra-short global shortcuts for most common commands
	cmd.AddCommand(newEnsureShortcut())
	cmd.AddCommand(newSetShortcut())
	cmd.AddCommand(newQueryShortcut())
	cmd.AddCommand(newFixShortcut())
	cmd.AddCommand(newCheckShortcut())

	// Add completion command as well for more standard approach
	cmd.AddCommand(newCompletionCommand())

	// Set up custom completions
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

		// Add specific completions for frontmatter commands
		if subCmd.Name() == "frontmatter" {
			setupFrontmatterCompletions(subCmd)
		}

		// Add specific completions for rename command
		if subCmd.Name() == "rename" {
			setupRenameCompletions(subCmd)
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
		case "ensure", "set", "cast", "sync", "check", "download":
			// All frontmatter subcommands take paths
			subCmd.ValidArgsFunction = CompleteDirs
		}

		// Special completions for specific commands
		if subCmd.Name() == "download" {
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
