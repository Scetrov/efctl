package doctor

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"
)

// CrashCommand is one structured helper invocation. Arguments are never shell-interpolated.
type CrashCommand struct {
	Name         string
	Args         []string
	Env          []string
	Deadline     time.Duration
	Budget       time.Duration
	CaptureLimit int
}

// CrashRunner executes a crash helper. Tests inject this; production uses runCrashCommand.
type CrashRunner func(context.Context, CrashCommand) CommandResult

func crashPagerEnv() []string {
	return []string{"SYSTEMD_PAGER=cat", "SYSTEMD_PAGERSECURE=1", "PAGER=cat"}
}

func crashMetaCommand(name string, args ...string) CrashCommand {
	return CrashCommand{
		Name: name, Args: args, Env: crashPagerEnv(),
		Deadline: crashMetaDeadline, Budget: crashBudget, CaptureLimit: captureLimit,
	}
}

func (o CrashOptions) run(ctx context.Context, cmd CrashCommand) CommandResult {
	if cmd.Deadline <= 0 {
		cmd.Deadline = crashMetaDeadline
	}
	if cmd.Budget == 0 {
		cmd.Budget = crashBudget
	}
	cctx, cancel := context.WithTimeout(ctx, cmd.Deadline)
	defer cancel()
	if o.Runner != nil {
		return o.Runner(cctx, cmd)
	}
	return runCrashCommand(cctx, cmd)
}

func allowedCrashCommand(name string) bool {
	switch name {
	case "coredumpctl", "journalctl", "docker", "podman":
		return true
	default:
		return false
	}
}

type limitedCapture struct {
	data     []byte
	limit    int
	overflow bool
}

func (b *limitedCapture) Write(p []byte) (int, error) {
	n := len(p)
	if b.limit <= 0 {
		b.limit = captureLimit
	}
	remaining := b.limit - len(b.data)
	if remaining < 0 {
		remaining = 0
	}
	if n > remaining {
		b.overflow = true
		p = p[:remaining]
	}
	b.data = append(b.data, p...)
	return n, nil
}

func runCrashCommand(ctx context.Context, cmd CrashCommand) CommandResult {
	if !allowedCrashCommand(cmd.Name) {
		return CommandResult{Reason: "crash helper not allowlisted"}
	}
	limit := cmd.CaptureLimit
	if limit <= 0 {
		limit = captureLimit
	}
	var stdout, stderr limitedCapture
	stdout.limit, stderr.limit = limit, limit
	execCmd := exec.CommandContext(ctx, cmd.Name, cmd.Args...) // #nosec G204 -- allowlisted crash helpers and validated structured arguments; never shell-interpolated
	execCmd.Env = append(os.Environ(), crashPagerEnv()...)
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr
	execCmd.WaitDelay = 100 * time.Millisecond
	err := execCmd.Run()
	res := CommandResult{Stdout: stdout.data, Stderr: stderr.data}
	switch {
	case ctx.Err() != nil:
		res.Reason = ctx.Err().Error()
	case stdout.overflow || stderr.overflow:
		res.Reason = "output exceeded capture limit"
	case err != nil:
		res.Reason = crashExecReason(err)
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			res.ExitCode = exit.ExitCode()
		}
	}
	return res
}

func crashExecReason(err error) string {
	var exit *exec.ExitError
	switch {
	case errors.As(err, &exit):
		return SanitizeDiagnostic(exit.ProcessState.String())
	case errors.Is(err, exec.ErrNotFound) || errors.Is(err, os.ErrNotExist):
		return "executable not found"
	default:
		return "command could not execute"
	}
}

func crashFailureReason(res CommandResult) string {
	reason := res.Reason
	if reason == "" && res.ExitCode != 0 {
		reason = "command failed"
	}
	lower := strings.ToLower(reason)
	switch {
	case strings.Contains(lower, "deadline") || strings.Contains(lower, "timeout"):
		return "timeout"
	case strings.Contains(lower, "cancel"):
		return "cancelled"
	default:
		cleaned := SanitizeDiagnostic(reason)
		if cleaned == "" {
			return "crash helper unavailable"
		}
		return cleaned
	}
}

func crashCollectionContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, crashBudget)
}
