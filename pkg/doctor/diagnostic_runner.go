package doctor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
	"unicode"
)

const captureLimit = 64 * 1024

// CommandResult contains bounded output and an error that never includes stderr.
// Output is untrusted; callers must select fields and sanitize them before reporting.
type CommandResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	Reason   string
}

// DiagnosticRunner is injectable so probes can be tested without a container engine.
type DiagnosticRunner func(context.Context, string, ...string) CommandResult

type boundedCapture struct {
	data     []byte
	overflow bool
}

func (b *boundedCapture) Write(p []byte) (int, error) {
	n := len(p)
	remaining := captureLimit - len(b.data)
	if n > remaining {
		b.overflow = true
		p = p[:remaining]
	}
	b.data = append(b.data, p...)
	return n, nil
}

func newProbeContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, 20*time.Second)
}

func runDiagnostic(parent context.Context, name string, args ...string) CommandResult {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	var stdout, stderr boundedCapture
	cmd := exec.CommandContext(ctx, name, args...) // #nosec G204 -- fixed diagnostic commands and structured arguments; never shell-interpolated
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// Bound waiting for inherited output descriptors as well as the child process.
	cmd.WaitDelay = 100 * time.Millisecond
	err := cmd.Run()
	r := CommandResult{Stdout: stdout.data, Stderr: stderr.data}
	switch {
	case ctx.Err() != nil:
		r.Reason = ctx.Err().Error()
	case stdout.overflow || stderr.overflow:
		r.Reason = "output exceeded 64 KiB capture limit"
	case err != nil:
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			r.ExitCode = exit.ExitCode()
			// ProcessState's description includes the observed signal when available,
			// without exposing executable paths, arguments or subprocess stderr.
			r.Reason = SanitizeDiagnostic(exit.ProcessState.String())
		} else if errors.Is(err, exec.ErrNotFound) || errors.Is(err, os.ErrNotExist) {
			r.Reason = "executable not found"
		} else {
			r.Reason = "command could not execute"
		}
	}
	return r
}

var secretDiagnostic = regexp.MustCompile(`(?i)(password|passwd|token|secret|credential|authorization|private[ _-]?key|api[ _-]?key|basic[ _-]?auth|bearer|recovery[ _-]?phrase|mnemonic|https?://[^\s/]*@)`)

// SanitizeDiagnostic deliberately omits a whole field if it may contain secrets.
// Diagnostic metadata does not need URLs, authentication values or key material.
func SanitizeDiagnostic(s string) string { return sanitizeDiagnosticLimit(s, 512) }

// SanitizeDiagnosticValue retains complete bounded feature sets while making
// metadata safe to print. Only error excerpts use the smaller 512-rune limit.
func SanitizeDiagnosticValue(s string) string { return sanitizeDiagnosticLimit(s, captureLimit) }

func sanitizeDiagnosticLimit(s string, limit int) string {
	if secretDiagnostic.MatchString(s) {
		return "[redacted sensitive diagnostic]"
	}
	var b strings.Builder
	for _, r := range strings.TrimSpace(s) {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			fmt.Fprintf(&b, "\\u%04x", r)
		} else {
			b.WriteRune(r)
		}
	}
	runes := []rune(b.String())
	if len(runes) > limit {
		return string(runes[:limit-3]) + "..."
	}
	return string(runes)
}
