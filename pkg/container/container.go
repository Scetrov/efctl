package container

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	dockercontainer "github.com/docker/docker/api/types/container"
	dockerfilters "github.com/docker/docker/api/types/filters"
	dockerimage "github.com/docker/docker/api/types/image"
	dockermount "github.com/docker/docker/api/types/mount"
	dockernetwork "github.com/docker/docker/api/types/network"
	dockervolume "github.com/docker/docker/api/types/volume"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/moby/go-archive"

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

// Client wraps the Docker/Podman SDK and implements ContainerClient.
type Client struct {
	Engine      string // "docker" or "podman"
	docker      *dockerclient.Client
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
	dockerAuthZAdvisoryID     = "GHSA-x744-4wpc-v9h2"
	dockerAuthZCVEID          = "CVE-2026-34040"
)

// NewClient returns a new container client backed by the Docker SDK.
// It auto-detects the engine (Docker or Podman) and connects to the
// appropriate socket.
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
		dc, err := dockerclient.NewClientWithOpts(dockerClientOpts(candidate)...)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s client init failed: %v", describeCandidate(candidate), err))
			continue
		}

		pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, err = dc.Ping(pingCtx)
		cancel()
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s ping failed: %v", describeCandidate(candidate), err))
			_ = dc.Close()
			continue
		}

		securityCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err = dockerDaemonSecurityError(securityCtx, candidate.engine, dc)
		cancel()
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s security check failed: %v", describeCandidate(candidate), err))
			_ = dc.Close()
			continue
		}

		ui.Debug.Println(fmt.Sprintf("NewClient: using engine=%s host=%s", candidate.engine, candidateDisplayHost(candidate)))
		return &Client{Engine: candidate.engine, docker: dc}, nil
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
			if goos != "linux" {
				add(clientConnectionCandidate{engine: "podman", useFromEnv: true, host: dockerHost})
				continue
			}

			if looksLikePodmanHost(dockerHost) && hostExists(dockerHost) {
				add(clientConnectionCandidate{engine: "podman", host: dockerHost})
			}
			for _, host := range podmanSocketHosts(uid) {
				if hostExists(host) {
					add(clientConnectionCandidate{engine: "podman", host: host})
				}
			}
		case "docker":
			add(clientConnectionCandidate{engine: "docker", useFromEnv: true, host: dockerHost})
		}
	}

	return candidates
}

