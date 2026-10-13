package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"efctl/pkg/doctor"

	"github.com/pterm/pterm"
)

func TestCPUWarningsUseCollectedEvidenceAndWriter(t *testing.T) {
	for _, tc := range []struct {
		name, arch, flags, reason string
		warning                   bool
	}{
		{"x86 missing", "x86_64", "sse", "", true},
		{"x86 compatible", "amd64", "adx bmi2", "", false},
		{"ARM", "aarch64", "fp asimd aes", "", false},
		{"unknown architecture", "", "sse", "", false},
		{"unknown flags", "amd64", "", "incomplete CPU metadata", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := diagnosticFixture()
			d.LocalCPU = doctor.CPUInfo{Architecture: doctor.Evidence{Source: "recorded architecture", Value: tc.arch}, Flags: doctor.Evidence{Source: "recorded flags", Value: tc.flags, Reason: tc.reason}}
			d.CPU = doctor.CPUInfo{}
			var out bytes.Buffer
			renderRuntimeDiagnostics(&out, d)
			got := strings.Contains(out.String(), "local CPU warning:")
			if got != tc.warning {
				t.Fatalf("warning=%v want=%v; report=%s", got, tc.warning, out.String())
			}
			if !tc.warning && strings.Contains(out.String(), "WARNING") {
				t.Fatal("CPU warning badge rendered without a warranted warning")
			}
			if got && (!strings.Contains(out.String(), "Some x86-64 Sui builds") || strings.Contains(out.String(), "Sui release requires")) {
				t.Fatal("warning must be conditional, not a diagnosis")
			}
		})
	}
}

func TestCPUWarningsRetainVisibleBadge(t *testing.T) {
	wasColor := pterm.PrintColor
	t.Cleanup(func() {
		if wasColor {
			pterm.EnableColor()
		} else {
			pterm.DisableColor()
		}
	})
	for _, color := range []bool{false, true} {
		mode := "plain"
		if color {
			mode = "color"
		}
		for _, boundary := range []string{"local", "runtime"} {
			t.Run(mode+"/"+boundary, func(t *testing.T) {
				if color {
					pterm.EnableColor()
				} else {
					pterm.DisableColor()
				}
				d := diagnosticFixture()
				d.LocalCPU.Heterogeneous = false
				d.LocalCPU.Flags.Value = "adx bmi2"
				d.CPU.Flags.Value = "adx bmi2"
				if boundary == "local" {
					d.LocalCPU.Flags.Value = "adx"
				} else {
					d.CPU.Flags.Value = "adx"
				}
				var out bytes.Buffer
				renderRuntimeDiagnostics(&out, d)
				assertCPUWarningBadge(t, out.String(), boundary, color)
			})
		}
	}
}

func assertCPUWarningBadge(t *testing.T, report, boundary string, color bool) {
	t.Helper()
	ansi := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	plain := ansi.ReplaceAllString(report, "")
	if strings.Count(plain, "WARNING") != 1 || strings.Count(plain, boundary+" CPU warning:") != 1 {
		t.Fatalf("expected one visible WARNING badge in supplied writer: %s", report)
	}
	lines := strings.Split(report, "\n")
	for i, line := range lines {
		if !strings.Contains(line, boundary+" CPU warning:") {
			continue
		}
		for _, want := range []string{"WARNING", "BMI2", "Some x86-64 Sui builds"} {
			if !strings.Contains(line, want) {
				t.Fatalf("warning row missing %q: %s", want, line)
			}
		}
		if ansi.MatchString(line) != color {
			t.Fatalf("warning styling does not match color mode %v: %q", color, line)
		}
		if color && !strings.Contains(line, "43m") {
			t.Fatalf("warning badge lost its yellow background: %q", line)
		}
		if i == 0 || i+1 >= len(lines) || lines[i-1] != "" || lines[i+1] != "" {
			t.Fatalf("warning must be separated from adjacent rows: %s", report)
		}
	}
}

func TestCPUWarningsKeepLocalAndRuntimeBoundaries(t *testing.T) {
	d := diagnosticFixture()
	d.LocalCPU.Flags.Value = "adx bmi2"
	d.CPU.Flags.Value = "adx"
	var out bytes.Buffer
	renderRuntimeDiagnostics(&out, d)
	if strings.Contains(out.String(), "local CPU warning:") {
		t.Fatal("compatible local CPU incorrectly warned")
	}
	if !strings.Contains(out.String(), "runtime CPU warning:") || !strings.Contains(out.String(), "BMI2") {
		t.Fatal("runtime evidence warning absent")
	}
}

func TestRenderCPUWarningsDoesNotProbe(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("shell fixture requires Unix")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "probe-ran")
	script := "#!/bin/sh\nprintf touched > \"$CPU_PROBE_MARKER\"\nprintf 'x86_64\\n'\n"
	if err := os.WriteFile(filepath.Join(dir, "uname"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("CPU_PROBE_MARKER", marker)
	var out bytes.Buffer
	renderRuntimeDiagnostics(&out, diagnosticFixture())
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("rendering launched a CPU probe: %v", err)
	}
}
