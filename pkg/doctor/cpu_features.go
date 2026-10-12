package doctor

import "strings"

var requiredCPUFeatures = []string{"adx", "bmi2"}

// MissingCPUFeatures reports missing features used by some x86-64 Sui builds.
// Callers interpreting Sui compatibility must first check observed architecture.
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

// MissingSuiCPUFeatures interprets collected evidence only; it launches no probes.
// ARM and unknown architectures do not establish x86 instruction requirements.
func MissingSuiCPUFeatures(cpu CPUInfo) ([]string, bool) {
	arch := cpu.Architecture
	if arch.Source == "" || arch.Reason != "" || NormalizeArchitecture(arch.Value) != "amd64" {
		return nil, false
	}
	return MissingCPUFeatures(cpu.Flags)
}