func dockerClientOpts(candidate clientConnectionCandidate) []dockerclient.Opt {
	opts := []dockerclient.Opt{dockerclient.WithAPIVersionNegotiation()}
	if candidate.useFromEnv {
		opts = append(opts, dockerclient.FromEnv)
	}
	if candidate.host != "" {
		opts = append(opts, dockerclient.WithHost(candidate.host))
	}
	return opts
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

func dockerDaemonSecurityError(ctx context.Context, engine string, dc *dockerclient.Client) error {
	if engine != "docker" || dc == nil {
		return nil
	}

	info, err := dc.Info(ctx)
	if err != nil {
		ui.Debug.Println(fmt.Sprintf("Docker security check skipped: failed to query daemon info: %v", err))
		return nil
	}

	return dockerAuthZVulnerabilityError(info.ServerVersion, info.Plugins.Authorization)
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

// BuildImage builds a Docker image from a Dockerfile in the given context
// directory.
func (c *Client) BuildImage(ctx context.Context, contextDir string, dockerfileName string, tag string) error {
	spinner, _ := ui.Spin(fmt.Sprintf("Building image %s...", tag))

	tarReader, err := archive.TarWithOptions(contextDir, &archive.TarOptions{})
	if err != nil {
		spinner.Fail("Failed to create build context")
		return fmt.Errorf("tar build context: %w", err)
	}

	resp, err := c.docker.ImageBuild(ctx, tarReader, dockertypes.ImageBuildOptions{
		Tags:       []string{tag},
		Dockerfile: dockerfileName,
		NoCache:    true,
		Remove:     true,
	})
	if err != nil {
		spinner.Fail("Failed to build image")
		return fmt.Errorf("image build: %w", err)
	}
	defer resp.Body.Close()

	// Drain build output to completion — detect errors in the stream.
	// We use a JSON decoder to parse the stream and detect failures.
	decoder := json.NewDecoder(resp.Body)
	for {
		var msg struct {
			Stream string `json:"stream"`
			Error  string `json:"error"`
		}
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			spinner.Fail("Failed to parse build stream")
			return fmt.Errorf("decode build stream: %w", err)
		}
		if msg.Error != "" {
			spinner.Fail("Build failed")
			return fmt.Errorf("image build error: %s", msg.Error)
		}
		if msg.Stream != "" {
			ui.Debug.Print(msg.Stream)
		}
	}

	spinner.Success(fmt.Sprintf("Image %s built successfully", tag))
	return nil
}

// CreateNetwork creates a bridge network with the given name.
// It is a no-op if the network already exists.
func (c *Client) CreateNetwork(ctx context.Context, name string) error {
	// Check if it already exists.
	list, err := c.docker.NetworkList(ctx, dockernetwork.ListOptions{
		Filters: dockerfilters.NewArgs(dockerfilters.Arg("name", name)),
	})
	if err == nil {
		for _, n := range list {
			if n.Name == name {
				return nil
			}
		}
	}

	_, err = c.docker.NetworkCreate(ctx, name, dockernetwork.CreateOptions{
		Driver: "bridge",
	})
	if err != nil {
		return fmt.Errorf("create network %s: %w", name, err)
	}
	return nil
}

// RemoveNetwork removes a network by name, ignoring "not found" errors.
func (c *Client) RemoveNetwork(ctx context.Context, name string) error {
	if err := c.docker.NetworkRemove(ctx, name); err != nil && !dockerclient.IsErrNotFound(err) {
		return fmt.Errorf("remove network %s: %w", name, err)
	}
	return nil
}

// CreateVolume creates a named volume.  It is a no-op if it already exists.
func (c *Client) CreateVolume(ctx context.Context, name string) error {
	_, err := c.docker.VolumeCreate(ctx, dockervolume.CreateOptions{Name: name})
	if err != nil {
		return fmt.Errorf("create volume %s: %w", name, err)
	}
	return nil
}

// CreateContainer creates (but does not start) a container from the given config.
// Remote images (containing "/" or ":") are automatically pulled if not present locally.
func (c *Client) CreateContainer(ctx context.Context, cfg ContainerConfig) error {
	ui.Debug.Println(fmt.Sprintf("Creating container %s (image=%s)", cfg.Name, cfg.Image))
	for _, m := range cfg.Mounts {
		ui.Debug.Println(fmt.Sprintf("  mount: type=%s src=%s → %s", m.Type, m.Source, m.Target))
	}
	for _, e := range cfg.Env {
		ui.Debug.Println(fmt.Sprintf("  env: %s", e))
	}

	// --- pull remote images if needed ---
	if strings.Contains(cfg.Image, "/") || strings.Contains(cfg.Image, ":") {
		if err := c.ensureImage(ctx, cfg.Image); err != nil {
			return fmt.Errorf("pull image %s: %w", cfg.Image, err)
		}
	}

	exposedPorts, portBindings := c.preparePortConfig(cfg.Ports)
	mounts := c.prepareMountConfig(cfg.Mounts)
	hc := c.prepareHealthConfig(cfg.Healthcheck)
	networkConfig, hostNetworkMode := c.prepareNetworkConfig(cfg.NetworkName, cfg.Aliases)

	// --- userns mode (Podman keep-id) ---
	usernsMode := dockercontainer.UsernsMode("")
	if cfg.UsernsMode != "" {
		usernsMode = dockercontainer.UsernsMode(cfg.UsernsMode)
	}

	containerCfg := &dockercontainer.Config{
		Image:        cfg.Image,
		Env:          cfg.Env,
		Cmd:          cfg.Cmd,
		Entrypoint:   cfg.Entrypoint,
		WorkingDir:   cfg.WorkingDir,
		Tty:          cfg.Tty,
		OpenStdin:    cfg.OpenStdin,
		ExposedPorts: exposedPorts,
		Healthcheck:  hc,
	}

	hostCfg := &dockercontainer.HostConfig{
		PortBindings: portBindings,
		Mounts:       mounts,
		UsernsMode:   usernsMode,
		NetworkMode:  hostNetworkMode,
		AutoRemove:   false,
	}

	_, err := c.docker.ContainerCreate(ctx, containerCfg, hostCfg, networkConfig, nil, cfg.Name)
	if err != nil {
		return fmt.Errorf("create container %s: %w", cfg.Name, err)
	}

	// Store healthcheck test for the exec-based fallback probe (Podman compat).
	if cfg.Healthcheck != nil && len(cfg.Healthcheck.Test) > 0 {
		if c.healthTests == nil {
			c.healthTests = make(map[string][]string)
		}
		c.healthTests[cfg.Name] = cfg.Healthcheck.Test
	}

	return nil
}

func (c *Client) preparePortConfig(ports map[int]int) (nat.PortSet, nat.PortMap) {
	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}
	for host, ctr := range ports {
		cp := nat.Port(fmt.Sprintf("%d/tcp", ctr))
		exposedPorts[cp] = struct{}{}
		portBindings[cp] = []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: fmt.Sprintf("%d", host)}}
	}
	return exposedPorts, portBindings
}

