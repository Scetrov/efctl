package doctor

import (
	"context"
	"strings"
	"testing"
	"time"

	"efctl/pkg/env"
)

func TestManagedProbesIndependent(t *testing.T) {
	p := probes{ctx: context.Background(), run: func(_ context.Context, _ string, args ...string) CommandResult {
		if strings.Contains(strings.Join(args, " "), "sui --version") {
			return CommandResult{Reason: "signal: illegal instruction", ExitCode: -1}
		}
		if args[len(args)-1] == "-m" {
			return CommandResult{Stdout: []byte("x86_64")}
		}
		return CommandResult{Stdout: []byte("processor : 0\nflags : sse avx\n")}
	}}
	version, cpu := executionEvidence(p, "docker", "sui-playground", "managed container")
	if !strings.Contains(version.Reason, "illegal instruction") || cpu.Flags.Value != "avx sse" || cpu.Architecture.Value != "x86_64" {
		t.Fatalf("lost independent evidence: %+v %+v", version, cpu)
	}
}

func TestManagedExitRaceFallsBackOnce(t *testing.T) {
	for _, mode := range []string{"running", "race", "removed"} {
		t.Run(mode, func(t *testing.T) {
			runManagedExitRace(t, mode)
		})
	}
}

func runManagedExitRace(t *testing.T, mode string) {
	t.Helper()
	state := &raceProbe{t: t, mode: mode, id: "sha256:" + strings.Repeat("a", 64)}
	p := probes{ctx: context.Background(), run: state.run}
	got := gatherDiagnostics(p, &env.CheckResult{HasDocker: true})
	assertRaceReport(t, mode, got, state.runs)
}

type raceProbe struct {
	t     *testing.T
	mode  string
	id    string
	runs  int
	token string
}

func (s *raceProbe) run(_ context.Context, name string, args ...string) CommandResult {
	if name == "sui" || name == "uname" {
		return localBinaryResult(name)
	}
	switch args[0] {
	case "--version", "version", "info":
		return staticEngineResult(args[0])
	case "image":
		return staticImageResult(s.id)
	case "container":
		return s.container(args)
	case "run":
		return s.recordRun(args)
	case "exec":
		return s.exec(args)
	default:
		s.t.Fatalf("unexpected %s %v", name, args)
		return CommandResult{}
	}
}

func localBinaryResult(name string) CommandResult {
	if name == "sui" {
		return CommandResult{Reason: "executable not found"}
	}
	return CommandResult{Stdout: []byte("x86_64")}
}

func staticEngineResult(kind string) CommandResult {
	switch kind {
	case "--version":
		return CommandResult{Stdout: []byte("Docker version 29")}
	case "version":
		return CommandResult{Stdout: []byte(`{"Client":"29","Server":"29"}`)}
	default:
		return CommandResult{Stdout: []byte(`{"OS":"linux","Architecture":"amd64"}`)}
	}
}

func staticImageResult(id string) CommandResult {
	return CommandResult{Stdout: []byte(`{"ID":"` + id + `","OS":"linux","Architecture":"amd64","Volumes":0}`)}
}

func (s *raceProbe) container(args []string) CommandResult {
	if args[1] == "rm" || args[1] == "ls" || args[1] == "kill" {
		return CommandResult{}
	}
	if strings.Contains(args[3], "Labels") {
		return CommandResult{Stdout: []byte(s.token)}
	}
	if args[3] == "{{.State.Running}}" {
		return s.runningState()
	}
	return CommandResult{Stdout: []byte(`{"State":"running","Running":true,"ExitCode":0,"Reference":"rebuilt-tag","ID":"` + s.id + `"}`)}
}

func (s *raceProbe) runningState() CommandResult {
	if s.mode == "removed" {
		return CommandResult{Reason: "container removed"}
	}
	return CommandResult{Stdout: []byte("false")}
}

func (s *raceProbe) recordRun(args []string) CommandResult {
	s.runs++
	for i, arg := range args {
		if arg == "--label" {
			s.token = strings.SplitN(args[i+1], "=", 2)[1]
		}
	}
	return CommandResult{Stdout: []byte("probe-id")}
}

