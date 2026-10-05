package doctor

import (
	"io"
	"strconv"
	"strings"
	"time"
)

const (
	crashBudget        = 20 * time.Second
	crashMetaDeadline  = 5 * time.Second
	crashDebugDeadline = 8 * time.Second
	crashUnwindCapture = 8 * 1024
	crashUnwindLines   = 20
	crashLogTailLines  = 200
	crashMessageID     = "fc2e22bc6ee647b6b90729ab34a250b1"

	crashBoundaryNote = "container exit code and log text are shell/container evidence, not the child sui start status, and do not by themselves attribute a journal crash"
	crashReviewNote   = "review before sharing; do not attach core files, crash environment, or maps"
)

// CrashHints are container/shell observations. They never attribute a journal record.
type CrashHints struct {
	ContainerID         string
	IdentityUnavailable bool
	Status              string
	ExitCode            string
	SIGILLCompatible    bool
	LogExcerpt          string
	LogUnavailable      string
	BoundaryNote        string
	StartedAt           time.Time
	FinishedAt          time.Time
	Running             bool
}

// CrashRecord is allowlisted crash metadata. Environment, maps, and core paths are not fields.
type CrashRecord struct {
	Timestamp        time.Time
	PID              int
	Signal           int
	SignalName       string
	Executable       string
	Cgroup           string
	ContainerCmdline string
	CorePresent      bool
	CoreKnown        bool
}

// CrashCorrelation is either skipped, unscoped, attributed, or empty after a search.
type CrashCorrelation struct {
	Skipped         bool
	Reason          string
	AttributedCount int
	UnscopedCount   int
	Selected        *CrashRecord
	Unscoped        []CrashRecord
}

// CrashUnwind is a bounded debugger view. Metadata remains even when this is unavailable.
type CrashUnwind struct {
	Attempted bool
	Available bool
	Reason    string
	Lines     []string
	Truncated bool
}

// CrashResult is the opt-in crash section. Default doctor does not populate it.
type CrashResult struct {
	Hints       CrashHints
	Correlation CrashCorrelation
	Unwind      CrashUnwind
}

// Kind distinguishes hints, a skipped search, unscoped records, and one attributed record.
func (r CrashResult) Kind() string {
	switch {
	case r.Correlation.Selected != nil:
		return "attributed"
	case r.Correlation.UnscopedCount > 0:
		return "unscoped"
	case r.Correlation.Skipped:
		return "skipped"
	default:
		return "hints"
	}
}

// RenderCrash writes the credential-safe crash section body.
func RenderCrash(w io.Writer, r CrashResult) {
	note := r.Hints.BoundaryNote
	if note == "" {
		note = crashBoundaryNote
	}
	crashLine(w, "boundary", note)
	renderCrashHints(w, r.Hints)
	renderCrashCorrelation(w, r.Correlation)
	renderCrashUnwind(w, r.Unwind)
	crashLine(w, "review", crashReviewNote)
}

func renderCrashHints(w io.Writer, h CrashHints) {
	if h.IdentityUnavailable || h.ContainerID == "" {
		crashLine(w, "container id", "unavailable: managed container identity is unavailable")
	} else {
		crashLine(w, "container id", h.ContainerID)
	}
	if h.SIGILLCompatible {
		crashLine(w, "container status hint", "container status compatible with SIGILL at the shell/container boundary; not proof the background sui start child exited")
	} else if h.ExitCode != "" {
		crashLine(w, "recorded container exit", h.ExitCode+" (shell/container boundary, not child sui start status)")
	}
	if h.LogExcerpt != "" {
		crashLine(w, "container log text", h.LogExcerpt)
	} else if h.LogUnavailable != "" {
		crashLine(w, "container log text", "unavailable: "+h.LogUnavailable)
	}
}

func renderCrashCorrelation(w io.Writer, c CrashCorrelation) {
	if c.Skipped {
		crashLine(w, "correlation", "skipped: "+fallback(c.Reason, "unavailable"))
		return
	}
	if c.UnscopedCount > 0 {
		crashLine(w, "unscoped", formatUnscoped(c))
	}
	if c.Selected == nil {
		if c.UnscopedCount == 0 {
			crashLine(w, "correlation", "no attributed crash record")
		}
		return
	}
	crashLine(w, "attributed", formatAttributedCount(c.AttributedCount))
	renderRecord(w, *c.Selected)
}

func renderRecord(w io.Writer, rec CrashRecord) {
	crashLine(w, "crash time", rec.Timestamp.UTC().Format(time.RFC3339))
	crashLine(w, "crash pid", strconv.Itoa(rec.PID))
	crashLine(w, "crash signal", formatSignal(rec.SignalName, rec.Signal))
	crashLine(w, "crash executable", rec.Executable)
	crashLine(w, "crash cgroup", rec.Cgroup)
	if rec.ContainerCmdline != "" {
		crashLine(w, "crash container cmdline", rec.ContainerCmdline)
	}
	crashLine(w, "core file present", corePresenceLabel(rec))
}

func renderCrashUnwind(w io.Writer, u CrashUnwind) {
	if u.Available {
		crashLine(w, "unwind", "instruction view and at most 8 frames")
		for _, line := range u.Lines {
			crashLine(w, "unwind line", line)
		}
		if u.Truncated || u.Reason != "" {
			crashLine(w, "unwind limit", fallback(u.Reason, "truncated"))
		}
		return
	}
	if u.Reason != "" || u.Attempted {
		crashLine(w, "unwind", "unavailable: "+fallback(u.Reason, "debugger unavailable"))
	}
}

func crashLine(w io.Writer, label, value string) {
	value = sanitizeCrashLine(value)
	if value == "" {
		value = "[omitted]"
	}
	_, _ = io.WriteString(w, "  "+label+": "+value+"\n")
}

func formatSignal(name string, num int) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "unspecified"
	}
	if num == 0 {
		return name
	}
	return name + " " + strconv.Itoa(num)
}

func corePresenceLabel(rec CrashRecord) string {
	if !rec.CoreKnown {
		return "unknown"
	}
	if rec.CorePresent {
		return "yes"
	}
	return "no"
}

func formatAttributedCount(n int) string {
	if n < 1 {
		n = 1
	}
	if n == 1 {
		return "1 record attributed to sui-playground"
	}
	return strconv.Itoa(n) + " records attributed to sui-playground; unwind considers only the newest"
}

func formatUnscoped(c CrashCorrelation) string {
	return strconv.Itoa(c.UnscopedCount) + " record(s) not attributed (no full container id in cgroup); not unwound"
}

func fallback(value, empty string) string {
	if strings.TrimSpace(value) == "" {
		return empty
	}
	return value
}
