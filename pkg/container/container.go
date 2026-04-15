package container

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"efctl/pkg/env"
	"efctl/pkg/ui"
)

// ── Container configuration ────────────────────────────────────────

// MountDef describes a volume or bind mount for a container.
type MountDef struct {
	Type    string // "bind" or "volume"
	Source  string // host path (bind) or volume name (volume)
	Target  string // container path
	SELinux bool   // append :z for SELinux relabelling
}

// HealthcheckDef describes a container healthcheck.
type HealthcheckDef struct {
	Test        []string
	Interval    time.Duration
	Timeout     time.Duration
	Retries     int
	StartPeriod time.Duration
}

// ContainerConfig holds everything needed to create & start a container.
type ContainerConfig struct {
	Name        string
	Image       string
	Ports       map[int]int // host → container
	Mounts      []MountDef
	Env         []string // KEY=VALUE
	NetworkName string
	Aliases     []string // DNS aliases in the network
	WorkingDir  string
	Cmd         []string
	Entrypoint  []string
	Tty         bool
	OpenStdin   bool
	Healthcheck *HealthcheckDef
	UsernsMode  string // e.g. "keep-id" for Podman
}

// ── Interface ──────────────────────────────────────────────────────

// ContainerClient defines the interface for container operations.
// All consumers should accept this interface to enable testing with mocks.
type ContainerClient interface {
	// Lifecycle primitives
	BuildImage(ctx context.Context, contextDir string, dockerfilePath string, tag string) error
	CreateNetwork(ctx context.Context, name string) error
	RemoveNetwork(ctx context.Context, name string) error
	CreateVolume(ctx context.Context, name string) error
	CreateContainer(ctx context.Context, cfg ContainerConfig) error
	StartContainer(ctx context.Context, name string) error
	StopContainer(ctx context.Context, name string) error
	RemoveContainer(ctx context.Context, name string) error
	WaitHealthy(ctx context.Context, name string, timeout time.Duration) error

	// Inspection / interaction (carried over from previous interface)
	GetEngine() string
	NetworkName() string
	ContainerRunning(name string) bool
	ContainerLogs(name string, tail int) string
	ContainerExitCode(name string) (int, error)
	WaitForLogs(ctx context.Context, containerName string, searchString string) error
	InteractiveShell(containerName string) error
	Exec(ctx context.Context, containerName string, command []string) error
	ExecCapture(ctx context.Context, containerName string, command []string) (string, error)
	RemoveImages(names []string)
	Cleanup() error
}

// ── Client ─────────────────────────────────────────────────────────

// Client wraps the Docker/Podman CLI and implements ContainerClient.
type Client struct {
	Engine      string // "docker" or "podman"
	host        string
	useFromEnv  bool
	network     string              // dynamic network name
	healthTests map[string][]string // container name → healthcheck Test (for exec fallback)
}

type clientConnectionCandidate struct {
	engine     string
	host       string
	useFromEnv bool
}

// Compile-time check that Client implements ContainerClient.
var _ ContainerClient = (*Client)(nil)

const (
	dockerAuthZPatchedVersion = "29.3.1"
	dockerAuthZAdvisoryPrefix = "GHSA"
	dockerAuthZAdvisorySuffix = "x744-4wpc-v9h2"
	dockerAuthZCVEID          = "CVE-2026-34040"
)

var dockerAuthZAdvisoryID = dockerAuthZAdvisoryPrefix + "-" + dockerAuthZAdvisorySuffix

// NewClient returns a new container client backed by the Docker/Podman CLI.
// It auto-detects the engine (Docker or Podman) and connects to the
// appropriate daemon context.
func NewClient() (*Client, error) {
	res := env.CheckPrerequisites()
	if !res.HasDocker && !res.HasPodman {
		return nil, fmt.Errorf("no container engine found")
	}

	candidates := connectionCandidates(res, runtime.GOOS, os.Getuid(), os.Getenv("DOCKER_HOST"), socketHostExists)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no reachable container daemon found: podman socket not found and docker host unavailable")
	}

	var errs []string
	for _, candidate := range candidates {
		probeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := probeCandidate(probeCtx, candidate)
		cancel()
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s probe failed: %v", describeCandidate(candidate), err))
			continue
		}

		ui.Debug.Println(fmt.Sprintf("NewClient: using engine=%s host=%s", candidate.engine, candidateDisplayHost(candidate)))
		return &Client{Engine: candidate.engine, host: candidate.host, useFromEnv: candidate.useFromEnv}, nil
	}

	return nil, fmt.Errorf("failed to connect to a reachable container daemon: %s", strings.Join(errs, "; "))
}

func preferredEngineOrder(res *env.CheckResult) []string {
	order := make([]string, 0, 2)
	add := func(engine string, available bool) {
		if !available {
			return
		}
		for _, existing := range order {
			if existing == engine {
				return
			}
		}
		order = append(order, engine)
	}

	switch os.Getenv("EFCTL_ENGINE") {
	case "docker":
		add("docker", res.HasDocker)
		add("podman", res.HasPodman)
	case "podman":
		add("podman", res.HasPodman)
		add("docker", res.HasDocker)
	default:
		add("podman", res.HasPodman)
		add("docker", res.HasDocker)
	}

	return order
}

