package container

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"efctl/pkg/env"

	dockercontainer "github.com/docker/docker/api/types/container"
	dockermount "github.com/docker/docker/api/types/mount"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	// Attempt to create a client. If the system has docker or podman, it should succeed.
	// We can manipulate the environment variable to force a specific engine if needed.

	os.Setenv("EFCTL_ENGINE", "docker")
	defer os.Unsetenv("EFCTL_ENGINE")

	client, err := NewClient()

	// If the runner has docker or podman, it should work.
	if err != nil {
		t.Logf("NewClient returned error (possibly missing docker/podman): %v", err)
	} else if client == nil {
		t.Errorf("Expected non-nil client if err is nil")
	}
}

func TestNetworkNameForWorkspace_Deterministic(t *testing.T) {
	name1 := NetworkNameForWorkspace("/home/user/project")
	name2 := NetworkNameForWorkspace("/home/user/project")
	if name1 != name2 {
		t.Errorf("Expected deterministic network name, got %q and %q", name1, name2)
	}
	if name1[:5] != "efctl" {
		t.Errorf("Expected network name to start with 'efctl', got %q", name1)
	}
}

func TestNetworkNameForWorkspace_UniquePerPath(t *testing.T) {
	name1 := NetworkNameForWorkspace("/home/user/project-a")
	name2 := NetworkNameForWorkspace("/home/user/project-b")
	if name1 == name2 {
		t.Errorf("Expected different network names for different paths, both got %q", name1)
	}
}

func TestSuiDevConfig_Ports(t *testing.T) {
	cfg := SuiDevConfig("/workspace", "efctl-test", "docker", false, "sui", "pass", "db", nil)
	if _, ok := cfg.Ports[9000]; !ok {
		t.Error("Expected port 9000 in SuiDevConfig")
	}
	if _, ok := cfg.Ports[9125]; ok {
		t.Error("Port 9125 should not be present without graphql")
	}

	cfgGql := SuiDevConfig("/workspace", "efctl-test", "docker", true, "sui", "pass", "db", nil)
	if _, ok := cfgGql.Ports[9125]; !ok {
		t.Error("Expected port 9125 with graphql enabled")
	}
	if len(cfgGql.Env) == 0 {
		t.Error("Expected env vars with graphql enabled")
	}
}

func TestSuiDevConfig_PodmanUserns(t *testing.T) {
	// The sui-dev container must use keep-id to avoid host permission
	// issues with bind mounts in Podman rootless mode.
	cfg := SuiDevConfig("/workspace", "efctl-test", "podman", false, "sui", "pass", "db", nil)
	if cfg.UsernsMode != "keep-id" {
		t.Errorf("Expected UsernsMode 'keep-id' for Podman sui-dev, got %q", cfg.UsernsMode)
	}

	cfgDocker := SuiDevConfig("/workspace", "efctl-test", "docker", false, "sui", "pass", "db", nil)
	if cfgDocker.UsernsMode != "" {
		t.Errorf("Expected empty UsernsMode for Docker, got %q", cfgDocker.UsernsMode)
	}
}

func TestSuiDevConfig_AdditionalBindMounts(t *testing.T) {
	cfg := SuiDevConfig("/workspace", "efctl-test", "docker", false, "sui", "pass", "db", []AdditionalBindMount{{
		Source:     "/tmp/contracts",
		Identifier: "contracts_mount",
	}})

	require.Len(t, cfg.Mounts, 4)
	assert.Equal(t, "/tmp/contracts", cfg.Mounts[3].Source)
	assert.Equal(t, "/workspace/mounts/contracts_mount", cfg.Mounts[3].Target)
	assert.True(t, cfg.Mounts[3].SELinux)
}

func TestPostgresConfig_Healthcheck(t *testing.T) {
	cfg := PostgresConfig("efctl-test", "sui", "pass", "db")
	if cfg.Healthcheck == nil {
		t.Fatal("Expected healthcheck for postgres")
	}
	if cfg.Healthcheck.Retries != 30 {
		t.Errorf("Expected 30 retries, got %d", cfg.Healthcheck.Retries)
	}
	if cfg.Name != ContainerPostgres {
		t.Errorf("Expected container name %q, got %q", ContainerPostgres, cfg.Name)
	}
}

