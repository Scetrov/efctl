package setup

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"efctl/pkg/ui"
	"github.com/jedib0t/go-pretty/v6/table"
)

// ExtractedObjectIds represents the JSON structure from world-object-ids.json
type ExtractedObjectIds struct {
	Network string `json:"network"`
	World   struct {
		PackageId             string `json:"packageId"`
		GovernorCap           string `json:"governorCap"`
		ServerAddressRegistry string `json:"serverAddressRegistry"`
		AdminAcl              string `json:"adminAcl"`
		ObjectRegistry        string `json:"objectRegistry"`
		EnergyConfig          string `json:"energyConfig"`
		FuelConfig            string `json:"fuelConfig"`
		GateConfig            string `json:"gateConfig"`
	} `json:"world"`
}

var characterRegex = regexp.MustCompile(`Pre-computed Character ID:\s*(0x[a-fA-F0-9]+)`)
var nwnRegex = regexp.MustCompile(`NWN Object Id:\s*(0x[a-fA-F0-9]+)`)
var ssuRegex = regexp.MustCompile(`Storage Unit Object Id:\s*(0x[a-fA-F0-9]+)`)
var gateRegex = regexp.MustCompile(`Gate Object Id:\s*(0x[a-fA-F0-9]+)`)

func PrintDeploymentSummary(workspace string) {
	fmt.Println()
	ui.Info.Println("Generating Deployment Summary...")

	// 1. Read JSON Object IDs
	jsonPath := filepath.Join(workspace, "world-contracts", "deployments", "localnet", "extracted-object-ids.json")
	bytes, err := os.ReadFile(jsonPath) // #nosec G304

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Component Type", "Object ID / Address"})

	t.SetStyle(table.StyleRounded)

	if err == nil {
		var extracted ExtractedObjectIds
		if err := json.Unmarshal(bytes, &extracted); err == nil {
			t.AppendRows([]table.Row{
				{"World Package ID", extracted.World.PackageId},
				{"Governor Cap", extracted.World.GovernorCap},
				{"Admin ACL", extracted.World.AdminAcl},
				{"Object Registry", extracted.World.ObjectRegistry},
			})
		}
	} else {
		ui.Warn.Println("Could not read extracted-object-ids.json, skipping core world IDs...")
	}

	// 2. Read Log File for dynamic resources
	logPath := filepath.Join(workspace, "world-contracts", "deployments", "localnet", "deploy.log")
	file, err := os.Open(logPath) // #nosec G304
	if err == nil {
		defer file.Close()

		scanner := bufio.NewScanner(file)
		var characters []string
		var nwns []string
		var ssus []string
		var gates []string

		for scanner.Scan() {
			line := scanner.Text()

			if match := characterRegex.FindStringSubmatch(line); match != nil {
				characters = append(characters, match[1])
			} else if match := nwnRegex.FindStringSubmatch(line); match != nil {
				nwns = append(nwns, match[1])
			} else if match := ssuRegex.FindStringSubmatch(line); match != nil {
				ssus = append(ssus, match[1])
			} else if match := gateRegex.FindStringSubmatch(line); match != nil {
				gates = append(gates, match[1])
			}
		}

		for i, id := range characters {
			t.AppendRow(table.Row{fmt.Sprintf("Character %d", i+1), id})
		}
		for i, id := range nwns {
			t.AppendRow(table.Row{fmt.Sprintf("Network Node %d", i+1), id})
		}
		for i, id := range ssus {
			t.AppendRow(table.Row{fmt.Sprintf("Smart Storage Unit %d", i+1), id})
		}
		for i, id := range gates {
			t.AppendRow(table.Row{fmt.Sprintf("Smart Gate %d", i+1), id})
		}
	} else {
		ui.Warn.Println("Could not read deploy.log, skipping dynamic resource IDs...")
	}

	t.Render()

	fmt.Println()
	ui.Success.Println("Explore the generated World:")
	fmt.Println("🔗 https://custom.suiscan.xyz/custom/home/?network=http%3A%2F%2Flocalhost%3A9000")
	fmt.Println()
}
