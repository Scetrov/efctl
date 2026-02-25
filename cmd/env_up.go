package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"efctl/pkg/env"
	"efctl/pkg/setup"
	"efctl/pkg/ui"
	"github.com/spf13/cobra"
)

var envUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Bring up the local environment",
	Long:  `Runs check, setup, start, and deploy sequentially to bring up a fully working EVE Frontier Smart Assembly testing environment.`,
	Run: func(cmd *cobra.Command, args []string) {
		ui.Info.Println("Checking prerequisites...")
		res := env.CheckPrerequisites()

		if !res.HasNode {
			ui.Error.Println("Node.js is not installed. Please install Node.js >= 20.0.0 to continue.")
			os.Exit(1)
		}
		if strings.HasPrefix(res.NodeVer, "v") {
			parts := strings.Split(res.NodeVer[1:], ".")
			if len(parts) >= 1 {
				major, err := strconv.Atoi(parts[0])
				if err == nil {
					if major < 20 {
						ui.Error.Println("Node.js version must be 20.0.0 or higher. Found: " + res.NodeVer)
						os.Exit(1)
					} else if major != 24 {
						ui.Warn.Println("Node.js version is within range but different from project standard (24.x.x). Found: " + res.NodeVer)
					}
				}
			}
		}

		if !res.HasDocker && !res.HasPodman {
			ui.Error.Println("Neither Docker nor Podman is installed. Please install one to continue.")
			os.Exit(1)
		}
		if !res.HasGit {
			ui.Error.Println("Git is not installed.")
			os.Exit(1)
		}
		if !env.IsPortAvailable(9000) {
			ui.Error.Println("Port 9000 is already in use by another process. Please free it up before initializing.")
			os.Exit(1)
		}
		if withGraphql {
			if !env.IsPortAvailable(8000) {
				ui.Error.Println("Port 8000 (GraphQL) is already in use by another process. Please free it up.")
				os.Exit(1)
			}
			if !env.IsPortAvailable(5432) {
				ui.Error.Println("Port 5432 (PostgreSQL) is already in use by another process. Please free it up.")
				os.Exit(1)
			}
		}

		ui.Info.Println("Setting up workspace...")
		if err := setup.CloneRepositories(workspacePath); err != nil {
			ui.Error.Println("Setup failed: " + err.Error())
			ui.Warn.Println("The environment may be partially initialized. It is recommended to run `efctl down` before trying again.")
			os.Exit(1)
		}

		ui.Info.Println("Starting environment...")

		if err := setup.StartEnvironment(workspacePath, withGraphql); err != nil {
			ui.Error.Println("Start failed: " + err.Error())
			ui.Warn.Println("The environment may be partially initialized. It is recommended to run `efctl down` before trying again.")
			os.Exit(1)
		}

		ui.Info.Println("Deploying world contracts...")
		if err := setup.DeployWorld(workspacePath); err != nil {
			ui.Error.Println("Deployment failed: " + err.Error())
			ui.Warn.Println("The environment may be partially initialized. It is recommended to run `efctl down` before trying again.")
			os.Exit(1)
		}

		setup.PrintDeploymentSummary(workspacePath)

		ui.Success.Println(fmt.Sprintf("\n%s Environment is up! The Sui playground is running and gates are spawned.", ui.GlobeEmoji))
	},
}

var withGraphql bool

func init() {
	envUpCmd.Flags().BoolVar(&withGraphql, "with-graphql", false, "Enable the SQL Indexer and GraphQL API")
	envCmd.AddCommand(envUpCmd)
}
