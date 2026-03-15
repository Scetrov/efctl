package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
		targetDir := filepath.Dir(cleanPath)

		if info, err := os.Stat(cleanPath); err == nil && !info.IsDir() && !initForce {
			return fmt.Errorf("config file already exists at %s; rerun with --force to overwrite", cleanPath)
		} else if err == nil && info.IsDir() {
			return fmt.Errorf("config path %s is a directory", cleanPath)
		} else if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to inspect config path %s: %w", cleanPath, err)
		}

		if err := os.MkdirAll(targetDir, 0750); err != nil {
			return fmt.Errorf("failed to create config directory for %s: %w", cleanPath, err)
		}

		if err := os.WriteFile(cleanPath, []byte(config.DefaultConfigYAML()), 0600); err != nil {
			return fmt.Errorf("failed to write config file %s: %w", cleanPath, err)
		}

		ui.Success.Printf("Created config file at %s\n", cleanPath)

		// 1. Git Init
		if _, err := os.Stat(filepath.Join(targetDir, ".git")); os.IsNotExist(err) {
			gitCmd := exec.Command("git", "init")
			gitCmd.Dir = targetDir
			if err := gitCmd.Run(); err != nil {
				ui.Warn.Printf("Failed to initialize Git repository: %v\n", err)
			} else {
				ui.Success.Println("Initialized empty Git repository")
			}
		}

		// 2. .gitignore Update
		gitignorePath := filepath.Join(targetDir, ".gitignore")
		entries := []string{"world-contracts/", "builder-scaffold/"}
		if err := appendToGitignore(gitignorePath, entries); err != nil {
			ui.Warn.Printf("Failed to update .gitignore: %v\n", err)
		} else {
			ui.Success.Println("Updated .gitignore")
		}

		// 3. Create Bind-Mount Directory
		extensionDir := filepath.Join(targetDir, "my-extension")
		if err := os.MkdirAll(extensionDir, 0750); err != nil {
			ui.Warn.Printf("Failed to create example extension directory: %v\n", err)
		} else {
			ui.Success.Println("Created example extension directory")
		}

		return nil
	},
}

// appendToGitignore appends unique entries to a .gitignore file.
func appendToGitignore(path string, entries []string) error {
	existing, err := readGitignore(path)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600) // #nosec G304
	if err != nil {
		return err
	}
	defer f.Close()

	// Ensure file ends with newline if it's not empty
	info, err := f.Stat()
	if err == nil && info.Size() > 0 {
		data, err := os.ReadFile(path) // #nosec G304
		if err == nil && len(data) > 0 && data[len(data)-1] != '\n' {
			if _, err := f.WriteString("\n"); err != nil {
				return err
			}
		}
	}

	for _, entry := range entries {
		cleanEntry := strings.TrimSpace(entry)
		if cleanEntry != "" && !existing[cleanEntry] && !existing["/"+cleanEntry] {
			if _, err := f.WriteString(cleanEntry + "\n"); err != nil {
				return err
			}
		}
	}

	return nil
}

func readGitignore(path string) (map[string]bool, error) {
	existing := make(map[string]bool)
	f, err := os.Open(path) // #nosec G304
	if err != nil {
		if os.IsNotExist(err) {
			return existing, nil
		}
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			existing[line] = true
		}
	}
	return existing, scanner.Err()
}

func init() {
	initCmd.Flags().BoolVar(&initForce, "force", false, "Overwrite an existing config file")
	rootCmd.AddCommand(initCmd)
}
