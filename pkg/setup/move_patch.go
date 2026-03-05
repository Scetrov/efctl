package setup

import (
	"log"
	"os"
	"path/filepath"

	"efctl/pkg/ui"
)

// cleanStaleMoveLocks removes Move.lock files from world-contracts so
// that `sui client test-publish --build-env testnet` resolves framework
// dependencies from the Sui binary rather than from stale pinned git
// revisions that may no longer exist upstream.
func cleanStaleMoveLocks(workspace string) {
	contractsDir := filepath.Join(workspace, "world-contracts", "contracts")

	entries, err := os.ReadDir(contractsDir)
	if err != nil {
		log.Printf("move_patch: cannot read contracts dir: %v", err)
		return
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		lockPath := filepath.Join(contractsDir, e.Name(), "Move.lock")
		if err := os.Remove(lockPath); err == nil {
			ui.Debug.Printfln("Removed stale %s", filepath.Join("contracts", e.Name(), "Move.lock"))
		}
	}
}
