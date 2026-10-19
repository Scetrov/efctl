package doctor

import (
	"encoding/json"
	"io"
	"os"
	"sort"
	"strings"
	"time"
)

const commandTimeout = 5 * time.Second

func available(source, value string) Evidence {
	return Evidence{Source: SanitizeDiagnostic(source), Value: SanitizeDiagnosticValue(value)}
}
func unavailable(source, reason string) Evidence {
	return Evidence{Source: SanitizeDiagnostic(source), Reason: SanitizeDiagnostic(reason)}
}
func (p probes) evidence(source, name string, args ...string) Evidence {
	r := p.command(name, args...)
	if r.Reason != "" {
		return unavailable(source, r.Reason)
	}
	value := strings.TrimSpace(string(r.Stdout))
	if value == "" {
		return unavailable(source, "command returned no data")
	}
	return available(source, value)
}

// NormalizeArchitecture compares aliases without claiming physical CPU identity.
func NormalizeArchitecture(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "x86_64", "amd64", "x64":
		return "amd64"
	case "aarch64", "arm64":
		return "arm64"
	case "i386", "i686", "x86", "386":
		return "386"
	case "armv7l", "armv6l", "arm":
		return "arm"
	}
	return strings.ToLower(strings.TrimSpace(value))
}
func sortedFlags(raw string) string {
	set := map[string]bool{}
	for _, flag := range strings.Fields(strings.ToLower(raw)) {
		set[flag] = true
	}
	flags := make([]string, 0, len(set))
	for f := range set {
		flags = append(flags, f)
	}
	sort.Strings(flags)
	return strings.Join(flags, " ")
}

func parseLinuxCPU(raw, source string) CPUInfo {
	info := CPUInfo{Model: unavailable(source, "CPU model unavailable"), Flags: unavailable(source, "CPU flags unavailable")}
	sets, models, incomplete := linuxCPUBlocks(raw)
	if model := joinedUnique(models); model != "" {
		info.Model = available(source, model)
	}
	if incomplete {
		info.Flags = unavailable(source, "incomplete per-CPU flags")
		return info
	}
	flags, heterogeneous, ok := intersectFlagSets(sets)
	if ok {
		info.Flags = available(source, flags)
		info.Heterogeneous = heterogeneous
	}
	return info
}

type linuxCPUBlock struct {
	processor bool
	hasFlags  bool
	flags     map[string]bool
	models    []string
}

func linuxCPUBlocks(raw string) ([]map[string]bool, []string, bool) {
	var sets []map[string]bool
	var models []string
	incomplete := false
	for _, block := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n\n") {
		if strings.TrimSpace(block) == "" {
			continue
		}
		parsed := parseLinuxCPUBlock(block)
		models = append(models, parsed.models...)
		if parsed.processor || parsed.hasFlags {
			if !parsed.hasFlags {
				incomplete = true
			}
			sets = append(sets, parsed.flags)
		}
	}
	return sets, models, incomplete
}

func parseLinuxCPUBlock(block string) linuxCPUBlock {
	parsed := linuxCPUBlock{flags: map[string]bool{}}
	for _, line := range strings.Split(block, "\n") {
		key, value, ok := splitCPUField(line)
		if !ok {
			continue
		}
		switch key {
		case "processor":
			parsed.processor = true
		case "model name", "Processor", "Hardware":
			if value != "" {
				parsed.models = append(parsed.models, value)
			}
		case "flags", "Features":
			parsed.hasFlags = true
			recordFlags(parsed.flags, value)
		}
	}
	return parsed
}

func splitCPUField(line string) (string, string, bool) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
}

func recordFlags(flags map[string]bool, value string) {
	for _, flag := range strings.Fields(value) {
		flags[flag] = true
	}
}

func joinedUnique(values []string) string {
	if len(values) == 0 {
		return ""
	}
	sort.Strings(values)
	unique := values[:0]
	for _, value := range values {
		if len(unique) == 0 || unique[len(unique)-1] != value {
			unique = append(unique, value)
		}
	}
	return strings.Join(unique, "; ")
}

