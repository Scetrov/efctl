//go:build integration && linux

package doctor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"efctl/pkg/container"
	"efctl/pkg/env"
)

// This test intentionally refuses to use a normal host engine. The companion
// Bubblewrap harness exports an existing local image into a dedicated store.
// Sui here is a version-only shell fixture, not the real upstream executable.
func TestDoctorRuntimeIntegration(t *testing.T) {
	store, base := isolatedRuntimeFixture(t)
	run := isolatedPodman(t)
	contextDir := prepareRuntimeFixture(t, store, run)
	good := buildRuntimeFixture(t, run, contextDir, base, "good", container.ImageSuiDev, false)
	marker, sentinel := writeManagedSentinel(t, store)
	originalVolumes := run("volume", "ls", "--format", "{{.Name}}")
	t.Cleanup(func() {
		_ = runDiagnostic(context.Background(), "podman", "container", "rm", "--force", "--volumes", container.ContainerSuiPlayground)
	})
	for _, engine := range []string{"podman", "docker"} {
		t.Logf("checking engine %s", engine)
		verifyRuntimeEngine(t, runtimeScenario{
			run: run, engine: engine, store: store, contextDir: contextDir, base: base,
			good: good, marker: marker, sentinel: sentinel, originalVolumes: originalVolumes,
		})
	}
	assertNoImagePulls(t, store)
}

type fixtureRun func(args ...string) string

type runtimeScenario struct {
	run                             fixtureRun
	engine, store, contextDir, base string
	good, marker, sentinel          string
	originalVolumes                 string
}

func isolatedRuntimeFixture(t *testing.T) (string, string) {
	t.Helper()
	store := os.Getenv("EFCTL_DOCTOR_ISOLATED_STORE")
	if store == "" {
		t.Skip("use tools/test-doctor-runtime-isolated.sh with an existing local image ID")
	}
	base := os.Getenv("EFCTL_DOCTOR_BASE_ID")
	if !immutableImage.MatchString(base) {
		t.Fatal("immutable base fixture required")
	}
	return store, base
}

func isolatedPodman(t *testing.T) fixtureRun {
	t.Helper()
	return func(args ...string) string {
		t.Helper()
		return runIsolatedPodman(t, args)
	}
}

func runIsolatedPodman(t *testing.T, args []string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "podman", args...).Output() // #nosec G204 -- fixed fixture engine with structured local-only arguments
	if err != nil {
		return failPodman(t, args, err, output)
	}
	// Podman build progress is not metadata; only inspect/info output is consumed.
	return strings.TrimSpace(string(output))
}

func failPodman(t *testing.T, args []string, err error, output []byte) string {
	t.Helper()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		output = exit.Stderr
	}
	t.Fatalf("fixture command %v: %v (%s)", args, err, SanitizeDiagnostic(string(output)))
	return ""
}

func prepareRuntimeFixture(t *testing.T, store string, run fixtureRun) string {
	t.Helper()
	if got := run("info", "--format", "{{.Store.GraphRoot}}"); got != filepath.Join(store, "storage") {
		t.Fatalf("refusing non-isolated engine store: %s", got)
	}
	contextDir := filepath.Join(store, "fixture")
	if err := os.MkdirAll(contextDir, 0700); err != nil {
		t.Fatal(err)
	}
	return contextDir
}

func runtimeFixtureBody(mode string) string {
	body := "#!/bin/sh\nif [ \"$1\" != --version ]; then exit 99; fi\n"
	if mode == "hang" {
		return body + "sleep 30\n"
	}
	return body + "echo 'sui doctor-fixture 1.0'\n"
}

func buildRuntimeFixture(t *testing.T, run fixtureRun, contextDir, base, mode, tag string, volumes bool) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(contextDir, "sui"), []byte(runtimeFixtureBody(mode)), 0700); err != nil {
		t.Fatal(err)
	}
	file := fmt.Sprintf("FROM %s\nCOPY --chmod=0755 sui /usr/local/bin/sui\n", base)
	if volumes {
		file += "VOLUME /data\n"
	}
	if err := os.WriteFile(filepath.Join(contextDir, "Containerfile"), []byte(file), 0600); err != nil {
		t.Fatal(err)
	}
	run("build", "--pull=never", "--network=none", "--tag", tag, contextDir)
	return run("image", "inspect", "--format", "{{.Id}}", tag)
}

func writeManagedSentinel(t *testing.T, store string) (string, string) {
	t.Helper()
	marker := filepath.Join(store, "managed-data")
	if err := os.MkdirAll(marker, 0700); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(marker, "sentinel")
	if err := os.WriteFile(sentinel, []byte("unchanged"), 0600); err != nil {
		t.Fatal(err)
	}
	return marker, sentinel
}

