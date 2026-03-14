package doctor

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"efctl/pkg/config"
	"efctl/pkg/env"
)

func TestGatherSystem_NonEmpty(t *testing.T) {
	info := gatherSystem()
	if info.OS == "" {
		t.Error("expected non-empty OS name")
	}
	if info.Platform == "" {
		t.Error("expected non-empty Platform")
	}
	if !strings.HasPrefix(info.GoVersion, "go") {
		t.Errorf("expected GoVersion to start with 'go', got %q", info.GoVersion)
	}
	if info.Platform != runtime.GOOS+"/"+runtime.GOARCH {
		t.Errorf("Platform mismatch: got %q, want %q", info.Platform, runtime.GOOS+"/"+runtime.GOARCH)
	}
}

func TestDetectWSLFrom(t *testing.T) {
	if !detectWSLFrom("5.15.167.4-microsoft-standard-WSL2") {
		t.Error("expected WSL kernel release to be detected")
	}
	if !detectWSLFrom("WSLInterop") {
		t.Error("expected non-empty WSL environment indicator to be detected")
	}
	if detectWSLFrom("", "") {
		t.Error("did not expect empty values to be detected")
	}
	if detectWSLFrom("6.8.0-generic") {
		t.Error("did not expect non-WSL values to be detected")
	}
}

func TestGatherPorts_FiveEntries(t *testing.T) {
	ports := gatherPorts()
	if len(ports) != 5 {
		t.Fatalf("expected 5 port entries, got %d", len(ports))
	}
	expected := []int{9000, 9123, 9125, 5432, 5173}
	for i, p := range ports {
		if p.Port != expected[i] {
			t.Errorf("port[%d]: expected %d, got %d", i, expected[i], p.Port)
		}
	}
}

func TestGatherRepo_Missing(t *testing.T) {
	tmp := t.TempDir()
	nonExistent := filepath.Join(tmp, "no-such-repo")

	info := gatherRepo("test-repo", nonExistent)
	if info.Found {
		t.Error("expected Found=false for non-existent directory")
	}
	if info.Name != "test-repo" {
		t.Errorf("expected Name=%q, got %q", "test-repo", info.Name)
	}
}

func TestGatherRepo_NotGit(t *testing.T) {
	tmp := t.TempDir()
	// Directory exists but is not a git repo.
	info := gatherRepo("test-repo", tmp)
	if info.Found {
		t.Errorf("expected Found=false for non-git directory, got Found=true with error=%q", info.Error)
	}
}

func TestGatherRepo_Present(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	tmp := t.TempDir()

	// Initialise a minimal git repository with one commit.
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = tmp
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	run("init", "-b", "main")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(tmp, "README.md"), []byte("hello"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	run("add", ".")
	run("commit", "-m", "init")

	info := gatherRepo("test-repo", tmp)
	if !info.Found {
		t.Fatalf("expected Found=true; error=%q", info.Error)
	}
	if len(info.Commit) != 40 {
		t.Errorf("expected 40-char commit hash, got %q", info.Commit)
	}
	if info.Branch != "main" {
		t.Errorf("expected branch=main, got %q", info.Branch)
	}
	if info.IsDirty {
		t.Error("freshly committed repo should not be dirty")
	}
}

func TestGatherRepo_Dirty(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	tmp := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = tmp
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	run("init", "-b", "main")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(tmp, "README.md"), []byte("hello"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	run("add", ".")
	run("commit", "-m", "init")

	// Introduce an uncommitted change.
	if err := os.WriteFile(filepath.Join(tmp, "new.txt"), []byte("dirty"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	run("add", "new.txt")

	info := gatherRepo("test-repo", tmp)
	if !info.Found {
		t.Fatalf("expected Found=true; error=%q", info.Error)
	}
	if !info.IsDirty {
		t.Error("expected IsDirty=true with staged but uncommitted file")
	}
}

func TestGather_NonNil(t *testing.T) {
	prereqs := &env.CheckResult{
		HasGit:    false,
		HasDocker: false,
		HasPodman: false,
		HasNode:   false,
	}

	r := Gather(Options{
		Workspace: t.TempDir(),
		Version:   "test",
		CommitSHA: "abc123",
		BuildDate: "2026-01-01",
		Prereqs:   prereqs,
	})

	if r == nil {
		t.Fatal("Gather returned nil")
	}
	if r.Efctl.Version != "test" {
		t.Errorf("expected Version=test, got %q", r.Efctl.Version)
	}
	if len(r.Ports) != 5 {
		t.Errorf("expected 5 ports, got %d", len(r.Ports))
	}
	if len(r.Repos) != 2 {
		t.Errorf("expected 2 repos, got %d", len(r.Repos))
	}
}

func TestGatherConfigEntries(t *testing.T) {
	withFrontend := true
	withGraphql := false
	entries := gatherConfigEntries(&config.Config{
		WithFrontend:       &withFrontend,
		WithGraphql:        &withGraphql,
		WorldContractsURL:  "https://example.com/world-contracts.git",
		WorldContractsRef:  "main",
		BuilderScaffoldURL: "https://example.com/builder-scaffold.git",
		BuilderScaffoldRef: "v1.2.3",
	})

	if len(entries) != 6 {
		t.Fatalf("expected 6 config entries, got %d", len(entries))
	}
	if entries[0].Key != "with-frontend" || entries[0].Value != "true" {
		t.Fatalf("unexpected first config entry: %+v", entries[0])
	}
	if entries[1].Key != "with-graphql" || entries[1].Value != "false" {
		t.Fatalf("unexpected second config entry: %+v", entries[1])
	}
	if entries[5].Key != "builder-scaffold-ref" || entries[5].Value != "v1.2.3" {
		t.Fatalf("unexpected last config entry: %+v", entries[5])
	}
}

func TestParseContainerVersion(t *testing.T) {
	cases := []struct {
		engine string
		raw    string
		want   string
	}{
		{"docker", "Docker version 28.5.2, build 123abc", "28.5.2"},
		{"podman", "podman version 4.9.0", "4.9.0"},
		{"docker", "Docker version 24.0.5, build ced0996", "24.0.5"},
	}
	for _, c := range cases {
		got := parseContainerVersion(c.engine, c.raw)
		if got != c.want {
			t.Errorf("parseContainerVersion(%q, %q) = %q, want %q", c.engine, c.raw, got, c.want)
		}
	}
}

func TestGatherRepos_ReturnsBoth(t *testing.T) {
	tmp := t.TempDir()
	repos := gatherRepos(tmp)
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
	if repos[0].Name != "builder-scaffold" {
		t.Errorf("expected repos[0].Name=builder-scaffold, got %q", repos[0].Name)
	}
	if repos[1].Name != "world-contracts" {
		t.Errorf("expected repos[1].Name=world-contracts, got %q", repos[1].Name)
	}
	// Neither directory exists → both should be not-found.
	for _, repo := range repos {
		if repo.Found {
			t.Errorf("repo %q: expected Found=false in empty tmp dir", repo.Name)
		}
	}
}
