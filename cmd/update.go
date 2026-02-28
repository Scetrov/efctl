//go:build !windows

package cmd

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"efctl/pkg/ui"

	"github.com/spf13/cobra"
)

const (
	// maxUpdateBinarySize is the maximum allowed size for a downloaded update binary (100 MB).
	maxUpdateBinarySize int64 = 100 * 1024 * 1024
	// updateHTTPTimeout is the timeout for the update HTTP client.
	updateHTTPTimeout = 120 * time.Second
	// releaseBaseURL is the base URL for downloading release assets.
	releaseBaseURL = "https://github.com/Scetrov/efctl/releases/latest/download"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update efctl to the latest version",
	Long:  `Downloads the latest efctl binary for your OS and architecture from GitHub Releases, verifies its SHA-256 checksum, and replaces the current executable.`,
	Run: func(cmd *cobra.Command, args []string) {
		goos := runtime.GOOS
		goarch := runtime.GOARCH

		binaryName := fmt.Sprintf("efctl-%s-%s", goos, goarch)
		if goos == "windows" {
			binaryName += ".exe"
		}

		binaryURL := fmt.Sprintf("%s/%s", releaseBaseURL, binaryName)
		checksumsURL := fmt.Sprintf("%s/checksums.txt", releaseBaseURL)

		ui.Info.Println(fmt.Sprintf("Downloading latest efctl for %s/%s...", goos, goarch))

		// Fetch checksums first
		expectedHash, err := fetchExpectedChecksum(checksumsURL, binaryName)
		if err != nil {
			ui.Error.Println(fmt.Sprintf("Failed to fetch checksums: %s", err.Error()))
			os.Exit(1)
		}

		spinner, _ := ui.Spin(fmt.Sprintf("Downloading %s", binaryURL))

		client := &http.Client{Timeout: updateHTTPTimeout}
		resp, err := client.Get(binaryURL) // #nosec G107 -- URL constructed from hardcoded releaseBaseURL constant
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

		// Download with size limit and compute SHA-256 simultaneously
		hasher := sha256.New()
		limitedReader := io.LimitReader(resp.Body, maxUpdateBinarySize)
		teeReader := io.TeeReader(limitedReader, hasher)

		_, err = io.Copy(tmpFile, teeReader)
		if closeErr := tmpFile.Close(); closeErr != nil {
			ui.Warn.Println(fmt.Sprintf("Warning: failed to close temp file: %s", closeErr.Error()))
		}
		if err != nil {
			if removeErr := os.Remove(tmpPath); removeErr != nil {
				ui.Warn.Println(fmt.Sprintf("Warning: failed to clean up temp file: %s", removeErr.Error()))
			}
			if spinner != nil {
				_ = spinner.Stop()
			}
			ui.Error.Println(fmt.Sprintf("Failed to write update: %s", err.Error()))
			os.Exit(1)
		}

		// Verify SHA-256 checksum
		actualHash := hex.EncodeToString(hasher.Sum(nil))
		if actualHash != expectedHash {
			if removeErr := os.Remove(tmpPath); removeErr != nil {
				ui.Warn.Println(fmt.Sprintf("Warning: failed to clean up temp file: %s", removeErr.Error()))
			}
			if spinner != nil {
				_ = spinner.Stop()
			}
			ui.Error.Println(fmt.Sprintf("Checksum verification failed!\n  Expected: %s\n  Actual:   %s\nThe downloaded binary may have been tampered with.", expectedHash, actualHash))
			os.Exit(1)
		}

		// Make executable
		if err := os.Chmod(tmpPath, 0700); err != nil {
			if removeErr := os.Remove(tmpPath); removeErr != nil {
				ui.Warn.Println(fmt.Sprintf("Warning: failed to clean up temp file: %s", removeErr.Error()))
			}
			if spinner != nil {
				_ = spinner.Stop()
			}
			ui.Error.Println(fmt.Sprintf("Failed to set permissions: %s", err.Error()))
			os.Exit(1)
		}

		// Atomic swap: rename current binary out of the way, then move new one in
		oldPath := execPath + ".old"
		if err := os.Rename(execPath, oldPath); err != nil {
			if removeErr := os.Remove(tmpPath); removeErr != nil {
				ui.Warn.Println(fmt.Sprintf("Warning: failed to clean up temp file: %s", removeErr.Error()))
			}
			if spinner != nil {
				_ = spinner.Stop()
			}
			ui.Error.Println(fmt.Sprintf("Failed to replace binary: %s", err.Error()))
			os.Exit(1)
		}

		if err := os.Rename(tmpPath, execPath); err != nil {
			// Try to restore the old binary
			if restoreErr := os.Rename(oldPath, execPath); restoreErr != nil {
				ui.Warn.Println(fmt.Sprintf("Warning: failed to restore old binary: %s", restoreErr.Error()))
			}
			if spinner != nil {
				_ = spinner.Stop()
			}
			ui.Error.Println(fmt.Sprintf("Failed to replace binary: %s", err.Error()))
			os.Exit(1)
		}

		// Best-effort cleanup of the old binary
		if removeErr := os.Remove(oldPath); removeErr != nil {
			ui.Warn.Println(fmt.Sprintf("Warning: could not remove old binary: %s", removeErr.Error()))
		}

		if spinner != nil {
			_ = spinner.Stop()
		}

		ui.Success.Println(fmt.Sprintf("Checksum verified: %s", actualHash))
		ui.Success.Println("efctl has been updated to the latest version!")
		os.Exit(0)
	},
}

// fetchExpectedChecksum downloads the checksums.txt file and extracts the expected SHA-256 hash
// for the given binary name.
func fetchExpectedChecksum(checksumsURL, binaryName string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(checksumsURL) // #nosec G107 -- URL constructed from hardcoded releaseBaseURL constant
	if err != nil {
		return "", fmt.Errorf("failed to download checksums: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download checksums: HTTP %d", resp.StatusCode)
	}

	// Limit checksums file to 1 MB (should be tiny)
	limitedBody := io.LimitReader(resp.Body, 1024*1024)
	scanner := bufio.NewScanner(limitedBody)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Format: <sha256sum>  <filename>
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == binaryName {
			hash := strings.ToLower(parts[0])
			if len(hash) != 64 {
				return "", fmt.Errorf("invalid checksum length for %s", binaryName)
			}
			return hash, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading checksums: %w", err)
	}

	return "", fmt.Errorf("no checksum found for %s in checksums.txt", binaryName)
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
