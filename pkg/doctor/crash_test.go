package doctor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

const crashFixtureSecret = "efctl-coredump-fixture-9f3a7c"

func TestCrashResultKinds(t *testing.T) {
	id := crashContainerID()
	attributed := CrashRecord{Timestamp: time.Unix(1_700_000_000, 0).UTC(), PID: 42, Signal: 4, SignalName: "SIGILL", Executable: "/usr/bin/sui", Cgroup: "/docker/" + id, CoreKnown: true, CorePresent: true}
	cases := []struct {
		name string
		in   CrashResult
		want string
		text string
	}{
		{name: "hints", in: CrashResult{Hints: CrashHints{ContainerID: id, SIGILLCompatible: true, LogExcerpt: "Illegal instruction"}}, want: "hints", text: "shell/container boundary"},
		{name: "skipped", in: CrashResult{Hints: CrashHints{IdentityUnavailable: true}, Correlation: CrashCorrelation{Skipped: true, Reason: "remote/VM boundary not established"}}, want: "skipped", text: "correlation: skipped:"},
		{name: "unscoped", in: CrashResult{Correlation: CrashCorrelation{UnscopedCount: 1, Unscoped: []CrashRecord{{Executable: "/usr/bin/sui", PID: 7}}}}, want: "unscoped", text: "not unwound"},
		{name: "attributed", in: CrashResult{Correlation: CrashCorrelation{AttributedCount: 1, Selected: &attributed}}, want: "attributed", text: "attributed to sui-playground"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.in.Kind() != tc.want {
				t.Fatalf("kind %s", tc.in.Kind())
			}
			if !strings.Contains(renderCrash(tc.in), tc.text) {
				t.Fatalf("render missing %s: %s", tc.text, renderCrash(tc.in))
			}
		})
	}
}

func TestCrashFixtureSecretIsDropped(t *testing.T) {
	parsed, index, line := crashSecretFixtures(t)
	result := CrashResult{Correlation: CrashCorrelation{AttributedCount: 1, Selected: &parsed[0].record}}
	assertNoCrashSecret(t, resultBlob(result)+renderCrash(result))
	opts := qualifyingCrashOptions()
	opts.Runner = secretCrashRunner(line, index)
	got := CollectCrash(opts)
	assertNoCrashSecret(t, resultBlob(got)+renderCrash(got))
	if got.Kind() != "attributed" || got.Correlation.Selected == nil || !got.Correlation.Selected.CorePresent {
		t.Fatalf("attribution/core: %+v", got.Correlation)
	}
}

func TestCrashExitAndLogHints(t *testing.T) {
	id := crashContainerID()
	image := "sha256:" + strings.Repeat("e", 64)
	opts := qualifyingCrashOptions()
	opts.Boundary = Evidence{Reason: "remote/VM boundary not established"}
	opts.LookPath = func(string) (string, error) { t.Fatal("helper lookup"); return "", nil }
	var formats []string
	opts.Runner = exitLogRunner(t, id, image, &formats)
	got := CollectCrash(opts)
	assertExitLogHint(t, got, id, formats)
}

func TestCrashAbsentContainerSkipsCorrelation(t *testing.T) {
	opts := qualifyingCrashOptions()
	opts.Runner = func(_ context.Context, cmd CrashCommand) CommandResult {
		if cmd.Name != "docker" || hasExactArg(cmd.Args, "logs") {
			t.Fatalf("unexpected %s %v", cmd.Name, cmd.Args)
		}
		return CommandResult{Reason: "No such container"}
	}
	got := CollectCrash(opts)
	if got.Kind() != "skipped" || !strings.Contains(renderCrash(got), "managed container identity is unavailable") {
		t.Fatalf("%+v", got)
	}
}

