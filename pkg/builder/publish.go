package builder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"efctl/pkg/config"
	"efctl/pkg/container"
	"efctl/pkg/setup"
	"efctl/pkg/ui"
	"github.com/lithammer/fuzzysearch/fuzzy"
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

type PublishSearchRoot struct {
	HostPath      string
	ContainerPath string
}

type PublishCandidate struct {
	Name          string
	HostPath      string
	ContainerPath string
}

const worldDependencyMarker = "world = {"

// PublishExtension publishes the custom extension to the smart assembly testnet
// and updates the builder-scaffold/.env with the extracted package IDs.
func PublishExtension(c container.ContainerClient, workspace string, network string, candidate PublishCandidate) error {

	ui.Info.Printf("Publishing extension contract from %s...\n", candidate.HostPath)

	ui.Info.Printf("Executing publish inside container at %s...\n", candidate.ContainerPath)

	// Ensure standard dependencies are available via symlinks if we are publishing from /workspace/mounts
	if strings.HasPrefix(candidate.ContainerPath, "/workspace/mounts/") {
		if err := ensureMountDependencies(c); err != nil {
			return fmt.Errorf("failed to setup mount dependencies: %w", err)
		}
	}

	publishCmd, err := buildPublishCmd(workspace, network, candidate.ContainerPath)
	if err != nil {
		return err
	}

	// Clean stale Move.lock files before publishing to avoid framework drift issues
	setup.CleanStaleMoveLocks(workspace)
	if err := setup.PatchBuilderExampleMoveTomls(workspace); err != nil {
		return err
	}

	ui.Warn.Println("Publish logging will be piped below:")

	output, err := c.ExecCapture(context.Background(), container.ContainerSuiPlayground, []string{"/bin/bash", "-c", publishCmd})
	if output != "" {
		fmt.Print(output)
	}
	if err != nil {
		return fmt.Errorf("publish command failed: %w", err)
	}

	return writePublishedIDs(workspace, output)
}

func resolvePublishContractDir(workspace string) (PublishCandidate, error) {
	searchRoots, err := GetPublishSearchRoots(workspace)
	if err != nil {
		return PublishCandidate{}, err
	}

	candidates, err := DiscoverPublishCandidates(searchRoots)
	if err != nil {
		return PublishCandidate{}, err
	}

	if len(candidates) == 0 {
		searchedRoots := make([]string, 0, len(searchRoots))
		for _, root := range searchRoots {
			searchedRoots = append(searchedRoots, root.HostPath)
		}
		return PublishCandidate{}, fmt.Errorf("no publishable extension found; searched immediate child directories under: %s", strings.Join(searchedRoots, ", "))
	}

	if len(candidates) > 1 {
		matchingPaths := make([]string, 0, len(candidates))
		for _, candidate := range candidates {
			matchingPaths = append(matchingPaths, candidate.HostPath)
		}
		return PublishCandidate{}, fmt.Errorf("multiple publishable extensions found; aborting: %s", strings.Join(matchingPaths, ", "))
	}

	return candidates[0], nil
}

