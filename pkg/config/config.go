package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// safeBranchRe matches valid git branch names (alphanumeric, hyphens, underscores, dots, slashes).
var safeBranchRe = regexp.MustCompile(`^[a-zA-Z0-9._/-]+$`)
var safeMountIdentifierRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
var safeHostnameRe = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)

// AdditionalBindMount represents a user-configured host directory that should be
// bind-mounted into the container environment.
type AdditionalBindMount struct {
	HostPath   string `yaml:"hostPath"`
	Identifier string `yaml:"identifier"`
}

// ResolvedAdditionalBindMount represents an additional bind mount after its host
// path has been resolved to an absolute directory.
type ResolvedAdditionalBindMount struct {
	HostPath   string
	Identifier string
}

// Config represents the structure of an efctl.yaml configuration file.
type Config struct {
	WithFrontend          *bool                 `yaml:"with-frontend"`
	WithGraphql           *bool                 `yaml:"with-graphql"`
	WorldContractsURL     string                `yaml:"world-contracts-url"`
	WorldContractsRef     string                `yaml:"world-contracts-ref"`
	WorldContractsBranch  string                `yaml:"world-contracts-branch"` // Deprecated: use world-contracts-ref
	BuilderScaffoldURL    string                `yaml:"builder-scaffold-url"`
	BuilderScaffoldRef    string                `yaml:"builder-scaffold-ref"`
	BuilderScaffoldBranch string                `yaml:"builder-scaffold-branch"` // Deprecated: use builder-scaffold-ref
	GitAutoCRLF           *bool                 `yaml:"git-autocrlf"`
	ContainerEngine       string                `yaml:"container-engine"`
	AdditionalBindMounts  []AdditionalBindMount `yaml:"additional-bind-mounts"`
	Host                  string                `yaml:"host"`
	ExposePostgres        bool                  `yaml:"expose-postgres"`

	// Internal field to track if a config file was actually loaded
	configFileLoaded bool
	configDir        string
}

// DefaultWorldContractsURL is the default git clone URL for world-contracts.
const DefaultWorldContractsURL = "https://github.com/evefrontier/world-contracts.git"

// DefaultBuilderScaffoldURL is the default git clone URL for builder-scaffold.
const DefaultBuilderScaffoldURL = "https://github.com/evefrontier/builder-scaffold.git"

// DefaultBranch is the canonical upstream branch name when branch semantics are needed.
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
	if absPath, absErr := filepath.Abs(cleanPath); absErr == nil {
		cfg.configDir = filepath.Dir(absPath)
	} else {
		cfg.configDir = filepath.Dir(cleanPath)
	}
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
	for _, validate := range []func(*Config) error{
		validateGitURLs,
		validateGitRefs,
		validateConfiguredHost,
		validateAdditionalBindMounts,
	} {
		if err := validate(c); err != nil {
			return err
		}
	}
	return nil
}

func validateGitURLs(c *Config) error {
	// Validate git URLs — only https:// is allowed to prevent git:// / ssh:// / file:// protocol abuse
	for _, entry := range []struct {
		name, url string
	}{
		{"world-contracts-url", c.WorldContractsURL},
		{"builder-scaffold-url", c.BuilderScaffoldURL},
	} {
		if entry.url != "" && !strings.HasPrefix(entry.url, "https://") {
			return fmt.Errorf("%s must use https:// scheme, got: %s", entry.name, entry.url)
		}
	}
	return nil
}

func validateGitRefs(c *Config) error {
	// Validate ref names — prevent argument injection via git checkout
	for _, entry := range []struct {
		name, ref string
	}{
		{"world-contracts-ref", c.GetWorldContractsRef()},
		{"builder-scaffold-ref", c.GetBuilderScaffoldRef()},
	} {
		if err := validateGitRef(entry.name, entry.ref); err != nil {
			return err
		}
	}
	return nil
}

func validateGitRef(name, ref string) error {
	if ref == "" {
		return nil
	}
	isCommit, _ := regexp.MatchString(`^[0-9a-fA-F]{40}$`, ref)
	if !isCommit && !safeBranchRe.MatchString(ref) {
		return fmt.Errorf("%s contains invalid characters: %s (allowed: alphanumeric, hyphens, underscores, dots, slashes, or 40-char commit hash)", name, ref)
	}
	if strings.HasPrefix(ref, "-") {
		return fmt.Errorf("%s must not start with a hyphen: %s", name, ref)
	}
	return nil
}

func validateConfiguredHost(c *Config) error {
	return validateHostValue("host", c.Host, c.Host != "")
}

func validateAdditionalBindMounts(c *Config) error {
	seenIdentifiers := make(map[string]struct{}, len(c.AdditionalBindMounts))
	for index, mount := range c.AdditionalBindMounts {
		if err := validateAdditionalBindMount(index, mount, seenIdentifiers); err != nil {
			return err
		}
	}
	return nil
}

func validateAdditionalBindMount(index int, mount AdditionalBindMount, seenIdentifiers map[string]struct{}) error {
	hostPath := strings.TrimSpace(mount.HostPath)
	identifier := strings.TrimSpace(mount.Identifier)

	if hostPath == "" {
		return fmt.Errorf("additional-bind-mounts[%d].hostPath must not be empty", index)
	}
	if strings.ContainsRune(hostPath, 0) {
		return fmt.Errorf("additional-bind-mounts[%d].hostPath contains null bytes", index)
	}
	if identifier == "" {
		return fmt.Errorf("additional-bind-mounts[%d].identifier must not be empty", index)
	}
	if !safeMountIdentifierRe.MatchString(identifier) {
		return fmt.Errorf("additional-bind-mounts[%d].identifier contains invalid characters: %s", index, identifier)
	}
	if _, exists := seenIdentifiers[identifier]; exists {
		return fmt.Errorf("additional-bind-mounts[%d].identifier duplicates %q", index, identifier)
	}
	seenIdentifiers[identifier] = struct{}{}
	return nil
}

