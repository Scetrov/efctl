package doctor

import (
	"context"
	"strings"
)

const crashDebuggerArguments = "-batch -ex 'set pagination off' -ex 'x/i $pc' -ex 'bt 8'"

func unwindOne(ctx context.Context, opts CrashOptions, rec CrashRecord, presenceReason string) CrashUnwind {
	if presenceReason != "" {
		return CrashUnwind{Reason: presenceReason}
	}
	if !rec.CoreKnown || !rec.CorePresent {
		return CrashUnwind{Reason: "core file not present"}
	}
	if _, err := opts.lookPath("gdb"); err != nil {
		return CrashUnwind{Reason: "gdb unavailable"}
	}
	cmd, err := debugCommand(rec.PID)
	if err != nil {
		return CrashUnwind{Reason: "invalid debugger arguments"}
	}
	res := opts.run(ctx, cmd)
	return unwindFromOutput(res)
}

func debugCommand(pid int) (CrashCommand, error) {
	arg, err := validatedPIDArg(pid)
	if err != nil {
		return CrashCommand{}, err
	}
	if unsafeDebuggerArguments(crashDebuggerArguments) {
		return CrashCommand{}, errMalformedCrash
	}
	cmd := crashMetaCommand("coredumpctl",
		"--no-pager",
		"debug",
		arg,
		"--debugger=gdb",
		"--debugger-arguments="+crashDebuggerArguments,
	)
	cmd.Deadline = crashDebugDeadline
	cmd.CaptureLimit = crashUnwindCapture
	return cmd, nil
}

func unsafeDebuggerArguments(args string) bool {
	lower := strings.ToLower(args)
	for _, bad := range []string{"--output", "info locals", "bt full", "show environment", " dump"} {
		if strings.Contains(lower, bad) {
			return true
		}
	}
	return !strings.Contains(args, "-batch") || !strings.Contains(args, "x/i") || !strings.Contains(args, "bt 8")
}

func unwindFromOutput(res CommandResult) CrashUnwind {
	if res.Reason != "" && len(bytesTrim(res.Stdout)) == 0 {
		return CrashUnwind{Attempted: true, Reason: crashFailureReason(res)}
	}
	lines, truncated, overflow := limitUnwind(string(res.Stdout))
	uw := CrashUnwind{Attempted: true, Available: len(lines) > 0, Lines: lines, Truncated: truncated || overflow}
	switch {
	case overflow:
		uw.Reason = "output exceeded 8 KiB capture limit"
	case truncated:
		uw.Reason = "unwind truncated to 20 lines"
	case res.Reason != "" && !uw.Available:
		uw.Reason = crashFailureReason(res)
	}
	return uw
}

func limitUnwind(text string) (lines []string, truncated, overflow bool) {
	if len(text) > crashUnwindCapture {
		text = text[:crashUnwindCapture]
		overflow = true
	}
	raw := strings.Split(text, "\n")
	for _, line := range raw {
		cleaned := sanitizeCrashLine(line)
		if cleaned == "" || cleaned == "[redacted sensitive diagnostic]" {
			if strings.TrimSpace(line) != "" && cleaned == "[redacted sensitive diagnostic]" {
				lines = append(lines, cleaned)
			}
			if len(lines) == crashUnwindLines {
				break
			}
			continue
		}
		lines = append(lines, cleaned)
		if len(lines) == crashUnwindLines {
			break
		}
	}
	truncated = unwindHadMore(raw, lines)
	return lines, truncated, overflow
}

func unwindHadMore(raw []string, kept []string) bool {
	nonEmpty := 0
	for _, line := range raw {
		if strings.TrimSpace(line) != "" {
			nonEmpty++
		}
	}
	return nonEmpty > len(kept)
}

func bytesTrim(b []byte) string {
	return strings.TrimSpace(string(b))
}

func sanitizeCrashLine(line string) string {
	if strings.Contains(line, "/var/lib/systemd/coredump") {
		return ""
	}
	lower := strings.ToLower(line)
	for _, denied := range []string{"coredump_environ", "coredump_proc_maps", "coredump_open_fds", "coredump_proc_limits", "coredump_proc_auxv", "coredump_filename"} {
		if strings.Contains(lower, denied) {
			return ""
		}
	}
	return SanitizeDiagnostic(line)
}
