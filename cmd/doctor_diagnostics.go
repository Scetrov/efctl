package cmd

import (
	"fmt"
	"io"
	"os"

	"efctl/pkg/doctor"
)

func printRuntimeDiagnosticSection(r *doctor.Report) {
	renderRuntimeDiagnostics(os.Stdout, r.Diagnostics)
}

func printCrashSection(r doctor.CrashResult) {
	fmt.Fprintln(os.Stdout, "crash diagnostics:")
	doctor.RenderCrash(os.Stdout, r)
	fmt.Fprintln(os.Stdout)
}

func diagnosticLabel(e doctor.Evidence) string {
	source := doctor.SanitizeDiagnostic(e.Source)
	if source == "" {
		source = "not collected"
	}
	value := doctor.SanitizeDiagnosticValue(e.Value)
	if e.Reason != "" {
		value = "unavailable: " + doctor.SanitizeDiagnostic(e.Reason)
	} else if value == "" {
		if e.Source == "" {
			value = "unavailable"
		} else {
			value = "none reported"
		}
	}
	return value + " [" + source + "]"
}

func renderRuntimeDiagnostics(w io.Writer, d doctor.RuntimeDiagnostics) {
	line := func(label string, e doctor.Evidence) { fmt.Fprintf(w, doctorFmt, label+":", diagnosticLabel(e)) }
	cpu := func(prefix string, c doctor.CPUInfo) {
		line(prefix+" CPU model", c.Model)
		line(prefix+" machine arch", c.Architecture)
		line(prefix+" CPU flags", c.Flags)
		if c.Heterogeneous {
			fmt.Fprintf(w, doctorFmt, prefix+" CPU scope:", "per-CPU features differ; flags are their conservative intersection")
		}
	}
	cpu("local", d.LocalCPU)
	line("runtime client version", d.Runtime.Client)
	line("runtime server version", d.Runtime.Server)
	line("runtime server OS", d.Runtime.OS)
	line("runtime server arch", d.Runtime.Architecture)
	line("runtime boundary", d.Runtime.Boundary)
	line("managed Sui state", d.Image.State)
	line("recorded container exit", d.Image.ExitCode)
	line("Sui image reference", d.Image.Reference)
	line("Sui image ID", d.Image.ID)
	line("Sui image OS", d.Image.OS)
	line("Sui image arch", d.Image.Architecture)
	line("Sui image digests", d.Image.Digests)
	line("host Sui version", d.HostSui)
	line("runtime Sui version", d.Sui)
	cpu("runtime", d.CPU)
	if d.ExecFailure.Source != "" {
		line("managed exec failure", d.ExecFailure)
	}
	if d.Cleanup.Source != "" {
		line("probe cleanup", d.Cleanup)
	}
	if d.Runtime.Architecture.Reason == "" && d.Image.Architecture.Reason == "" && d.Runtime.Architecture.Value != "" && d.Image.Architecture.Value != "" && doctor.NormalizeArchitecture(d.Runtime.Architecture.Value) != doctor.NormalizeArchitecture(d.Image.Architecture.Value) {
		fmt.Fprintf(w, doctorFmt, "architecture note:", "server/image architectures differ: possible emulation; not confirmed (not a crash diagnosis)")
	}
	fmt.Fprintf(w, doctorFmt, "diagnostic scope:", "version-only probes do not establish node startup health")
	fmt.Fprintln(w)
}