func connectionCandidates(res *env.CheckResult, goos string, uid int, dockerHost string, hostExists func(string) bool) []clientConnectionCandidate {
	order := preferredEngineOrder(res)
	seen := make(map[string]struct{})
	candidates := make([]clientConnectionCandidate, 0, len(order))
	add := func(candidate clientConnectionCandidate) {
		key := fmt.Sprintf("%s|%t|%s", candidate.engine, candidate.useFromEnv, candidate.host)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		candidates = append(candidates, candidate)
	}

	for _, engine := range order {
		switch engine {
		case "podman":
			if looksLikePodmanHost(dockerHost) && hostExists(dockerHost) {
				add(clientConnectionCandidate{engine: "podman", host: dockerHost})
			}
			if goos == "linux" {
				for _, host := range podmanSocketHosts(uid) {
					if hostExists(host) {
						add(clientConnectionCandidate{engine: "podman", host: host})
					}
				}
			}
			add(clientConnectionCandidate{engine: "podman"})
		case "docker":
			add(clientConnectionCandidate{engine: "docker", useFromEnv: true, host: dockerHost})
		}
	}

	return candidates
}

func describeCandidate(candidate clientConnectionCandidate) string {
	return fmt.Sprintf("engine=%s host=%s", candidate.engine, candidateDisplayHost(candidate))
}

func candidateDisplayHost(candidate clientConnectionCandidate) string {
	if candidate.host != "" {
		return candidate.host
	}
	if candidate.useFromEnv {
		if host := os.Getenv("DOCKER_HOST"); host != "" {
			return host
		}
		return "from-env/default"
	}
	return "default"
}

type dockerInfoSummary struct {
	ServerVersion string `json:"ServerVersion"`
	Plugins       struct {
		Authorization []string `json:"Authorization"`
	} `json:"Plugins"`
}

type containerInspectResult struct {
	State      *containerState      `json:"State"`
	HostConfig *containerHostConfig `json:"HostConfig"`
}

type containerState struct {
	Running  bool             `json:"Running"`
	ExitCode int              `json:"ExitCode"`
	Health   *containerHealth `json:"Health"`
}

type containerHealth struct {
	Status string `json:"Status"`
}

type containerHostConfig struct {
	UsernsMode string `json:"UsernsMode"`
}

func probeCandidate(ctx context.Context, candidate clientConnectionCandidate) error {
	if candidate.engine == "docker" {
		info, err := dockerInfoForCandidate(ctx, candidate)
		if err != nil {
			return err
		}
		return dockerAuthZVulnerabilityError(info.ServerVersion, info.Plugins.Authorization)
	}

	output, err := candidateCommandOutput(ctx, candidate, "info")
	if err != nil {
		return fmt.Errorf("%w%s", err, trimmedCommandOutputSuffix(output))
	}
	if len(output) > 0 {
		ui.Debug.Println(fmt.Sprintf("%s info probe succeeded", candidate.engine))
	}
	return nil
}

func dockerInfoForCandidate(ctx context.Context, candidate clientConnectionCandidate) (dockerInfoSummary, error) {
	var info dockerInfoSummary
	output, err := candidateCommandOutput(ctx, candidate, "info", "--format", "{{json .}}")
	if err != nil {
		return info, fmt.Errorf("docker info failed: %w%s", err, trimmedCommandOutputSuffix(output))
	}
	if err := json.Unmarshal(output, &info); err != nil {
		return info, fmt.Errorf("decode docker info: %w", err)
	}
	return info, nil
}

func dockerAuthZVulnerabilityError(serverVersion string, authPlugins []string) error {
	if len(authPlugins) == 0 {
		return nil
	}

	isSafe, err := versionAtLeast(serverVersion, dockerAuthZPatchedVersion)
	if err != nil {
		return fmt.Errorf(
			"docker daemon uses authorization plugins (%s), but its version %q could not be validated against the minimum safe version %s for %s / %s; upgrade Docker Engine or disable AuthZ plugins that inspect request bodies",
			strings.Join(authPlugins, ", "),
			serverVersion,
			dockerAuthZPatchedVersion,
			dockerAuthZAdvisoryID,
			dockerAuthZCVEID,
		)
	}
	if isSafe {
		return nil
	}

	return fmt.Errorf(
		"docker daemon %q uses authorization plugins (%s) and is below the minimum safe version %s for %s / %s; upgrade Docker Engine, disable AuthZ plugins that inspect request bodies, or use Podman",
		serverVersion,
		strings.Join(authPlugins, ", "),
		dockerAuthZPatchedVersion,
		dockerAuthZAdvisoryID,
		dockerAuthZCVEID,
	)
}

func versionAtLeast(version string, minimum string) (bool, error) {
	actualParts, actualHasSuffix, err := parseComparableVersion(version)
	if err != nil {
		return false, err
	}
	minimumParts, _, err := parseComparableVersion(minimum)
	if err != nil {
		return false, err
	}

	for i := 0; i < 3; i++ {
		if actualParts[i] < minimumParts[i] {
			return false, nil
		}
		if actualParts[i] > minimumParts[i] {
			return true, nil
		}
	}

	if actualHasSuffix {
		return false, nil
	}

	return true, nil
}

