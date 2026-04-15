package setup

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"efctl/pkg/container"
	"efctl/pkg/ui"
)

// containerEnvPath is the path to the world-contracts .env file inside the
// sui-playground container.
const containerEnvPath = "/workspace/world-contracts/.env"

// CleanStaleMoveLocks removes Move.lock files from world-contracts so
// that `sui client test-publish --build-env testnet` resolves framework
// dependencies from the Sui binary rather than from stale pinned git
// revisions that may no longer exist upstream.
func CleanStaleMoveLocks(workspace string) {
	removeMoveLocksInSubdirs(filepath.Join(workspace, "world-contracts", "contracts"), "contracts")
	removeMoveLocksInSubdirs(filepath.Join(workspace, "builder-scaffold", "move-contracts"), filepath.Join("builder-scaffold", "move-contracts"))
}

// PatchBuilderExampleMoveTomls removes legacy named-address sections from the
// bundled builder-scaffold example packages so they remain publishable against
// CCP's recommended world-contracts v0.0.23 checkout.
func PatchBuilderExampleMoveTomls(workspace string) error {
	for _, contractName := range []string{"smart_gate_extension", "storage_unit_extension"} {
		moveTomlPath := filepath.Join(workspace, "builder-scaffold", "move-contracts", contractName, "Move.toml")
		changed, err := removeManifestSection(moveTomlPath, "addresses")
		if err != nil {
			return err
		}
		if changed {
			ui.Debug.Printfln("Patched legacy [addresses] section in %s", filepath.Join("builder-scaffold", "move-contracts", contractName, "Move.toml"))
		}
	}

	return nil
}

func removeMoveLocksInSubdirs(root string, debugPrefix string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		log.Printf("move_patch: cannot read %s: %v", root, err)
		return
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		lockPath := filepath.Join(root, e.Name(), "Move.lock")
		if err := os.Remove(lockPath); err == nil {
			ui.Debug.Printfln("Removed stale %s", filepath.Join(debugPrefix, e.Name(), "Move.lock"))
		}
	}
}

func removeManifestSection(filePath string, sectionName string) (bool, error) {
	content, err := os.ReadFile(filePath) // #nosec G304 -- path is workspace-local and constructed by caller
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to read %s: %w", filePath, err)
	}

	lines := strings.Split(string(content), "\n")
	sectionHeader := fmt.Sprintf("[%s]", sectionName)
	updated := make([]string, 0, len(lines))
	removed := false
	inSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !inSection && trimmed == sectionHeader {
			inSection = true
			removed = true
			continue
		}

		if inSection {
			if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
				inSection = false
			} else {
				continue
			}
		}

		updated = append(updated, line)
	}

	if !removed {
		return false, nil
	}

	updated = trimRepeatedBlankLines(updated)
	if err := os.WriteFile(filePath, []byte(strings.Join(updated, "\n")), 0600); err != nil {
		return false, fmt.Errorf("failed to write %s: %w", filePath, err)
	}

	return true, nil
}

func trimRepeatedBlankLines(lines []string) []string {
	compacted := make([]string, 0, len(lines))
	prevBlank := false

	for _, line := range lines {
		isBlank := strings.TrimSpace(line) == ""
		if isBlank && prevBlank {
			continue
		}
		compacted = append(compacted, line)
		prevBlank = isBlank
	}

	for len(compacted) > 0 && compacted[0] == "" {
		compacted = compacted[1:]
	}
	for len(compacted) > 0 && compacted[len(compacted)-1] == "" {
		compacted = compacted[:len(compacted)-1]
	}

	if len(compacted) == 0 {
		return []string{}
	}

	if !slices.Equal(compacted, lines) {
		return compacted
	}

	return compacted
}

// ensureWorldSponsorAddresses backfills SPONSOR_ADDRESS and SPONSOR_ADDRESSES
// from ADMIN_ADDRESS when upstream env-generation scripts fail to populate them.
//
// The .env file is created by a script running as root inside the container,
// so it is owned by root on the host.  To avoid permission-denied errors we
// read and write the file through the container using ExecCapture / Exec.
func ensureWorldSponsorAddresses(c container.ContainerClient, containerName string) {
	data, err := c.ExecCapture(context.Background(), containerName, []string{"cat", containerEnvPath})
	if err != nil {
		log.Printf("move_patch: cannot read world env file via container: %v", err)
		return
	}

	lines := strings.Split(data, "\n")
	admin := ""
	sponsorVal := ""
	sponsorsVal := ""

	for _, line := range lines {
		if strings.HasPrefix(line, "ADMIN_ADDRESS=") {
			admin = strings.TrimSpace(strings.TrimPrefix(line, "ADMIN_ADDRESS="))
		}
		if strings.HasPrefix(line, "SPONSOR_ADDRESS=") {
			sponsorVal = strings.TrimSpace(strings.TrimPrefix(line, "SPONSOR_ADDRESS="))
		}
		if strings.HasPrefix(line, "SPONSOR_ADDRESSES=") {
			sponsorsVal = strings.TrimSpace(strings.TrimPrefix(line, "SPONSOR_ADDRESSES="))
		}
	}

	if admin == "" || (sponsorVal != "" && sponsorsVal != "") {
		return
	}

	// Use sed to replace empty values or append if missing.
	sedCmds := []string{}
	if sponsorVal == "" {
		ui.Debug.Printfln("move_patch: patching SPONSOR_ADDRESS=%s", admin)
		sedCmds = append(sedCmds, fmt.Sprintf(
			`grep -q '^SPONSOR_ADDRESS=' '%s' && `+
				`sed -i 's/^SPONSOR_ADDRESS=.*/SPONSOR_ADDRESS=%s/' '%s' || `+
				`echo 'SPONSOR_ADDRESS=%s' >> '%s'`,
			containerEnvPath, admin, containerEnvPath, admin, containerEnvPath,
		))
	}
	if sponsorsVal == "" {
		ui.Debug.Printfln("move_patch: patching SPONSOR_ADDRESSES=%s", admin)
		sedCmds = append(sedCmds, fmt.Sprintf(
			`grep -q '^SPONSOR_ADDRESSES=' '%s' && `+
				`sed -i 's/^SPONSOR_ADDRESSES=.*/SPONSOR_ADDRESSES=%s/' '%s' || `+
				`echo 'SPONSOR_ADDRESSES=%s' >> '%s'`,
			containerEnvPath, admin, containerEnvPath, admin, containerEnvPath,
		))
	}

	if len(sedCmds) == 0 {
		return
	}

	fullCmd := strings.Join(sedCmds, " && ")
	if execErr := c.Exec(context.Background(), containerName, []string{"/bin/bash", "-c", fullCmd}); execErr != nil {
		log.Printf("move_patch: cannot write world env file via container: %v", execErr)
		return
	}

	ui.Debug.Println("Backfilled missing sponsor fields from ADMIN_ADDRESS in world-contracts/.env")
}