func (s *raceProbe) exec(args []string) CommandResult {
	if args[1] == "sui-playground" && s.mode != "running" {
		return CommandResult{Reason: "container exited"}
	}
	return probeExecPayload(args)
}

func probeExecPayload(args []string) CommandResult {
	last := args[len(args)-1]
	if last == "--version" {
		return CommandResult{Stdout: []byte("sui 1.0")}
	}
	if last == "-m" {
		return CommandResult{Stdout: []byte("x86_64")}
	}
	return CommandResult{Stdout: []byte("processor : 0\nflags : sse avx\n")}
}

func assertRaceReport(t *testing.T, mode string, got RuntimeDiagnostics, runs int) {
	t.Helper()
	if got.HostSui.Reason != "executable not found" || got.Sui.Value != "sui 1.0" || got.CPU.Flags.Value != "avx sse" {
		t.Fatalf("lost independent evidence: %+v", got)
	}
	if mode == "running" {
		assertNoRaceFallback(t, got, runs)
		return
	}
	assertOneRaceFallback(t, got, runs)
}

func assertNoRaceFallback(t *testing.T, got RuntimeDiagnostics, runs int) {
	t.Helper()
	if runs != 0 || !strings.Contains(got.Sui.Source, "managed container") {
		t.Fatal("healthy managed container replaced")
	}
}

func assertOneRaceFallback(t *testing.T, got RuntimeDiagnostics, runs int) {
	t.Helper()
	if runs != 1 || got.ExecFailure.Reason == "" || !strings.Contains(got.Sui.Source, "local-image") {
		t.Fatalf("race fallback: %+v, %d runs", got, runs)
	}
}

func TestDisposableIsolationAndCleanup(t *testing.T) {
	id := "sha256:" + strings.Repeat("a", 64)
	for _, mode := range []string{"success", "runfailure", "execfailure", "timeout", "cancel", "cleanupfailure", "collision"} {
		t.Run(mode, func(t *testing.T) {
			runDisposableMode(t, mode, id)
		})
	}
	t.Run("volumes", func(t *testing.T) {
		assertVolumeProbeSkipped(t, id)
	})
}

func runDisposableMode(t *testing.T, mode, id string) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	probe := &disposableProbe{t: t, mode: mode, cancel: cancel}
	p := probes{ctx: ctx, run: probe.run}
	version, cpu, cleanup := disposableEvidence(p, "docker", id, true)
	assertDisposableOutcome(t, mode, probe.name, probe.calls, version, cpu, cleanup)
}

type disposableProbe struct {
	t      *testing.T
	mode   string
	cancel context.CancelFunc
	calls  [][]string
	name   string
	label  string
}

func (d *disposableProbe) run(ctx context.Context, _ string, args ...string) CommandResult {
	d.calls = append(d.calls, append([]string(nil), args...))
	switch args[0] {
	case "run":
		return d.handleRun(args)
	case "exec":
		return d.handleExec(args)
	case "container":
		return d.handleContainer(ctx, args)
	default:
		d.t.Fatalf("unexpected %v", args)
		return CommandResult{}
	}
}

func (d *disposableProbe) handleRun(args []string) CommandResult {
	d.captureIdentity(args)
	assertIsolationArgs(d.t, args, d.name)
	return d.runOutcome()
}

func (d *disposableProbe) captureIdentity(args []string) {
	for i, arg := range args {
		if arg == "--name" {
			d.name = args[i+1]
		}
		if arg == "--label" {
			d.label = strings.SplitN(args[i+1], "=", 2)[1]
		}
	}
}

func isolationRequirements() []string {
	return []string{"--pull never", "--network none", "--read-only", "--cap-drop ALL", "--security-opt no-new-privileges", "--memory 256m", "--cpus 1", "--pids-limit 32", "--ulimit core=0:0", "--entrypoint /bin/sh", "--no-healthcheck", "--log-driver none"}
}

func assertIsolationArgs(t *testing.T, args []string, name string) {
	t.Helper()
	joined := strings.Join(args, " ")
	for _, required := range isolationRequirements() {
		if !strings.Contains(joined, required) {
			t.Fatalf("missing isolation: %s", required)
		}
	}
	assertNoUnsafeProbeArgs(t, args)
	if name == "sui-playground" || !strings.HasPrefix(name, "efctl-doctor-") {
		t.Fatal("unsafe name")
	}
}

