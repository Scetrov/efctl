package cmd

import (
	"github.com/spf13/cobra"
)

var worldCmd = &cobra.Command{
	Use:   "world",
	Short: "Interact with the EVE Frontier local world contracts",
	Long:  `Provides tools to query and interact with the localnet world contracts and smart assemblies.`,
}

var Network string

func init() {
	worldCmd.PersistentFlags().StringVarP(&GraphqlEndpoint, "endpoint", "e", "http://localhost:9125/graphql", "Sui GraphQL RPC endpoint")
	worldCmd.PersistentFlags().StringVarP(&Network, "network", "n", "localnet", "The network to query (localnet, devnet, testnet, mainnet)")
	rootCmd.AddCommand(worldCmd)
}
