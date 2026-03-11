package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"efctl/pkg/config"
	"efctl/pkg/doctor"
	"efctl/pkg/env"

	"github.com/spf13/cobra"
)

var doctorWorkspace string

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Print diagnostic information about the environment",
	Long: `Prints a non-destructive summary of the local environment useful for debugging
and bug reports, including: efctl version, OS details, container runtime, Node.js,
git, the state of running containers, port availability, and the git ref of any
checked-out builder-scaffold and world-contracts repositories.`,
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
			Workspace:    doctorWorkspace,
			Version:      Version,
			CommitSHA:    CommitSHA,
			BuildDate:    BuildDate,
			Prereqs:      prereqs,
			ConfigLoaded: cfgLoaded,
			ConfigPath:   cfgPath,
			Config:       config.Loaded,
		})

		printDoctorReport(r)
	},
}

const doctorFmt = "%-22s %s\n"

func printDoctorReport(r *doctor.Report) {
	// ── efctl identity ────────────────────────────────────────────
	fmt.Printf(doctorFmt, "efctl:", fmt.Sprintf(
		"%s (%s) built %s %s/%s",
		r.Efctl.Version, r.Efctl.CommitSHA, r.Efctl.BuildDate,
		r.Efctl.GOOS, r.Efctl.GOARCH,
	))
	fmt.Printf(doctorFmt, "os:", r.System.OS+" ("+r.System.Platform+")")
	fmt.Printf(doctorFmt, "wsl:", yesNo(r.System.IsWSL))
	fmt.Printf(doctorFmt, "go runtime:", r.System.GoVersion)
	fmt.Println()

	// ── Tool versions ─────────────────────────────────────────────
	if r.Container.Found {
		fmt.Printf(doctorFmt, "container runtime:", fmt.Sprintf(
			"%s %s (%s)", r.Container.Engine, r.Container.Version, r.Container.Path,
		))
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

	// ── Environment state ─────────────────────────────────────────
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

	// ── Port availability ─────────────────────────────────────────
	for _, p := range r.Ports {
		avail := "free"
		if !p.Available {
			avail = "in use"
		}
		fmt.Printf(doctorFmt, fmt.Sprintf("port %d:", p.Port), avail)
	}
	fmt.Println()

	// ── Repository state ──────────────────────────────────────────
	for _, repo := range r.Repos {
		fmt.Printf(doctorFmt, repo.Name+":", repoLabel(repo))
	}
	fmt.Println()

	// ── Config file ───────────────────────────────────────────────
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

	return fmt.Sprintf("%s on %s (%s)", shortSHA, branch, dirtyStr)
}

func init() {
	doctorCmd.Flags().StringVarP(&doctorWorkspace, "workspace", "w", ".", "Path to the workspace directory")
	rootCmd.AddCommand(doctorCmd)
}
