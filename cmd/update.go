//go:build !windows

package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"efctl/pkg/ui"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update efctl to the latest version",
	Long:  `Downloads the latest efctl binary for your OS and architecture from GitHub Releases and replaces the current executable.`,
	Run: func(cmd *cobra.Command, args []string) {
		goos := runtime.GOOS
		goarch := runtime.GOARCH

		binaryName := fmt.Sprintf("efctl-%s-%s", goos, goarch)
		if goos == "windows" {
			binaryName += ".exe"
		}

		url := fmt.Sprintf("https://github.com/Scetrov/efctl/releases/latest/download/%s", binaryName)

		ui.Info.Println(fmt.Sprintf("Downloading latest efctl for %s/%s...", goos, goarch))

		spinner, _ := ui.Spin(fmt.Sprintf("Downloading %s", url))

		resp, err := http.Get(url)
		if err != nil {
			if spinner != nil {
				_ = spinner.Stop()
			}
			ui.Error.Println(fmt.Sprintf("Failed to download update: %s", err.Error()))
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			if spinner != nil {
				_ = spinner.Stop()
			}
			ui.Error.Println(fmt.Sprintf("Failed to download update: HTTP %d", resp.StatusCode))
			os.Exit(1)
		}

		// Write to a temp file in the same directory as the executable
		execPath, err := os.Executable()
		if err != nil {
			if spinner != nil {
				_ = spinner.Stop()
			}
			ui.Error.Println(fmt.Sprintf("Failed to determine executable path: %s", err.Error()))
			os.Exit(1)
		}
		execPath, err = filepath.EvalSymlinks(execPath)
		if err != nil {
			if spinner != nil {
				_ = spinner.Stop()
			}
			ui.Error.Println(fmt.Sprintf("Failed to resolve executable path: %s", err.Error()))
			os.Exit(1)
		}

		execDir := filepath.Dir(execPath)
		tmpFile, err := os.CreateTemp(execDir, "efctl-update-*")
		if err != nil {
			if spinner != nil {
				_ = spinner.Stop()
			}
			ui.Error.Println(fmt.Sprintf("Failed to create temp file: %s", err.Error()))
			os.Exit(1)
		}
		tmpPath := tmpFile.Name()

		_, err = io.Copy(tmpFile, resp.Body)
		tmpFile.Close()
		if err != nil {
			os.Remove(tmpPath)
			if spinner != nil {
				_ = spinner.Stop()
			}
			ui.Error.Println(fmt.Sprintf("Failed to write update: %s", err.Error()))
			os.Exit(1)
		}

		// Make executable (no-op on Windows, but harmless)
		if err := os.Chmod(tmpPath, 0755); err != nil {
			os.Remove(tmpPath)
			if spinner != nil {
				_ = spinner.Stop()
			}
			ui.Error.Println(fmt.Sprintf("Failed to set permissions: %s", err.Error()))
			os.Exit(1)
		}

		// Atomic swap: rename current binary out of the way, then move new one in
		oldPath := execPath + ".old"
		if err := os.Rename(execPath, oldPath); err != nil {
			os.Remove(tmpPath)
			if spinner != nil {
				_ = spinner.Stop()
			}
			ui.Error.Println(fmt.Sprintf("Failed to replace binary: %s", err.Error()))
			os.Exit(1)
		}

		if err := os.Rename(tmpPath, execPath); err != nil {
			// Try to restore the old binary
			_ = os.Rename(oldPath, execPath)
			if spinner != nil {
				_ = spinner.Stop()
			}
			ui.Error.Println(fmt.Sprintf("Failed to replace binary: %s", err.Error()))
			os.Exit(1)
		}

		// Best-effort cleanup of the old binary (may fail on Windows while running)
		_ = os.Remove(oldPath)

		if spinner != nil {
			_ = spinner.Stop()
		}

		ui.Success.Println("efctl has been updated to the latest version!")
		os.Exit(0)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
