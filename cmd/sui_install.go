package cmd

import (
	"efctl/pkg/sui"
	"efctl/pkg/ui"
	"github.com/spf13/cobra"
)

var suiInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install suiup and the Sui client",
	Run: func(cmd *cobra.Command, args []string) {
		if !sui.IsSuiUpInstalled() {
			if ui.Confirm("suiup is not installed. Would you like to install it now?") {
				if err := sui.InstallSuiUp(); err != nil {
					ui.Error.Println("Failed to install suiup: " + err.Error())
					return
				}
				ui.Success.Println("suiup installed successfully.")
			} else {
				ui.Warn.Println("suiup installation skipped. Cannot proceed with Sui installation.")
				return
			}
		} else {
			ui.Info.Println("suiup is already installed.")
		}

		if !sui.IsSuiInstalled() {
			if ui.Confirm("Sui client is not installed. Would you like to install it now using suiup?") {
				if err := sui.InstallSui(); err != nil {
					ui.Error.Println("Failed to install Sui: " + err.Error())
					return
				}
				ui.Success.Println("Sui client installed successfully.")
			} else {
				ui.Warn.Println("Sui installation skipped.")
			}
		} else {
			ui.Info.Println("Sui client is already installed.")
		}
	},
}

func init() {
	suiCmd.AddCommand(suiInstallCmd)
}
