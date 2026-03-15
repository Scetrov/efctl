package container

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"efctl/pkg/ui"
)

// CheckPodmanConfig performs diagnostics on the Podman configuration.
// 1. Checks if ~/.config/containers/containers.conf exists.
// 2. If it exists, logs an INF message.
// 3. Parses the file to ensure firewall_driver = "iptables" under [network].
// 4. If missing, logs a WARN message.
func CheckPodmanConfig() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	configPath := filepath.Join(home, ".config/containers/containers.conf")
	data, err := os.ReadFile(configPath) // #nosec G304
	if err != nil {
		// File does not exist or cannot be read, which is common if user is using defaults.
		// The prompt says: "If the engine is podman, check for the existence of the file ~/.config/containers/containers.conf."
		// "If the file exists, log an INF level message exactly as: ~/.config/containers/containers.conf exists"
		return
	}

	ui.Info.Println("~/.config/containers/containers.conf exists")

	if !hasIptablesConfig(string(data)) {
		ui.Warn.Println("The firewall driver is not set. You may need to add:\n\n[network]\nfirewall_driver = \"iptables\"\n\nif you receive error messages about `nft` failing.")
	}
}

func hasIptablesConfig(content string) bool {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var currentSection string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Remove inline comments
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.Trim(line, "[]")
			continue
		}

		if currentSection == "network" {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
				if key == "firewall_driver" && val == "iptables" {
					return true
				}
			}
		}
	}
	return false
}
