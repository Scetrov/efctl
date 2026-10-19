package doctor

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"runtime"
	"strconv"
	"strings"
	"time"

	"efctl/pkg/container"
	"efctl/pkg/env"
)

func formatExitCode(code int) string { return strconv.Itoa(code) }
func missingCPU(source, reason string) CPUInfo {
	v := unavailable(source, reason)
	return CPUInfo{Model: v, Architecture: v, Flags: v}
}

func gatherDiagnostics(p probes, prereqs *env.CheckResult) RuntimeDiagnostics {
	d := RuntimeDiagnostics{LocalCPU: gatherLocalCPU(p, runtime.GOOS), HostSui: p.evidence("host version-only", "sui", "--version")}
	d.Sui = unavailable("runtime", "no container engine")
	d.CPU = missingCPU("runtime", "no container engine")
	d.Image = unavailableImage("runtime", "no container engine")
	engine, err := prereqs.Engine()
	if err != nil {
		v := unavailable("runtime", "no container engine")
		d.Runtime = RuntimeServerInfo{Client: v, Server: v, OS: v, Architecture: v, Boundary: v}
		return d
	}
	d.Runtime = gatherRuntimeServer(p, engine)
	image, running, safe := gatherImage(p, engine)
	d.Image = image
	if running {
		d.Sui, d.CPU = executionEvidence(p, engine, container.ContainerSuiPlayground, "managed container")
		// A failed version command (including SIGILL) is not evidence of a container exit.
		// Recheck state only when an exec probe failed, and fall back at most once.
		if d.Sui.Reason != "" || d.CPU.Architecture.Reason != "" || d.CPU.Flags.Reason != "" {
			state := p.evidence("managed container", engine, "container", "inspect", "--format", "{{.State.Running}}", container.ContainerSuiPlayground)
			exited := state.Value == "false"
			if state.Reason != "" {
				listing := p.command(engine, "container", "ls", "--all", "--filter", "name=^"+container.ContainerSuiPlayground+"$", "--format", "{{.Names}}")
				exited = listing.Reason == "" && strings.TrimSpace(string(listing.Stdout)) == ""
			}
			if exited {
				d.ExecFailure = unavailable("managed container exec before exit", d.Sui.Reason+"; "+d.CPU.Architecture.Reason+"; "+d.CPU.Flags.Reason)
				d.Sui, d.CPU, d.Cleanup = disposableEvidence(p, engine, image.ID.Value, safe)
			}
		}
	} else {
		d.Sui, d.CPU, d.Cleanup = disposableEvidence(p, engine, image.ID.Value, safe)
	}
	return d
}

func executionEvidence(p probes, engine, target, source string) (Evidence, CPUInfo) {
	version := p.evidence(source+" version-only", engine, "exec", target, "sui", "--version")
	architecture := p.evidence(source+" uname", engine, "exec", target, "uname", "-m")
	raw := p.command(engine, "exec", target, "cat", "/proc/cpuinfo")
	cpu := missingCPU(source, "CPU source unavailable")
	if raw.Reason == "" {
		cpu = parseLinuxCPU(string(raw.Stdout), source+" /proc/cpuinfo")
	} else {
		cpu = missingCPU(source, raw.Reason)
	}
	cpu.Architecture = architecture
	return version, cpu
}

func disposableEvidence(p probes, engine, image string, safe bool) (version Evidence, cpu CPUInfo, cleanup Evidence) {
	source := "local-image probe (current server view, not historical managed context)"
	reason := "immutable local image unavailable"
	if immutableImage.MatchString(image) && !safe {
		reason = "image-declared volumes present or unknown; isolation cannot be guaranteed"
	}
	version = unavailable(source, reason)
	cpu = missingCPU(source, reason)
	if !immutableImage.MatchString(image) || !safe {
		return
	}
	if err := p.ctx.Err(); err != nil {
		version = unavailable(source, err.Error())
		cpu = missingCPU(source, err.Error())
		return
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		version = unavailable(source, "unique probe identity unavailable")
		return
	}
	token := hex.EncodeToString(nonce)
	name := "efctl-doctor-" + token
	// Track identity before creation, but only remove an object whose random label
	// proves ownership. Name alone is insufficient after a collision or failed run.
	defer func() { cleanup = cleanupProbe(p.run, engine, name, token) }()
	args := []string{"run", "--detach", "--name", name, "--label", "org.efctl.doctor-probe=" + token, "--pull", "never", "--network", "none", "--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--memory", "256m", "--cpus", "1", "--pids-limit", "32", "--ulimit", "core=0:0", "--no-healthcheck", "--log-driver", "none", "--user", "65534:65534", "--entrypoint", "/bin/sh", image, "-c", "sleep 30"}
	result := p.command(engine, args...)
	if result.Reason != "" {
		version = unavailable(source, "isolated probe could not start: "+result.Reason)
		cpu = missingCPU(source, version.Reason)
		return
	}
	version, cpu = executionEvidence(p, engine, name, source)
	return
}

func cleanupProbe(run DiagnosticRunner, engine, name, token string) Evidence {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	source := "disposable probe cleanup"
	ownership := run(ctx, engine, "container", "inspect", "--format", `{{index .Config.Labels "org.efctl.doctor-probe"}}`, name)
	if ownership.Reason != "" {
		return unavailable(source, "could not verify probe ownership; inspect tracked target "+name+" before removal: "+ownership.Reason)
	}
	if strings.TrimSpace(string(ownership.Stdout)) != token {
		return unavailable(source, "probe ownership not confirmed; no object removed: "+name)
	}
	// Podman force-removal otherwise waits its stop grace period. Kill only the
	// ownership-verified probe first; an already-exited probe can reject kill
	// harmlessly, and removal must still be attempted within the same allowance.
	_ = run(ctx, engine, "container", "kill", "--signal", "KILL", name)
	result := run(ctx, engine, "container", "rm", "--force", "--volumes", name)
	if result.Reason != "" {
		return unavailable(source, "cleanup failed; owned probe removal target: "+name+": "+result.Reason)
	}
	return available(source, "removed "+name)
}
