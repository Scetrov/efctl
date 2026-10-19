package doctor

import (
	"context"
	"runtime"
	"time"
)

// CrashOptions configures opt-in crash collection. It does not start containers or enable dumps.
type CrashOptions struct {
	Context     context.Context
	GOOS        string
	Engine      string
	Boundary    Evidence
	Now         time.Time
	Runner      CrashRunner
	LookPath    func(string) (string, error)
	runtimeGOOS string
}

func (o CrashOptions) now() time.Time {
	if o.Now.IsZero() {
		return time.Now().UTC()
	}
	return o.Now.UTC()
}

// CollectCrash gathers opt-in crash hints and, when the local kernel boundary is proven, journal metadata.
// It is not called by Gather. Unavailable crash data is a section reason, not a doctor failure.
func CollectCrash(opts CrashOptions) CrashResult {
	if opts.runtimeGOOS == "" {
		opts.runtimeGOOS = runtime.GOOS
	}
	ctx, cancel := crashCollectionContext(opts.Context)
	defer cancel()
	result := CrashResult{Hints: collectHints(ctx, opts)}
	if result.Hints.IdentityUnavailable {
		return skipCorrelation(result, "managed container identity is unavailable")
	}
	if reason, ok := allowCrashQuery(opts); !ok {
		return skipCorrelation(result, reason)
	}
	window, ok := buildWindow(result.Hints, opts.now())
	if !ok {
		return skipCorrelation(result, "container start time unavailable")
	}
	records, reason := loadJournal(ctx, opts, window)
	if reason != "" {
		return skipCorrelation(result, reason)
	}
	return finishCorrelation(ctx, opts, result, records, window)
}

func skipCorrelation(result CrashResult, reason string) CrashResult {
	result.Correlation.Skipped = true
	result.Correlation.Reason = reason
	return result
}

func finishCorrelation(ctx context.Context, opts CrashOptions, result CrashResult, records []parsedCrash, window crashSpan) CrashResult {
	attributed, unscoped := classifyCrashes(records, result.Hints.ContainerID, window)
	result.Correlation.UnscopedCount = len(unscoped)
	result.Correlation.AttributedCount = len(attributed)
	if len(unscoped) > maxUnscopedKept {
		unscoped = unscoped[:maxUnscopedKept]
	}
	result.Correlation.Unscoped = unscoped
	if len(attributed) == 0 {
		return result
	}
	selected := newestCrash(attributed)
	var presenceReason string
	selected, presenceReason = applyCorePresence(ctx, opts, window, selected)
	result.Correlation.Selected = &selected
	result.Unwind = unwindOne(ctx, opts, selected, presenceReason)
	return result
}