func pathExists(filePath string) (bool, error) {
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func GetPublishSearchRoots(workspace string) ([]PublishSearchRoot, error) {
	roots := []PublishSearchRoot{
		{
			HostPath:      filepath.Join(workspace, "builder-scaffold", "move-contracts"),
			ContainerPath: "/workspace/builder-scaffold/move-contracts",
		},
		{
			HostPath:      filepath.Join(workspace, "world-contracts", "contracts"),
			ContainerPath: "/workspace/world-contracts/contracts",
		},
	}

	if config.Loaded == nil {
		return roots, nil
	}

	resolvedMounts, err := config.Loaded.ResolveAdditionalBindMounts(workspace)
	if err != nil {
		return nil, err
	}

	for _, mount := range resolvedMounts {
		roots = append(roots, PublishSearchRoot{
			HostPath:      mount.HostPath,
			ContainerPath: path.Join("/workspace/mounts", mount.Identifier),
		})
	}

	return roots, nil
}

func DiscoverPublishCandidates(searchRoots []PublishSearchRoot) ([]PublishCandidate, error) {
	var candidates []PublishCandidate

	for _, root := range searchRoots {
		entries, err := os.ReadDir(root.HostPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to read publish root %s: %w", root.HostPath, err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			hostCandidatePath := filepath.Join(root.HostPath, entry.Name())
			manifestPath := filepath.Join(hostCandidatePath, "Move.toml")
			moveTomlExists, err := pathExists(manifestPath)
			if err != nil {
				return nil, fmt.Errorf("failed to stat Move.toml for %s: %w", hostCandidatePath, err)
			}
			if !moveTomlExists {
				continue
			}

			isExtension, err := isExtensionManifest(manifestPath)
			if err != nil {
				return nil, fmt.Errorf("failed to inspect Move.toml for %s: %w", hostCandidatePath, err)
			}
			if !isExtension {
				continue
			}

			candidates = append(candidates, PublishCandidate{
				Name:          entry.Name(),
				HostPath:      hostCandidatePath,
				ContainerPath: path.Join(root.ContainerPath, entry.Name()),
			})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].HostPath < candidates[j].HostPath
	})

	return candidates, nil
}

func isExtensionManifest(manifestPath string) (bool, error) {
	manifestContent, err := os.ReadFile(manifestPath) // #nosec G304 -- manifestPath is constructed from workspace-local directories discovered via os.ReadDir
	if err != nil {
		return false, err
	}

	return strings.Contains(string(manifestContent), worldDependencyMarker), nil
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
			"cd %s && sui client test-publish --with-unpublished-dependencies --build-env testnet --pubfile-path /workspace/builder-scaffold/deployments/localnet/Pub.extension.toml --json",
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

// ensureMountDependencies creates symlinks in /workspace/mounts so that extensions
// can resolve relative dependencies like ../../world-contracts.
func ensureMountDependencies(c container.ContainerClient) error {
	setupCmd := "mkdir -p /workspace/mounts && " +
		"ln -sf /workspace/world-contracts /workspace/mounts/world-contracts && " +
		"ln -sf /workspace/builder-scaffold /workspace/mounts/builder-scaffold"

	return c.Exec(context.Background(), container.ContainerSuiPlayground, []string{"/bin/bash", "-c", setupCmd})
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

// GetCandidate finds a candidate by its container path.
func GetCandidate(workspace, containerPath string) (PublishCandidate, error) {
	searchRoots, err := GetPublishSearchRoots(workspace)
	if err != nil {
		return PublishCandidate{}, err
	}

	candidates, err := DiscoverPublishCandidates(searchRoots)
	if err != nil {
		return PublishCandidate{}, err
	}

	for _, c := range candidates {
		// Check for absolute match
		if c.ContainerPath == containerPath {
			return c, nil
		}
		// Check for relative match (relative to /workspace)
		if strings.TrimPrefix(c.ContainerPath, "/workspace/") == containerPath {
			return c, nil
		}
	}

	return PublishCandidate{}, fmt.Errorf("extension %q not found", containerPath)
}

// FindClosestMatch returns a list of candidates (relative to /workspace) sorted by Levenshtein distance to the target.
func FindClosestMatch(workspace, target string) []string {
	searchRoots, err := GetPublishSearchRoots(workspace)
	if err != nil {
		return nil
	}

	candidates, err := DiscoverPublishCandidates(searchRoots)
	if err != nil {
		return nil
	}

	type match struct {
		name     string
		distance int
	}
	var matches []match
	for _, c := range candidates {
		rel := strings.TrimPrefix(c.ContainerPath, "/workspace/")
		// Calculate distance against both absolute and relative paths
		distAbs := fuzzy.LevenshteinDistance(target, c.ContainerPath)
		distRel := fuzzy.LevenshteinDistance(target, rel)

		minDist := distAbs
		if distRel < minDist {
			minDist = distRel
		}

		matches = append(matches, match{rel, minDist})
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].distance == matches[j].distance {
			return matches[i].name < matches[j].name
		}
		return matches[i].distance < matches[j].distance
	})

	var result []string
	for i := 0; i < len(matches) && i < 3; i++ {
		result = append(result, matches[i].name)
	}
	return result
}
