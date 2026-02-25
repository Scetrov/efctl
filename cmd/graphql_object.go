package cmd

import (
	"os"

	"efctl/pkg/graphql"
	"efctl/pkg/ui"
	"github.com/spf13/cobra"
)

var graphqlObjectCmd = &cobra.Command{
	Use:   "object [id]",
	Short: "Query an object by ID",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		ui.Info.Printf("Querying object %s at %s...\n", id, graphqlEndpoint)

		if err := graphql.QueryObject(graphqlEndpoint, id); err != nil {
			ui.Error.Println("GraphQL query failed: " + err.Error())
			os.Exit(1)
		}
	},
}

func init() {
	graphqlCmd.AddCommand(graphqlObjectCmd)
}
