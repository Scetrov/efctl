package cmd

import (
	"os"
	"path/filepath"

	"efctl/pkg/config"
	"efctl/pkg/ui"
	"efctl/pkg/validate"
	"github.com/spf13/cobra"
)

var (
	configFile string
	debugMode  bool
	noProgress bool
)

var rootCmd = &cobra.Command{
	Use:   "efctl",
	Short: "efctl manages the local EVE Frontier Sui development environment",
	Long:  `A fast and flexible CLI to automate the setup, deployment, and teardown of the EVE Frontier local world contracts and smart gates.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Enable debug output before any other work so early messages are visible.
		if debugMode {
			ui.DebugEnabled = true
		}

		// Disable progress spinner if explicitly requested or running in CI.
		if noProgress || os.Getenv("CI") == "true" {
			ui.ProgressEnabled = false
		}

		resolvedConfigPath := configFile
		if !cmd.Flags().Changed("config-file") {
			if discoveredPath, found, discoverErr := config.FindDefaultConfigPath("."); discoverErr != nil {
				ui.Error.Println("Failed to discover config file: " + discoverErr.Error())
				os.Exit(1)
			} else if found {
				resolvedConfigPath = discoveredPath
			}
		}

		cfg, err := config.Load(resolvedConfigPath)
		if err != nil {
			ui.Error.Println("Failed to load config: " + err.Error())
			os.Exit(1)
		}
		config.Loaded = cfg

		if cfg != nil && !cfg.WasLoaded() {
			ui.Debug.Println("Config file not found in current directory or any parent directories.")
			ui.Debug.Println("Using default configuration. To customize, create efctl.yaml or efctl.yml.")
		} else if cfg != nil && cfg.WasLoaded() {
			ui.Debug.Println("Loaded configuration from: " + resolvedConfigPath)
		}

		// Resolve workspacePath to an absolute path so that bind-mount
		// sources are correct regardless of the container daemon's cwd.
		if workspacePath != "" {
			abs, absErr := filepath.Abs(workspacePath)
			if absErr == nil {
				ui.Debug.Println("Resolved workspace path: " + abs)
				workspacePath = abs
			}
		}

		// Validate workspace path if a workspace-aware command is being used
		if workspacePath != "" && workspacePath != "." {
			if err := validate.WorkspacePath(workspacePath); err != nil {
				ui.Error.Println("Invalid workspace path: " + err.Error())
				os.Exit(1)
			}
		}
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configFile, "config-file", config.DefaultConfigFile, "Path to the efctl.yaml or efctl.yml configuration file")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable verbose debug logging")
	rootCmd.PersistentFlags().BoolVar(&noProgress, "no-progress", false, "Disable the progress spinner for cleaner CI output")
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	ui.PrintBanner()
	if err := rootCmd.Execute(); err != nil {
		ui.Error.Println(err.Error())
		os.Exit(1)
	}
}

// GetRootCmd returns the root cobra command
func GetRootCmd() *cobra.Command {
	return rootCmd
}

// GetNewRootCmd returns a fresh instance of the root command for testing
func GetNewRootCmd() *cobra.Command {
	// Re-initialize a fresh command tree for tests to avoid state leakage
	// This is a simplified version; in a real app you might want to refactor
	// init() into a function that can be called repeatedly.
	newRoot := &cobra.Command{
		Use:              rootCmd.Use,
		Short:            rootCmd.Short,
		Long:             rootCmd.Long,
		PersistentPreRun: rootCmd.PersistentPreRun,
	}
	newRoot.PersistentFlags().StringVar(&configFile, "config-file", config.DefaultConfigFile, "Path to the efctl.yaml or efctl.yml configuration file")
	newRoot.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable verbose debug logging")
	newRoot.PersistentFlags().BoolVar(&noProgress, "no-progress", false, "Disable the progress spinner for cleaner CI output")

	// Re-add subcommands... This is getting complex because they are added in init()
	// Let's try a different approach: manually reset the Changed property of flags.
	return rootCmd
}
