package setup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// patchPnpmDependencies creates pnpm-workspace.yaml in each repo directory
// to allow esbuild build scripts. Since pnpm v10.26+, onlyBuiltDependencies
// was replaced by allowBuilds in pnpm-workspace.yaml. The .npmrc file only
// reads auth/registry settings and is ignored for build-related configs.
//
// pnpm-workspace.yaml format (valid for pnpm v10.26+ and v11+):
//
//	allowBuilds:
//	  esbuild: true
func patchPnpmDependencies(workspace string) error {
	repos := []string{"builder-scaffold", "world-contracts"}
	var errs []error
	for _, repo := range repos {
		repoDir := filepath.Join(workspace, repo)

		// Create pnpm-workspace.yaml with allowBuilds for esbuild.
		// This is the only supported location for build-related settings
		// in pnpm v10.26+ (allowBuilds replaces onlyBuiltDependencies in v11).
		workspacePath := filepath.Join(repoDir, "pnpm-workspace.yaml")
		if err := patchPnpmWorkspaceYaml(workspacePath); err != nil {
			errs = append(errs, fmt.Errorf("patch pnpm-workspace.yaml in %s: %w", repo, err))
		}
	}
	return errors.Join(errs...)
}

// patchPnpmWorkspaceYaml creates or updates pnpm-workspace.yaml with
// allowBuilds configuration for esbuild. Idempotent — safe to re-run.
func patchPnpmWorkspaceYaml(path string) error {
	// Read existing content if present
	existing, err := os.ReadFile(path) // #nosec G304 -- path is within workspace
	if err != nil {
		if os.IsNotExist(err) {
			// Create new file
			content := "allowBuilds:\n  esbuild: true\n"
			return os.WriteFile(path, []byte(content), 0600) // #nosec G306 -- path validated; restricted permissions
		}
		return err
	}

	content := string(existing)
	// Check if already patched
	if containsAllowBuildsForEsbuild(content) {
		return nil
	}

	// If there's already an allowBuilds: block without esbuild, merge it
	if hasAllowBuildsBlock(content) {
		content = mergeAllowBuildsEsbuild(content)
		return os.WriteFile(path, []byte(content), 0600) // #nosec G306 G703 -- path validated; restricted permissions
	}

	// Append allowBuilds section
	content += "\nallowBuilds:\n  esbuild: true\n"
	return os.WriteFile(path, []byte(content), 0600) // #nosec G306 G703 -- path validated; restricted permissions
}

// hasAllowBuildsBlock checks if content contains an allowBuilds: top-level key
// (without esbuild: beneath it, since containsAllowBuildsForEsbuild already checked that).
var allowBuildsBlockRe = regexp.MustCompile(`(?m)^allowBuilds:\s*$`)

func hasAllowBuildsBlock(content string) bool {
	return allowBuildsBlockRe.MatchString(content)
}

// mergeAllowBuildsEsbuild inserts "esbuild: true" at the correct indentation
// level (2 spaces) into the first allowBuilds: block.
func mergeAllowBuildsEsbuild(content string) string {
	// Find the allowBuilds: line and insert esbuild right after it.
	return allowBuildsBlockRe.ReplaceAllStringFunc(content, func(match string) string {
		return match + "\n  esbuild: true"
	})
}

// containsAllowBuildsForEsbuild checks if the content already has
// allowBuilds with esbuild (true or false) — used to ensure idempotency.
func containsAllowBuildsForEsbuild(content string) bool {
	// Single regex: match allowBuilds block containing an indented esbuild entry.
	// This prevents false positives where esbuild appears as a sibling key outside
	// the allowBuilds section.
	pat := regexp.MustCompile(`(?m)^allowBuilds:\s*\n(\s+.*\n)*?\s+esbuild:\s+(true|false)\s*$`)
	return pat.MatchString(content)
}

// pnpmWorkspaceYamlContent returns the canonical pnpm-workspace.yaml content
// for allowing esbuild build scripts. Used by tests.
const pnpmWorkspaceYamlContent = "allowBuilds:\n  esbuild: true\n"

// pnpmWorkspaceYamlRelPath returns the relative path to pnpm-workspace.yaml
// within a repo directory. Used by tests.
func pnpmWorkspaceYamlRelPath() string {
	return "pnpm-workspace.yaml"
}
