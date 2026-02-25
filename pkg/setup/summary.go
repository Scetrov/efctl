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
var adminRegex = regexp.MustCompile(`Admin Address:\s*(0x[a-fA-F0-9]+)`)
var playerARegex = regexp.MustCompile(`Player A Address:\s*(0x[a-fA-F0-9]+)`)
var playerBRegex = regexp.MustCompile(`Player B Address:\s*(0x[a-fA-F0-9]+)`)

func PrintDeploymentSummary(workspace string) {
	fmt.Println()
	ui.Info.Println("Generating Deployment Summary...")

	tPackages := table.NewWriter()
	tPackages.SetOutputMirror(os.Stdout)
	tPackages.AppendHeader(table.Row{"Package Type", "Package ID"})
	tPackages.SetStyle(table.StyleRounded)

	tObjects := table.NewWriter()
	tObjects.SetOutputMirror(os.Stdout)
	tObjects.AppendHeader(table.Row{"Component Type", "Object ID"})
	tObjects.SetStyle(table.StyleRounded)

	tAddresses := table.NewWriter()
	tAddresses.SetOutputMirror(os.Stdout)
	tAddresses.AppendHeader(table.Row{"Role", "Address"})
	tAddresses.SetStyle(table.StyleRounded)

	extractWorldIds(workspace, tPackages, tObjects)
	extractDynamicIds(workspace, tObjects, tAddresses)

	ui.Info.Println("Packages")
	tPackages.Render()

	ui.Info.Println("Objects")
	tObjects.Render()

	ui.Info.Println("Addresses")
	if tAddresses.Length() > 0 {
		tAddresses.Render()
	} else {
		fmt.Println("No addresses extracted (ensure log parsing is configured)")
	}

	fmt.Println()
	ui.Success.Println("Explore the generated World:")
	fmt.Println("🔗 https://custom.suiscan.xyz/custom/home/?network=http%3A%2F%2Flocalhost%3A9000")
	fmt.Println()
}

func extractWorldIds(workspace string, tPackages, tObjects table.Writer) {
	jsonPath := filepath.Join(workspace, "world-contracts", "deployments", "localnet", "extracted-object-ids.json")
	bytes, err := os.ReadFile(jsonPath) // #nosec G304
	if err != nil {
		ui.Warn.Println("Could not read extracted-object-ids.json, skipping core world IDs...")
		return
	}

	var extracted ExtractedObjectIds
	if err := json.Unmarshal(bytes, &extracted); err == nil {
		tPackages.AppendRow(table.Row{"World Package ID", extracted.World.PackageId})
		tObjects.AppendRows([]table.Row{
			{"Governor Cap", extracted.World.GovernorCap},
			{"Admin ACL", extracted.World.AdminAcl},
			{"Object Registry", extracted.World.ObjectRegistry},
		})
	} else {
		ui.Warn.Println("Failed to parse extracted-object-ids.json...")
	}
}

type ParsedObjIds struct {
	characters []string
	nwns       []string
	ssus       []string
	gates      []string
	admin      []string
	playerA    []string
	playerB    []string
}

func parseDeployLog(scanner *bufio.Scanner) ParsedObjIds {
	var ids ParsedObjIds
	for scanner.Scan() {
		line := scanner.Text()

		if match := characterRegex.FindStringSubmatch(line); match != nil {
			ids.characters = append(ids.characters, match[1])
		} else if match := nwnRegex.FindStringSubmatch(line); match != nil {
			ids.nwns = append(ids.nwns, match[1])
		} else if match := ssuRegex.FindStringSubmatch(line); match != nil {
			ids.ssus = append(ids.ssus, match[1])
		} else if match := gateRegex.FindStringSubmatch(line); match != nil {
			ids.gates = append(ids.gates, match[1])
		} else if match := adminRegex.FindStringSubmatch(line); match != nil {
			ids.admin = append(ids.admin, match[1])
		} else if match := playerARegex.FindStringSubmatch(line); match != nil {
			ids.playerA = append(ids.playerA, match[1])
		} else if match := playerBRegex.FindStringSubmatch(line); match != nil {
			ids.playerB = append(ids.playerB, match[1])
		}
	}
	return ids
}

func extractDynamicIds(workspace string, tObjects, tAddresses table.Writer) {
	logPath := filepath.Join(workspace, "world-contracts", "deployments", "localnet", "deploy.log")
	file, err := os.Open(logPath) // #nosec G304
	if err != nil {
		ui.Warn.Println("Could not read deploy.log, skipping dynamic resource IDs...")
		return
	}
	defer file.Close()

	ids := parseDeployLog(bufio.NewScanner(file))

	for i, id := range ids.characters {
		tObjects.AppendRow(table.Row{fmt.Sprintf("Character %d", i+1), id})
	}
	for i, id := range ids.nwns {
		tObjects.AppendRow(table.Row{fmt.Sprintf("Network Node %d", i+1), id})
	}
	for i, id := range ids.ssus {
		tObjects.AppendRow(table.Row{fmt.Sprintf("Smart Storage Unit %d", i+1), id})
	}
	for i, id := range ids.gates {
		tObjects.AppendRow(table.Row{fmt.Sprintf("Smart Gate %d", i+1), id})
	}

	if len(ids.admin) > 0 {
		tAddresses.AppendRow(table.Row{"Admin", ids.admin[0]})
	}
	if len(ids.playerA) > 0 {
		tAddresses.AppendRow(table.Row{"Player A", ids.playerA[0]})
	}
	if len(ids.playerB) > 0 {
		tAddresses.AppendRow(table.Row{"Player B", ids.playerB[0]})
	}
}
