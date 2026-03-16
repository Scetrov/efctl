package assembly

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"efctl/pkg/sui"
	"efctl/pkg/ui"
)

type ObjectIds struct {
	WorldPackageId string
	ObjectRegistry string
	AdminAcl       string
	CharacterId    string
	NetworkNodeId  string
}

type AssemblyType string

const (
	TypeGate        AssemblyType = "gate"
	TypeTurret      AssemblyType = "turret"
	TypeStorageUnit AssemblyType = "storage_unit"
)

var (
	characterRegex = regexp.MustCompile(`Pre-computed Character ID:\s*(0x[a-fA-F0-9]+)`)
	nwnRegex       = regexp.MustCompile(`NWN Object Id:\s*(0x[a-fA-F0-9]+)`)
)

func LoadObjectIds(workspace string) (*ObjectIds, error) {
	ids := &ObjectIds{}

	// 1. Load from extracted-object-ids.json
	jsonPath := filepath.Join(workspace, "world-contracts", "deployments", "localnet", "extracted-object-ids.json")
	bytes, err := os.ReadFile(jsonPath) // #nosec G304 -- path constructed from known workspace prefix
	if err != nil {
		return nil, fmt.Errorf("failed to read world IDs: %w", err)
	}

	var extracted struct {
		World struct {
			PackageId      string `json:"packageId"`
			AdminAcl       string `json:"adminAcl"`
			ObjectRegistry string `json:"objectRegistry"`
		} `json:"world"`
	}
	if err := json.Unmarshal(bytes, &extracted); err != nil {
		return nil, fmt.Errorf("failed to parse world IDs: %w", err)
	}

	ids.WorldPackageId = extracted.World.PackageId
	ids.AdminAcl = extracted.World.AdminAcl
	ids.ObjectRegistry = extracted.World.ObjectRegistry

	// 2. Load from deploy.log (for character and NWN)
	logPath := filepath.Join(workspace, "world-contracts", "deployments", "localnet", "deploy.log")
	file, err := os.Open(logPath) // #nosec G304 -- path constructed from known workspace prefix
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if match := characterRegex.FindStringSubmatch(line); match != nil {
				ids.CharacterId = match[1]
			} else if match := nwnRegex.FindStringSubmatch(line); match != nil {
				ids.NetworkNodeId = match[1]
			}
		}
	}

	return ids, nil
}

func DeployAssembly(workspace string, assemblyType AssemblyType, itemId, typeId uint64, locationHash string, online bool) (string, error) {
	ids, err := LoadObjectIds(workspace)
	if err != nil {
		return "", err
	}

	ui.Info.Printf("Deploying %s (ItemId: %d, TypeId: %d)...\n", assemblyType, itemId, typeId)

	module := string(assemblyType)
	if assemblyType == TypeStorageUnit {
		module = "storage_unit"
	}

	// 1. anchor
	args := []string{
		"--package", ids.WorldPackageId,
		"--module", module,
		"--function", "anchor",
		"--args",
		ids.ObjectRegistry,
		ids.NetworkNodeId,
		ids.CharacterId,
		ids.AdminAcl,
		fmt.Sprintf("%d", itemId),
		fmt.Sprintf("%d", typeId),
		fmt.Sprintf("vector<u8>:%s", locationHash),
	}

	out, err := sui.CallMove(args, "50000000")
	if err != nil {
		return "", fmt.Errorf("anchor failed: %w\nOutput: %s", err, out)
	}

	// Extract Assembly ID and OwnerCap ID from events
	assemblyId, ownerCapId, err := extractCreatedIds(out, module)
	if err != nil {
		return "", err
	}

	// 2. share_*
	shareFunc := "share_" + module
	if assemblyType == TypeTurret {
		shareFunc = "share_turret" // Turret might have different share function or use common one
	}

	shareArgs := []string{
		"--package", ids.WorldPackageId,
		"--module", module,
		"--function", shareFunc,
		"--args",
		assemblyId,
		ids.AdminAcl,
	}

	_, err = sui.CallMove(shareArgs, "10000000")
	if err != nil {
		return "", fmt.Errorf("sharing %s failed: %w", assemblyType, err)
	}

	if online {
		if err := OnlineAssembly(workspace, assemblyType, assemblyId, ids.NetworkNodeId, ownerCapId); err != nil {
			ui.Warn.Printf("Deployment succeeded but onlining failed: %v\n", err)
		} else {
			ui.Success.Println("Assembly onlined successfully.")
		}
	}

	return assemblyId, nil
}

func OnlineAssembly(workspace string, assemblyType AssemblyType, assemblyId, nwnId, ownerCapId string) error {
	ids, err := LoadObjectIds(workspace)
	if err != nil {
		return err
	}

	// For online we need EnergyConfig as well
	jsonPath := filepath.Join(workspace, "world-contracts", "deployments", "localnet", "extracted-object-ids.json")
	bytes, _ := os.ReadFile(jsonPath) // #nosec G304 -- path constructed from known workspace prefix
	var extracted struct {
		World struct {
			EnergyConfig string `json:"energyConfig"`
		} `json:"world"`
	}
	json.Unmarshal(bytes, &extracted)

	module := string(assemblyType)

	// Transaction: character::borrow_owner_cap -> module::online -> character::return_owner_cap
	// Since we are using the CLI, it's easier to use the OwnerCap directly if we have it or can find it.
	// In local dev we usually have the OwnerCap as a standalone object if not borrowed yet.

	args := []string{
		"--package", ids.WorldPackageId,
		"--module", module,
		"--function", "online",
		"--args",
		assemblyId,
		nwnId,
		extracted.World.EnergyConfig,
		ownerCapId,
	}

	_, err = sui.CallMove(args, "20000000")
	return err
}