func parseComparableVersion(raw string) ([3]int, bool, error) {
	var parts [3]int
	trimmed := strings.TrimSpace(strings.TrimPrefix(raw, "v"))
	if trimmed == "" {
		return parts, false, fmt.Errorf("empty version")
	}

	var builder strings.Builder
	hasSuffix := false
	for _, r := range trimmed {
		if (r >= '0' && r <= '9') || r == '.' {
			builder.WriteRune(r)
			continue
		}
		hasSuffix = true
		break
	}

	prefix := strings.Trim(builder.String(), ".")
	if prefix == "" {
		return parts, hasSuffix, fmt.Errorf("invalid version %q", raw)
	}

	segments := strings.Split(prefix, ".")
	if len(segments) > len(parts) {
		segments = segments[:len(parts)]
	}
	for i, segment := range segments {
		if segment == "" {
			return parts, hasSuffix, fmt.Errorf("invalid version %q", raw)
		}
		value, err := strconv.Atoi(segment)
		if err != nil {
			return parts, hasSuffix, fmt.Errorf("invalid version %q: %w", raw, err)
		}
		parts[i] = value
	}

	return parts, hasSuffix, nil
}

func candidateCommandOutput(ctx context.Context, candidate clientConnectionCandidate, args ...string) ([]byte, error) {
	cmd := commandForEngineContext(ctx, candidate.engine, candidate.host, candidate.useFromEnv, args...)
	return cmd.CombinedOutput()
}

func (c *Client) engineCommandOutput(ctx context.Context, args ...string) ([]byte, error) {
	if c == nil || c.Engine == "" {
		return nil, fmt.Errorf("container engine not configured")
	}
	cmd := commandForEngineContext(ctx, c.Engine, c.host, c.useFromEnv, args...)
	return cmd.CombinedOutput()
}

func commandForEngineContext(ctx context.Context, engine string, host string, useFromEnv bool, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, engine, args...) // #nosec G204 -- arguments are constructed programmatically without shell expansion
	envVars := os.Environ()
	if host != "" {
		envVars = withEnvVar(envVars, "DOCKER_HOST", host)
	} else if !useFromEnv {
		envVars = withoutEnvVar(envVars, "DOCKER_HOST")
	}
	cmd.Env = envVars
	return cmd
}

func withEnvVar(envVars []string, key string, value string) []string {
	prefix := key + "="
	filtered := make([]string, 0, len(envVars)+1)
	for _, envVar := range envVars {
		if strings.HasPrefix(envVar, prefix) {
			continue
		}
		filtered = append(filtered, envVar)
	}
	filtered = append(filtered, prefix+value)
	return filtered
}

func withoutEnvVar(envVars []string, key string) []string {
	prefix := key + "="
	filtered := make([]string, 0, len(envVars))
	for _, envVar := range envVars {
		if strings.HasPrefix(envVar, prefix) {
			continue
		}
		filtered = append(filtered, envVar)
	}
	return filtered
}

func trimmedCommandOutputSuffix(output []byte) string {
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return ""
	}
	if len(trimmed) > 500 {
		trimmed = trimmed[:500]
	}
	return ": " + trimmed
}

func (c *Client) inspectContainer(ctx context.Context, name string) (containerInspectResult, error) {
	var result containerInspectResult
	output, err := c.engineCommandOutput(ctx, "container", "inspect", name)
	if err != nil {
		return result, fmt.Errorf("inspect container %s: %w%s", name, err, trimmedCommandOutputSuffix(output))
	}
	var items []containerInspectResult
	if err := json.Unmarshal(output, &items); err != nil {
		return result, fmt.Errorf("decode container inspect for %s: %w", name, err)
	}
	if len(items) == 0 {
		return result, fmt.Errorf("container inspect for %s returned no results", name)
	}
	return items[0], nil
}

func isContainerNotFound(output []byte) bool {
	text := strings.ToLower(string(output))
	return strings.Contains(text, "no such container") || strings.Contains(text, "no container with name or id")
}

func isResourceMissing(output []byte) bool {
	text := strings.ToLower(string(output))
	return strings.Contains(text, "no such") || strings.Contains(text, "not found") || strings.Contains(text, "no container with name or id")
}

func looksLikePodmanHost(host string) bool {
	return strings.Contains(host, "podman.sock")
}

func podmanSocketHosts(uid int) []string {
	return []string{
		fmt.Sprintf("unix:///run/user/%d/podman/podman.sock", uid),
		"unix:///run/podman/podman.sock",
		"unix:///var/run/podman/podman.sock",
	}
}

func socketHostExists(host string) bool {
	if host == "" {
		return false
	}
	path := strings.TrimPrefix(host, "unix://")
	if path == host {
		return false
	}
	path = filepath.Clean(path)
	allowed := false
	for _, candidate := range podmanSocketHosts(os.Getuid()) {
		if filepath.Clean(strings.TrimPrefix(candidate, "unix://")) == path {
			allowed = true
			break
		}
	}
	if !allowed {
		return false
	}
	_, err := os.Stat(path) // #nosec G304 -- path is restricted to known Podman socket locations
	return err == nil
}

