package setup

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

// patchPnpmDependencies injects pnpm configuration into package.json files
// and .npmrc to allow esbuild build scripts, avoiding pnpm warnings during
// install. Writing to both package.json and .npmrc ensures pnpm 10+ with
// corepack picks up the config regardless of which resolution path is used.
func patchPnpmDependencies(workspace string) {
	repos := []string{"builder-scaffold", "world-contracts"}
	for _, repo := range repos {
		repoDir := filepath.Join(workspace, repo)

		// Patch package.json
		pkgPath := filepath.Join(repoDir, "package.json")
		if err := patchPackageJSON(pkgPath); err != nil {
			log.Printf("pnpm_patch: failed to patch %s: %v", pkgPath, err)
		}

		// Also patch .npmrc for pnpm 10+ corepack compatibility.
		// This is the most reliable way to configure onlyBuiltDependencies
		// because .npmrc is always read regardless of packageManager field.
		npmrcPath := filepath.Join(repoDir, ".npmrc")
		if err := patchNpmrc(npmrcPath); err != nil {
			log.Printf("pnpm_patch: failed to patch %s: %v", npmrcPath, err)
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

// patchNpmrc injects pnpm configuration into .npmrc files
// to allow esbuild build scripts. This complements the package.json patch
// and is the most reliable method for pnpm 10+ with corepack.
func patchNpmrc(path string) error {
	data, err := os.ReadFile(path) // #nosec G304 -- path is validated to be within workspace
	if err != nil {
		if os.IsNotExist(err) {
			// Create .npmrc if it doesn't exist
			content := "onlyBuiltDependencies=esbuild\n"
			return os.WriteFile(path, []byte(content), 0600) // #nosec G304
		}
		return err
	}

	content := string(data)
	if strings.Contains(content, "onlyBuiltDependencies=esbuild") {
		return nil
	}

	// Append the onlyBuiltDependencies config
	content = strings.TrimRight(content, "\n") + "\nonlyBuiltDependencies=esbuild\n"

	return os.WriteFile(path, []byte(content), 0600) // #nosec G306 G703 -- path validated by safePath; G703 false positive from taint analysis
}
