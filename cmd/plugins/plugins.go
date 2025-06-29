package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/eoinhurrell/mdnotes/internal/config"
	"github.com/eoinhurrell/mdnotes/pkg/plugins"
	"github.com/spf13/cobra"
)

// NewPluginsCommand creates the plugins command
func NewPluginsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "plugins",
		Aliases: []string{"plugin", "p"},
		Short:   "Manage mdnotes plugins",
		Long: `Manage mdnotes plugins including loading, listing, enabling, and disabling plugins.

Plugins extend mdnotes functionality by providing custom processing capabilities.
They can be executed at various hook points during command execution.

PLUGIN CONFIGURATION:
Configure plugins in your mdnotes.yaml file:

plugins:
  enabled: true
  paths: 
    - "~/.mdnotes/plugins"
    - "./plugins"
  plugins:
    auto-frontmatter:
      required_fields: ["created", "modified", "tags"]
    content-enhancer:
      fix_spacing: true
      fix_newlines: true

HOOK POINTS:
Plugins can register for the following hook points:
- pre-command: Before any command processing
- per-file: For each file during processing  
- post-command: After command processing completes
- export-complete: After export operations complete

PLUGIN DEVELOPMENT:
Plugins are compiled Go shared objects (.so files) that implement the Plugin interface.
See the examples in pkg/plugins/examples.go for reference implementations.`,
		Example: `  # List all loaded plugins
  mdnotes plugins list

  # List plugins with detailed information
  mdnotes plugins list --verbose

  # Enable a specific plugin
  mdnotes plugins enable auto-frontmatter

  # Disable a specific plugin
  mdnotes plugins disable content-enhancer

  # Show plugin information in JSON format
  mdnotes plugins list --format json`,
	}

	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newEnableCommand())
	cmd.AddCommand(newDisableCommand())
	cmd.AddCommand(newInfoCommand())

	return cmd
}

func newListCommand() *cobra.Command {
	var (
		outputFormat string
		verbose      bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List loaded plugins",
		Long:  `List all currently loaded plugins with their status and basic information.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			cfg, err := loadConfig(cmd)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			// Create plugin manager
			managerConfig := plugins.ManagerConfig{
				Enabled:     cfg.Plugins.Enabled,
				SearchPaths: cfg.Plugins.SearchPaths,
				Plugins:     cfg.Plugins.Plugins,
			}
			manager := plugins.NewPluginManager(managerConfig)

			// Load plugins
			if err := manager.LoadPlugins(); err != nil {
				return fmt.Errorf("loading plugins: %w", err)
			}

			// Get plugin list
			pluginList := manager.ListPlugins()

			// Output results
			if outputFormat == "json" {
				return outputJSON(pluginList)
			} else {
				return outputTable(pluginList, verbose)
			}
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "format", "f", "table", "Output format (table, json)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed plugin information")

	return cmd
}

func newEnableCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable <plugin-name>",
		Short: "Enable a specific plugin",
		Long:  `Enable a specific plugin that is currently disabled.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pluginName := args[0]

			// Load configuration
			cfg, err := loadConfig(cmd)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			// Create plugin manager
			managerConfig := plugins.ManagerConfig{
				Enabled:     cfg.Plugins.Enabled,
				SearchPaths: cfg.Plugins.SearchPaths,
				Plugins:     cfg.Plugins.Plugins,
			}
			manager := plugins.NewPluginManager(managerConfig)

			// Load plugins
			if err := manager.LoadPlugins(); err != nil {
				return fmt.Errorf("loading plugins: %w", err)
			}

			// Enable plugin
			if err := manager.EnablePlugin(pluginName); err != nil {
				return fmt.Errorf("enabling plugin: %w", err)
			}

			fmt.Printf("✅ Plugin '%s' enabled successfully\n", pluginName)
			return nil
		},
	}

	return cmd
}

func newDisableCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable <plugin-name>",
		Short: "Disable a specific plugin",
		Long:  `Disable a specific plugin that is currently enabled.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pluginName := args[0]

			// Load configuration
			cfg, err := loadConfig(cmd)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			// Create plugin manager
			managerConfig := plugins.ManagerConfig{
				Enabled:     cfg.Plugins.Enabled,
				SearchPaths: cfg.Plugins.SearchPaths,
				Plugins:     cfg.Plugins.Plugins,
			}
			manager := plugins.NewPluginManager(managerConfig)

			// Load plugins
			if err := manager.LoadPlugins(); err != nil {
				return fmt.Errorf("loading plugins: %w", err)
			}

			// Disable plugin
			if err := manager.DisablePlugin(pluginName); err != nil {
				return fmt.Errorf("disabling plugin: %w", err)
			}

			fmt.Printf("✅ Plugin '%s' disabled successfully\n", pluginName)
			return nil
		},
	}

	return cmd
}

func newInfoCommand() *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "info <plugin-name>",
		Short: "Show detailed information about a plugin",
		Long:  `Show detailed information about a specific plugin including configuration and capabilities.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pluginName := args[0]

			// Load configuration
			cfg, err := loadConfig(cmd)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			// Create plugin manager
			managerConfig := plugins.ManagerConfig{
				Enabled:     cfg.Plugins.Enabled,
				SearchPaths: cfg.Plugins.SearchPaths,
				Plugins:     cfg.Plugins.Plugins,
			}
			manager := plugins.NewPluginManager(managerConfig)

			// Load plugins
			if err := manager.LoadPlugins(); err != nil {
				return fmt.Errorf("loading plugins: %w", err)
			}

			// Get plugin info
			info, exists := manager.GetPluginInfo(pluginName)
			if !exists {
				return fmt.Errorf("plugin '%s' not found", pluginName)
			}

			// Output results
			if outputFormat == "json" {
				return outputJSON(info)
			} else {
				return outputPluginInfo(info)
			}
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "Output format (text, json)")

	return cmd
}

func loadConfig(cmd *cobra.Command) (*config.Config, error) {
	configPath, _ := cmd.Flags().GetString("config")

	if configPath != "" {
		return config.LoadConfigFromFile(configPath)
	}

	return config.LoadConfigWithFallback(config.GetDefaultConfigPaths())
}

func outputJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func outputTable(pluginList []*plugins.PluginInfo, verbose bool) error {
	if len(pluginList) == 0 {
		fmt.Println("No plugins loaded.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	if verbose {
		fmt.Fprintln(w, "NAME\tVERSION\tENABLED\tDESCRIPTION\tHOOKS\tPATH")
		fmt.Fprintln(w, "----\t-------\t-------\t-----------\t-----\t----")
		for _, plugin := range pluginList {
			status := "✅"
			if !plugin.Enabled {
				status = "❌"
			}

			hooks := ""
			for i, hook := range plugin.SupportedHooks {
				if i > 0 {
					hooks += ", "
				}
				hooks += string(hook)
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				plugin.Name, plugin.Version, status, plugin.Description, hooks, plugin.LoadPath)
		}
	} else {
		fmt.Fprintln(w, "NAME\tVERSION\tENABLED\tDESCRIPTION")
		fmt.Fprintln(w, "----\t-------\t-------\t-----------")
		for _, plugin := range pluginList {
			status := "✅"
			if !plugin.Enabled {
				status = "❌"
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				plugin.Name, plugin.Version, status, plugin.Description)
		}
	}

	return w.Flush()
}

func outputPluginInfo(info *plugins.PluginInfo) error {
	fmt.Printf("Plugin Information\n")
	fmt.Printf("==================\n\n")
	fmt.Printf("Name:        %s\n", info.Name)
	fmt.Printf("Version:     %s\n", info.Version)
	fmt.Printf("Description: %s\n", info.Description)
	fmt.Printf("Enabled:     %t\n", info.Enabled)
	fmt.Printf("Load Path:   %s\n", info.LoadPath)

	fmt.Printf("\nSupported Hooks:\n")
	for _, hook := range info.SupportedHooks {
		fmt.Printf("  • %s\n", hook)
	}

	if len(info.Config) > 0 {
		fmt.Printf("\nConfiguration:\n")
		for key, value := range info.Config {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	return nil
}