func TestFrontendConfig_WorkingDir(t *testing.T) {
	cfg := FrontendConfig("/workspace", "efctl-test", "docker")
	if cfg.WorkingDir != "/workspace/builder-scaffold/dapps" {
		t.Errorf("Expected working dir /workspace/builder-scaffold/dapps, got %q", cfg.WorkingDir)
	}
	if cfg.Name != ContainerFrontend {
		t.Errorf("Expected container name %q, got %q", ContainerFrontend, cfg.Name)
	}
}

func TestPrepareMountConfig_DockerSkipsSharedPropagation(t *testing.T) {
	c := &Client{Engine: "docker"}
	mounts := c.prepareMountConfig([]MountDef{{
		Type:    "bind",
		Source:  "/tmp/workspace/builder-scaffold",
		Target:  "/workspace/builder-scaffold",
		SELinux: true,
	}})

	if len(mounts) != 1 {
		t.Fatalf("expected 1 mount, got %d", len(mounts))
	}
	if mounts[0].Type != dockermount.TypeBind {
		t.Fatalf("expected bind mount, got %q", mounts[0].Type)
	}
	if mounts[0].BindOptions != nil {
		t.Fatalf("expected docker bind mount to omit propagation options, got %+v", mounts[0].BindOptions)
	}
}

func TestPrepareMountConfig_PodmanUsesSharedPropagation(t *testing.T) {
	c := &Client{Engine: "podman"}
	mounts := c.prepareMountConfig([]MountDef{{
		Type:    "bind",
		Source:  "/tmp/workspace/builder-scaffold",
		Target:  "/workspace/builder-scaffold",
		SELinux: true,
	}})

	if len(mounts) != 1 {
		t.Fatalf("expected 1 mount, got %d", len(mounts))
	}
	if mounts[0].BindOptions == nil {
		t.Fatal("expected podman bind mount to include bind options")
	}
	if mounts[0].BindOptions.Propagation != dockermount.PropagationShared {
		t.Fatalf("expected podman bind mount propagation %q, got %q", dockermount.PropagationShared, mounts[0].BindOptions.Propagation)
	}
}

func TestPreferredEngineOrder(t *testing.T) {
	res := &env.CheckResult{HasDocker: true, HasPodman: true}

	t.Setenv("EFCTL_ENGINE", "docker")
	if got := preferredEngineOrder(res); !reflect.DeepEqual(got, []string{"docker", "podman"}) {
		t.Fatalf("expected docker preference order, got %v", got)
	}

	t.Setenv("EFCTL_ENGINE", "podman")
	if got := preferredEngineOrder(res); !reflect.DeepEqual(got, []string{"podman", "docker"}) {
		t.Fatalf("expected podman preference order, got %v", got)
	}

	t.Setenv("EFCTL_ENGINE", "")
	if got := preferredEngineOrder(res); !reflect.DeepEqual(got, []string{"podman", "docker"}) {
		t.Fatalf("expected default podman-first order, got %v", got)
	}
}

func TestConnectionCandidates_FallbackToDockerWhenPodmanSocketMissing(t *testing.T) {
	t.Setenv("EFCTL_ENGINE", "")
	res := &env.CheckResult{HasDocker: true, HasPodman: true}
	candidates := connectionCandidates(res, "linux", 1001, "unix:///var/run/docker.sock", func(host string) bool {
		return false
	})

	if len(candidates) != 1 {
		t.Fatalf("expected only docker candidate, got %d (%+v)", len(candidates), candidates)
	}
	if candidates[0].engine != "docker" {
		t.Fatalf("expected docker fallback, got %+v", candidates[0])
	}
	if !candidates[0].useFromEnv {
		t.Fatalf("expected docker candidate to use environment host, got %+v", candidates[0])
	}
}

func TestConnectionCandidates_UsePodmanSocketWhenAvailable(t *testing.T) {
	t.Setenv("EFCTL_ENGINE", "")
	res := &env.CheckResult{HasDocker: true, HasPodman: true}
	podmanHost := "unix:///run/user/1001/podman/podman.sock"
	candidates := connectionCandidates(res, "linux", 1001, "unix:///var/run/docker.sock", func(host string) bool {
		return host == podmanHost
	})

	if len(candidates) < 2 {
		t.Fatalf("expected podman and docker candidates, got %+v", candidates)
	}
	if candidates[0].engine != "podman" || candidates[0].host != podmanHost {
		t.Fatalf("expected first candidate to use podman socket, got %+v", candidates[0])
	}
	if candidates[1].engine != "docker" {
		t.Fatalf("expected docker fallback candidate second, got %+v", candidates[1])
	}
}