// NewClientWithNetwork creates a client and derives a deterministic network
// name from the workspace path.  Callers that need network isolation (env up)
// should use this constructor.
func NewClientWithNetwork(workspace string) (*Client, error) {
	c, err := NewClient()
	if err != nil {
		return nil, err
	}
	c.network = NetworkNameForWorkspace(workspace)
	return c, nil
}

// NetworkNameForWorkspace returns a deterministic network name for a workspace
// path.  Format: efctl-<first 8 hex chars of SHA-256>.
func NetworkNameForWorkspace(workspace string) string {
	abs, err := filepath.Abs(workspace)
	if err != nil {
		abs = workspace
	}
	h := sha256.Sum256([]byte(abs))
	return fmt.Sprintf("%s%x", NetworkPrefix, h[:4])
}

// GetEngine returns the underlying container engine ("docker" or "podman")
func (c *Client) GetEngine() string {
	return c.Engine
}

// NetworkName returns the dynamic network name for this client.
func (c *Client) NetworkName() string {
	return c.network
}

// ── Lifecycle primitives ───────────────────────────────────────────

// BuildImage builds an image from a Dockerfile in the given context directory.
func (c *Client) BuildImage(ctx context.Context, contextDir string, dockerfileName string, tag string) error {
	spinner, _ := ui.Spin(fmt.Sprintf("Building image %s...", tag))
	output, err := c.engineCommandOutput(ctx, "build", "--no-cache", "--rm", "-t", tag, "-f", dockerfileName, contextDir)
	if err != nil {
		spinner.Fail("Failed to build image")
		return fmt.Errorf("image build: %w%s", err, trimmedCommandOutputSuffix(output))
	}
	if len(output) > 0 {
		ui.Debug.Print(string(output))
	}

	spinner.Success(fmt.Sprintf("Image %s built successfully", tag))
	return nil
}

// CreateNetwork creates a bridge network with the given name.
// It is a no-op if the network already exists.
func (c *Client) CreateNetwork(ctx context.Context, name string) error {
	if _, err := c.engineCommandOutput(ctx, "network", "inspect", name); err == nil {
		return nil
	}

	output, err := c.engineCommandOutput(ctx, "network", "create", name)
	if err != nil {
		return fmt.Errorf("create network %s: %w%s", name, err, trimmedCommandOutputSuffix(output))
	}
	return nil
}

// RemoveNetwork removes a network by name, ignoring "not found" errors.
func (c *Client) RemoveNetwork(ctx context.Context, name string) error {
	output, err := c.engineCommandOutput(ctx, "network", "rm", name)
	if err != nil && !isResourceMissing(output) {
		return fmt.Errorf("remove network %s: %w%s", name, err, trimmedCommandOutputSuffix(output))
	}
	return nil
}

// CreateVolume creates a named volume.  It is a no-op if it already exists.
func (c *Client) CreateVolume(ctx context.Context, name string) error {
	output, err := c.engineCommandOutput(ctx, "volume", "create", name)
	if err != nil {
		return fmt.Errorf("create volume %s: %w%s", name, err, trimmedCommandOutputSuffix(output))
	}
	return nil
}

// CreateContainer creates (but does not start) a container from the given config.
// Remote images (containing "/" or ":") are automatically pulled if not present locally.
func (c *Client) CreateContainer(ctx context.Context, cfg ContainerConfig) error {
	c.logContainerConfig(cfg)
	if err := c.ensureContainerImage(ctx, cfg.Image); err != nil {
		return fmt.Errorf("pull image %s: %w", cfg.Image, err)
	}

	args := c.buildCreateContainerArgs(cfg)
	output, err := c.engineCommandOutput(ctx, args...)
	if err != nil {
		return fmt.Errorf("create container %s: %w%s", cfg.Name, err, trimmedCommandOutputSuffix(output))
	}

	c.storeHealthTest(cfg)
	return nil
}

func (c *Client) logContainerConfig(cfg ContainerConfig) {
	ui.Debug.Println(fmt.Sprintf("Creating container %s (image=%s)", cfg.Name, cfg.Image))
	for _, mount := range cfg.Mounts {
		ui.Debug.Println(fmt.Sprintf("  mount: type=%s src=%s → %s", mount.Type, mount.Source, mount.Target))
	}
	for _, envVar := range cfg.Env {
		ui.Debug.Println(fmt.Sprintf("  env: %s", envVar))
	}
}

func (c *Client) ensureContainerImage(ctx context.Context, image string) error {
	if !strings.Contains(image, "/") && !strings.Contains(image, ":") {
		return nil
	}
	return c.ensureImage(ctx, image)
}

func (c *Client) buildCreateContainerArgs(cfg ContainerConfig) []string {
	args := []string{"create", "--name", cfg.Name}
	args = append(args, c.preparePortConfig(cfg.Ports)...)
	args = append(args, c.prepareMountConfig(cfg.Mounts)...)
	args = append(args, c.prepareHealthConfig(cfg.Healthcheck)...)
	args = append(args, c.prepareNetworkConfig(cfg.NetworkName, cfg.Aliases)...)
	args = append(args, c.containerCreateOptionArgs(cfg)...)
	args = append(args, cfg.Image)
	args = append(args, c.containerCommandArgs(cfg)...)
	return args
}

