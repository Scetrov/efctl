package builder

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"efctl/pkg/ui"
)

// InitExtensionEnv performs Step 6 and Step 7 of the builder flow.
// It copies the world artifacts from world-contracts to builder-scaffold,
// and configures the builder-scaffold's .env file inline.
func InitExtensionEnv(workspace string, network string) error {
	worldContractsDir := filepath.Join(workspace, "world-contracts")
	builderScaffoldDir := filepath.Join(workspace, "builder-scaffold")

	// Step 6: Copy world artifacts
	// mkdir -p /workspace/builder-scaffold/deployments/$NETWORK/
	builderDeploymentsDir := filepath.Join(builderScaffoldDir, "deployments", network)
	if err := os.MkdirAll(builderDeploymentsDir, 0750); err != nil { // #nosec G301
		return fmt.Errorf("failed to create deployments dir: %w", err)
	}

	// cp -r deployments/* /workspace/builder-scaffold/deployments/
	// (we selectively copy the network folder for safety and test-resources.json)
	srcDeployNetworkDir := filepath.Join(worldContractsDir, "deployments", network)
	if err := copyDir(srcDeployNetworkDir, builderDeploymentsDir); err != nil {
		return fmt.Errorf("failed to copy deployment network dir: %w", err)
	}

	// cp test-resources.json /workspace/builder-scaffold/test-resources.json
	srcTestResources := filepath.Join(worldContractsDir, "test-resources.json")
	dstTestResources := filepath.Join(builderScaffoldDir, "test-resources.json")
	if err := copyFile(srcTestResources, dstTestResources); err != nil {
		return fmt.Errorf("failed to copy test-resources.json: %w", err)
	}

	// cp "contracts/world/Pub.localnet.toml" "/workspace/builder-scaffold/deployments/localnet/Pub.localnet.toml"
	pubTomlFile := fmt.Sprintf("Pub.%s.toml", network)
	srcPubToml := filepath.Join(worldContractsDir, "contracts", "world", pubTomlFile)
	dstPubToml := filepath.Join(builderDeploymentsDir, pubTomlFile)
	if err := copyFile(srcPubToml, dstPubToml); err != nil {
		ui.Warn.Printf("Could not copy %s (may not exist): %v\n", pubTomlFile, err)
	}

	ui.Info.Println("Copied world artifacts into builder-scaffold deployments.")

	// Step 7: Configure builder-scaffold .env
	// cp .env.example .env
	srcEnvExample := filepath.Join(builderScaffoldDir, ".env.example")
	dstEnv := filepath.Join(builderScaffoldDir, ".env")
	if err := copyFile(srcEnvExample, dstEnv); err != nil {
		return fmt.Errorf("failed to copy .env.example to .env: %w", err)
	}

	// Read world-contracts/.env to fetch admin/player keys
	worldEnvFile := filepath.Join(worldContractsDir, ".env")
	worldEnvMap, err := parseDotEnv(worldEnvFile)
	if err != nil {
		return fmt.Errorf("failed to parse world .env: %w", err)
	}

	// Read world package id
	extractedIdsFile := filepath.Join(builderDeploymentsDir, "extracted-object-ids.json")
	worldPackageId, err := extractWorldPackageId(extractedIdsFile)
	if err != nil {
		return fmt.Errorf("failed to extract world.packageId: %w", err)
	}

	// Update builder-scaffold/.env
	envUpdates := map[string]string{
		"SUI_NETWORK":          network,
		"WORLD_PACKAGE_ID":     worldPackageId,
		"ADMIN_ADDRESS":        worldEnvMap["ADMIN_ADDRESS"],
		"ADMIN_PRIVATE_KEY":    worldEnvMap["ADMIN_PRIVATE_KEY"],
		"PLAYER_A_ADDRESS":     worldEnvMap["PLAYER_A_ADDRESS"],
		"PLAYER_A_PRIVATE_KEY": worldEnvMap["PLAYER_A_PRIVATE_KEY"],
		"PLAYER_B_ADDRESS":     worldEnvMap["PLAYER_B_ADDRESS"],
		"PLAYER_B_PRIVATE_KEY": worldEnvMap["PLAYER_B_PRIVATE_KEY"],
	}

	if err := updateEnvFile(dstEnv, envUpdates); err != nil {
		return fmt.Errorf("failed to update builder-scaffold .env: %w", err)
	}

	ui.Info.Println("Configured builder-scaffold .env.")

	return nil
}

// Helpers
func copyFile(src, dst string) error {
	in, err := os.Open(src) // #nosec G304
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst) // #nosec G304
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := os.MkdirAll(dstPath, 0750); err != nil { // #nosec G301
				return err
			}
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func parseDotEnv(path string) (map[string]string, error) {
	file, err := os.Open(path) // #nosec G304
	if err != nil {
		return nil, err
	}
	defer file.Close()

	envMap := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			envMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return envMap, scanner.Err()
}

func extractWorldPackageId(path string) (string, error) {
	content, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		return "", err
	}
	var data struct {
		World struct {
			PackageId string `json:"packageId"`
		} `json:"world"`
	}

	if err := json.Unmarshal(content, &data); err != nil {
		return "", err
	}

	return data.World.PackageId, nil
}

func updateEnvFile(path string, updates map[string]string) error {
	content, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	updatedMap := make(map[string]bool)

	var newLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			newLines = append(newLines, line)
			continue
		}

		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			if val, ok := updates[key]; ok {
				newLines = append(newLines, fmt.Sprintf("%s=%s", key, val))
				updatedMap[key] = true
				continue
			}
		}
		newLines = append(newLines, line)
	}

	// Append any missing keys
	for k, v := range updates {
		if !updatedMap[k] {
			newLines = append(newLines, fmt.Sprintf("%s=%s", k, v))
		}
	}

	cleanPath := filepath.Clean(path)
	return os.WriteFile(cleanPath, []byte(strings.Join(newLines, "\n")), 0600) // #nosec G306 G703 -- path is constructed from workspace-local filepath.Join in caller
}
