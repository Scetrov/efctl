package doctor

import (
	"context"
	"testing"

	"efctl/pkg/env"
)

func TestDiagnosticContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got := gatherRuntimeDiagnostics(Options{Context: ctx, Runner: func(context.Context, string, ...string) CommandResult {
		t.Fatal("cancelled diagnostics invoked a command")
		return CommandResult{}
	}}, &env.CheckResult{HasDocker: true})
	if got.HostSui.Source != "host version-only" || got.HostSui.Reason == "" || got.Runtime.Architecture.Reason == "" || got.Sui.Reason == "" {
		t.Fatalf("unavailable evidence not preserved: %+v", got)
	}
}