func AuthorizeExtension(workspace string, assemblyType AssemblyType, assemblyId, extensionConfigId string) error {
	ids, err := LoadObjectIds(workspace)
	if err != nil {
		return err
	}

	// We need to find the OwnerCap for this assembly.
	// For simplicity in this implementation, we assume the user provides it or we find it in the character's inventory.
	// A more robust way would be to query the blockchain via GraphQL or Sui CLI for owned objects.

	// For now, let's try to find it via character borrowing flow if we want to be "correct",
	// but the user's request just asked to "authorize an extension".

	// The TS scripts use a specific AuthType: ${builderPackageId}::extension::XAuth
	// We need the builder package ID.
	envPath := filepath.Join(workspace, "builder-scaffold", ".env")
	builderPackageId := ""
	if bytes, err := os.ReadFile(envPath); err == nil { // #nosec G304 -- path constructed from known workspace prefix
		re := regexp.MustCompile(`BUILDER_PACKAGE_ID=(0x[a-fA-F0-9]+)`)
		if match := re.FindStringSubmatch(string(bytes)); match != nil {
			builderPackageId = match[1]
		}
	}

	if builderPackageId == "" {
		return fmt.Errorf("BUILDER_PACKAGE_ID not found in builder-scaffold/.env")
	}

	authType := fmt.Sprintf("%s::extension::XAuth", builderPackageId)

	ui.Info.Printf("Authorizing extension %s for %s %s...\n", authType, assemblyType, assemblyId)

	// We need the OwnerCap. In a real scenario, we'd query for it.
	// For the initial implementation, let's assume we need to borrow it from the character.

	module := string(assemblyType)
	// typeArg := fmt.Sprintf("%s::%s::%s", ids.WorldPackageId, module, strings.Title(module))
	// if assemblyType == TypeStorageUnit {
	// 	typeArg = fmt.Sprintf("%s::storage_unit::StorageUnit", ids.WorldPackageId)
	// }

	// This is a complex PTB. The Sui CLI 'call' command is Limited for PTBs.
	// We might need to use a temporary TS script or a more advanced Go Sui SDK if we want complex PTBs.
	// However, we can try to do it in steps if the Move functions allow, or use a simplified approach for local dev.

	// Let's check if there's a simpler 'authorize' function or if we must use the borrow flow.
	// The Move code: public fun authorize_extension<Auth: drop>(gate: &mut Gate, owner_cap: &OwnerCap<Gate>)

	// If the user has the OwnerCap in their address (not tucked in character), it's easy.
	// In the seed script, OwnerCaps are transferred to the character address.

	// Let's try to find the OwnerCap ID first.
	ownerCapId := findOwnerCap(ids.CharacterId, assemblyId)
	if ownerCapId == "" {
		return fmt.Errorf("could not find OwnerCap for assembly %s owned by %s", assemblyId, ids.CharacterId)
	}

	args := []string{
		"--package", ids.WorldPackageId,
		"--module", module,
		"--function", "authorize_extension",
		"--type-args", authType,
		"--args",
		assemblyId,
		ownerCapId,
	}

	_, err = sui.CallMove(args, "20000000")
	return err
}

func extractCreatedIds(jsonOut, module string) (assemblyId, ownerCapId string, err error) {
	var data struct {
		Events []struct {
			Type   string                 `json:"type"`
			Parsed map[string]interface{} `json:"parsedJson"`
		} `json:"events"`
	}

	if err := json.Unmarshal([]byte(jsonOut), &data); err != nil {
		return "", "", fmt.Errorf("failed to parse event JSON: %w", err)
	}

	eventSuffix := "::" + module + "::" + strings.Title(module) + "CreatedEvent"
	if module == "storage_unit" {
		eventSuffix = "::storage_unit::StorageUnitCreatedEvent"
	}

	for _, e := range data.Events {
		if strings.HasSuffix(e.Type, eventSuffix) {
			if id, ok := e.Parsed["assembly_id"].(string); ok {
				assemblyId = id
			}
			if capId, ok := e.Parsed["owner_cap_id"].(string); ok {
				ownerCapId = capId
			}
			return
		}
	}

	return "", "", fmt.Errorf("CreatedEvent not found in transaction output")
}

func findOwnerCap(ownerAddr, assemblyId string) string {
	// sui client objects --json
	executor := &sui.DefaultExecutor{}
	out, err := executor.ExecCapture("sui", "client", "objects", "--json")
	if err != nil {
		return ""
	}

	var objects []struct {
		Data struct {
			ObjectId string `json:"objectId"`
			Content  struct {
				Type   string                 `json:"type"`
				Fields map[string]interface{} `json:"fields"`
			} `json:"content"`
		} `json:"data"`
	}

	if err := json.Unmarshal([]byte(out), &objects); err != nil {
		return ""
	}

	for _, obj := range objects {
		if strings.Contains(obj.Data.Content.Type, "OwnerCap") {
			if target, ok := obj.Data.Content.Fields["authorized_object_id"].(string); ok && target == assemblyId {
				return obj.Data.ObjectId
			}
		}
	}

	return ""
}