func validateHostValue(name string, value string, explicitlyConfigured bool) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		if explicitlyConfigured {
			return fmt.Errorf("%s must not be empty", name)
		}
		return nil
	}
	if trimmed != value {
		value = trimmed
	}
	if strings.EqualFold(value, "localhost") {
		return nil
	}
	if strings.Contains(value, ":") {
		return fmt.Errorf("%s does not support IPv6 host values yet: %s", name, value)
	}
	if ip := net.ParseIP(value); ip != nil {
		if ip.To4() == nil {
			return fmt.Errorf("%s does not support IPv6 host values yet: %s", name, value)
		}
		return nil
	}
	if len(value) > 253 || !safeHostnameRe.MatchString(value) {
		return fmt.Errorf("%s must be localhost, a valid IPv4 address, or a valid hostname: %s", name, value)
	}
	return nil
}

// ResolveAdditionalBindMounts resolves configured mount paths into absolute host
// directories. Relative paths are resolved against the loaded config directory,
// or fallbackBaseDir when the config was constructed in-memory.
func (c *Config) ResolveAdditionalBindMounts(fallbackBaseDir string) ([]ResolvedAdditionalBindMount, error) {
	if c == nil || len(c.AdditionalBindMounts) == 0 {
		return nil, nil
	}

	baseDir := strings.TrimSpace(c.configDir)
	if baseDir == "" {
		baseDir = strings.TrimSpace(fallbackBaseDir)
	}
	if baseDir != "" {
		absBaseDir, err := filepath.Abs(baseDir)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve base directory for additional bind mounts: %w", err)
		}
		baseDir = absBaseDir
	}

	resolved := make([]ResolvedAdditionalBindMount, 0, len(c.AdditionalBindMounts))
	for index, mount := range c.AdditionalBindMounts {
		cleanHostPath := filepath.Clean(strings.TrimSpace(mount.HostPath))
		if !filepath.IsAbs(cleanHostPath) {
			if baseDir != "" {
				cleanHostPath = filepath.Join(baseDir, cleanHostPath)
			}
		}

		absHostPath, err := filepath.Abs(cleanHostPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve additional-bind-mounts[%d].hostPath %q: %w", index, mount.HostPath, err)
		}

		info, err := os.Stat(absHostPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("additional-bind-mounts[%d].hostPath does not exist: %s", index, absHostPath)
			}
			return nil, fmt.Errorf("failed to stat additional-bind-mounts[%d].hostPath %q: %w", index, absHostPath, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("additional-bind-mounts[%d].hostPath must point to a directory: %s", index, absHostPath)
		}

		resolved = append(resolved, ResolvedAdditionalBindMount{
			HostPath:   absHostPath,
			Identifier: mount.Identifier,
		})
	}

	return resolved, nil
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

// GetWorldContractsRef returns the configured world-contracts ref, falling back to the
// deprecated branch field, then the recommended compatible default.
func (c *Config) GetWorldContractsRef() string {
	if c != nil {
		if c.WorldContractsRef != "" {
			return c.WorldContractsRef
		}
		if c.WorldContractsBranch != "" {
			return c.WorldContractsBranch
		}
	}
	return RecommendedWorldContractsRef
}

// GetBuilderScaffoldRef returns the configured builder-scaffold ref, falling back to the
// deprecated branch field, then the recommended compatible default.
func (c *Config) GetBuilderScaffoldRef() string {
	if c != nil {
		if c.BuilderScaffoldRef != "" {
			return c.BuilderScaffoldRef
		}
		if c.BuilderScaffoldBranch != "" {
			return c.BuilderScaffoldBranch
		}
	}
	return RecommendedBuilderScaffoldRef
}

// GetGitAutoCRLF returns the configured git-autocrlf option, falling back to false.
func (c *Config) GetGitAutoCRLF() bool {
	if c != nil && c.GitAutoCRLF != nil {
		return *c.GitAutoCRLF
	}
	return false
}

// GetContainerEngine returns the configured container-engine option, falling back to auto-detect.
func (c *Config) GetContainerEngine() string {
	if c != nil && c.ContainerEngine != "" {
		return c.ContainerEngine
	}
	return "auto-detect"
}

// GetHost returns the configured bind address for service container ports, defaulting to 127.0.0.1.
func (c *Config) GetHost() string {
	if c != nil {
		if host := strings.TrimSpace(c.Host); host != "" {
			return host
		}
	}
	return "127.0.0.1"
}

// GetPostgresHost returns the PostgreSQL bind address. PostgreSQL stays local-only
// unless explicitly exposed, in which case it uses the validated service host.
func (c *Config) GetPostgresHost() string {
	if c != nil && c.ExposePostgres {
		return c.GetHost()
	}
	return "127.0.0.1"
}

// WasLoaded returns true if a config file was successfully loaded (not just defaulted).
func (c *Config) WasLoaded() bool {
	if c == nil {
		return false
	}
	return c.configFileLoaded
}
