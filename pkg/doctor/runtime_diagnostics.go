package doctor

import (
	"context"
	"efctl/pkg/env"
)

// Evidence distinguishes unavailable data from an observed empty value.
type Evidence struct{ Source, Value, Reason string }

// CPUInfo describes the CPU view visible at the labelled execution boundary.
type CPUInfo struct {
	Model, Architecture, Flags Evidence
	Heterogeneous              bool
}

// RuntimeServerInfo keeps client identity separate from the server platform.
type RuntimeServerInfo struct{ Client, Server, OS, Architecture, Boundary Evidence }

// ImageInfo describes the actual container image or an explicitly labelled fallback.
type ImageInfo struct{ State, ExitCode, Reference, ID, OS, Architecture, Digests Evidence }

// RuntimeDiagnostics contains additive, independently available observations.
type RuntimeDiagnostics struct {
	LocalCPU    CPUInfo
	HostSui     Evidence
	Runtime     RuntimeServerInfo
	Image       ImageInfo
	Sui         Evidence
	CPU         CPUInfo
	ExecFailure Evidence
	Cleanup     Evidence
}

type probes struct {
	ctx context.Context
	run DiagnosticRunner
}

func (p probes) command(name string, args ...string) CommandResult {
	if err := p.ctx.Err(); err != nil {
		return CommandResult{Reason: err.Error()}
	}
	ctx, cancel := context.WithTimeout(p.ctx, commandTimeout)
	defer cancel()
	return p.run(ctx, name, args...)
}

func gatherRuntimeDiagnostics(opts Options, prereqs *env.CheckResult) RuntimeDiagnostics {
	parent := opts.Context
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := newProbeContext(parent)
	defer cancel()
	run := opts.Runner
	if run == nil {
		run = runDiagnostic
	}
	p := probes{ctx: ctx, run: run}
	return gatherDiagnostics(p, prereqs)
}
