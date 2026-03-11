package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"efctl/pkg/config"
	"efctl/pkg/ui"
	"github.com/spf13/cobra"
)

var initForce bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create an efctl.yaml configuration file",
	Long:  `Scaffold an efctl.yaml configuration file with the current recommended defaults for efctl.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		targetPath := config.DefaultConfigFile
		if cmd.Flags().Changed("config-file") {
			targetPath = configFile
		}

		cleanPath := filepath.Clean(targetPath)
		if info, err := os.Stat(cleanPath); err == nil && !info.IsDir() && !initForce {
			return fmt.Errorf("config file already exists at %s; rerun with --force to overwrite", cleanPath)
		} else if err == nil && info.IsDir() {
			return fmt.Errorf("config path %s is a directory", cleanPath)
		} else if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to inspect config path %s: %w", cleanPath, err)
		}

		if err := os.MkdirAll(filepath.Dir(cleanPath), 0750); err != nil {
			return fmt.Errorf("failed to create config directory for %s: %w", cleanPath, err)
		}

		if err := os.WriteFile(cleanPath, []byte(config.DefaultConfigYAML()), 0600); err != nil {
			return fmt.Errorf("failed to write config file %s: %w", cleanPath, err)
		}

		ui.Success.Printf("Created config file at %s\n", cleanPath)
		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&initForce, "force", false, "Overwrite an existing config file")
	rootCmd.AddCommand(initCmd)
}
