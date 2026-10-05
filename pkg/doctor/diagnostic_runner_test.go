package doctor

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestDiagnosticHelper(t *testing.T) {
	if os.Getenv("EFCTL_DIAGNOSTIC_HELPER") != "1" {
		return
	}
	switch os.Args[len(os.Args)-1] {
	case "success":
		os.Stdout.WriteString("sui 1.0\n")
	case "malformed":
		os.Stdout.WriteString("not JSON")
	case "failure":
		os.Stderr.WriteString("password=secret recovery phrase private key")
		os.Exit(7)
	case "hang":
		time.Sleep(10 * time.Second)
	case "large":
		os.Stdout.WriteString(strings.Repeat("x", captureLimit+1))
		os.Stderr.WriteString(strings.Repeat("y", captureLimit+1))
	}
	os.Exit(0)
}

func TestDiagnosticRunner(t *testing.T) {
	t.Setenv("EFCTL_DIAGNOSTIC_HELPER", "1")
	// Race-instrumented helper processes otherwise sleep for one second at exit,
	// which tests the detector's shutdown delay rather than command deadlines.
	// The parent detector is already initialized; keep detection enabled in both.
	t.Setenv("GORACE", os.Getenv("GORACE")+" atexit_sleep_ms=0")
	for _, tc := range []struct {
		mode   string
		failed bool
	}{{"success", false}, {"malformed", false}, {"failure", true}, {"large", true}, {"hang", true}} {
		t.Run(tc.mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			got := runDiagnostic(ctx, os.Args[0], "-test.run=^TestDiagnosticHelper$", "--", tc.mode)
			if (got.Reason != "") != tc.failed {
				t.Fatalf("unexpected result: %+v", got)
			}
			if len(got.Stdout) > captureLimit || len(got.Stderr) > captureLimit {
				t.Fatal("capture exceeded limit")
			}
			if strings.Contains(got.Reason, "secret") {
				t.Fatal("stderr leaked")
			}
			if tc.mode == "success" && string(got.Stdout) != "sui 1.0\n" {
				t.Fatalf("stdout: %q", got.Stdout)
			}
			if tc.mode == "failure" && got.ExitCode != 7 {
				t.Fatalf("exit: %d", got.ExitCode)
			}
		})
	}
	t.Run("missing", func(t *testing.T) {
		if r := runDiagnostic(context.Background(), "efctl-nonexistent-diagnostic-command"); r.Reason == "" {
			t.Fatal("missing command not reported")
		}
	})
	t.Run("cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if r := runDiagnostic(ctx, os.Args[0]); !strings.Contains(r.Reason, "cancel") {
			t.Fatalf("reason: %s", r.Reason)
		}
	})
}

func TestSanitizeDiagnostic(t *testing.T) {
	for _, raw := range []string{"password=secret", "token:secret", "https://user:secret@example.org", "private key: secret", "recovery phrase: words", "mnemonic=words"} {
		if strings.Contains(SanitizeDiagnostic(raw), "secret") || strings.Contains(SanitizeDiagnostic(raw), "words") {
			t.Fatalf("credential retained: %q", raw)
		}
	}
	got := SanitizeDiagnostic("model\x1b[31m\r\n\x00")
	if strings.ContainsAny(got, "\x1b\r\n\x00") {
		t.Fatalf("unsafe controls: %q", got)
	}
	if len([]rune(SanitizeDiagnostic(strings.Repeat("é", 1000)))) > 512 {
		t.Fatal("excerpt too long")
	}
}

func TestDiagnosticSignal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix signal fixture")
	}
	got := runDiagnostic(context.Background(), "sh", "-c", "ulimit -c 0; kill -ILL $$")
	if got.ExitCode != -1 || !strings.Contains(got.Reason, "illegal instruction") {
		t.Fatalf("signal not observed: %+v", got)
	}
}

func TestProbeCommandDeadline(t *testing.T) {
	p := probes{ctx: context.Background(), run: func(ctx context.Context, _ string, _ ...string) CommandResult {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 5*time.Second {
			t.Fatal("command deadline not bounded")
		}
		return CommandResult{}
	}}
	p.command("fixture")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p.ctx = ctx
	p.run = func(context.Context, string, ...string) CommandResult {
		t.Fatal("cancelled probe executed")
		return CommandResult{}
	}
	if got := p.command("fixture"); got.Reason == "" {
		t.Fatal("cancelled budget ignored")
	}
}

func TestDiagnosticBudget(t *testing.T) {
	ctx, cancel := newProbeContext(context.Background())
	defer cancel()
	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) > 20*time.Second {
		t.Fatal("missing shared budget")
	}
}