func TestCrashImageIDIsNotContainerID(t *testing.T) {
	image := "sha256:" + strings.Repeat("e", 64)
	opts := qualifyingCrashOptions()
	opts.Runner = func(_ context.Context, cmd CrashCommand) CommandResult {
		if hasExactArg(cmd.Args, "logs") {
			t.Fatal("logs without container id")
		}
		return CommandResult{Stdout: []byte(`{"ID":"` + image + `","Image":"` + image + `","Status":"exited","ExitCode":132}`)}
	}
	got := CollectCrash(opts)
	if got.Hints.ContainerID != "" || strings.Contains(resultBlob(got), strings.Repeat("e", 64)) {
		t.Fatalf("image id used: %+v", got.Hints)
	}
}

func TestCrashInspectTemplates(t *testing.T) {
	for _, format := range []string{crashContainerFields, crashContainerPodmanFields} {
		if strings.Contains(format, ".Image") || !strings.Contains(format, "StartedAt") || !strings.Contains(format, "FinishedAt") || !strings.Contains(format, "ExitCode") {
			t.Fatalf("template: %s", format)
		}
	}
	if !strings.Contains(crashContainerFields, "{{json .Id}}") || !strings.Contains(crashContainerPodmanFields, "{{json .ID}}") {
		t.Fatal("container id field missing")
	}
	id := crashContainerID()
	opts := qualifyingCrashOptions()
	opts.Engine = "podman"
	opts.Boundary = Evidence{Reason: "remote/VM boundary not established"}
	var formats []string
	opts.Runner = func(_ context.Context, cmd CrashCommand) CommandResult {
		if cmd.Name != "podman" {
			t.Fatalf("engine %s", cmd.Name)
		}
		if hasExactArg(cmd.Args, "logs") {
			return CommandResult{}
		}
		format := argAfter(cmd.Args, "--format")
		formats = append(formats, format)
		if strings.Contains(format, "{{json .ID}}") {
			return crashInspectResult(id, "exited", intPtr(1), "2024-01-02T03:00:00Z", "2024-01-02T03:10:00Z", "")
		}
		return CommandResult{Reason: "exit status 125"}
	}
	got := CollectCrash(opts)
	if got.Hints.ContainerID != id || len(formats) != 2 {
		t.Fatalf("podman formats: %v id %s", formats, got.Hints.ContainerID)
	}
}

func TestCrashQueryGateSkipsHelpers(t *testing.T) {
	id := crashContainerID()
	cases := []struct {
		name     string
		goos     string
		boundary Evidence
		lookup   func(string) (string, error)
	}{
		{name: "remote", boundary: Evidence{Source: "runtime", Value: "engine reports remote service"}},
		{name: "desktop", boundary: Evidence{Source: "runtime", Value: "Docker Desktop server; VM boundary possible"}},
		{name: "unestablished", boundary: Evidence{Reason: "remote/VM boundary not established"}},
		{name: "non-linux", goos: "darwin", boundary: Evidence{Source: "runtime", Value: "local engine kernel"}},
		{name: "missing coredumpctl", boundary: Evidence{Source: "runtime", Value: "local engine kernel"}, lookup: func(string) (string, error) { return "", errors.New("missing") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := qualifyingCrashOptions()
			opts.Boundary = tc.boundary
			if tc.goos != "" {
				opts.GOOS = tc.goos
			}
			if tc.lookup != nil {
				opts.LookPath = tc.lookup
			}
			var helpers []string
			opts.Runner = func(_ context.Context, cmd CrashCommand) CommandResult {
				if cmd.Name == "coredumpctl" || cmd.Name == "journalctl" || strings.Contains(strings.Join(cmd.Args, " "), "ssh") {
					helpers = append(helpers, cmd.Name)
				}
				if cmd.Name == "docker" && hasExactArg(cmd.Args, "inspect") {
					return crashInspectResult(id, "exited", intPtr(132), "2024-01-02T03:00:00Z", "2024-01-02T03:10:00Z", "")
				}
				return CommandResult{}
			}
			got := CollectCrash(opts)
			if len(helpers) != 0 || !got.Correlation.Skipped || got.Correlation.Reason == "" {
				t.Fatalf("helpers %v result %+v", helpers, got.Correlation)
			}
		})
	}
}