func (c *Client) containerCreateOptionArgs(cfg ContainerConfig) []string {
	args := make([]string, 0, len(cfg.Env)*2+8)
	if cfg.UsernsMode != "" {
		args = append(args, "--userns", cfg.UsernsMode)
	}
	for _, envVar := range cfg.Env {
		args = append(args, "-e", envVar)
	}
	if cfg.WorkingDir != "" {
		args = append(args, "-w", cfg.WorkingDir)
	}
	if cfg.Tty {
		args = append(args, "-t")
	}
	if cfg.OpenStdin {
		args = append(args, "-i")
	}
	if len(cfg.Entrypoint) > 0 {
		args = append(args, "--entrypoint", cfg.Entrypoint[0])
	}
	return args
}

func (c *Client) containerCommandArgs(cfg ContainerConfig) []string {
	if len(cfg.Entrypoint) > 1 {
		command := make([]string, 0, len(cfg.Entrypoint)-1+len(cfg.Cmd))
		command = append(command, cfg.Entrypoint[1:]...)
		command = append(command, cfg.Cmd...)
		return command
	}
	return cfg.Cmd
}

func (c *Client) storeHealthTest(cfg ContainerConfig) {
	if cfg.Healthcheck == nil || len(cfg.Healthcheck.Test) == 0 {
		return
	}
	if c.healthTests == nil {
		c.healthTests = make(map[string][]string)
	}
	c.healthTests[cfg.Name] = cfg.Healthcheck.Test
}

func (c *Client) preparePortConfig(ports map[int]int) []string {
	if len(ports) == 0 {
		return nil
	}
	hostPorts := make([]int, 0, len(ports))
	for hostPort := range ports {
		hostPorts = append(hostPorts, hostPort)
	}
	sort.Ints(hostPorts)
	args := make([]string, 0, len(hostPorts)*2)
	for _, hostPort := range hostPorts {
		args = append(args, "-p", fmt.Sprintf("127.0.0.1:%d:%d/tcp", hostPort, ports[hostPort]))
	}
	return args
}

func (c *Client) prepareMountConfig(mountDefs []MountDef) []string {
	args := make([]string, 0, len(mountDefs)*2)
	for _, m := range mountDefs {
		switch m.Type {
		case "bind":
			spec := fmt.Sprintf("%s:%s", m.Source, m.Target)
			if m.SELinux && c.Engine == "podman" {
				spec += ":z"
			}
			args = append(args, "-v", spec)
		default: // volume
			args = append(args, "-v", fmt.Sprintf("%s:%s", m.Source, m.Target))
		}
	}
	return args
}

func (c *Client) prepareHealthConfig(h *HealthcheckDef) []string {
	if h == nil {
		return nil
	}
	args := make([]string, 0, 8)
	if healthCmd := healthCommand(h.Test); healthCmd != "" {
		args = append(args, "--health-cmd", healthCmd)
	}
	if h.Interval > 0 {
		args = append(args, "--health-interval", h.Interval.String())
	}
	if h.Timeout > 0 {
		args = append(args, "--health-timeout", h.Timeout.String())
	}
	if h.Retries > 0 {
		args = append(args, "--health-retries", strconv.Itoa(h.Retries))
	}
	if h.StartPeriod > 0 {
		args = append(args, "--health-start-period", h.StartPeriod.String())
	}
	return args
}

func healthCommand(test []string) string {
	if len(test) == 0 {
		return ""
	}
	switch test[0] {
	case "CMD-SHELL", "CMD":
		return strings.Join(test[1:], " ")
	default:
		return strings.Join(test, " ")
	}
}

func (c *Client) prepareNetworkConfig(networkName string, aliases []string) []string {
	if networkName != "" {
		args := []string{"--network", networkName}
		for _, alias := range aliases {
			args = append(args, "--network-alias", alias)
		}
		return args
	}
	return nil
}

// ensureImage pulls a remote image if it is not already present locally.
func (c *Client) ensureImage(ctx context.Context, ref string) error {
	if _, err := c.engineCommandOutput(ctx, "image", "inspect", ref); err == nil {
		return nil // already present
	}

	ui.Info.Println(fmt.Sprintf("Pulling image %s ...", ref))
	output, err := c.engineCommandOutput(ctx, "pull", ref)
	if err != nil {
		return fmt.Errorf("pull image %s: %w%s", ref, err, trimmedCommandOutputSuffix(output))
	}
	if len(output) > 0 {
		ui.Debug.Print(string(output))
	}
	return nil
}

const ProjectIssuesURL = "https://github.com/evefrontier/efctl/issues"