func (c *Client) prepareMountConfig(mountDefs []MountDef) []dockermount.Mount {
	mounts := make([]dockermount.Mount, 0, len(mountDefs))
	for _, m := range mountDefs {
		mt := dockermount.Mount{
			Target: m.Target,
		}
		switch m.Type {
		case "bind":
			mt.Type = dockermount.TypeBind
			mt.Source = m.Source
			if m.SELinux && c.Engine == "podman" {
				mt.BindOptions = &dockermount.BindOptions{
					Propagation: dockermount.PropagationShared,
				}
			}
		default: // volume
			mt.Type = dockermount.TypeVolume
			mt.Source = m.Source
		}
		mounts = append(mounts, mt)
	}
	return mounts
}

func (c *Client) prepareHealthConfig(h *HealthcheckDef) *dockercontainer.HealthConfig {
	if h == nil {
		return nil
	}
	return &dockercontainer.HealthConfig{
		Test:        h.Test,
		Interval:    h.Interval,
		Timeout:     h.Timeout,
		Retries:     h.Retries,
		StartPeriod: h.StartPeriod,
	}
}

func (c *Client) prepareNetworkConfig(networkName string, aliases []string) (*dockernetwork.NetworkingConfig, dockercontainer.NetworkMode) {
	var networkConfig *dockernetwork.NetworkingConfig
	var hostNetworkMode dockercontainer.NetworkMode

	if networkName != "" {
		hostNetworkMode = dockercontainer.NetworkMode(networkName)
		endpointSettings := &dockernetwork.EndpointSettings{}
		if len(aliases) > 0 {
			endpointSettings.Aliases = aliases
		}
		networkConfig = &dockernetwork.NetworkingConfig{
			EndpointsConfig: map[string]*dockernetwork.EndpointSettings{
				networkName: endpointSettings,
			},
		}
	}
	return networkConfig, hostNetworkMode
}

// ensureImage pulls a remote image if it is not already present locally.
func (c *Client) ensureImage(ctx context.Context, ref string) error {
	// Check if the image exists locally first.
	_, _, err := c.docker.ImageInspectWithRaw(ctx, ref)
	if err == nil {
		return nil // already present
	}

	ui.Info.Println(fmt.Sprintf("Pulling image %s ...", ref))
	reader, err := c.docker.ImagePull(ctx, ref, dockerimage.PullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()
	// Drain the reader to complete the pull; discard progress output.
	_, _ = io.Copy(io.Discard, reader)
	return nil
}

const ProjectIssuesURL = "https://github.com/evefrontier/efctl/issues"

// StartContainer starts an existing container by name.
func (c *Client) StartContainer(ctx context.Context, name string) error {
	if err := c.docker.ContainerStart(ctx, name, dockercontainer.StartOptions{}); err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "netavark") && strings.Contains(errStr, "nftables") {
			return fmt.Errorf("start container %s: %w\n\nTIP: This error often occurs on WSL with Podman's default networking. Try setting 'firewall_driver = \"iptables\"' in your ~/.config/containers/containers.conf and running 'podman system reset' if the issue persists.\n\nPlease report this issue at %s - include the output of 'efctl doctor'.", name, err, ProjectIssuesURL)
		}
		return fmt.Errorf("start container %s: %w\n\nPlease report this issue at %s - include the output of 'efctl doctor'.", name, err, ProjectIssuesURL)
	}
	return nil
}

// StopContainer stops a running container by name (10s timeout).
func (c *Client) StopContainer(ctx context.Context, name string) error {
	timeout := 10
	if err := c.docker.ContainerStop(ctx, name, dockercontainer.StopOptions{Timeout: &timeout}); err != nil && !dockerclient.IsErrNotFound(err) {
		return fmt.Errorf("stop container %s: %w", name, err)
	}
	return nil
}