func TestCrashPagerEnvAndCancellation(t *testing.T) {
	t.Run("pager", func(t *testing.T) {
		opts := qualifyingCrashOptions()
		id := crashContainerID()
		opts.Runner = func(ctx context.Context, cmd CrashCommand) CommandResult {
			if cmd.Name == "journalctl" {
				assertPager(t, ctx, cmd, crashMetaDeadline)
				return CommandResult{}
			}
			if cmd.Name == "docker" && hasExactArg(cmd.Args, "inspect") {
				return crashInspectResult(id, "exited", intPtr(1), "2024-01-02T03:00:00Z", "2024-01-02T03:10:00Z", "")
			}
			return CommandResult{}
		}
		if CollectCrash(opts).Correlation.Skipped {
			t.Fatal("qualifying query skipped")
		}
	})
	t.Run("cancel", func(t *testing.T) {
		parent, cancel := context.WithCancel(context.Background())
		defer cancel()
		opts := qualifyingCrashOptions()
		opts.Context = parent
		id := crashContainerID()
		started := make(chan struct{})
		opts.Runner = func(ctx context.Context, cmd CrashCommand) CommandResult {
			if cmd.Name == "docker" && hasExactArg(cmd.Args, "inspect") {
				return crashInspectResult(id, "exited", intPtr(1), "2024-01-02T03:00:00Z", "2024-01-02T03:10:00Z", "")
			}
			if cmd.Name == "docker" {
				return CommandResult{}
			}
			select {
			case <-started:
			default:
				close(started)
			}
			cancel()
			<-ctx.Done()
			return CommandResult{Reason: ctx.Err().Error()}
		}
		done := make(chan CrashResult, 1)
		go func() { done <- CollectCrash(opts) }()
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatal("helper did not start")
		}
		select {
		case got := <-done:
			if !strings.Contains(got.Correlation.Reason, "cancel") {
				t.Fatalf("reason %s", got.Correlation.Reason)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("cancellation did not stop crash collection")
		}
	})
}

func TestCrashParsers(t *testing.T) {
	id := crashContainerID()
	ts := time.Date(2024, 1, 2, 3, 1, 0, 0, time.UTC)
	line := journalLine(id, ts, 9, 4, `,"COREDUMP_ENVIRON":"`+crashFixtureSecret+`","unexpected":"extra"`)
	parsed, err := parseJournalExport([]byte(line))
	if err != nil || parsed[0].record.PID != 9 || parsed[0].record.Signal != 4 || !strings.Contains(parsed[0].record.Cgroup, id) {
		t.Fatalf("journal: %v %+v", err, parsed)
	}
	if strings.Contains(fmt.Sprintf("%+v", parsed[0].record), crashFixtureSecret) {
		t.Fatal("extra key retained")
	}
	index := []byte(`[{"time":1700000000000000,"pid":9,"sig":4,"exe":"/usr/bin/sui","corefile":"present","size":8,"COREDUMP_ENVIRON":"` + crashFixtureSecret + `"}]`)
	entries, err := parseCoredumpIndex(index)
	if err != nil || entries[0].PID != 9 || !entries[0].CorePresent {
		t.Fatalf("index: %v %+v", err, entries)
	}
	if _, err = parseJournalExport([]byte("{")); err == nil {
		t.Fatal("malformed journal accepted")
	}
	if _, err = parseCoredumpIndex([]byte("not-json")); err == nil {
		t.Fatal("malformed index accepted")
	}
	if got, err := parseJournalExport(nil); err != nil || len(got) != 0 {
		t.Fatalf("empty journal: %v %d", err, len(got))
	}
}

