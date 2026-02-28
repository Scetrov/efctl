package setup

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

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

var adminAddressRegex = regexp.MustCompile(`^ADMIN_ADDRESS=(0x[a-fA-F0-9]+)`)
var adminKeyRegex = regexp.MustCompile(`^ADMIN_PRIVATE_KEY=(suiprivkey[a-z0-9]+)`)
var playerAAddressRegex = regexp.MustCompile(`^PLAYER_A_ADDRESS=(0x[a-fA-F0-9]+)`)
var playerAKeyRegex = regexp.MustCompile(`^PLAYER_A_PRIVATE_KEY=(suiprivkey[a-z0-9]+)`)
var playerBAddressRegex = regexp.MustCompile(`^PLAYER_B_ADDRESS=(0x[a-fA-F0-9]+)`)
var playerBKeyRegex = regexp.MustCompile(`^PLAYER_B_PRIVATE_KEY=(suiprivkey[a-z0-9]+)`)

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

	// Check if frontend is enabled by looking at the override file
	overridePath := filepath.Join(workspace, "builder-scaffold", "docker", "docker-compose.override.yml")
	if data, err := os.ReadFile(overridePath); err == nil { // #nosec G304
		if strings.Contains(string(data), "frontend:") {
			fmt.Println("💻 Frontend dApp: http://localhost:5173")
		}
	}

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
		}
	}
	return ids
}

type ParsedEnv struct {
	adminAddress   string
	adminKey       string
	playerAAddress string
	playerAKey     string
	playerBAddress string
	playerBKey     string
}

func parseEnvLog(scanner *bufio.Scanner) ParsedEnv {
	var env ParsedEnv
	for scanner.Scan() {
		line := scanner.Text()
		if match := adminAddressRegex.FindStringSubmatch(line); match != nil {
			env.adminAddress = match[1]
		} else if match := adminKeyRegex.FindStringSubmatch(line); match != nil {
			env.adminKey = match[1]
		} else if match := playerAAddressRegex.FindStringSubmatch(line); match != nil {
			env.playerAAddress = match[1]
		} else if match := playerAKeyRegex.FindStringSubmatch(line); match != nil {
			env.playerAKey = match[1]
		} else if match := playerBAddressRegex.FindStringSubmatch(line); match != nil {
			env.playerBAddress = match[1]
		} else if match := playerBKeyRegex.FindStringSubmatch(line); match != nil {
			env.playerBKey = match[1]
		}
	}
	return env
}

func extractDynamicIds(workspace string, tObjects, tAddresses table.Writer) {
	extractDeployLogIds(workspace, tObjects)
	extractEnvAddresses(workspace, tAddresses)
}

func extractDeployLogIds(workspace string, tObjects table.Writer) {
	logPath := filepath.Join(workspace, "world-contracts", "deployments", "localnet", "deploy.log")
	file, err := os.Open(logPath) // #nosec G304
	if err == nil {
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
	} else {
		ui.Warn.Println("Could not read deploy.log, skipping dynamic resource IDs...")
	}
}

func extractEnvAddresses(workspace string, tAddresses table.Writer) {
	envPath := filepath.Join(workspace, "world-contracts", ".env")
	envFile, err := os.Open(envPath) // #nosec G304
	if err == nil {
		defer envFile.Close()
		env := parseEnvLog(bufio.NewScanner(envFile))

		appendRoleAddress(tAddresses, "Admin", "ef-admin", env.adminAddress, env.adminKey)
		appendRoleAddress(tAddresses, "Player A", "ef-player-a", env.playerAAddress, env.playerAKey)
		appendRoleAddress(tAddresses, "Player B", "ef-player-b", env.playerBAddress, env.playerBKey)
	} else {
		ui.Warn.Println("Could not read .env, skipping addresses...")
	}
}

func appendRoleAddress(t table.Writer, role, alias, address, key string) {
	if address != "" {
		t.AppendRow(table.Row{role, address})
		return
	}
	addr := resolveAddress(alias)
	if addr != "" {
		t.AppendRow(table.Row{role, addr})
	} else if key != "" {
		t.AppendRow(table.Row{role, "(Derived via Key: " + key[:16] + "...)"})
	}
}

func resolveAddress(alias string) string {
	// sui client addresses --json
	out, err := exec.Command("sui", "client", "addresses", "--json").Output()
	if err != nil {
		return ""
	}

	// Sui 1.66 JSON structure: {"activeAddress": "...", "addresses": [["alias", "0x..."], ...]}
	var data struct {
		Addresses [][]string `json:"addresses"`
	}
	if err := json.Unmarshal(out, &data); err != nil {
		// Fallback for older versions which might return a simple map[string]string or similar
		var fallback map[string]string
		if err := json.Unmarshal(out, &fallback); err == nil {
			for addr, a := range fallback {
				if a == alias || addr == alias {
					return addr
				}
			}
		}
		return ""
	}

	for _, pair := range data.Addresses {
		if len(pair) >= 2 {
			if pair[0] == alias {
				return pair[1]
			}
		}
	}

	return ""
}
