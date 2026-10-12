package cmd

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"efctl/pkg/config"
	"efctl/pkg/doctor"
	"efctl/pkg/env"

	"github.com/spf13/cobra"
)

var doctorWorkspace string
var doctorCrash bool

// collectDoctorCrash is replaced in command tests so default doctor can prove it
// does not query crash helpers.
var collectDoctorCrash = doctor.CollectCrash

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Print diagnostic information about the environment",
	Long: `Prints a non-destructive summary of the local environment useful for debugging
and bug reports, including: efctl version, OS details, container runtime, Node.js,
git, the state of running containers, port availability, and the git ref of any
checked-out builder-scaffold and world-contracts repositories.

Also reports source-labelled CPU features, runtime client/server platforms, actual
Sui image identity, and host/container Sui versions. When the managed Sui container
is stopped or absent, doctor may create a transient isolated probe of an existing
local immutable image. It never pulls images, starts the managed environment,
initializes a Sui client, or mounts host/managed data. Probes override the entrypoint,
disable networking, use a read-only root and resource limits, and are cleaned up.
Unsupported isolation or image-declared volumes cause probing to be skipped.

New probes share a 20-second budget with five seconds per command and up to three
additional seconds for cleanup; existing report gatherers are not covered by this
budget. Local, remote/VM server, image, and execution-time CPU views are distinct.
Unavailable fields include reasons. Version success is not proof of node health;
architecture mismatch suggests possible emulation, not a confirmed crash cause.
Update efctl, run doctor from your workspace, and review the report before sharing
it with support. A cleanup failure identifies the probe to inspect/remove.

--crash is opt-in and off by default. Default doctor does not query systemd,
coredumpctl, journalctl, or a debugger, and it does not add a crash section.
With --crash, doctor appends container exit/log hints and, only on a proven-local
Linux kernel, field-limited host crash metadata. It does not enable core dumps,
extract a core, print crash environment or maps, install helpers, or change the
doctor exit status when crash evidence is unavailable. Crash collection has its
own 20-second budget, five seconds per metadata command, and eight seconds for
one debugger invocation. Review that section before sharing it; do not attach
core files.`,
	Run: func(cmd *cobra.Command, args []string) {
		prereqs := env.CheckPrerequisites()

		cfgLoaded := false
		cfgPath := configFile
		if config.Loaded != nil && config.Loaded.WasLoaded() {
			cfgLoaded = true
		}

		abs, err := filepath.Abs(doctorWorkspace)
		if err == nil {
			doctorWorkspace = abs
		}

		r := doctor.Gather(doctor.Options{
			Context:      cmd.Context(),
			Workspace:    doctorWorkspace,
			Version:      Version,
			CommitSHA:    CommitSHA,
			BuildDate:    BuildDate,
			Prereqs:      prereqs,
			ConfigLoaded: cfgLoaded,
			ConfigPath:   cfgPath,
			Config:       config.Loaded,
			Crash:        doctorCrash,
		})

		var crash *doctor.CrashResult
		if doctorCrash {
			collected := collectDoctorCrash(doctor.CrashOptions{
				Context:  cmd.Context(),
				GOOS:     runtime.GOOS,
				Engine:   r.Container.Engine,
				Boundary: r.Diagnostics.Runtime.Boundary,
			})
			crash = &collected
		}

		printDoctorReport(r)
		if crash != nil {
			printCrashSection(*crash)
		}
	},
}

const doctorFmt = "%-22s %s\n"

func printDoctorReport(r *doctor.Report) {
	printIdentitySection(r)
	printToolsSection(r)
	printRuntimeDiagnosticSection(r)
	printEnvSection(r)
	printPortsSection(r)
	printSuiSection(r)
	printReposSection(r)
	printConfigSection(r)
}

func printIdentitySection(r *doctor.Report) {
	fmt.Printf(doctorFmt, "efctl:", fmt.Sprintf(
		"%s (%s) built %s %s/%s",
		r.Efctl.Version, r.Efctl.CommitSHA, r.Efctl.BuildDate,
		r.Efctl.GOOS, r.Efctl.GOARCH,
	))
	fmt.Printf(doctorFmt, "os:", r.System.OS+" ("+r.System.Platform+")")
	fmt.Printf(doctorFmt, "wsl:", yesNo(r.System.IsWSL))
	fmt.Printf(doctorFmt, "go runtime:", r.System.GoVersion)
	fmt.Println()
}

