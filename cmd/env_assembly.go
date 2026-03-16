package cmd

import (
	"fmt"
	"os"

	"efctl/pkg/assembly"
	"efctl/pkg/ui"
	"efctl/pkg/validate"
	"github.com/spf13/cobra"
)

var (
	assemblyOnBehalfOf string
	assemblyOnline     bool
	assemblyItemId     uint64
	assemblyTypeId     uint64
	assemblyLocHash    string
	asmType            string
)

var assemblyCmd = &cobra.Command{
	Use:   "assembly",
	Short: "Manage Smart Assemblies",
	Long:  `Deploy, online, and authorize Smart Assemblies (Gates, Turrets, Storage Units).`,
}

var assemblyDeployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy a new assembly",
}

var deployGateCmd = &cobra.Command{
	Use:   "gate",
	Short: "Deploy a Smart Gate",
	Run: func(cmd *cobra.Command, args []string) {
		runDeploy(assembly.TypeGate)
	},
}

var deployTurretCmd = &cobra.Command{
	Use:   "turret",
	Short: "Deploy a Smart Turret",
	Run: func(cmd *cobra.Command, args []string) {
		runDeploy(assembly.TypeTurret)
	},
}

var deployStorageCmd = &cobra.Command{
	Use:   "storage",
	Short: "Deploy a Smart Storage Unit",
	Run: func(cmd *cobra.Command, args []string) {
		runDeploy(assembly.TypeStorageUnit)
	},
}

var assemblyAuthorizeCmd = &cobra.Command{
	Use:   "authorize [extension-config-id] [assembly-id]",
	Short: "Authorize an extension for an assembly",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		extId := args[0]
		asmId := args[1]

		if err := validate.SuiAddress(extId); err != nil {
			ui.Error.Println("Invalid extension config ID: " + err.Error())
			os.Exit(1)
		}
		if err := validate.SuiAddress(asmId); err != nil {
			ui.Error.Println("Invalid assembly ID: " + err.Error())
			os.Exit(1)
		}

		aType := assembly.AssemblyType(asmType)
		if aType == "" {
			aType = assembly.TypeGate
		}

		spin, _ := ui.Spin("Authorizing extension...")
		err := assembly.AuthorizeExtension(workspacePath, aType, asmId, extId)
		if err != nil {
			spin.Fail("Authorization failed: ", err)
			os.Exit(1)
		}
		spin.Success("Extension authorized successfully.")
	},
}

func runDeploy(t assembly.AssemblyType) {
	if assemblyItemId == 0 {
		ui.Error.Println("Item ID (--item-id) is required and must be non-zero")
		os.Exit(1)
	}
	if assemblyTypeId == 0 {
		ui.Error.Println("Type ID (--type-id) is required and must be non-zero")
		os.Exit(1)
	}

	spin, _ := ui.Spin(fmt.Sprintf("Deploying %s...", t))
	id, err := assembly.DeployAssembly(workspacePath, t, assemblyItemId, assemblyTypeId, assemblyLocHash, assemblyOnline)
	if err != nil {
		spin.Fail("Deployment failed: ", err)
		os.Exit(1)
	}
	spin.Success(fmt.Sprintf("%s deployed successfully: %s", t, id))
}

func init() {
	assemblyDeployCmd.PersistentFlags().Uint64Var(&assemblyItemId, "item-id", 0, "Unique Item ID for the assembly")
	assemblyDeployCmd.PersistentFlags().Uint64Var(&assemblyTypeId, "type-id", 0, "Type ID for the assembly")
	assemblyDeployCmd.PersistentFlags().StringVar(&assemblyLocHash, "location-hash", "0x0000000000000000000000000000000000000000000000000000000000000000", "Location hash (hex)")
	assemblyDeployCmd.PersistentFlags().BoolVar(&assemblyOnline, "online", false, "Automatically online the assembly after deployment")
	assemblyDeployCmd.PersistentFlags().StringVar(&assemblyOnBehalfOf, "on-behalf-of", "", "Character alias or ID (optional)")

	assemblyDeployCmd.AddCommand(deployGateCmd, deployTurretCmd, deployStorageCmd)
	assemblyAuthorizeCmd.Flags().StringVar(&asmType, "type", "gate", "Assembly type (gate, turret, storage)")
	assemblyCmd.AddCommand(assemblyDeployCmd, assemblyAuthorizeCmd)
	envCmd.AddCommand(assemblyCmd)
}
