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
	WorldContractsBranch  string `yaml:"world-contracts-branch"`
	BuilderScaffoldURL    string `yaml:"builder-scaffold-url"`
	BuilderScaffoldBranch string `yaml:"builder-scaffold-branch"`
}

// DefaultWorldContractsURL is the default git clone URL for world-contracts.
const DefaultWorldContractsURL = "https://github.com/evefrontier/world-contracts.git"

// DefaultBuilderScaffoldURL is the default git clone URL for builder-scaffold.
const DefaultBuilderScaffoldURL = "https://github.com/evefrontier/builder-scaffold.git"

// DefaultBranch is the default git branch to checkout.
const DefaultBranch = "main"

// DefaultConfigFile is the default configuration file name.
const DefaultConfigFile = "efctl.yaml"

// Loaded holds the currently loaded configuration (populated after Load).
var Loaded *Config

// Load reads and parses the config file at the given path.
// If the file does not exist and the path is the default, an empty config is returned without error.
func Load(path string) (*Config, error) {
	cfg := &Config{}

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

	// Validate branch names — prevent argument injection via git checkout
	for _, entry := range []struct {
		name, branch string
	}{
		{"world-contracts-branch", c.WorldContractsBranch},
		{"builder-scaffold-branch", c.BuilderScaffoldBranch},
	} {
		if entry.branch != "" {
			if !safeBranchRe.MatchString(entry.branch) {
				return fmt.Errorf("%s contains invalid characters: %s (allowed: alphanumeric, hyphens, underscores, dots, slashes)", entry.name, entry.branch)
			}
			// Reject branches starting with "-" to prevent argument injection
			if strings.HasPrefix(entry.branch, "-") {
				return fmt.Errorf("%s must not start with a hyphen: %s", entry.name, entry.branch)
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

// GetWorldContractsBranch returns the configured world-contracts branch, falling back to default.
func (c *Config) GetWorldContractsBranch() string {
	if c != nil && c.WorldContractsBranch != "" {
		return c.WorldContractsBranch
	}
	return DefaultBranch
}

// GetBuilderScaffoldBranch returns the configured builder-scaffold branch, falling back to default.
func (c *Config) GetBuilderScaffoldBranch() string {
	if c != nil && c.BuilderScaffoldBranch != "" {
		return c.BuilderScaffoldBranch
	}
	return DefaultBranch
}