func TestCrashCorrelationRules(t *testing.T) {
	id := crashContainerID()
	start := time.Date(2024, 1, 2, 3, 0, 0, 0, time.UTC)
	finish := start.Add(10 * time.Minute)
	window, ok := buildWindow(CrashHints{StartedAt: start, FinishedAt: finish}, finish.Add(time.Hour))
	if !ok || !window.Until.Equal(finish.Add(crashFinishGrace)) || !window.Since.Equal(start.Add(-crashStartSlack)) {
		t.Fatalf("window %+v", window)
	}
	inside := start.Add(time.Minute)
	records := []parsedCrash{
		{record: CrashRecord{Timestamp: inside, PID: 1, Executable: "/usr/bin/sui", Cgroup: "/docker/" + id[:12]}, related: true},
		{record: CrashRecord{Timestamp: inside.Add(time.Second), PID: 2, Executable: "/usr/bin/sui", Cgroup: "/other"}, related: true},
		{record: CrashRecord{Timestamp: inside.Add(2 * time.Second), PID: 3, Executable: "/usr/bin/sui", Cgroup: "/docker/" + id}, related: true},
		{record: CrashRecord{Timestamp: inside.Add(3 * time.Second), PID: 4, Executable: "/usr/bin/sui", Cgroup: "/docker/" + id}, related: true},
		{record: CrashRecord{Timestamp: finish.Add(31 * time.Second), PID: 5, Executable: "/usr/bin/sui", Cgroup: "/docker/" + id}, related: true},
		{record: CrashRecord{Timestamp: start.Add(-3 * time.Second), PID: 6, Executable: "/usr/bin/sui", Cgroup: "/docker/" + id}, related: true},
	}
	attributed, unscoped := classifyCrashes(records, id, window)
	if len(unscoped) != 2 || len(attributed) != 2 {
		t.Fatalf("attributed %d unscoped %d", len(attributed), len(unscoped))
	}
	if newestCrash(attributed).PID != 4 {
		t.Fatalf("newest %+v", newestCrash(attributed))
	}
	if _, ok := buildWindow(CrashHints{}, time.Now()); ok {
		t.Fatal("missing start searched")
	}
	longStart := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	longFinish := longStart.Add(10 * time.Hour)
	capped, ok := buildWindow(CrashHints{StartedAt: longStart, FinishedAt: longFinish}, longFinish)
	if !ok || capped.Until.Sub(capped.Since) != crashWindowCap || !capped.Until.Equal(longFinish.Add(crashFinishGrace)) {
		t.Fatalf("cap %+v", capped)
	}
	runningUntil := time.Date(2024, 1, 2, 4, 0, 0, 0, time.UTC)
	running, ok := buildWindow(CrashHints{StartedAt: start, FinishedAt: finish, Running: true}, runningUntil)
	if !ok || !running.Until.Equal(runningUntil) {
		t.Fatalf("running window %+v", running)
	}
}

func TestCrashUnsupportedFieldSelectionDoesNotFallBack(t *testing.T) {
	id := crashContainerID()
	opts := qualifyingCrashOptions()
	var commands []CrashCommand
	opts.Runner = func(_ context.Context, cmd CrashCommand) CommandResult {
		commands = append(commands, cmd)
		return unsupportedJournal(id, cmd)
	}
	got := CollectCrash(opts)
	if !got.Correlation.Skipped || !strings.Contains(got.Correlation.Reason, "unrecognized") {
		t.Fatalf("reason %s", got.Correlation.Reason)
	}
	assertNoCrashFallback(t, commands)
}

func TestCrashJournalWindowArguments(t *testing.T) {
	id := crashContainerID()
	opts := qualifyingCrashOptions()
	opts.Now = time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)
	opts.Runner = func(_ context.Context, cmd CrashCommand) CommandResult {
		if cmd.Name == "journalctl" {
			assertJournalWindow(t, strings.Join(cmd.Args, " "))
			return CommandResult{}
		}
		if hasExactArg(cmd.Args, "inspect") {
			return crashInspectResult(id, "exited", intPtr(1), "2024-01-02T00:00:00Z", "2024-01-02T10:00:00Z", "")
		}
		return CommandResult{}
	}
	_ = CollectCrash(opts)
}

