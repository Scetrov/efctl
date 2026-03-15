package config

import "fmt"

// Recommended refs track the CCP-documented compatible repository pair.
const RecommendedWorldContractsRef = "v0.0.18"

const RecommendedBuilderScaffoldRef = "v0.0.2"

// DefaultConfigYAML returns the scaffolded efctl config file content.
func DefaultConfigYAML() string {
	return fmt.Sprintf(`# efctl.yaml — Configuration file for efctl CLI
# All properties are optional. CLI flags (e.g. --with-frontend) override these values.

# Enable the builder-scaffold web frontend (Vite dev server on port 5173)
with-frontend: true

# Enable the SQL Indexer and GraphQL API
with-graphql: true

# Git clone URL for the world-contracts repository
world-contracts-url: %q

# Ref (branch, tag, or commit) to checkout for world-contracts (default: main)
world-contracts-ref: %q
# world-contracts-branch: %q # Deprecated: use world-contracts-ref

# Git clone URL for the builder-scaffold repository
builder-scaffold-url: %q

# Ref (branch, tag, or commit) to checkout for builder-scaffold (default: main)
builder-scaffold-ref: %q
# builder-scaffold-branch: %q # Deprecated: use builder-scaffold-ref

# Configure Git core.autocrlf for cloned repositories (default: false)
git-autocrlf: false

# Preferred container engine: "docker", "podman", or "auto-detect" (default: auto-detect)
# If Podman networking fails on WSL, try setting this to "docker".
container-engine: auto-detect

# Additional host directories to bind-mount into the container environment.
# additional-bind-mounts:
#   - hostPath: ./my-extension
#     identifier: my-extension
`, DefaultWorldContractsURL, RecommendedWorldContractsRef, RecommendedWorldContractsRef, DefaultBuilderScaffoldURL, RecommendedBuilderScaffoldRef, RecommendedBuilderScaffoldRef)
}