// StartContainer starts an existing container by name.
func (c *Client) StartContainer(ctx context.Context, name string) error {
	output, err := c.engineCommandOutput(ctx, "start", name)
	if err != nil {
		errStr := err.Error()
		if len(output) > 0 {
			errStr = string(output)
		}
		if strings.Contains(errStr, "netavark") && strings.Contains(errStr, "nftables") {
			return fmt.Errorf("start container %s: %w\n\nTIP: This error often occurs on WSL with Podman's default networking. Try setting 'firewall_driver = \"iptables\"' in your ~/.config/containers/containers.conf and running 'podman system reset' if the issue persists.\n\nPlease report this issue at %s - include the output of 'efctl doctor'.", name, err, ProjectIssuesURL)
		}
		return fmt.Errorf("start container %s: %w\n\nPlease report this issue at %s - include the output of 'efctl doctor'.", name, err, ProjectIssuesURL)
	}
	return nil
}

// StopContainer stops a running container by name (10s timeout).
func (c *Client) StopContainer(ctx context.Context, name string) error {
	output, err := c.engineCommandOutput(ctx, "stop", "-t", "10", name)
	if err != nil && !isContainerNotFound(output) {
		return fmt.Errorf("stop container %s: %w%s", name, err, trimmedCommandOutputSuffix(output))
	}
	return nil
}

// RemoveContainer removes a container by name, ignoring "not found" errors.
func (c *Client) RemoveContainer(ctx context.Context, name string) error {
	output, err := c.engineCommandOutput(ctx, "rm", "-f", name)
	if err != nil && !isContainerNotFound(output) {
		return fmt.Errorf("remove container %s: %w%s", name, err, trimmedCommandOutputSuffix(output))
	}
	return nil
}

// WaitHealthy polls a container's health status until it reports "healthy"
// or the timeout expires. If the engine's native healthcheck never succeeds
// (common with Podman), it falls back to executing the healthcheck command
// inside the container via exec.
func (c *Client) WaitHealthy(ctx context.Context, name string, timeout time.Duration) error {
	ui.Debug.Println(fmt.Sprintf("WaitHealthy: container=%s timeout=%s engine=%s", name, timeout, c.Engine))
	spinner, _ := ui.Spin(fmt.Sprintf("Waiting for %s to become healthy...", name))
	deadline := time.After(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// After a few ticks without a "healthy" status, switch to exec-based probing.
	unhealthyTicks := 0
	const execFallbackThreshold = 3

	for {
		select {
		case <-deadline:
			spinner.Fail(fmt.Sprintf("%s did not become healthy in %s", name, timeout))
			return fmt.Errorf("timeout waiting for %s healthcheck", name)
		case <-ctx.Done():
			spinner.Fail(fmt.Sprintf("%s healthcheck cancelled", name))
			return ctx.Err()
		case <-ticker.C:
			info, err := c.inspectContainer(ctx, name)
			if err != nil {
				continue
			}
			if info.State == nil {
				continue
			}
			// If the container exited, there is no point waiting.
			if !info.State.Running {
				spinner.Fail(fmt.Sprintf("%s exited before becoming healthy", name))
				return fmt.Errorf("container %s is not running (exit code %d)", name, info.State.ExitCode)
			}

			// Check native healthcheck status.
			healthStatus := "none"
			if info.State.Health != nil {
				healthStatus = info.State.Health.Status
			}
			ui.Debug.Println(fmt.Sprintf("WaitHealthy: %s tick=%d health=%s running=%v", name, unhealthyTicks, healthStatus, info.State.Running))

			if healthStatus == "healthy" {
				spinner.Success(fmt.Sprintf("%s is healthy", name))
				return nil
			}

			// Native healthcheck absent or stuck (e.g. Podman "starting") —
			// fall back to running the probe command via exec.
			unhealthyTicks++
			if unhealthyTicks >= execFallbackThreshold {
				ok := c.execHealthProbe(ctx, name)
				ui.Debug.Println(fmt.Sprintf("WaitHealthy: %s exec fallback result=%v", name, ok))
				if ok {
					spinner.Success(fmt.Sprintf("%s is healthy", name))
					return nil
				}
			}
		}
	}
}

// execHealthProbe runs the container's healthcheck command via exec.
// Returns true if the command exits 0.
func (c *Client) execHealthProbe(ctx context.Context, name string) bool {
	if c == nil || c.Engine == "" {
		return false
	}
	// Look up the healthcheck test stored during CreateContainer.
	test := c.healthTests[name]
	if len(test) == 0 {
		return false
	}

	var cmd []string
	switch test[0] {
	case "CMD-SHELL":
		cmd = []string{"sh", "-c", strings.Join(test[1:], " ")}
	case "CMD":
		cmd = test[1:]
	default:
		cmd = test
	}
	if len(cmd) == 0 {
		return false
	}
	output, err := c.engineCommandOutput(ctx, append([]string{"exec", name}, cmd...)...)
	if err != nil {
		if len(output) > 0 {
			ui.Debug.Println(fmt.Sprintf("Health check exec for %s failed: %s", name, strings.TrimSpace(string(output))))
		}
		return false
	}
	return true
}

// ── Inspection / interaction ────────────────────────────────────────

// ContainerRunning checks if a container is currently running.
func (c *Client) ContainerRunning(name string) bool {
	ctx := context.Background()
	info, err := c.inspectContainer(ctx, name)
	if err != nil {
		return false
	}
	return info.State != nil && info.State.Running
}

// ContainerLogs returns the last N lines of a container's logs.
func (c *Client) ContainerLogs(name string, tail int) string {
	ctx := context.Background()
	output, err := c.engineCommandOutput(ctx, "logs", "--tail", fmt.Sprintf("%d", tail), name)
	if err != nil {
		return fmt.Sprintf("(could not retrieve logs: %v%s)", err, trimmedCommandOutputSuffix(output))
	}
	return strings.TrimSpace(string(output))
}

// ContainerExitCode returns the exit code of a stopped container.
func (c *Client) ContainerExitCode(name string) (int, error) {
	ctx := context.Background()
	info, err := c.inspectContainer(ctx, name)
	if err != nil {
		return -1, fmt.Errorf("failed to inspect container %s: %w", name, err)
	}
	if info.State == nil {
		return -1, fmt.Errorf("no state for container %s", name)
	}
	return info.State.ExitCode, nil
}

// WaitForLogs waits for a specific string in the container logs
func (c *Client) WaitForLogs(ctx context.Context, containerName string, searchString string) error {
	spinner, _ := ui.Spin(fmt.Sprintf("Waiting for %s to initialize...", containerName))
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Timeout reached — check if container is still running and healthy as fallback.
			// This handles cases where the log string changed or was never printed but container is actually ready.
			if c.ContainerRunning(containerName) {
				ui.Debug.Println(fmt.Sprintf("WaitForLogs timeout reached but %s is still running - checking health", containerName))
				spinner.UpdateText(fmt.Sprintf("Log string not found, verifying %s health...", containerName))

				// Give the container a moment to stabilize
				time.Sleep(2 * time.Second)

				// Create a short timeout for the health check
				healthCtx, healthCancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer healthCancel()

				// Try exec-based health probe as fallback
				if c.execHealthProbe(healthCtx, containerName) {
					spinner.Success(fmt.Sprintf("%s is ready (health check passed)", containerName))
					ui.Warn.Println(fmt.Sprintf("Log string %q not found, but container is healthy", searchString))
					return nil
				}
			}

			spinner.Fail("Timed out waiting for logs")
			lastLogs := c.ContainerLogs(containerName, 50)
			return fmt.Errorf("timeout waiting for %q in %s logs.\n\nLast 50 lines of container logs:\n%s", searchString, containerName, lastLogs)
		case <-ticker.C:
			logs := c.ContainerLogs(containerName, 200)
			if strings.Contains(logs, searchString) {
				spinner.Success(fmt.Sprintf("%s is ready", containerName))
				return nil
			}
			if !c.ContainerRunning(containerName) {
				// Container exited before the ready string appeared — provide diagnostics
				exitCode, exitErr := c.ContainerExitCode(containerName)
				lastLogs := c.ContainerLogs(containerName, 50)
				diag := fmt.Sprintf("Container %s exited before becoming ready.", containerName)
				if exitErr == nil {
					diag += fmt.Sprintf(" Exit code: %d.", exitCode)
				}
				diag += fmt.Sprintf("\n\nLast 50 lines of container logs:\n%s", lastLogs)
				spinner.Fail(fmt.Sprintf("%s exited unexpectedly (search string %q not found)", containerName, searchString))
				return fmt.Errorf("%s", diag)
			}
		}
	}
}

