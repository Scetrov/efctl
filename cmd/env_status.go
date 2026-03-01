package cmd

import (
	"fmt"
	"os"
	"sort"

	"efctl/pkg/env"
	"efctl/pkg/status"
	"efctl/pkg/ui"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

var envStatusRPCURL string

var envStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show environment status without launching the dashboard",
	Long:  `Shows container status, port usage, chain health, and deployed world metadata in a lightweight non-interactive output.`,
	Run: func(cmd *cobra.Command, args []string) {
		res := env.CheckPrerequisites()
		engine, err := res.Engine()
		if err != nil {
			ui.Warn.Println("Container engine not detected (docker/podman). Container status may be incomplete.")
			engine = ""
		}

		st := status.Gather(engine, workspacePath, envStatusRPCURL)

		renderContainerTable(st.Containers)
		renderPortTable(st.Ports)
		renderChainTable(st.Chain)
		renderWorldTable(st.World)
	},
}

func renderContainerTable(containers []status.ContainerStat) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)
	t.AppendHeader(table.Row{"Container", "Status", "CPU", "Memory"})

	for _, c := range containers {
		t.AppendRow(table.Row{c.Name, c.Status, c.CPU, c.Mem})
	}

	ui.Info.Println("Containers")
	t.Render()
	fmt.Println()
}

func renderPortTable(ports []status.PortStat) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)
	t.AppendHeader(table.Row{"Service", "Port", "State"})

	for _, p := range ports {
		state := "Available"
		if p.InUse {
			state = "In Use"
		}
		t.AppendRow(table.Row{p.Name, p.Port, state})
	}

	ui.Info.Println("Ports")
	t.Render()
	fmt.Println()
}

func renderChainTable(chain status.ChainStat) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)
	t.AppendHeader(table.Row{"RPC", "Checkpoint", "Epoch", "Tx Count"})
	t.AppendRow(table.Row{chain.RPCStatus, chain.Checkpoint, chain.Epoch, chain.TxCount})

	ui.Info.Println("Chain")
	t.Render()
	fmt.Println()
}

func renderWorldTable(world status.WorldInfo) {
	tWorld := table.NewWriter()
	tWorld.SetOutputMirror(os.Stdout)
	tWorld.SetStyle(table.StyleRounded)
	tWorld.AppendHeader(table.Row{"Property", "Value"})

	pkgValue := world.PackageID
	if pkgValue == "" {
		pkgValue = "Not found"
	}
	tWorld.AppendRow(table.Row{"World Package ID", pkgValue})

	ui.Info.Println("World")
	tWorld.Render()
	fmt.Println()

	tObjects := table.NewWriter()
	tObjects.SetOutputMirror(os.Stdout)
	tObjects.SetStyle(table.StyleRounded)
	tObjects.AppendHeader(table.Row{"Object", "ID"})

	objectKeys := make([]string, 0, len(world.Objects))
	for key := range world.Objects {
		objectKeys = append(objectKeys, key)
	}
	sort.Strings(objectKeys)
	for _, key := range objectKeys {
		tObjects.AppendRow(table.Row{key, world.Objects[key]})
	}

	ui.Info.Println("World Objects")
	if tObjects.Length() == 0 {
		fmt.Println("No world objects found.")
	} else {
		tObjects.Render()
	}
	fmt.Println()

	tAddr := table.NewWriter()
	tAddr.SetOutputMirror(os.Stdout)
	tAddr.SetStyle(table.StyleRounded)
	tAddr.AppendHeader(table.Row{"Address Key", "Value"})

	addrKeys := make([]string, 0, len(world.Addresses))
	for key := range world.Addresses {
		addrKeys = append(addrKeys, key)
	}
	sort.Strings(addrKeys)
	for _, key := range addrKeys {
		tAddr.AppendRow(table.Row{key, world.Addresses[key]})
	}

	ui.Info.Println("Addresses")
	if tAddr.Length() == 0 {
		fmt.Println("No addresses found in world-contracts/.env")
	} else {
		tAddr.Render()
	}
	fmt.Println()
}

func init() {
	envStatusCmd.Flags().StringVar(&envStatusRPCURL, "rpc-url", "http://localhost:9000", "Sui JSON-RPC endpoint URL")
	envCmd.AddCommand(envStatusCmd)
}