// RemoveContainer removes a container by name, ignoring "not found" errors.
func (c *Client) RemoveContainer(ctx context.Context, name string) error {
	if err := c.docker.ContainerRemove(ctx, name, dockercontainer.RemoveOptions{Force: true}); err != nil && !dockerclient.IsErrNotFound(err) {
		return fmt.Errorf("remove container %s: %w", name, err)
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
			info, err := c.docker.ContainerInspect(ctx, name)
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
	if c.docker == nil {
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

	execCfg, err := c.docker.ContainerExecCreate(ctx, name, dockercontainer.ExecOptions{
		Cmd:          cmd,
		AttachStdout: false,
		AttachStderr: false,
	})
	if err != nil {
		return false
	}

	// Use Detach: true so the exec runs asynchronously without requiring
	// an attached stream. Podman's compat API rejects ExecStart when no
	// streams are attached and Detach is false ("must provide at least one
	// stream to attach to").
	if err := c.docker.ContainerExecStart(ctx, execCfg.ID, dockercontainer.ExecStartOptions{
		Detach: true,
	}); err != nil {
		return false
	}

	// Poll for exec completion with 30-second timeout
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		inspect, err := c.docker.ContainerExecInspect(ctx, execCfg.ID)
		if err != nil {
			return false
		}
		if !inspect.Running {
			return inspect.ExitCode == 0
		}
		time.Sleep(200 * time.Millisecond)
	}
	// Timeout reached, treat as unhealthy
	ui.Debug.Println(fmt.Sprintf("Health check exec for %s timed out after 30s", name))
	return false
}

// ── Inspection / interaction ────────────────────────────────────────

// ContainerRunning checks if a container is currently running.
func (c *Client) ContainerRunning(name string) bool {
	ctx := context.Background()
	info, err := c.docker.ContainerInspect(ctx, name)
	if err != nil {
		return false
	}
	return info.State != nil && info.State.Running
}

// ContainerLogs returns the last N lines of a container's logs.
func (c *Client) ContainerLogs(name string, tail int) string {
	ctx := context.Background()
	reader, err := c.docker.ContainerLogs(ctx, name, dockercontainer.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       fmt.Sprintf("%d", tail),
	})
	if err != nil {
		return fmt.Sprintf("(could not retrieve logs: %v)", err)
	}
	defer reader.Close()

	var sb strings.Builder
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Bytes()
		// Docker multiplexes logs with an 8-byte header; strip it when present.
		if len(line) > 8 {
			sb.Write(line[8:])
		} else {
			sb.Write(line)
		}
		sb.WriteByte('\n')
	}
	return strings.TrimSpace(sb.String())
}

// ContainerExitCode returns the exit code of a stopped container.
func (c *Client) ContainerExitCode(name string) (int, error) {
	ctx := context.Background()
	info, err := c.docker.ContainerInspect(ctx, name)
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

	reader, err := c.docker.ContainerLogs(ctx, containerName, dockercontainer.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	if err != nil {
		spinner.Fail("Failed to get logs stream")
		return err
	}
	defer reader.Close()

	// Channel to signal when search string is found
	done := make(chan bool, 1)

	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, searchString) {
				done <- true
				return
			}
		}
		done <- false
	}()

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
	case found := <-done:
		if !found {
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

	spinner.Success(fmt.Sprintf("%s is ready", containerName))
	return nil
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
		_ = c.docker.ContainerStop(ctx, name, dockercontainer.StopOptions{})
		_ = c.docker.ContainerRemove(ctx, name, dockercontainer.RemoveOptions{Force: true})
	}
}

// RemoveImages removes container images by name, ignoring errors for
// images that do not exist.  This is called before BuildImage to
// ensure Podman does not reuse a stale cached image.
func (c *Client) RemoveImages(names []string) {
	ctx := context.Background()
	for _, name := range names {
		_, _ = c.docker.ImageRemove(ctx, name, dockerimage.RemoveOptions{Force: true})
	}
}

func (c *Client) removeVolumes(ctx context.Context, names []string) {
	for _, vol := range names {
		_ = c.docker.VolumeRemove(ctx, vol, true)
	}
}

func (c *Client) normalizeBindMountPermissions(containerName string) {
	ctx := context.Background()
	info, err := c.docker.ContainerInspect(ctx, containerName)
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