func assertNoUnsafeProbeArgs(t *testing.T, args []string) {
	t.Helper()
	for _, arg := range args {
		if arg == "--volume" || arg == "-v" || arg == "--mount" || arg == "--env" || arg == "-e" {
			t.Fatalf("unsafe arg: %s", arg)
		}
	}
}

func (d *disposableProbe) runOutcome() CommandResult {
	switch d.mode {
	case "cancel":
		d.cancel()
		return CommandResult{Reason: "context canceled"}
	case "runfailure":
		return CommandResult{Reason: "unsupported isolation"}
	case "timeout":
		return CommandResult{Reason: "context deadline exceeded"}
	default:
		return CommandResult{Stdout: []byte("probe-id")}
	}
}

func (d *disposableProbe) handleExec(args []string) CommandResult {
	if d.mode == "execfailure" && args[len(args)-1] == "--version" {
		return CommandResult{Reason: "exit status 132"}
	}
	if args[len(args)-1] == "-m" {
		return CommandResult{Stdout: []byte("aarch64")}
	}
	return CommandResult{Stdout: []byte("processor : 0\nFeatures : aes neon\n")}
}

func (d *disposableProbe) handleContainer(ctx context.Context, args []string) CommandResult {
	assertCleanupBudget(d.t, ctx)
	switch args[1] {
	case "inspect":
		return d.inspectLabel()
	case "kill":
		d.assertOwned(args, "preexisting container killed")
		return CommandResult{}
	case "rm":
		return d.remove(args)
	default:
		d.t.Fatalf("unexpected %v", args)
		return CommandResult{}
	}
}

func assertCleanupBudget(t *testing.T, ctx context.Context) {
	t.Helper()
	deadline, ok := ctx.Deadline()
	if ctx.Err() != nil || !ok || time.Until(deadline) > 3*time.Second {
		t.Fatal("cleanup lacks independent bounded context")
	}
}

func (d *disposableProbe) inspectLabel() CommandResult {
	if d.mode == "collision" {
		return CommandResult{Stdout: []byte("someone-else")}
	}
	return CommandResult{Stdout: []byte(d.label)}
}

func (d *disposableProbe) assertOwned(args []string, message string) {
	d.t.Helper()
	if args[len(args)-1] != d.name || d.mode == "collision" {
		d.t.Fatal(message)
	}
}

func (d *disposableProbe) remove(args []string) CommandResult {
	d.assertOwned(args, "preexisting container removed")
	if d.mode == "cleanupfailure" {
		return CommandResult{Reason: "removal failed"}
	}
	return CommandResult{}
}

func assertDisposableOutcome(t *testing.T, mode, name string, calls [][]string, version Evidence, cpu CPUInfo, cleanup Evidence) {
	t.Helper()
	removals := countProbeArg(calls, "rm")
	if mode != "collision" && removals != 1 {
		t.Fatalf("cleanup attempts: %d", removals)
	}
	assertDisposableEvidence(t, mode, name, version, cpu, cleanup)
}

func countProbeArg(calls [][]string, want string) int {
	removals := 0
	for _, args := range calls {
		if len(args) > 1 && args[1] == want {
			removals++
		}
	}
	return removals
}

func assertDisposableEvidence(t *testing.T, mode, name string, version Evidence, cpu CPUInfo, cleanup Evidence) {
	t.Helper()
	if mode == "success" && (version.Reason != "" || cpu.Flags.Value != "aes neon") {
		t.Fatalf("missing evidence: %+v %+v", version, cpu)
	}
	if mode == "cleanupfailure" && !strings.Contains(cleanup.Reason, name) {
		t.Fatal("safe cleanup target omitted")
	}
}

func assertVolumeProbeSkipped(t *testing.T, id string) {
	t.Helper()
	p := probes{ctx: context.Background(), run: func(context.Context, string, ...string) CommandResult {
		t.Fatal("unsafe image executed")
		return CommandResult{}
	}}
	version, _, _ := disposableEvidence(p, "docker", id, false)
	if version.Reason == "" {
		t.Fatal("skip not reported")
	}
}
