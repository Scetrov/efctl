package doctor

import (
	"os/exec"
	"strings"
)

func (o CrashOptions) goos() string {
	if o.GOOS != "" {
		return o.GOOS
	}
	return o.runtimeGOOS
}

func (o CrashOptions) lookPath(name string) (string, error) {
	if o.LookPath != nil {
		return o.LookPath(name)
	}
	return exec.LookPath(name)
}

func allowCrashQuery(opts CrashOptions) (string, bool) {
	if opts.goos() != "linux" {
		return "crash helpers require the local Linux kernel running efctl", false
	}
	if reason, ok := establishedLocalBoundary(opts.Boundary); !ok {
		return reason, false
	}
	if _, err := opts.lookPath("coredumpctl"); err != nil {
		return "coredumpctl unavailable on PATH", false
	}
	return "", true
}

func establishedLocalBoundary(boundary Evidence) (string, bool) {
	text := strings.ToLower(boundary.Reason + " " + boundary.Value)
	if boundary.Reason != "" || strings.TrimSpace(boundary.Value) == "" || strings.Contains(text, "not established") {
		return "remote/VM boundary not established", false
	}
	if strings.Contains(strings.ToLower(boundary.Value), "docker desktop") {
		return "container kernel is not the local journal (Docker Desktop)", false
	}
	if strings.Contains(strings.ToLower(boundary.Value), "remote") {
		return "container kernel is not the local journal (engine reports a remote service)", false
	}
	return "", true
}
