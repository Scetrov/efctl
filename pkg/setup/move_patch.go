package setup

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"efctl/pkg/ui"
)

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
func ensureWorldSponsorAddresses(workspace string) {
	envPath := filepath.Join(workspace, "world-contracts", ".env")
	data, err := os.ReadFile(envPath) // #nosec G304
	if err != nil {
		log.Printf("move_patch: cannot read world env file: %v", err)
		return
	}

	content := string(data)
	lines := strings.Split(content, "\n")
	admin := ""
	sponsorIdx := -1
	sponsorVal := ""

	for i, line := range lines {
		if strings.HasPrefix(line, "ADMIN_ADDRESS=") {
			admin = strings.TrimSpace(strings.TrimPrefix(line, "ADMIN_ADDRESS="))
		}
		if strings.HasPrefix(line, "SPONSOR_ADDRESSES=") {
			sponsorIdx = i
			sponsorVal = strings.TrimSpace(strings.TrimPrefix(line, "SPONSOR_ADDRESSES="))
		}
	}

	if admin == "" || sponsorVal != "" {
		return
	}

	newLine := "SPONSOR_ADDRESSES=" + admin
	if sponsorIdx >= 0 {
		lines[sponsorIdx] = newLine
	} else {
		lines = append(lines, newLine)
	}

	updated := strings.Join(lines, "\n")
	mode := os.FileMode(0600)
	if fi, statErr := os.Stat(envPath); statErr == nil {
		mode = fi.Mode().Perm()
	}

	if writeErr := os.WriteFile(envPath, []byte(updated), mode); writeErr != nil { // #nosec G703
		log.Printf("move_patch: cannot write world env file: %v", writeErr)
		return
	}

	ui.Debug.Println("Backfilled SPONSOR_ADDRESSES from ADMIN_ADDRESS in world-contracts/.env")
}
