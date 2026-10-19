package doctor

import (
	"context"
	"encoding/json"
	"strings"

	"efctl/pkg/container"
)

const crashContainerFields = `{"Id":{{json .Id}},"Status":{{json .State.Status}},"ExitCode":{{json .State.ExitCode}},"StartedAt":{{json .State.StartedAt}},"FinishedAt":{{json .State.FinishedAt}}}`
const crashContainerPodmanFields = `{"Id":{{json .ID}},"Status":{{json .State.Status}},"ExitCode":{{json .State.ExitCode}},"StartedAt":{{json .State.StartedAt}},"FinishedAt":{{json .State.FinishedAt}}}`

type crashInspect struct {
	ID         string
	Status     string
	ExitCode   *int
	StartedAt  string
	FinishedAt string
}

func collectHints(ctx context.Context, opts CrashOptions) CrashHints {
	hints := CrashHints{BoundaryNote: crashBoundaryNote, IdentityUnavailable: true}
	if opts.Engine != "docker" && opts.Engine != "podman" {
		hints.LogUnavailable = "container engine unavailable"
		return hints
	}
	meta, reason := inspectCrashContainer(ctx, opts)
	if reason != "" || meta.ID == "" {
		hints.LogUnavailable = fallback(reason, "managed container identity is unavailable")
		return hints
	}
	return hintsFromInspect(ctx, opts, meta)
}

func hintsFromInspect(ctx context.Context, opts CrashOptions, meta crashInspect) CrashHints {
	hints := CrashHints{
		ContainerID: meta.ID, Status: SanitizeDiagnostic(meta.Status), BoundaryNote: crashBoundaryNote,
		StartedAt: parseCrashTime(meta.StartedAt), FinishedAt: parseCrashTime(meta.FinishedAt),
		Running: strings.EqualFold(meta.Status, "running"),
	}
	if meta.ExitCode != nil {
		hints.ExitCode = formatExitCode(*meta.ExitCode)
		hints.SIGILLCompatible = *meta.ExitCode == 132
	}
	applyLogHint(ctx, opts, &hints)
	return hints
}

func inspectCrashContainer(ctx context.Context, opts CrashOptions) (crashInspect, string) {
	first, reason := decodeCrashInspect(opts.run(ctx, inspectCommand(opts.Engine, crashContainerFields)))
	if reason == "" && first.ID != "" {
		return first, ""
	}
	second, reason2 := decodeCrashInspect(opts.run(ctx, inspectCommand(opts.Engine, crashContainerPodmanFields)))
	if reason2 == "" && second.ID != "" {
		return second, ""
	}
	if reason == "" {
		reason = reason2
	}
	return crashInspect{}, fallback(reason, "managed container identity is unavailable")
}

func inspectCommand(engine, format string) CrashCommand {
	return crashMetaCommand(engine, "container", "inspect", "--format", format, container.ContainerSuiPlayground)
}

func decodeCrashInspect(res CommandResult) (crashInspect, string) {
	if res.Reason != "" {
		return crashInspect{}, crashFailureReason(res)
	}
	if len(res.Stdout) > captureLimit {
		return crashInspect{}, "output exceeded capture limit"
	}
	var raw struct {
		ID         string `json:"Id"`
		Status     string `json:"Status"`
		ExitCode   *int   `json:"ExitCode"`
		StartedAt  string `json:"StartedAt"`
		FinishedAt string `json:"FinishedAt"`
	}
	if json.Unmarshal(res.Stdout, &raw) != nil {
		return crashInspect{}, "malformed selected metadata"
	}
	id := strings.ToLower(strings.TrimSpace(raw.ID))
	if !containerIDPattern.MatchString(id) {
		id = ""
	}
	return crashInspect{ID: id, Status: raw.Status, ExitCode: raw.ExitCode, StartedAt: raw.StartedAt, FinishedAt: raw.FinishedAt}, ""
}

func applyLogHint(ctx context.Context, opts CrashOptions, hints *CrashHints) {
	res := opts.run(ctx, crashMetaCommand(opts.Engine, "logs", "--tail", formatExitCode(crashLogTailLines), container.ContainerSuiPlayground))
	if res.Reason != "" {
		hints.LogUnavailable = crashFailureReason(res)
		return
	}
	hints.LogExcerpt = illegalInstructionExcerpt(string(res.Stdout))
}

func illegalInstructionExcerpt(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(strings.ToLower(line), "illegal instruction") {
			return SanitizeDiagnostic(line)
		}
	}
	return ""
}