func TestCrashMissingStartSkipsJournal(t *testing.T) {
	id := crashContainerID()
	opts := qualifyingCrashOptions()
	opts.Runner = func(_ context.Context, cmd CrashCommand) CommandResult {
		if cmd.Name == "journalctl" || cmd.Name == "coredumpctl" {
			t.Fatal("searched without start time")
		}
		if hasExactArg(cmd.Args, "inspect") {
			return crashInspectResult(id, "exited", intPtr(1), "", "2024-01-02T10:00:00Z", "")
		}
		return CommandResult{}
	}
	got := CollectCrash(opts)
	if !strings.Contains(got.Correlation.Reason, "start time") {
		t.Fatalf("reason %s", got.Correlation.Reason)
	}
}

func TestCrashUnwindSkipsWithoutGDB(t *testing.T) {
	id := crashContainerID()
	newer := time.Date(2024, 1, 2, 3, 5, 0, 0, time.UTC)
	journal := journalLine(id, newer.Add(-time.Minute), 10, 11, "") + "\n" + journalLine(id, newer, 20, 4, "")
	var debugs []CrashCommand
	var lookups []string
	opts := qualifyingCrashOptions()
	opts.LookPath = gdbMissing(&lookups)
	opts.Runner = crashRunner(id, "2024-01-02T03:00:00Z", journal, []byte(`[{"pid":10,"corefile":"present"},{"pid":20,"corefile":"present"}]`), &debugs, "")
	got := CollectCrash(opts)
	text := renderCrash(got)
	if got.Correlation.AttributedCount != 2 || got.Correlation.Selected.PID != 20 || !strings.Contains(text, "/usr/bin/sui") {
		t.Fatalf("metadata not retained: %+v", got.Correlation)
	}
	if len(debugs) != 0 || !strings.Contains(text, "gdb unavailable") || !strings.Contains(strings.Join(lookups, ","), "gdb") {
		t.Fatalf("gdb gate debugs=%d text=%s", len(debugs), text)
	}
}

func TestCrashUnwindSkipsMissingCore(t *testing.T) {
	id := crashContainerID()
	newer := time.Date(2024, 1, 2, 3, 5, 0, 0, time.UTC)
	var debugs []CrashCommand
	opts := qualifyingCrashOptions()
	opts.Runner = crashRunner(id, "2024-01-02T03:00:00Z", journalLine(id, newer, 20, 4, ""), []byte(`[{"pid":20,"corefile":"missing"}]`), &debugs, "")
	got := CollectCrash(opts)
	if len(debugs) != 0 || got.Unwind.Reason != "core file not present" || got.Correlation.Selected == nil {
		t.Fatalf("missing core: %+v debugs %d", got.Unwind, len(debugs))
	}
}

func TestCrashUnwindArgumentsAndCapture(t *testing.T) {
	id := crashContainerID()
	newer := time.Date(2024, 1, 2, 3, 5, 0, 0, time.UTC)
	var debugs []CrashCommand
	opts := qualifyingCrashOptions()
	opts.Runner = debugCaptureRunner(t, id, newer, &debugs)
	got := CollectCrash(opts)
	if len(debugs) != 1 || got.Correlation.Selected.PID != 20 {
		t.Fatalf("debug invocations %d", len(debugs))
	}
	assertSafeDebugArgs(t, debugs[0].Args)
	if len(got.Unwind.Lines) != crashUnwindLines || !got.Unwind.Truncated || !strings.Contains(renderCrash(got), "8 KiB") {
		t.Fatalf("unwind bound: %+v", got.Unwind)
	}
	if strings.Contains(renderCrash(got), "\x1b") {
		t.Fatal("terminal escape rendered")
	}
}

func TestCrashBudgets(t *testing.T) {
	parent := context.Background()
	ctx, cancel := crashCollectionContext(parent)
	defer cancel()
	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) < crashBudget-time.Second || time.Until(deadline) > crashBudget+time.Second {
		t.Fatalf("budget deadline %s", time.Until(deadline))
	}
	if crashBudget != 20*time.Second || crashMetaDeadline != 5*time.Second || crashDebugDeadline != 8*time.Second {
		t.Fatal("budget constants drifted")
	}
}

