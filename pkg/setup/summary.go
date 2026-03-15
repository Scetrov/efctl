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

	"efctl/pkg/sui"
	"efctl/pkg/ui"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/pterm/pterm"
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

var adminAddressRegex = regexp.MustCompile(`ADMIN_ADDRESS\s*=\s*["']?(0x[a-fA-F0-9]+)["']?`)
var adminKeyRegex = regexp.MustCompile(`ADMIN_PRIVATE_KEY\s*=\s*["']?(suiprivkey[a-zA-Z0-9]+)["']?`)
var playerAAddressRegex = regexp.MustCompile(`PLAYER_A_ADDRESS\s*=\s*["']?(0x[a-fA-F0-9]+)["']?`)
var playerAKeyRegex = regexp.MustCompile(`PLAYER_A_PRIVATE_KEY\s*=\s*["']?(suiprivkey[a-zA-Z0-9]+)["']?`)
var playerBAddressRegex = regexp.MustCompile(`PLAYER_B_ADDRESS\s*=\s*["']?(0x[a-fA-F0-9]+)["']?`)
var playerBKeyRegex = regexp.MustCompile(`PLAYER_B_PRIVATE_KEY\s*=\s*["']?(suiprivkey[a-zA-Z0-9]+)["']?`)

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

	extractWorldIds(workspace, tPackages, tObjects)
	addresses := extractDynamicIds(workspace, tObjects)

	ui.Info.Println("Packages")
	tPackages.Render()

	ui.Info.Println("Objects")
	tObjects.Render()

	ui.Info.Println("Addresses")
	if len(addresses) > 0 {
		width := pterm.GetTerminalWidth()
		if width < 150 {
			for _, addr := range addresses {
				fmt.Printf("Role:        %s\n", addr.Role)
				fmt.Printf("Address:     %s\n", addr.Address)
				fmt.Printf("Private Key: %s\n", addr.Key)
				fmt.Println("---")
			}
		} else {
			tAddresses := table.NewWriter()
			tAddresses.SetOutputMirror(os.Stdout)
			tAddresses.AppendHeader(table.Row{"Role", "Address", "Private Key"})
			tAddresses.SetStyle(table.StyleRounded)
			for _, addr := range addresses {
				tAddresses.AppendRow(table.Row{addr.Role, addr.Address, addr.Key})
			}
			tAddresses.Render()
		}
	} else {
		fmt.Println("No addresses extracted (ensure log parsing is configured)")
	}

	fmt.Println()
	ui.Success.Println("Explore the generated World:")
	fmt.Println("🔗 https://custom.suiscan.xyz/custom/home/?network=http%3A%2F%2Flocalhost%3A9000")

	// Check if optional services are enabled by looking at the override file
	overridePath := filepath.Join(workspace, "builder-scaffold", "docker", "docker-compose.override.yml")
	if data, err := os.ReadFile(overridePath); err == nil { // #nosec G304 -- path is filepath.Join(workspace, hardcoded-sub-path); workspace is set by the user's own config
		content := string(data)
		if strings.Contains(content, "postgres:") || strings.Contains(content, "SUI_GRAPHQL_ENABLED") {
			fmt.Println("📊 GraphQL API:   http://localhost:9125/graphql")
		}
		if strings.Contains(content, "frontend:") {
			fmt.Println("💻 Frontend dApp: http://localhost:5173")
		}
	}

	fmt.Println()
}

func extractWorldIds(workspace string, tPackages, tObjects table.Writer) {
	jsonPath := filepath.Join(workspace, "world-contracts", "deployments", "localnet", "extracted-object-ids.json")
	bytes, err := os.ReadFile(jsonPath) // #nosec G304 -- path is filepath.Join(workspace, hardcoded-sub-path); workspace is set by the user's own config
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

func extractDynamicIds(workspace string, tObjects table.Writer) []AddressInfo {
	extractDeployLogIds(workspace, tObjects)
	return extractEnvAddresses(workspace)
}

func extractDeployLogIds(workspace string, tObjects table.Writer) {
	logPath := filepath.Join(workspace, "world-contracts", "deployments", "localnet", "deploy.log")
	file, err := os.Open(logPath) // #nosec G304 -- path is filepath.Join(workspace, hardcoded-sub-path); workspace is set by the user's own config
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

type AddressInfo struct {
	Role    string
	Address string
	Key     string
}

func extractEnvAddresses(workspace string) []AddressInfo {
	var addresses []AddressInfo
	envPath := filepath.Join(workspace, "world-contracts", ".env")
	envFile, err := os.Open(envPath) // #nosec G304 -- path is filepath.Join(workspace, hardcoded-sub-path); workspace is set by the user's own config
	if err == nil {
		defer envFile.Close()
		env := parseEnvLog(bufio.NewScanner(envFile))

		addresses = append(addresses, deriveRoleAddress("Admin", "ef-admin", env.adminAddress, env.adminKey))
		addresses = append(addresses, deriveRoleAddress("Player A", "ef-player-a", env.playerAAddress, env.playerAKey))
		addresses = append(addresses, deriveRoleAddress("Player B", "ef-player-b", env.playerBAddress, env.playerBKey))
	} else {
		ui.Warn.Println("Could not read .env, skipping addresses...")
	}
	return addresses
}

func deriveRoleAddress(role, alias, address, key string) AddressInfo {
	addr := address
	if addr == "" {
		addr = resolveAddress(alias)
	}
	if addr == "" && key != "" {
		addr = deriveAddress(key)
	}
	if addr == "" {
		addr = "N/A"
	}
	return AddressInfo{Role: role, Address: addr, Key: key}
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

func deriveAddress(key string) string {
	addr, err := sui.DeriveAddressFromPrivateKey(key)
	if err != nil {
		ui.Debug.Println(fmt.Sprintf("Failed to derive address from key: %v", err))
		return ""
	}
	return addr
}
