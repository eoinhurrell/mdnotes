package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/eoinhurrell/mdnotes/internal/errors"
)

// HandleError processes errors consistently across all commands
func HandleError(cmd *cobra.Command, err error) {
	if err == nil {
		return
	}

	// Get verbosity flags
	verbose, _ := cmd.Flags().GetBool("verbose")
	quiet, _ := cmd.Flags().GetBool("quiet")

	// Create error handler
	errorHandler := errors.NewErrorHandler(verbose, quiet)

	// Format and display error
	errorMessage := errorHandler.Handle(err)
	
	if !quiet {
		cmd.PrintErrln(errorMessage)
	}

	// Exit with appropriate code
	os.Exit(errors.ExitCode(err))
}

// WithErrorHandling wraps a command function with consistent error handling
func WithErrorHandling(fn func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		if err := fn(cmd, args); err != nil {
			HandleError(cmd, err)
		}
	}
}

// CommonErrorSuggestions provides suggestions for common error scenarios
type CommonErrorSuggestions struct{}

// ForFileOperation suggests solutions for file operation errors
func (s CommonErrorSuggestions) ForFileOperation(operation, file string, err error) string {
	switch operation {
	case "scan":
		return "Ensure the path exists and you have read permissions. Use --verbose to see which files are being processed."
	case "parse":
		return "Check that the file contains valid YAML frontmatter between --- delimiters. Use --dry-run to test without making changes."
	case "write":
		return "Ensure you have write permissions and sufficient disk space. Consider using --backup to create a backup first."
	default:
		return "Use --help to see available options, or --verbose for more detailed output."
	}
}

// ForValidationOperation suggests solutions for validation errors
func (s CommonErrorSuggestions) ForValidationOperation(field, expectedType string) string {
	switch expectedType {
	case "date":
		return "Use ISO date format (YYYY-MM-DD) or datetime format (YYYY-MM-DDTHH:MM:SSZ). Example: created: 2023-01-15"
	case "array":
		return "Use YAML array syntax: tags: [tag1, tag2] or YAML list syntax:\ntags:\n  - tag1\n  - tag2"
	case "number":
		return "Use numeric values without quotes: priority: 5 or weight: 3.14"
	case "boolean":
		return "Use true or false without quotes: published: true"
	default:
		return "Check the field format in your frontmatter. Use 'mdnotes frontmatter validate --help' for more information."
	}
}

// ForConfigOperation suggests solutions for configuration errors
func (s CommonErrorSuggestions) ForConfigOperation(configFile string) string {
	return "Check YAML syntax, ensure required fields are present, and verify file permissions. " +
		"Use 'mdnotes batch validate " + configFile + "' to check configuration validity."
}

// ForNetworkOperation suggests solutions for network errors
func (s CommonErrorSuggestions) ForNetworkOperation(service string) string {
	switch service {
	case "linkding":
		return "Verify LINKDING_URL and LINKDING_TOKEN environment variables are set correctly. " +
			"Test the connection with: curl -H 'Authorization: Token $LINKDING_TOKEN' $LINKDING_URL/api/bookmarks/"
	default:
		return "Check internet connection, service availability, and authentication credentials."
	}
}