// ── execHealthProbe unit tests ─────────────────────────────────────

func TestExecHealthProbe_NoHealthTests(t *testing.T) {
	// When healthTests is nil or doesn't contain the container, return false.
	c := &Client{healthTests: nil}
	if c.execHealthProbe(context.Background(), "no-such-container") {
		t.Error("expected false when healthTests is nil")
	}

	c2 := &Client{healthTests: map[string][]string{"other": {"CMD", "true"}}}
	if c2.execHealthProbe(context.Background(), "no-such-container") {
		t.Error("expected false when container not in healthTests")
	}
}

func TestExecHealthProbe_EmptyTest(t *testing.T) {
	c := &Client{healthTests: map[string][]string{"ctr": {}}}
	if c.execHealthProbe(context.Background(), "ctr") {
		t.Error("expected false for empty test slice")
	}
}

func TestExecHealthProbe_NilDockerClient(t *testing.T) {
	// When the docker client is nil, execHealthProbe must return false
	// without panicking.
	c := &Client{
		docker: nil,
		healthTests: map[string][]string{
			"ctr": {"CMD-SHELL", "pg_isready -U sui"},
		},
	}
	if c.execHealthProbe(context.Background(), "ctr") {
		t.Error("expected false when docker client is nil")
	}
}

// ── Integration: exec health probe against real Podman ─────────────

// TestExecHealthProbe_PodmanDetach validates that the exec health probe
// works with Podman by using Detach:true in ExecStartOptions.
// Requires a running Podman daemon; skipped otherwise.
func TestExecHealthProbe_PodmanDetach(t *testing.T) {
	if os.Getenv("EFCTL_INTEGRATION") == "" {
		t.Skip("set EFCTL_INTEGRATION=1 to run container integration tests")
	}

	uid := os.Getuid()
	sock := fmt.Sprintf("unix:///run/user/%d/podman/podman.sock", uid)
	dc, err := dockerclient.NewClientWithOpts(
		dockerclient.WithHost(sock),
		dockerclient.WithAPIVersionNegotiation(),
	)
	if err != nil {
		t.Skipf("cannot create docker client: %v", err)
	}

	ctx := context.Background()
	const name = "efctl-test-health-probe"

	// Cleanup any previous run.
	dc.ContainerRemove(ctx, name, dockercontainer.RemoveOptions{Force: true})
	t.Cleanup(func() {
		dc.ContainerRemove(ctx, name, dockercontainer.RemoveOptions{Force: true})
	})

	// Create and start a postgres container.
	cp := nat.Port("5432/tcp")
	_, err = dc.ContainerCreate(ctx, &dockercontainer.Config{
		Image:        ImagePostgres,
		Env:          []string{"POSTGRES_USER=sui", "POSTGRES_PASSWORD=sui", "POSTGRES_DB=sui_indexer"},
		ExposedPorts: map[nat.Port]struct{}{cp: {}},
		Healthcheck: &dockercontainer.HealthConfig{
			Test:        []string{"CMD-SHELL", "pg_isready -U sui -d sui_indexer"},
			Interval:    2 * time.Second,
			Timeout:     3 * time.Second,
			Retries:     30,
			StartPeriod: 5 * time.Second,
		},
	}, nil, nil, nil, name)
	if err != nil {
		t.Fatalf("create container: %v", err)
	}
	if err := dc.ContainerStart(ctx, name, dockercontainer.StartOptions{}); err != nil {
		t.Fatalf("start container: %v", err)
	}

	// Wait for Postgres to finish init (up to 30s).
	ready := false
	for i := 0; i < 15; i++ {
		time.Sleep(2 * time.Second)
		info, err := dc.ContainerInspect(ctx, name)
		if err != nil || info.State == nil || !info.State.Running {
			continue
		}
		ready = true
		break
	}
	if !ready {
		t.Fatal("postgres container never became running")
	}

	// Now test execHealthProbe with the Detach-based approach.
	c := &Client{
		Engine: "podman",
		docker: dc,
		healthTests: map[string][]string{
			name: {"CMD-SHELL", "pg_isready -U sui -d sui_indexer"},
		},
	}

	ok := c.execHealthProbe(ctx, name)
	if !ok {
		t.Error("execHealthProbe returned false; expected true for healthy postgres")
	}
}