// InteractiveShell opens an interactive shell in the container.
// This uses the CLI directly because the SDK exec-attach-hijack flow
// is non-trivial for raw TTY handling, and the CLI handles it perfectly.
func (c *Client) InteractiveShell(containerName string) error {
	cmd := exec.Command(c.Engine, "exec", "-it", containerName, "/bin/bash") // #nosec G204
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("interactive shell error: %w", err)
	}

	return nil
}

// Exec runs a command inside a container
func (c *Client) Exec(ctx context.Context, containerName string, command []string) error {
	spinner, _ := ui.Spin(fmt.Sprintf("Executing in %s...", containerName))

	// We use the CLI for exec because it handles TTY allocation and stream
	// multiplexing transparently.
	args := make([]string, 0, 2+len(command))
	args = append(args, "exec", containerName)
	args = append(args, command...)
	cmd := exec.CommandContext(ctx, c.Engine, args...) // #nosec G204

	output, err := cmd.CombinedOutput()

	// Print output if any, regardless of success/fail
	if len(output) > 0 {
		fmt.Printf("\n%s", string(output))
	}

	if err != nil {
		spinner.Fail("Execution failed")

		// Diagnostic: list containers on failure to catch "no such container" issues
		debugCmd := exec.Command(c.Engine, "ps", "-a") // #nosec G204
		debugOut, _ := debugCmd.CombinedOutput()
		ui.Warn.Println("Exec failed, current containers:")
		fmt.Println(string(debugOut))

		return fmt.Errorf("exec error: %w\n%s", err, string(output))
	}

	spinner.Success("Execution complete")
	return nil
}

// ExecCapture runs a command inside a container and returns the combined output.
func (c *Client) ExecCapture(ctx context.Context, containerName string, command []string) (string, error) {
	args := make([]string, 0, 2+len(command))
	args = append(args, "exec", containerName)
	args = append(args, command...)
	cmd := exec.CommandContext(ctx, c.Engine, args...) // #nosec G204

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("exec error: %w\n%s", err, string(output))
	}

	return string(output), nil
}