func TestRunCrashCommandRejectsAndCancels(t *testing.T) {
	if runCrashCommand(context.Background(), CrashCommand{Name: "ssh", Args: []string{"podman", "machine"}}).Reason == "" {
		t.Fatal("ssh helper allowlisted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	started := time.Now()
	got := runCrashCommand(ctx, crashMetaCommand("coredumpctl", "list"))
	if time.Since(started) > 2*time.Second {
		t.Fatal("cancelled helper was not bounded")
	}
	if got.Reason == "" {
		t.Fatalf("cancelled helper succeeded: %+v", got)
	}
	if !contains(crashPagerEnv(), "SYSTEMD_PAGER=cat") || !contains(crashPagerEnv(), "PAGER=cat") {
		t.Fatal("pager environment missing")
	}
}

func renderCrash(r CrashResult) string {
	var b strings.Builder
	RenderCrash(&b, r)
	return b.String()
}

func resultBlob(r CrashResult) string {
	raw, err := json.Marshal(r)
	if err != nil {
		return err.Error()
	}
	return string(raw) + fmt.Sprintf("%+v", r)
}

func qualifyingCrashOptions() CrashOptions {
	return CrashOptions{
		GOOS:     "linux",
		Engine:   "docker",
		Boundary: Evidence{Source: "test", Value: "local engine kernel"},
		Now:      time.Date(2024, 1, 2, 4, 0, 0, 0, time.UTC),
		LookPath: func(name string) (string, error) { return "/usr/bin/" + name, nil },
	}
}

func crashContainerID() string { return strings.Repeat("ab", 32) }

func crashInspectResult(id, status string, exit *int, started, finished, image string) CommandResult {
	payload := map[string]any{"Id": id, "Status": status, "StartedAt": started, "FinishedAt": finished, "Image": image, "ID": image}
	if exit != nil {
		payload["ExitCode"] = *exit
	}
	raw, _ := json.Marshal(payload)
	return CommandResult{Stdout: raw}
}

func journalLine(id string, ts time.Time, pid, sig int, extra string) string {
	return fmt.Sprintf(`{"COREDUMP_TIMESTAMP":"%d","COREDUMP_PID":"%d","COREDUMP_SIGNAL":"%d","COREDUMP_SIGNAL_NAME":"SIGILL","COREDUMP_EXE":"/usr/bin/sui","COREDUMP_COMM":"sui","COREDUMP_CGROUP":"/docker/%s","COREDUMP_CONTAINER_CMDLINE":"sui start"%s}`, ts.UnixMicro(), pid, sig, id, extra)
}

func crashRunner(id, started, journal string, index []byte, debugs *[]CrashCommand, debugOut string) CrashRunner {
	return func(_ context.Context, cmd CrashCommand) CommandResult {
		switch cmd.Name {
		case "docker":
			if hasExactArg(cmd.Args, "inspect") {
				return crashInspectResult(id, "exited", intPtr(1), started, "2024-01-02T03:10:00Z", "")
			}
			return CommandResult{}
		case "journalctl":
			return CommandResult{Stdout: []byte(journal)}
		case "coredumpctl":
			if hasExactArg(cmd.Args, "debug") {
				if debugs != nil {
					*debugs = append(*debugs, cmd)
				}
				return CommandResult{Stdout: []byte(debugOut)}
			}
			return CommandResult{Stdout: index}
		default:
			return CommandResult{Reason: "unexpected helper"}
		}
	}
}

func assertPager(t *testing.T, ctx context.Context, cmd CrashCommand, deadline time.Duration) {
	t.Helper()
	if !contains(cmd.Env, "SYSTEMD_PAGER=cat") || !hasExactArg(cmd.Args, "--no-pager") {
		t.Fatalf("pager not disabled: env %v args %v", cmd.Env, cmd.Args)
	}
	if cmd.Budget != crashBudget || cmd.Deadline != deadline {
		t.Fatalf("budget %s deadline %s", cmd.Budget, cmd.Deadline)
	}
	remaining, ok := ctx.Deadline()
	if !ok || time.Until(remaining) > deadline+time.Second || time.Until(remaining) < deadline-2*time.Second {
		t.Fatalf("context deadline %s want %s", time.Until(remaining), deadline)
	}
}

func hasExactArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

func hasPrefixArg(args []string, prefix string) bool {
	for _, arg := range args {
		if strings.HasPrefix(arg, prefix) {
			return true
		}
	}
	return false
}

func argAfter(args []string, flag string) string {
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func intPtr(v int) *int { return &v }

func crashSecretFixtures(t *testing.T) ([]parsedCrash, []byte, string) {
	t.Helper()
	id := crashContainerID()
	secret := crashFixtureSecret
	extra := fmt.Sprintf(`,"COREDUMP_ENVIRON":"password=supersecret %s","COREDUMP_PROC_MAPS":"%s","COREDUMP_OPEN_FDS":"%s","COREDUMP_FILENAME":"/var/lib/systemd/coredump/core.%s"`, secret, secret, secret, secret)
	line := journalLine(id, time.Date(2024, 1, 2, 3, 1, 0, 0, time.UTC), 42, 4, extra)
	parsed, err := parseJournalExport([]byte(line))
	if err != nil || len(parsed) != 1 {
		t.Fatalf("parse: %v %+v", err, parsed)
	}
	index := []byte(fmt.Sprintf(`[{"pid":42,"corefile":"present","size":3,"COREDUMP_ENVIRON":"%s","COREDUMP_PROC_MAPS":"%s"}]`, secret, secret))
	entries, err := parseCoredumpIndex(index)
	if err != nil || len(entries) != 1 || !entries[0].CorePresent {
		t.Fatalf("index: %v %+v", err, entries)
	}
	return parsed, index, line
}

func assertNoCrashSecret(t *testing.T, blob string) {
	t.Helper()
	for _, bad := range []string{crashFixtureSecret, "supersecret", "/var/lib/systemd/coredump"} {
		if strings.Contains(blob, bad) {
			t.Fatalf("secret retained %s in %s", bad, blob)
		}
	}
}

func secretCrashRunner(line string, index []byte) CrashRunner {
	id := crashContainerID()
	return func(_ context.Context, cmd CrashCommand) CommandResult {
		switch cmd.Name {
		case "docker":
			return crashInspectResult(id, "exited", intPtr(1), "2024-01-02T03:00:00Z", "2024-01-02T03:10:00Z", "")
		case "journalctl":
			return CommandResult{Stdout: []byte(line)}
		case "coredumpctl":
			if hasExactArg(cmd.Args, "info") || hasExactArg(cmd.Args, "dump") {
				return CommandResult{Reason: "unexpected fallback"}
			}
			return CommandResult{Stdout: index}
		default:
			return CommandResult{Reason: "unexpected helper"}
		}
	}
}

func exitLogRunner(t *testing.T, id, image string, formats *[]string) CrashRunner {
	return func(_ context.Context, cmd CrashCommand) CommandResult {
		if cmd.Name != "docker" {
			t.Fatalf("helper ran: %s %v", cmd.Name, cmd.Args)
		}
		if hasExactArg(cmd.Args, "logs") {
			return CommandResult{Stdout: []byte("ready\nIllegal Instruction (core dumped)\nalso illegal instruction\n")}
		}
		*formats = append(*formats, argAfter(cmd.Args, "--format"))
		return crashInspectResult(id, "exited", intPtr(132), "2024-01-02T03:00:00Z", "2024-01-02T03:10:00Z", image)
	}
}

func assertExitLogHint(t *testing.T, got CrashResult, id string, formats []string) {
	t.Helper()
	text := renderCrash(got)
	if got.Hints.ContainerID != id || strings.Contains(got.Hints.ContainerID, "sha256") {
		t.Fatalf("container id: %s", got.Hints.ContainerID)
	}
	if !got.Hints.SIGILLCompatible || !strings.Contains(text, "compatible with SIGILL") || strings.Contains(text, "child was the process that exited") {
		t.Fatalf("hint: %s", text)
	}
	if strings.Count(text, "Illegal Instruction") != 1 || !strings.Contains(text, "container log text") || !strings.Contains(text, crashBoundaryNote) || got.Kind() != "skipped" {
		t.Fatalf("excerpt: %s", text)
	}
	if len(formats) == 0 || strings.Contains(formats[0], ".Image") || formats[0] == containerFields || !strings.Contains(formats[0], "{{json .Id}}") {
		t.Fatalf("template: %v", formats)
	}
}

func unsupportedJournal(id string, cmd CrashCommand) CommandResult {
	switch {
	case cmd.Name == "docker" && hasExactArg(cmd.Args, "inspect"):
		return crashInspectResult(id, "exited", intPtr(1), "2024-01-02T00:00:00Z", "2024-01-02T10:00:00Z", "")
	case cmd.Name == "docker":
		return CommandResult{}
	case cmd.Name == "journalctl" && hasPrefixArg(cmd.Args, "--output-fields="):
		return CommandResult{ExitCode: 1, Reason: "unrecognized option '--output-fields'"}
	default:
		return CommandResult{Reason: "unexpected fallback"}
	}
}

func assertNoCrashFallback(t *testing.T, commands []CrashCommand) {
	t.Helper()
	for _, cmd := range commands {
		if cmd.Name == "coredumpctl" || hasExactArg(cmd.Args, "info") || hasExactArg(cmd.Args, "dump") || (cmd.Name == "journalctl" && !hasPrefixArg(cmd.Args, "--output-fields=")) {
			t.Fatalf("fallback: %s %v", cmd.Name, cmd.Args)
		}
	}
}

func assertJournalWindow(t *testing.T, joined string) {
	t.Helper()
	for _, want := range []string{"MESSAGE_ID=" + crashMessageID, "--since=2024-01-02T04:00:30Z", "--until=2024-01-02T10:00:30Z"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("window args missing %s in %s", want, joined)
		}
	}
	if strings.Contains(joined, "COREDUMP_ENVIRON") {
		t.Fatal("environ field requested")
	}
}

func gdbMissing(lookups *[]string) func(string) (string, error) {
	return func(name string) (string, error) {
		*lookups = append(*lookups, name)
		if name == "gdb" {
			return "", errors.New("missing")
		}
		return "/usr/bin/" + name, nil
	}
}

func debugCaptureRunner(t *testing.T, id string, newer time.Time, debugs *[]CrashCommand) CrashRunner {
	base := crashRunner(id, "2024-01-02T03:00:00Z", journalLine(id, newer, 20, 4, ""), []byte(`[{"pid":20,"corefile":"present"}]`), nil, "")
	return func(ctx context.Context, cmd CrashCommand) CommandResult {
		if !hasExactArg(cmd.Args, "debug") {
			return base(ctx, cmd)
		}
		*debugs = append(*debugs, cmd)
		assertPager(t, ctx, cmd, crashDebugDeadline)
		if cmd.CaptureLimit != crashUnwindCapture {
			t.Fatalf("capture %d", cmd.CaptureLimit)
		}
		return CommandResult{Stdout: []byte(debugOverflow())}
	}
}

func debugOverflow() string {
	var b strings.Builder
	for i := 0; i < 30; i++ {
		fmt.Fprintf(&b, "frame %d\n", i)
	}
	b.WriteString(strings.Repeat("A", crashUnwindCapture+10))
	b.WriteString("\x1b[31msecret-looking\n")
	return b.String()
}

func assertSafeDebugArgs(t *testing.T, args []string) {
	t.Helper()
	joined := strings.Join(args, " ")
	for _, bad := range []string{"--output", "info locals", "bt full", "show environment", " dump", "/tmp", "workspace"} {
		if strings.Contains(joined, bad) {
			t.Fatalf("unsafe debugger args %s", joined)
		}
	}
	for _, want := range []string{"COREDUMP_PID=20", "-batch", "x/i", "bt 8"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("args missing %s in %s", want, joined)
		}
	}
}
