package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"efctl/pkg/doctor"
)

func diagnosticFixture() doctor.RuntimeDiagnostics {
	return doctor.RuntimeDiagnostics{
		LocalCPU: doctor.CPUInfo{Model: doctor.Evidence{Source: "local view", Value: "CPU"}, Architecture: doctor.Evidence{Source: "local uname", Value: "x86_64"}, Flags: doctor.Evidence{Source: "local /proc/cpuinfo intersection", Value: "avx sse"}, Heterogeneous: true},
		Runtime:  doctor.RuntimeServerInfo{Client: doctor.Evidence{Source: "Docker client", Value: "29"}, Server: doctor.Evidence{Source: "remote server", Value: "28"}, OS: doctor.Evidence{Source: "remote server", Value: "linux"}, Architecture: doctor.Evidence{Source: "remote server", Value: "amd64"}},
		Image:    doctor.ImageInfo{State: doctor.Evidence{Source: "managed container recorded state", Value: "exited"}, ExitCode: doctor.Evidence{Source: "managed container recorded exit code (not Sui process status)", Value: "132"}, ID: doctor.Evidence{Source: "managed container actual image", Value: "sha256:actual"}, Architecture: doctor.Evidence{Source: "managed container actual image", Value: "x86_64"}, Digests: doctor.Evidence{Source: "managed container actual image", Value: "not recorded"}},
		HostSui:  doctor.Evidence{Source: "host version-only", Reason: "executable not found"},
		Sui:      doctor.Evidence{Source: "local-image probe", Reason: "signal: illegal instruction"},
		CPU:      doctor.CPUInfo{Flags: doctor.Evidence{Source: "local-image probe /proc/cpuinfo", Value: "avx sse"}, Architecture: doctor.Evidence{Source: "local-image probe uname", Value: "x86_64"}},
	}
}

func TestRenderRuntimeDiagnostics(t *testing.T) {
	d := diagnosticFixture()
	var out bytes.Buffer
	renderRuntimeDiagnostics(&out, d)
	text := out.String()
	for _, want := range []string{"local view", "remote server", "actual image", "host version-only", "local-image probe", "unavailable: executable not found", "signal: illegal instruction", "not Sui process status", "avx sse", "per-CPU features differ"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in %s", want, text)
		}
	}
	if strings.Contains(text, "possible emulation") {
		t.Fatal("architecture aliases incorrectly differ")
	}
	d.Image.Architecture.Value = "aarch64"
	out.Reset()
	renderRuntimeDiagnostics(&out, d)
	if !strings.Contains(out.String(), "possible emulation; not confirmed") || strings.Contains(out.String(), "caused") {
		t.Fatal("unsupported architecture diagnosis")
	}
	d.Image.ID.Source = "configured local image fallback (not proven startup image)"
	d.Runtime.Architecture.Reason = "server unavailable"
	out.Reset()
	renderRuntimeDiagnostics(&out, d)
	if !strings.Contains(out.String(), "fallback") || strings.Contains(out.String(), "possible emulation") {
		t.Fatal("fallback/unknown platform misrepresented")
	}
}

func TestUnavailableFlagsAreNotEmptyFeatures(t *testing.T) {
	unknown := diagnosticLabel(doctor.Evidence{Source: "CPU source", Reason: "incomplete per-CPU flags"})
	empty := diagnosticLabel(doctor.Evidence{Source: "CPU source"})
	if !strings.Contains(unknown, "unavailable") || !strings.Contains(empty, "none reported") || unknown == empty {
		t.Fatal("missing flags treated as confirmed absence")
	}
}

func TestCompleteDoctorReportDiagnosticSecurity(t *testing.T) {
	d := diagnosticFixture()
	d.HostSui.Value = "mnemonic=words"
	d.HostSui.Reason = ""
	d.Sui.Reason = "password=supersecret\x1b[31m"
	d.Image.Reference = doctor.Evidence{Source: "source\x1b[2J", Value: "https://user:supersecret@example.org"}
	d.Runtime.Server = doctor.Evidence{Source: "server", Value: "metadata\x1b[2J\r"}
	d.CPU.Model = doctor.Evidence{Source: "local-image", Value: "private key: supersecret"}
	report := doctor.Report{Diagnostics: d, Sui: doctor.SuiClientInfo{Found: true, ActiveEnv: "not configured"}}
	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	done := make(chan []byte, 1)
	go func() { data, _ := io.ReadAll(r); done <- data }()
	printDoctorReport(&report)
	_ = w.Close()
	os.Stdout = original
	data := <-done
	_ = r.Close()
	text := string(data)
	for _, unsafe := range []string{"supersecret", "mnemonic=words", "\x1b", "\r"} {
		if strings.Contains(text, unsafe) {
			t.Fatalf("unsafe diagnostic output: %q", unsafe)
		}
	}
	for _, section := range []string{"efctl:", "container runtime:", "env:", "sui active env:", "not configured", "config file:"} {
		if !strings.Contains(text, section) {
			t.Fatalf("existing report section lost: %s", section)
		}
	}
}
