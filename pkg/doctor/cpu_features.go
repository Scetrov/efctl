package doctor

import (
	"context"
	"runtime"
	"strings"
)

var requiredCPUFeatures = []string{"adx", "bmi2"}

// MissingCPUFeatures reports which required CPU features are absent.
// The boolean is false when the feature data is unavailable.
func MissingCPUFeatures(flags Evidence) ([]string, bool) {
	if flags.Reason != "" || flags.Source == "" {
		return nil, false
	}

	available := make(map[string]bool)
	for _, flag := range strings.Fields(strings.ToLower(flags.Value)) {
		available[flag] = true
	}

	var missing []string
	for _, feature := range requiredCPUFeatures {
		if !available[feature] {
			missing = append(missing, strings.ToUpper(feature))
		}
	}

	return missing, true

}

// LocalMissingCPUFeatures checks the CPU features exposed to this process.
func LocalMissingCPUFeatures() ([]string, bool) {
	ctx, cancel := newProbeContext(context.Background())
	defer cancel()

	p := probes{
		ctx: ctx,
		run: runDiagnostic,
	}

	cpu := gatherLocalCPU(p, runtime.GOOS)
	return MissingCPUFeatures(cpu.Flags)

}
