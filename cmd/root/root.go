package root

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/eoinhurrell/mdnotes/cmd/analyze"
	"github.com/eoinhurrell/mdnotes/cmd/batch"
	"github.com/eoinhurrell/mdnotes/cmd/frontmatter"
	"github.com/eoinhurrell/mdnotes/cmd/headings"
	"github.com/eoinhurrell/mdnotes/cmd/links"
	"github.com/eoinhurrell/mdnotes/cmd/linkding"
	"github.com/eoinhurrell/mdnotes/cmd/profile"
	"github.com/eoinhurrell/mdnotes/cmd/rename"
)

// NewRootCommand creates the root command for mdnotes
func NewRootCommand() *cobra.Command {
	var zshCompletion bool

	cmd := &cobra.Command{
		Use:   "mdnotes",
		Short: "A CLI tool for managing Obsidian markdown notes",
		Long: `mdnotes is a command-line tool designed to automate and standardize 
administrative tasks for Obsidian vaults. It provides powerful batch operations 
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
	cmd.PersistentFlags().Bool("dry-run", false, "Preview changes without applying them")
	cmd.PersistentFlags().Bool("verbose", false, "Verbose output")
	cmd.PersistentFlags().Bool("quiet", false, "Suppress non-error output")
	cmd.PersistentFlags().String("config", "", "Config file (default: .obsidian-admin.yaml)")
	
	// Add completion flag
	cmd.Flags().BoolVar(&zshCompletion, "zsh-completion", false, "Generate zsh completion script")

	// Add subcommands
	cmd.AddCommand(analyze.NewAnalyzeCommand())
	cmd.AddCommand(batch.NewBatchCommand())
	cmd.AddCommand(frontmatter.NewFrontmatterCommand())
	cmd.AddCommand(headings.NewHeadingsCommand())
	cmd.AddCommand(links.NewLinksCommand())
	cmd.AddCommand(linkding.NewLinkdingCommand())
	cmd.AddCommand(profile.NewProfileCommand())
	cmd.AddCommand(rename.NewRenameCommand())

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
	// Set completion for vault path arguments (directories)
	cmd.RegisterFlagCompletionFunc("config", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"yaml", "yml"}, cobra.ShellCompDirectiveFilterFileExt
	})
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