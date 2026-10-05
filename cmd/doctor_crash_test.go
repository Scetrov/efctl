package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"efctl/pkg/doctor"
)

func TestDefaultDoctorOmitsCrashSection(t *testing.T) {
	called := false
	prev := collectDoctorCrash
	collectDoctorCrash = func(doctor.CrashOptions) doctor.CrashResult {
		called = true
		return doctor.CrashResult{}
	}
	t.Cleanup(func() {
		collectDoctorCrash = prev
		doctorCrash = false
		_ = doctorCmd.Flags().Set("crash", "false")
		rootCmd.SetArgs(nil)
	})
	output := executeDoctor(t, "doctor", "--workspace", t.TempDir())
	if called {
		t.Fatal("default doctor invoked crash helpers")
	}
	if strings.Contains(output, "crash diagnostics:") {
		t.Fatalf("default report has crash section: %s", output)
	}
	if !strings.Contains(output, "efctl:") {
		t.Fatal("doctor report missing")
	}
}

func TestDoctorCrashFlagIsOptIn(t *testing.T) {
	prev := collectDoctorCrash
	collectDoctorCrash = func(doctor.CrashOptions) doctor.CrashResult {
		return doctor.CrashResult{Correlation: doctor.CrashCorrelation{Skipped: true, Reason: "coredumpctl unavailable on PATH"}}
	}
	t.Cleanup(func() {
		collectDoctorCrash = prev
		doctorCrash = false
		_ = doctorCmd.Flags().Set("crash", "false")
		rootCmd.SetArgs(nil)
	})
	output := executeDoctor(t, "doctor", "--crash", "--workspace", t.TempDir())
	if !strings.Contains(output, "crash diagnostics:") || !strings.Contains(output, "coredumpctl unavailable") {
		t.Fatalf("opt-in section missing: %s", output)
	}
	if !strings.Contains(output, "efctl:") || !strings.Contains(output, "review before sharing") {
		t.Fatal("existing report or review warning missing")
	}
	flag := doctorCmd.Flags().Lookup("crash")
	if flag == nil || flag.DefValue != "false" {
		t.Fatal("crash flag default is not false")
	}
}

func TestDoctorCrashHelpAndSupportDoc(t *testing.T) {
	for _, want := range []string{"--crash", "20-second", "five seconds", "eight seconds", "Review that section before sharing", "do not attach"} {
		if !strings.Contains(doctorCmd.Long, want) {
			t.Fatalf("help missing %q", want)
		}
	}
	body, err := os.ReadFile("../docs/doctor-support.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"efctl doctor --crash", "20-second", "five seconds", "eight seconds", "Review the section before sharing", "Do not attach core files"} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("support doc missing %q", want)
		}
	}
}

func executeDoctor(t *testing.T, args ...string) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	rootCmd.SetArgs(args)
	execErr := rootCmd.Execute()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	if execErr != nil {
		t.Fatalf("doctor exit: %v", execErr)
	}
	return buf.String()
}