func intersectFlagSets(sets []map[string]bool) (string, bool, bool) {
	if len(sets) == 0 {
		return "", false, false
	}
	intersection := map[string]bool{}
	for flag := range sets[0] {
		intersection[flag] = true
	}
	heterogeneous := false
	for _, set := range sets[1:] {
		if cpuFlagSetsDiffer(sets[0], set) {
			heterogeneous = true
		}
		for flag := range intersection {
			if !set[flag] {
				delete(intersection, flag)
			}
		}
	}
	return joinedFlags(intersection), heterogeneous, true
}

func cpuFlagSetsDiffer(base, set map[string]bool) bool {
	if len(set) != len(base) {
		return true
	}
	for flag := range base {
		if !set[flag] {
			return true
		}
	}
	return false
}

func joinedFlags(flags map[string]bool) string {
	values := make([]string, 0, len(flags))
	for flag := range flags {
		values = append(values, flag)
	}
	sort.Strings(values)
	return strings.Join(values, " ")
}

func gatherLocalCPU(p probes, platform string) CPUInfo {
	source := "local CPU view visible to efctl"
	info := CPUInfo{Model: unavailable(source, "unsupported CPU source"), Architecture: unavailable(source, "unsupported machine architecture source"), Flags: unavailable(source, "instruction flags unavailable")}
	switch platform {
	case "linux":
		// CPU metadata is bounded too; /proc is not assumed to be trusted or small.
		f, err := os.Open("/proc/cpuinfo")
		if err == nil {
			data, readErr := io.ReadAll(io.LimitReader(f, captureLimit+1))
			_ = f.Close()
			if readErr == nil && len(data) <= captureLimit {
				info = parseLinuxCPU(string(data), source+" /proc/cpuinfo")
			} else {
				info.Model = unavailable(source, "CPU source exceeds capture limit or cannot be read")
				info.Flags = info.Model
			}
		} else {
			info.Model = unavailable(source, "/proc/cpuinfo unavailable")
			info.Flags = info.Model
		}
		info.Architecture = p.evidence(source+" uname", "uname", "-m")
	case "darwin":
		info.Architecture = p.evidence(source+" uname", "uname", "-m")
		info.Model = p.evidence(source+" sysctl", "sysctl", "-n", "machdep.cpu.brand_string")
		if info.Model.Reason != "" {
			info.Model = p.evidence(source+" sysctl", "sysctl", "-n", "hw.model")
		}
		var features []string
		failed := false
		for _, key := range []string{"machdep.cpu.features", "machdep.cpu.leaf7_features", "machdep.cpu.extfeatures"} {
			value := p.evidence(source+" sysctl", "sysctl", "-n", key)
			if value.Reason != "" {
				failed = true
			} else {
				features = append(features, value.Value)
			}
		}
		if !failed {
			info.Flags = available(source+" sysctl", sortedFlags(strings.Join(features, " ")))
		} else {
			info.Flags = unavailable(source+" sysctl", "CPU flag sources incomplete or unsupported")
		}
	case "windows":
		r := p.command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", `$p=Get-CimInstance Win32_Processor; @{Model=($p.Name -join '; ');Architecture=[System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()} | ConvertTo-Json -Compress`)
		var value struct{ Model, Architecture string }
		if r.Reason == "" && json.Unmarshal(r.Stdout, &value) == nil {
			info.Model = fieldEvidence(source+" CIM", value.Model)
			info.Architecture = fieldEvidence(source+" OSArchitecture", value.Architecture)
		} else {
			reason := r.Reason
			if reason == "" {
				reason = "malformed CPU metadata"
			}
			info.Model = unavailable(source, reason)
			info.Architecture = info.Model
		}
		info.Flags = unavailable(source, "Windows CIM does not provide a complete instruction feature set")
	}
	return info
}
func fieldEvidence(source, value string) Evidence {
	if value == "" {
		return unavailable(source, "field not reported")
	}
	return available(source, value)
}