// ── Cleanup ────────────────────────────────────────────────────────

// Cleanup stops/removes all efctl containers, images, networks, and volumes.
// It also cleans up legacy compose-generated resources from older efctl versions.
func (c *Client) Cleanup() error {
	ctx := context.Background()

	spinner, _ := ui.Spin("Stopping and removing sui-playground container...")
	// Before removing the container, try to normalize permissions on bind-mounted volumes
	// so that the host user can clean up files created by root inside the container.
	c.normalizeBindMountPermissions(ContainerSuiPlayground)
	c.forceRemoveContainers(ctx, []string{ContainerSuiPlayground})
	spinner.Success(fmt.Sprintf("Container %s removal attempted", ContainerSuiPlayground))

	spinnerPg, _ := ui.Spin("Stopping and removing postgres container...")
	c.forceRemoveContainers(ctx, []string{ContainerPostgres, ContainerPostgresOld, ContainerPostgresOld2})
	spinnerPg.Success("Postgres container removal attempted")

	spinnerFe, _ := ui.Spin("Stopping and removing frontend container...")
	c.forceRemoveContainers(ctx, []string{ContainerFrontend, ContainerFrontendOld, ContainerFrontendOld2})
	spinnerFe.Success("Frontend container removal attempted")

	spinner2, _ := ui.Spin("Removing sui-dev images...")
	c.RemoveImages([]string{ImageSuiDev, ImageSuiDevOld, ImageSuiDevOld2})
	spinner2.Success("Images removal attempted")

	spinner3, _ := ui.Spin("Removing config and data volumes...")
	c.removeVolumes(ctx, []string{
		VolumeSuiConfig, VolumeSuiConfigOld, VolumeSuiConfigOld2,
		VolumePgData, VolumePgDataOld, VolumePgDataOld2,
		VolumeFrontendMods, VolumeFrontendModsOld, VolumeFrontendModsOld2,
	})
	spinner3.Success("Volumes removal attempted")

	// Remove any efctl networks
	if c.network != "" {
		spinnerNet, _ := ui.Spin("Removing network...")
		_ = c.RemoveNetwork(ctx, c.network)
		spinnerNet.Success("Network removal attempted")
	}

	return nil
}

func (c *Client) forceRemoveContainers(ctx context.Context, names []string) {
	for _, name := range names {
		ui.Debug.Println(fmt.Sprintf("forceRemoveContainers: stopping and removing %s", name))
		_, _ = c.engineCommandOutput(ctx, "stop", name)
		_, _ = c.engineCommandOutput(ctx, "rm", "-f", name)
	}
}

// RemoveImages removes container images by name, ignoring errors for
// images that do not exist.  This is called before BuildImage to
// ensure Podman does not reuse a stale cached image.
func (c *Client) RemoveImages(names []string) {
	ctx := context.Background()
	for _, name := range names {
		_, _ = c.engineCommandOutput(ctx, "rmi", "-f", name)
	}
}

func (c *Client) removeVolumes(ctx context.Context, names []string) {
	for _, vol := range names {
		_, _ = c.engineCommandOutput(ctx, "volume", "rm", "-f", vol)
	}
}

func (c *Client) normalizeBindMountPermissions(containerName string) {
	ctx := context.Background()
	info, err := c.inspectContainer(ctx, containerName)
	if err != nil {
		ui.Debug.Println(fmt.Sprintf("Permission normalization: failed to inspect %s: %v", containerName, err))
		return
	}

	if info.State == nil || !info.State.Running {
		ui.Debug.Println(fmt.Sprintf("Container %s not running, skipping permission normalization", containerName))
		return
	}

	// Get host UID/GID
	uid := os.Getuid()
	gid := os.Getgid()

	// Default: chown to the host user's IDs.
	// This works for Docker and Podman with keep-id.
	targetUID := uid
	targetGID := gid

	usernsMode := ""
	if info.HostConfig != nil {
		usernsMode = string(info.HostConfig.UsernsMode)
	}

	if c.Engine == "podman" && usernsMode != "keep-id" {
		// In Podman without keep-id, the host user maps to internal root (0).
		// So we chown to 0 inside to make it owned by the host user outside.
		ui.Debug.Println("Podman detected without keep-id: using internal root (0:0) for normalization")
		targetUID = 0
		targetGID = 0
	} else {
		ui.Debug.Println(fmt.Sprintf("Using host UID/GID (%d:%d) for normalization", targetUID, targetGID))
	}

	// Best-effort permission fix for common bind-mounted directories
	cmdStr := fmt.Sprintf(
		"chown -R %d:%d /workspace/world-contracts /workspace/builder-scaffold 2>/dev/null || true; "+
			"chmod -R u+rwX /workspace/world-contracts /workspace/builder-scaffold 2>/dev/null || true",
		targetUID, targetGID,
	)

	_, err = c.ExecCapture(ctx, containerName, []string{"/bin/bash", "-c", cmdStr})
	if err != nil {
		ui.Debug.Println(fmt.Sprintf("Permission normalization failed (non-critical): %v", err))
	} else {
		ui.Debug.Println(fmt.Sprintf("Normalized bind mount permissions for %s", containerName))
	}
}