func verifyRuntimeEngine(t *testing.T, scenario runtimeScenario) {
	t.Helper()
	t.Setenv("EFCTL_ENGINE", scenario.engine)
	startManagedFixture(scenario.run, scenario.good, scenario.marker)
	hanging := buildRuntimeFixture(t, scenario.run, scenario.contextDir, scenario.base, "hang", container.ImageSuiDev, false)
	if hanging == scenario.good {
		t.Fatal("fixture tag did not change identity")
	}
	assertRunningReport(t, collectRuntime(), scenario.good)
	scenario.run("container", "stop", "--time", "0", container.ContainerSuiPlayground)
	assertStoppedReport(t, collectRuntime(), scenario.good)
	assertImageOnlyReport(t, scenario.run, scenario.good)
	assertHungProbeCleanup(t, scenario.run, scenario.engine, hanging)
	assertManagedUnchanged(t, scenario.run, scenario.sentinel, scenario.originalVolumes)
	assertDeclaredVolumeSkipped(t, scenario)
}

func startManagedFixture(run fixtureRun, image, marker string) {
	run("run", "--detach", "--pull=never", "--network=none", "--name", container.ContainerSuiPlayground, "--volume", marker+":/data", "--entrypoint", "/bin/sh", image, "-c", "sleep 300")
}

func collectRuntime() RuntimeDiagnostics {
	return gatherRuntimeDiagnostics(Options{}, &env.CheckResult{HasDocker: true, HasPodman: true})
}

func assertRunningReport(t *testing.T, report RuntimeDiagnostics, good string) {
	t.Helper()
	if report.Image.ID.Value != good || report.Sui.Value != "sui doctor-fixture 1.0" || !strings.Contains(report.Sui.Source, "managed container") || report.CPU.Architecture.Value == "" {
		t.Fatalf("running report: %+v", report)
	}
}

func assertStoppedReport(t *testing.T, report RuntimeDiagnostics, good string) {
	t.Helper()
	if report.Image.ID.Value != good || report.Sui.Value != "sui doctor-fixture 1.0" || !strings.Contains(report.Sui.Source, "local-image probe") || report.Cleanup.Reason != "" {
		t.Fatalf("stopped report: %+v", report)
	}
}

func assertImageOnlyReport(t *testing.T, run fixtureRun, good string) {
	t.Helper()
	run("container", "rm", "--force", "--volumes", container.ContainerSuiPlayground)
	run("tag", good, container.ImageSuiDev)
	report := collectRuntime()
	if report.Sui.Value != "sui doctor-fixture 1.0" || !strings.Contains(report.Image.ID.Source, "fallback") || report.Cleanup.Reason != "" {
		t.Fatalf("image-only report: %+v", report)
	}
}

func assertHungProbeCleanup(t *testing.T, run fixtureRun, engine, hanging string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	version, _, cleanup := disposableEvidence(probes{ctx: ctx, run: runDiagnostic}, engine, hanging, true)
	cancel()
	if version.Reason == "" || cleanup.Reason != "" {
		t.Fatalf("timeout/cleanup: %+v %+v", version, cleanup)
	}
	names := run("container", "ls", "--all", "--format", "{{.Names}}")
	if strings.Contains(names, "efctl-doctor-") {
		t.Fatalf("surviving disposable probe: %s", names)
	}
}

func assertManagedUnchanged(t *testing.T, run fixtureRun, sentinel, originalVolumes string) {
	t.Helper()
	data, err := os.ReadFile(sentinel)
	if err != nil || string(data) != "unchanged" {
		t.Fatal("managed fixture data modified")
	}
	if after := run("volume", "ls", "--format", "{{.Name}}"); after != originalVolumes {
		t.Fatal("anonymous volume leak")
	}
}

func assertDeclaredVolumeSkipped(t *testing.T, scenario runtimeScenario) {
	t.Helper()
	buildRuntimeFixture(t, scenario.run, scenario.contextDir, scenario.base, "good", container.ImageSuiDev, true)
	skipped := collectRuntime()
	if !strings.Contains(skipped.Sui.Reason, "volumes") {
		t.Fatalf("unsafe image was not skipped: %+v", skipped)
	}
	if after := scenario.run("volume", "ls", "--format", "{{.Name}}"); after != scenario.originalVolumes {
		t.Fatal("declared-volume image caused a leak")
	}
	scenario.run("tag", scenario.good, container.ImageSuiDev)
}

func assertNoImagePulls(t *testing.T, store string) {
	t.Helper()
	log, err := os.ReadFile(filepath.Join(store, "commands.log"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(log), "\n") {
		if strings.HasPrefix(line, "pull ") {
			t.Fatalf("image pull attempted: %s", line)
		}
	}
}
