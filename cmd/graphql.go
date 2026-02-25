package cmd

import (
	"github.com/spf13/cobra"
)

var graphqlEndpoint string

var graphqlCmd = &cobra.Command{
	Use:   "graphql",
	Short: "Interact with the Sui GraphQL RPC",
	Long:  `Executes queries against the local or remote Sui GraphQL RPC endpoint.`,
}

func init() {
	graphqlCmd.PersistentFlags().StringVarP(&graphqlEndpoint, "endpoint", "e", "http://localhost:9125/graphql", "Sui GraphQL RPC endpoint")
	rootCmd.AddCommand(graphqlCmd)
}
