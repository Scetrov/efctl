package doctor

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"efctl/pkg/env"
	"testing"
)

func TestLinuxCPU(t *testing.T) {
	for _, tc := range []struct {
		name, input, flags, reason string
		heterogeneous              bool
	}{
		{"x86", "processor : 0\nmodel name : CPU\nflags : avx sse avx\n\nprocessor : 1\nmodel name : CPU\nflags : sse avx\n", "avx sse", "", false},
		{"heterogeneous", "processor : 0\nflags : avx sse\n\nprocessor : 1\nflags : sse\n", "sse", "", true},
		{"incomplete", "processor : 0\nflags : avx\n\nprocessor : 1\n", "", "incomplete", false},
		{"arm", "processor : 0\nmodel name : ARM\nFeatures : neon aes\n", "aes neon", "", false},
		{"empty", "", "", "unavailable", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := parseLinuxCPU(tc.input, "fixture")
			if got.Flags.Value != tc.flags || !strings.Contains(got.Flags.Reason, tc.reason) || got.Heterogeneous != tc.heterogeneous {
				t.Fatalf("got %+v", got)
			}
		})
	}
}

func TestArchitectureAliases(t *testing.T) {
	for _, tc := range [][2]string{{"x86_64", "amd64"}, {"aarch64", "arm64"}, {"AMD64", "amd64"}, {"armv7l", "arm"}, {"unknown", "unknown"}} {
		if got := NormalizeArchitecture(tc[0]); got != tc[1] {
			t.Fatalf("%s: %s", tc[0], got)
		}
	}
}

func TestPlatformCPUFixtures(t *testing.T) {
	for _, platform := range []string{"darwin", "windows", "unsupported"} {
		p := probes{ctx: context.Background(), run: func(ctx context.Context, name string, args ...string) CommandResult {
			switch name {
			case "uname":
				return CommandResult{Stdout: []byte("aarch64")}
			case "sysctl":
				return CommandResult{Stdout: []byte("aes neon aes")}
			case "powershell.exe":
				return CommandResult{Stdout: []byte(`{"Model":"Windows CPU","Architecture":"AMD64"}`)}
			}
			return CommandResult{Reason: "source unavailable"}
		}}
		got := gatherLocalCPU(p, platform)
		if platform == "darwin" && got.Flags.Value != "aes neon" {
			t.Fatalf("macOS: %+v", got)
		}
		if platform == "windows" && (got.Architecture.Value != "AMD64" || got.Flags.Reason == "") {
			t.Fatalf("Windows: %+v", got)
		}
		if platform == "unsupported" && got.Model.Reason == "" {
			t.Fatal("unsupported source guessed")
		}
	}
}

func TestUnconfiguredHostVersionDoesNotInitializeClient(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix executable fixture; Windows covered by injected runner")
	}
	home := t.TempDir()
	bin := t.TempDir()
	calls := filepath.Join(home, "calls")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("EFCTL_SUI_CALLS", calls)
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$EFCTL_SUI_CALLS"
if [ "$#" -eq 1 ] && [ "$1" = "--version" ]; then echo 'sui 1.0'; exit 0; fi
exit 99
`
	if err := os.WriteFile(filepath.Join(bin, "sui"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	got := gatherDiagnostics(probes{ctx: context.Background(), run: runDiagnostic}, &env.CheckResult{})
	if got.HostSui.Value != "sui 1.0" {
		t.Fatalf("host version: %+v", got.HostSui)
	}
	client := gatherSuiClient()
	if client.ActiveEnv != "not configured" {
		t.Fatalf("client guard: %+v", client)
	}
	data, err := os.ReadFile(calls)
	if err != nil || string(data) != "--version\n" {
		t.Fatalf("unexpected calls: %q %v", data, err)
	}
	entries, err := os.ReadDir(home)
	if err != nil || len(entries) != 1 || entries[0].Name() != "calls" {
		t.Fatalf("client generated data: %v %v", entries, err)
	}
}

func TestHostVersionOnly(t *testing.T) {
	calls := 0
	p := probes{ctx: context.Background(), run: func(ctx context.Context, name string, args ...string) CommandResult {
		calls++
		if name != "sui" || strings.Join(args, " ") != "--version" {
			t.Fatalf("unsafe host call: %s %v", name, args)
		}
		return CommandResult{Stdout: []byte("sui 1.0")}
	}}
	got := p.evidence("host", "sui", "--version")
	if got.Value != "sui 1.0" || calls != 1 {
		t.Fatalf("got %+v", got)
	}
	for _, reason := range []string{"executable not found", "exit status 1"} {
		p.run = func(context.Context, string, ...string) CommandResult { return CommandResult{Reason: reason} }
		if got := p.evidence("host", "sui", "--version"); got.Reason != reason {
			t.Fatalf("got %+v", got)
		}
	}
}
