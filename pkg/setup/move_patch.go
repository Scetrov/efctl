package setup

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"efctl/pkg/container"
	"efctl/pkg/ui"
)

// containerEnvPath is the path to the world-contracts .env file inside the
// sui-playground container.
const containerEnvPath = "/workspace/world-contracts/.env"

// cleanStaleMoveLocks removes Move.lock files from world-contracts so
// that `sui client test-publish --build-env testnet` resolves framework
// dependencies from the Sui binary rather than from stale pinned git
// revisions that may no longer exist upstream.
func cleanStaleMoveLocks(workspace string) {
	removeMoveLocksInSubdirs(filepath.Join(workspace, "world-contracts", "contracts"), "contracts")
	removeMoveLocksInSubdirs(filepath.Join(workspace, "builder-scaffold", "move-contracts"), filepath.Join("builder-scaffold", "move-contracts"))
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

// ensureWorldSponsorAddresses backfills SPONSOR_ADDRESSES from ADMIN_ADDRESS
// when upstream env-generation scripts fail to populate the sponsor list.
//
// The .env file is created by a script running as root inside the container,
// so it is owned by root on the host.  To avoid permission-denied errors we
// read and write the file through the container using ExecCapture / Exec.
func ensureWorldSponsorAddresses(c container.ContainerClient, containerName string) {
	data, err := c.ExecCapture(containerName, []string{"cat", containerEnvPath})
	if err != nil {
		log.Printf("move_patch: cannot read world env file via container: %v", err)
		return
	}

	lines := strings.Split(data, "\n")
	admin := ""
	sponsorVal := ""

	for _, line := range lines {
		if strings.HasPrefix(line, "ADMIN_ADDRESS=") {
			admin = strings.TrimSpace(strings.TrimPrefix(line, "ADMIN_ADDRESS="))
		}
		if strings.HasPrefix(line, "SPONSOR_ADDRESSES=") {
			sponsorVal = strings.TrimSpace(strings.TrimPrefix(line, "SPONSOR_ADDRESSES="))
		}
	}

	if admin == "" || sponsorVal != "" {
		return
	}

	// Use sed to replace an empty SPONSOR_ADDRESSES line, or append if missing.
	sedCmd := fmt.Sprintf(
		`grep -q '^SPONSOR_ADDRESSES=' '%s' && `+
			`sed -i 's/^SPONSOR_ADDRESSES=.*/SPONSOR_ADDRESSES=%s/' '%s' || `+
			`echo 'SPONSOR_ADDRESSES=%s' >> '%s'`,
		containerEnvPath, admin, containerEnvPath, admin, containerEnvPath,
	)

	if execErr := c.Exec(containerName, []string{"/bin/bash", "-c", sedCmd}); execErr != nil {
		log.Printf("move_patch: cannot write world env file via container: %v", execErr)
		return
	}

	ui.Debug.Println("Backfilled SPONSOR_ADDRESSES from ADMIN_ADDRESS in world-contracts/.env")
}
