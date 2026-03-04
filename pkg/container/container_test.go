package container

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	dockercontainer "github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
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
	cfg := SuiDevConfig("/workspace", "efctl-test", "docker", false)
	if _, ok := cfg.Ports[9000]; !ok {
		t.Error("Expected port 9000 in SuiDevConfig")
	}
	if _, ok := cfg.Ports[9125]; ok {
		t.Error("Port 9125 should not be present without graphql")
	}

	cfgGql := SuiDevConfig("/workspace", "efctl-test", "docker", true)
	if _, ok := cfgGql.Ports[9125]; !ok {
		t.Error("Expected port 9125 with graphql enabled")
	}
	if len(cfgGql.Env) == 0 {
		t.Error("Expected env vars with graphql enabled")
	}
}

func TestSuiDevConfig_PodmanUserns(t *testing.T) {
	// The sui-dev container must NOT use keep-id because the Dockerfile
	// installs everything under /root.  Podman rootless without keep-id
	// maps container UID 0 → host UID, preserving /root access.
	cfg := SuiDevConfig("/workspace", "efctl-test", "podman", false)
	if cfg.UsernsMode != "" {
		t.Errorf("Expected empty UsernsMode for Podman sui-dev, got %q", cfg.UsernsMode)
	}

	cfgDocker := SuiDevConfig("/workspace", "efctl-test", "docker", false)
	if cfgDocker.UsernsMode != "" {
		t.Errorf("Expected empty UsernsMode for Docker, got %q", cfgDocker.UsernsMode)
	}
}

func TestPostgresConfig_Healthcheck(t *testing.T) {
	cfg := PostgresConfig("efctl-test")
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
