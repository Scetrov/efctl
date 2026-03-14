package cmd

import (
	"efctl/pkg/sui"
	"efctl/pkg/ui"
	"github.com/spf13/cobra"
)

var faucetAddr string
var faucetUrl string

var envFaucetCmd = &cobra.Command{
	Use:   "faucet",
	Short: "Request gas from the local faucet",
	Long:  `Request gas coins from the local Sui faucet for a specific address.`,
	Run: func(cmd *cobra.Command, args []string) {
		spin, _ := ui.Spin("Requesting gas from faucet...")
		err := sui.RequestFaucet(faucetUrl, faucetAddr)
		if err != nil {
			spin.Fail("Faucet request failed: ", err)
			return
		}
		spin.Success("Gas request successful for ", faucetAddr)
	},
}

func init() {
	envFaucetCmd.Flags().StringVarP(&faucetAddr, "address", "a", "", "The address to receive gas (required)")
	_ = envFaucetCmd.MarkFlagRequired("address")
	envFaucetCmd.Flags().StringVar(&faucetUrl, "faucet-url", "http://localhost:9123", "The URL of the faucet")

	envCmd.AddCommand(envFaucetCmd)
}
