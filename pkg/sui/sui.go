package sui

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"efctl/pkg/ui"
)

var adminKeyRegex = regexp.MustCompile(`^ADMIN_PRIVATE_KEY=(suiprivkey[a-z0-9]+)`)
var playerAKeyRegex = regexp.MustCompile(`^PLAYER_A_PRIVATE_KEY=(suiprivkey[a-z0-9]+)`)
var playerBKeyRegex = regexp.MustCompile(`^PLAYER_B_PRIVATE_KEY=(suiprivkey[a-z0-9]+)`)

func IsSuiUpInstalled() bool {
	_, err := exec.LookPath("suiup")
	return err == nil
}

func InstallSuiUp() error {
	ui.Info.Println("Installing suiup...")
	// Official installation command for suiup
	cmd := exec.Command("bash", "-c", "set -o pipefail; curl -fsSL https://raw.githubusercontent.com/MystenLabs/suiup/main/install.sh | bash")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func IsSuiInstalled() bool {
	_, err := exec.LookPath("sui")
	return err == nil
}

func InstallSui() error {
	ui.Info.Println("Installing sui via suiup...")
	cmd := exec.Command("suiup", "install", "sui", "-y")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func ConfigureSui(workspace string) error {
	if !IsSuiInstalled() {
		return nil
	}

	ui.Info.Println("Configuring Sui client...")

	// 1. Add/Update environment
	// We use ef-localhost to avoid overriding existing localnet if any
	_ = exec.Command("sui", "client", "new-env", "--alias", "ef-localhost", "--rpc", "http://localhost:9000").Run()

	// Switch to it
	if err := exec.Command("sui", "client", "switch", "--env", "ef-localhost").Run(); err != nil {
		return fmt.Errorf("failed to switch to ef-localhost: %w", err)
	}

	// 2. Import keys from .env
	envPath := filepath.Join(workspace, "world-contracts", ".env")
	configs, err := extractKeyConfigs(envPath)
	if err != nil {
		ui.Warn.Println("Could not extract keys from .env: " + err.Error())
		return nil
	}

	for _, cfg := range configs {
		// sui keytool import <key> ed25519 --alias <alias>
		ui.Info.Println(fmt.Sprintf("Importing key for %s as alias: %s", cfg.Role, cfg.Alias))
		// #nosec G204 -- Key and Alias are securely extracted via regex
		if err := exec.Command("sui", "keytool", "import", cfg.Key, "ed25519", "--alias", cfg.Alias).Run(); err != nil {
			// If already exists, we might want to update or ignore. For now, ignore but log
			ui.Warn.Println(fmt.Sprintf("Failed to import key for %s (possibly already exists): %v", cfg.Role, err))
		}
	}

	ui.Success.Println("Sui client configured with ef-localhost environment and workspace keys.")
	return nil
}

func TeardownSui() error {
	if !IsSuiInstalled() {
		return nil
	}

	ui.Info.Println("Tearing down Sui client configuration...")

	// Remove aliases
	aliases := []string{"ef-admin", "ef-player-a", "ef-player-b"}
	for _, alias := range aliases {
		// #nosec G204 -- alias array contains safe hardcoded strings
		_ = exec.Command("sui", "client", "remove-address", alias).Run()
	}

	// Sui CLI doesn't have a direct 'remove-env' command easily accessible via simple 'sui client remove-env',
	// but we've switched to others if needed. For now, we mainly care about the aliases and the env being inactive.
	// Some versions might support removing from config file, but we'll stick to what's safe.

	ui.Success.Println("Sui client environment and aliases cleaned up.")
	return nil
}

type keyConfig struct {
	Role  string
	Key   string
	Alias string
}

func extractKeyConfigs(envPath string) ([]keyConfig, error) {
	// #nosec G304 -- envPath is constructed internally using filepath.Join with a known safe relative path
	file, err := os.Open(envPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var configs []keyConfig
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if match := adminKeyRegex.FindStringSubmatch(line); match != nil {
			configs = append(configs, keyConfig{Role: "Admin", Key: match[1], Alias: "ef-admin"})
		} else if match := playerAKeyRegex.FindStringSubmatch(line); match != nil {
			configs = append(configs, keyConfig{Role: "Player A", Key: match[1], Alias: "ef-player-a"})
		} else if match := playerBKeyRegex.FindStringSubmatch(line); match != nil {
			configs = append(configs, keyConfig{Role: "Player B", Key: match[1], Alias: "ef-player-b"})
		}
	}
	return configs, scanner.Err()
}