func printToolsSection(r *doctor.Report) {
	if r.Container.Found {
		fmt.Printf(doctorFmt, "container runtime:", fmt.Sprintf(
			"%s %s (%s)", r.Container.Engine, r.Container.Version, r.Container.Path,
		))
		printPodmanDetails(r)
	} else {
		fmt.Printf(doctorFmt, "container runtime:", "not found")
	}

	if r.Node.Found {
		fmt.Printf(doctorFmt, "node:", fmt.Sprintf("%s (%s)", r.Node.Version, r.Node.Path))
	} else {
		fmt.Printf(doctorFmt, "node:", "not found")
	}

	if r.Git.Found {
		fmt.Printf(doctorFmt, "git:", fmt.Sprintf("%s (%s)", r.Git.Version, r.Git.Path))
	} else {
		fmt.Printf(doctorFmt, "git:", "not found")
	}
	fmt.Println()
}

func printPodmanDetails(r *doctor.Report) {
	if r.Container.Engine != "podman" {
		return
	}
	if r.Container.PodmanNetns != "" {
		fmt.Printf(doctorFmt, "podman netns:", r.Container.PodmanNetns)
	}
	if r.Container.PodmanRuntime != "" {
		fmt.Printf(doctorFmt, "podman runtime:", r.Container.PodmanRuntime)
	}
	if r.Container.PodmanFirewallDriver != "" {
		fmt.Printf(doctorFmt, "podman firewall:", r.Container.PodmanFirewallDriver)
	}
}

func printEnvSection(r *doctor.Report) {
	fmt.Printf(doctorFmt, "env:", envStateLabel(r.Env))
	if len(r.Env.Logs) > 0 {
		fmt.Printf(doctorFmt, "container logs:", "last 10 lines from running containers")
		for _, log := range r.Env.Logs {
			fmt.Printf(doctorFmt, log.Name+":", "")
			for _, line := range strings.Split(log.Tail, "\n") {
				fmt.Printf("  %s\n", line)
			}
		}
	}
	fmt.Println()
}

func printPortsSection(r *doctor.Report) {
	for _, p := range r.Ports {
		avail := "free"
		if !p.Available {
			avail = "in use"
		}
		fmt.Printf(doctorFmt, fmt.Sprintf("port %d:", p.Port), avail)
	}
	fmt.Println()
}

func printSuiSection(r *doctor.Report) {
	if r.Sui.Found {
		fmt.Printf(doctorFmt, "sui active env:", r.Sui.ActiveEnv)
		fmt.Printf(doctorFmt, "sui active address:", r.Sui.ActiveAddress)
		fmt.Printf(doctorFmt, "sui rpc url:", r.Sui.ActiveEnvRpcUrl)
		fmt.Printf(doctorFmt, "sui faucet url:", r.Sui.ActiveEnvFaucetUrl)
	} else {
		fmt.Printf(doctorFmt, "sui client:", "not found")
	}
	fmt.Println()
}

func printReposSection(r *doctor.Report) {
	for _, repo := range r.Repos {
		fmt.Printf(doctorFmt, repo.Name+":", repoLabel(repo))
	}
	fmt.Println()
}

func printConfigSection(r *doctor.Report) {
	if r.Config.Loaded {
		fmt.Printf(doctorFmt, "config file:", r.Config.FilePath+" (loaded)")
		for _, entry := range r.Config.Entries {
			fmt.Printf(doctorFmt, "config "+entry.Key+":", entry.Value)
		}
	} else {
		fmt.Printf(doctorFmt, "config file:", "not found (using defaults)")
	}
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func envStateLabel(e doctor.EnvironmentInfo) string {
	switch e.State {
	case "up":
		return fmt.Sprintf("up (%d/%d containers running)", e.Running, e.Total)
	case "partial":
		return fmt.Sprintf("partial (%d/%d containers running)", e.Running, e.Total)
	case "down":
		return fmt.Sprintf("down (%d/%d containers running)", e.Running, e.Total)
	default:
		if e.Error != "" {
			return "unknown (" + e.Error + ")"
		}
		return "unknown"
	}
}

func repoLabel(r doctor.RepoInfo) string {
	if !r.Found {
		if r.Error != "" {
			return "not found (" + r.Error + ")"
		}
		return "not found"
	}

	shortSHA := r.Commit
	if len(shortSHA) > 7 {
		shortSHA = shortSHA[:7]
	}

	branch := r.Branch
	if branch == "" {
		branch = "detached HEAD"
	}

	dirtyStr := "clean"
	if r.IsDirty {
		dirtyStr = "modified"
	}

	res := fmt.Sprintf("%s on %s (%s)", shortSHA, branch, dirtyStr)
	if r.Remote != "" {
		res = r.Remote + " " + res
	}
	return res
}

func init() {
	doctorCmd.Flags().StringVarP(&doctorWorkspace, "workspace", "w", ".", "Path to the workspace directory")
	doctorCmd.Flags().BoolVar(&doctorCrash, "crash", false, "Opt-in local crash correlation; review before sharing and do not attach core files")
	rootCmd.AddCommand(doctorCmd)
}
