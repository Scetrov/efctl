package setup

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

// patchPnpmDependencies injects pnpm configuration into package.json files
// to allow esbuild build scripts, avoiding pnpm warnings during install.
func patchPnpmDependencies(workspace string) {
	repos := []string{"builder-scaffold", "world-contracts"}
	for _, repo := range repos {
		path := filepath.Join(workspace, repo, "package.json")
		if err := patchPackageJSON(path); err != nil {
			log.Printf("pnpm_patch: failed to patch %s: %v", path, err)
		}
	}
}

func patchPackageJSON(path string) error {
	data, err := os.ReadFile(path) // #nosec G304 -- path is validated to be within workspace
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	content := string(data)
	if strings.Contains(content, "\"onlyBuiltDependencies\"") && strings.Contains(content, "\"esbuild\"") {
		return nil
	}

	pnpmBlock := `  "pnpm": {
    "onlyBuiltDependencies": [
      "esbuild"
    ]
  },`

	// Try to inject after packageManager or before scripts
	if strings.Contains(content, "\"packageManager\":") {
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if strings.Contains(line, "\"packageManager\":") {
				lines[i] = line + "\n" + pnpmBlock
				break
			}
		}
		content = strings.Join(lines, "\n")
	} else if strings.Contains(content, "\"scripts\": {") {
		content = strings.Replace(content, "\"scripts\": {", pnpmBlock+"\n  \"scripts\": {", 1)
	} else {
		// Fallback: inject after the first opening brace
		content = strings.Replace(content, "{", "{\n"+pnpmBlock, 1)
	}

	return os.WriteFile(path, []byte(content), 0600) // #nosec G306 G703 -- restricted permissions and path is within workspace
}
