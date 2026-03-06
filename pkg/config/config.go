package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// safeBranchRe matches valid git branch names (alphanumeric, hyphens, underscores, dots, slashes).
var safeBranchRe = regexp.MustCompile(`^[a-zA-Z0-9._/-]+$`)

// Config represents the structure of an efctl.yaml configuration file.
type Config struct {
	WithFrontend          *bool  `yaml:"with-frontend"`
	WithGraphql           *bool  `yaml:"with-graphql"`
	WorldContractsURL     string `yaml:"world-contracts-url"`
	WorldContractsRef     string `yaml:"world-contracts-ref"`
	WorldContractsBranch  string `yaml:"world-contracts-branch"` // Deprecated: use world-contracts-ref
	BuilderScaffoldURL    string `yaml:"builder-scaffold-url"`
	BuilderScaffoldRef    string `yaml:"builder-scaffold-ref"`
	BuilderScaffoldBranch string `yaml:"builder-scaffold-branch"` // Deprecated: use builder-scaffold-ref

	// Internal field to track if a config file was actually loaded
	configFileLoaded bool
}

// DefaultWorldContractsURL is the default git clone URL for world-contracts.
const DefaultWorldContractsURL = "https://github.com/evefrontier/world-contracts.git"

// DefaultBuilderScaffoldURL is the default git clone URL for builder-scaffold.
const DefaultBuilderScaffoldURL = "https://github.com/evefrontier/builder-scaffold.git"

// DefaultBranch is the default git branch to checkout.
const DefaultBranch = "main"

// DefaultConfigFile is the default configuration file name.
const DefaultConfigFile = "efctl.yaml"

// AlternateDefaultConfigFile is the alternate supported config file name.
const AlternateDefaultConfigFile = "efctl.yml"

// DefaultConfigFiles lists default config names in preference order.
var DefaultConfigFiles = []string{DefaultConfigFile, AlternateDefaultConfigFile}

// Loaded holds the currently loaded configuration (populated after Load).
var Loaded *Config

// FindDefaultConfigPath searches from startDir upward for efctl.yaml/efctl.yml.
// Returns the first match in DefaultConfigFiles order.
func FindDefaultConfigPath(startDir string) (string, bool, error) {
	if startDir == "" {
		startDir = "."
	}

	dir, err := filepath.Abs(filepath.Clean(startDir))
	if err != nil {
		return "", false, fmt.Errorf("failed to resolve start directory %s: %w", startDir, err)
	}

	for {
		for _, name := range DefaultConfigFiles {
			candidate := filepath.Join(dir, name)
			info, statErr := os.Stat(candidate)
			if statErr == nil {
				if info.Mode().IsRegular() {
					return candidate, true, nil
				}
				continue
			}
			if !os.IsNotExist(statErr) {
				return "", false, fmt.Errorf("failed to stat config file %s: %w", candidate, statErr)
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", false, nil
}

// Load reads and parses the config file at the given path.
// If the file does not exist and the path is the default, an empty config is returned without error.
func Load(path string) (*Config, error) {
	cfg := &Config{}
	cfg.configFileLoaded = false

	cleanPath := filepath.Clean(path)
	data, err := os.ReadFile(cleanPath) // #nosec G304 -- config file path is intentionally user-specified via CLI flag
	if err != nil {
		if os.IsNotExist(err) && path == DefaultConfigFile {
			// Default config file is optional
			Loaded = cfg
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	cfg.configFileLoaded = true

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation error in %s: %w", path, err)
	}

	Loaded = cfg
	return cfg, nil
}

// Validate checks that all configured values are safe and well-formed.
func (c *Config) Validate() error {
	// Validate git URLs — only https:// is allowed to prevent git:// / ssh:// / file:// protocol abuse
	for _, entry := range []struct {
		name, url string
	}{
		{"world-contracts-url", c.WorldContractsURL},
		{"builder-scaffold-url", c.BuilderScaffoldURL},
	} {
		if entry.url != "" {
			if !strings.HasPrefix(entry.url, "https://") {
				return fmt.Errorf("%s must use https:// scheme, got: %s", entry.name, entry.url)
			}
		}
	}

	// Validate ref names — prevent argument injection via git checkout
	for _, entry := range []struct {
		name, ref string
	}{
		{"world-contracts-ref", c.GetWorldContractsRef()},
		{"builder-scaffold-ref", c.GetBuilderScaffoldRef()},
	} {
		if entry.ref != "" {
			// Allow commit hashes (40-character hex)
			isCommit, _ := regexp.MatchString(`^[0-9a-fA-F]{40}$`, entry.ref)
			if !isCommit {
				if !safeBranchRe.MatchString(entry.ref) {
					return fmt.Errorf("%s contains invalid characters: %s (allowed: alphanumeric, hyphens, underscores, dots, slashes, or 40-char commit hash)", entry.name, entry.ref)
				}
			}
			// Reject refs starting with "-" to prevent argument injection
			if strings.HasPrefix(entry.ref, "-") {
				return fmt.Errorf("%s must not start with a hyphen: %s", entry.name, entry.ref)
			}
		}
	}

	return nil
}

// GetWorldContractsURL returns the configured world-contracts URL, falling back to default.
func (c *Config) GetWorldContractsURL() string {
	if c != nil && c.WorldContractsURL != "" {
		return c.WorldContractsURL
	}
	return DefaultWorldContractsURL
}

// GetBuilderScaffoldURL returns the configured builder-scaffold URL, falling back to default.
func (c *Config) GetBuilderScaffoldURL() string {
	if c != nil && c.BuilderScaffoldURL != "" {
		return c.BuilderScaffoldURL
	}
	return DefaultBuilderScaffoldURL
}

// GetWorldContractsRef returns the configured world-contracts ref, falling back to branch, then default.
func (c *Config) GetWorldContractsRef() string {
	if c != nil {
		if c.WorldContractsRef != "" {
			return c.WorldContractsRef
		}
		if c.WorldContractsBranch != "" {
			return c.WorldContractsBranch
		}
	}
	return DefaultBranch
}

// GetBuilderScaffoldRef returns the configured builder-scaffold ref, falling back to branch, then default.
func (c *Config) GetBuilderScaffoldRef() string {
	if c != nil {
		if c.BuilderScaffoldRef != "" {
			return c.BuilderScaffoldRef
		}
		if c.BuilderScaffoldBranch != "" {
			return c.BuilderScaffoldBranch
		}
	}
	return DefaultBranch
}

// WasLoaded returns true if a config file was successfully loaded (not just defaulted).
func (c *Config) WasLoaded() bool {
	if c == nil {
		return false
	}
	return c.configFileLoaded
}
