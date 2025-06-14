package root

import (
	"github.com/spf13/cobra"
	"github.com/eoinhurrell/mdnotes/cmd/frontmatter"
	"github.com/eoinhurrell/mdnotes/cmd/headings"
	"github.com/eoinhurrell/mdnotes/cmd/links"
)

// NewRootCommand creates the root command for mdnotes
func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mdnotes",
		Short: "A CLI tool for managing Obsidian markdown notes",
		Long: `mdnotes is a command-line tool designed to automate and standardize 
administrative tasks for Obsidian vaults. It provides powerful batch operations 
for managing frontmatter, headings, links, and file organization.`,
		Version: "1.0.0",
	}

	// Add global flags
	cmd.PersistentFlags().Bool("dry-run", false, "Preview changes without applying them")
	cmd.PersistentFlags().Bool("verbose", false, "Verbose output")
	cmd.PersistentFlags().Bool("quiet", false, "Suppress non-error output")
	cmd.PersistentFlags().String("config", "", "Config file (default: .obsidian-admin.yaml)")

	// Add subcommands
	cmd.AddCommand(frontmatter.NewFrontmatterCommand())
	cmd.AddCommand(headings.NewHeadingsCommand())
	cmd.AddCommand(links.NewLinksCommand())

	return cmd
}