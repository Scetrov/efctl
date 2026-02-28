package builder

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"efctl/pkg/container"
	"efctl/pkg/ui"
)

// publishOutput represents the relevant parts of the JSON from `sui client publish --json`.
type publishOutput struct {
	ObjectChanges []objectChange `json:"objectChanges"`
}

type objectChange struct {
	Type       string `json:"type"`
	PackageID  string `json:"packageId"`
	ObjectID   string `json:"objectId"`
	ObjectType string `json:"objectType"`
}

// PublishExtension publishes the custom extension to the smart assembly testnet
// and updates the builder-scaffold/.env with the extracted package IDs.
func PublishExtension(c container.ContainerClient, workspace string, network string, contractPath string) error {

	// Ensure no leading slashes — treat it as relative to move-contracts
	cleanContractPath := filepath.Clean(contractPath)
	if strings.HasPrefix(cleanContractPath, "/") {
		return fmt.Errorf("contract path must be relative to builder-scaffold/move-contracts, got absolute: %s", contractPath)
	}

	containerContractDir := fmt.Sprintf("/workspace/builder-scaffold/move-contracts/%s", cleanContractPath)
	ui.Info.Printf("Executing publish inside container at %s...\n", containerContractDir)

	publishCmd, err := buildPublishCmd(workspace, network, containerContractDir)
	if err != nil {
		return err
	}

	ui.Warn.Println("Publish logging will be piped below:")

	output, err := c.ExecCapture(container.ContainerSuiPlayground, []string{"/bin/bash", "-c", publishCmd})
	if output != "" {
		fmt.Print(output)
	}
	if err != nil {
		return fmt.Errorf("publish command failed: %w", err)
	}

	return writePublishedIDs(workspace, output)
}

// buildPublishCmd constructs the sui publish command and, for localnet, deletes any
// stale ephemeral publication file so that re-running is idempotent.
func buildPublishCmd(workspace, network, containerContractDir string) (string, error) {
	switch network {
	case "localnet":
		pubFile := filepath.Join(workspace, "builder-scaffold", "deployments", network, "Pub.extension.toml")
		if err := os.Remove(pubFile); err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to remove previous publish file: %w", err)
		}
		return fmt.Sprintf(
			"cd %s && sui client test-publish --with-unpublished-dependencies --build-env testnet --pubfile-path ../../deployments/localnet/Pub.extension.toml --json",
			containerContractDir,
		), nil

	case "testnet":
		return fmt.Sprintf(
			"cd %s && sui client publish --with-unpublished-dependencies --build-env testnet --json",
			containerContractDir,
		), nil

	default:
		return "", fmt.Errorf("unsupported network %s", network)
	}
}

// writePublishedIDs parses the publish command JSON output and writes the discovered
// package and config IDs into builder-scaffold/.env.
func writePublishedIDs(workspace, output string) error {
	builderPackageID, extensionConfigID, parseErr := extractPublishIDs(output)
	if parseErr != nil {
		ui.Warn.Printf("Could not parse publish output as JSON: %v\n", parseErr)
	}

	if builderPackageID == "" {
		ui.Warn.Println("Could not automatically extract BUILDER_PACKAGE_ID. Please set it manually in builder-scaffold/.env")
	}
	if extensionConfigID == "" {
		ui.Warn.Println("Could not automatically extract EXTENSION_CONFIG_ID. Please set it manually in builder-scaffold/.env")
	}

	if builderPackageID == "" && extensionConfigID == "" {
		return nil
	}

	updates := map[string]string{}
	if builderPackageID != "" {
		updates["BUILDER_PACKAGE_ID"] = builderPackageID
	}
	if extensionConfigID != "" {
		updates["EXTENSION_CONFIG_ID"] = extensionConfigID
	}

	envFile := filepath.Join(workspace, "builder-scaffold", ".env")
	if err := updateEnvFile(envFile, updates); err != nil {
		return fmt.Errorf("failed to update builder-scaffold/.env: %w", err)
	}

	if builderPackageID != "" {
		ui.Info.Printf("BUILDER_PACKAGE_ID = %s\n", builderPackageID)
	}
	if extensionConfigID != "" {
		ui.Info.Printf("EXTENSION_CONFIG_ID = %s\n", extensionConfigID)
	}
	ui.Success.Println("builder-scaffold/.env updated with published IDs.")
	return nil
}

// extractPublishIDs parses the JSON from `sui client publish --json` and returns
// the newly published package ID and the ExtensionConfig object ID.
//
// The relevant portion of the JSON looks like:
//
//	"objectChanges": [
//	  { "type": "published", "packageId": "0x..." },
//	  { "type": "created", "objectType": "...::ExtensionConfig", "objectId": "0x..." }
//	]
func extractPublishIDs(output string) (builderPackageID, extensionConfigID string, err error) {
	// The sui CLI may emit non-JSON build logs before the JSON block.
	// Find the first '{' to locate the start of the JSON object.
	jsonStart := strings.Index(output, "{")
	if jsonStart == -1 {
		return "", "", fmt.Errorf("no JSON object found in output")
	}

	var result publishOutput
	if err := json.Unmarshal([]byte(output[jsonStart:]), &result); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal publish output: %w", err)
	}

	for _, change := range result.ObjectChanges {
		if change.Type == "published" && change.PackageID != "" {
			builderPackageID = change.PackageID
		}
		if change.Type == "created" &&
			strings.Contains(strings.ToLower(change.ObjectType), "extensionconfig") &&
			change.ObjectID != "" {
			extensionConfigID = change.ObjectID
		}
	}

	return builderPackageID, extensionConfigID, nil
}
