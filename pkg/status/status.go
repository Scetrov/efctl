package status

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"efctl/pkg/container"
	"efctl/pkg/env"
	"efctl/pkg/sui"
)

type ContainerStat struct {
	Name   string
	Status string
	CPU    string
	Mem    string
}

type PortStat struct {
	Name  string
	Port  int
	InUse bool
}

type ChainStat struct {
	RPCStatus  string
	Checkpoint string
	Epoch      string
	TxCount    string
}

type DiscoveredObject struct {
	ID   string
	Type string
	Name string
}

type WorldInfo struct {
	PackageID      string
	DiscoveredPkgs []DiscoveredPackage
	Objects        map[string]string
	Addresses      map[string]string
	Assemblies     []DiscoveredObject
	Extensions     []DiscoveredObject
	DiscoveryErr   string
}

type EnvironmentStatus struct {
	Containers []ContainerStat
	Ports      []PortStat
	Chain      ChainStat
	World      WorldInfo
}

func Gather(engine, workspace, rpcURL string) EnvironmentStatus {
	return EnvironmentStatus{
		Containers: GatherContainerStats(engine),
		Ports: []PortStat{
			{Name: "Sui RPC", Port: 9000, InUse: !env.IsPortAvailable(9000)},
			{Name: "GraphQL", Port: 9125, InUse: !env.IsPortAvailable(9125)},
			{Name: "PostgreSQL", Port: 5432, InUse: !env.IsPortAvailable(5432)},
			{Name: "Frontend", Port: 5173, InUse: !env.IsPortAvailable(5173)},
		},
		Chain: GatherChainHealth(rpcURL),
		World: GatherWorldInfo(workspace, rpcURL),
	}
}

func GatherContainerStats(engine string) []ContainerStat {
	sui := ContainerStat{Name: container.ContainerSuiPlayground, Status: "Stopped", CPU: "-", Mem: "-"}
	pg := ContainerStat{Name: container.ContainerPostgres, Status: "Stopped", CPU: "-", Mem: "-"}
	fe := ContainerStat{Name: container.ContainerFrontend, Status: "Stopped", CPU: "-", Mem: "-"}

	if engine == "" {
		return []ContainerStat{sui, pg, fe}
	}

	out, err := exec.Command(engine, "stats", "--no-stream", "--format", "{{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}").Output() // #nosec G204 -- engine is validated by env.CheckPrerequisites().Engine() to be "docker" or "podman"
	if err == nil {
		sui, pg, fe = parseStatsOutput(string(out), sui, pg, fe)
	}

	if sui.Status == "Stopped" && containerRunning(engine, container.ContainerSuiPlayground) {
		sui.Status = "Running"
	}
	if pg.Status == "Stopped" && containerRunning(engine, container.ContainerPostgres) {
		pg.Status = "Running"
	}
	if fe.Status == "Stopped" && containerRunning(engine, container.ContainerFrontend) {
		fe.Status = "Running"
	}

	return []ContainerStat{sui, pg, fe}
}

func parseStatsOutput(out string, sui, pg, fe ContainerStat) (ContainerStat, ContainerStat, ContainerStat) {
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		cpu := strings.TrimSpace(parts[1])
		mem := strings.TrimSpace(parts[2])

		switch name {
		case container.ContainerSuiPlayground:
			sui.Status = "Running"
			sui.CPU = cpu
			sui.Mem = mem
		case container.ContainerPostgres:
			pg.Status = "Running"
			pg.CPU = cpu
			pg.Mem = mem
		case container.ContainerFrontend:
			fe.Status = "Running"
			fe.CPU = cpu
			fe.Mem = mem
		}
	}
	return sui, pg, fe
}

func containerRunning(engine, name string) bool {
	out, err := exec.Command(engine, "inspect", "--format", "{{.State.Running}}", name).Output() // #nosec G204 -- engine is validated by env.CheckPrerequisites().Engine() to be "docker" or "podman"
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

func GatherChainHealth(rpcURL string) ChainStat {
	result := ChainStat{RPCStatus: "Offline", Checkpoint: "-", Epoch: "-", TxCount: "-"}
	client := &http.Client{Timeout: 1 * time.Second}

	var checkpoint string
	if err := rpcCall(client, rpcURL, `{"jsonrpc":"2.0","id":1,"method":"sui_getLatestCheckpointSequenceNumber","params":[]}`, &checkpoint); err == nil {
		result.Checkpoint = checkpoint
		result.RPCStatus = "Healthy"
	}

	var txCount string
	if err := rpcCall(client, rpcURL, `{"jsonrpc":"2.0","id":1,"method":"sui_getTotalTransactionBlocks","params":[]}`, &txCount); err == nil {
		result.TxCount = txCount
	}

	var epochRes struct {
		Epoch string `json:"epoch"`
	}
	if err := rpcCall(client, rpcURL, `{"jsonrpc":"2.0","id":1,"method":"sui_getLatestSuiSystemState","params":[]}`, &epochRes); err == nil {
		if epochRes.Epoch != "" {
			result.Epoch = epochRes.Epoch
		}
	}

	return result
}

func rpcCall(client *http.Client, rpcURL, payload string, result interface{}) error {
	req, err := http.NewRequest("POST", rpcURL, strings.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req) // #nosec G107 -- rpcURL is CLI input and intentionally configurable
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var envelope struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return err
	}
	if len(envelope.Result) == 0 {
		return fmt.Errorf("empty result")
	}
	return json.Unmarshal(envelope.Result, result)
}

func GatherWorldInfo(workspace, rpcURL string) WorldInfo {
	envVars := extractEnvVars(workspace)
	addresses := extractAddresses(envVars)
	objs, pkgID := extractWorldObjects(workspace)

	// Try to find builder package ID in multiple locations
	builderPkgID := extractBuilderPackageID(workspace)

	info := WorldInfo{
		PackageID:      pkgID,
		DiscoveredPkgs: []DiscoveredPackage{}, // Will be populated below
		Objects:        objs,
		Addresses:      addresses,
	}

	// Dynamic discovery via GraphQL if available
	// Shift port from 9000 (RPC) to 9125 (GraphQL)
	gqlURL := strings.Replace(rpcURL, ":9000", ":9125", 1)
	if !strings.HasSuffix(gqlURL, "/graphql") {
		gqlURL = strings.TrimSuffix(gqlURL, "/") + "/graphql"
	}

	assemblies, errA := DiscoverAssemblies(gqlURL, pkgID)
	if errA != nil {
		info.DiscoveryErr = fmt.Sprintf("Assemblies: %v", errA)
	}
	info.Assemblies = assemblies

	// Discovered all packages owned by our registry addresses
	ownerAddresses := make([]string, 0, len(addresses))
	for _, addr := range addresses {
		ownerAddresses = append(ownerAddresses, addr)
	}
	discoveredPkgs, errP := DiscoverPackages(gqlURL, ownerAddresses)
	if errP != nil {
		if info.DiscoveryErr != "" {
			info.DiscoveryErr += "; "
		}
		info.DiscoveryErr += fmt.Sprintf("Package discovery: %v", errP)
	}

	// Merge with builderPkgID from .env/Pub.toml as a fallback
	pkgMap := make(map[string]bool)
	var allPkgs []DiscoveredPackage
	var allPkgIDs []string
	for _, p := range discoveredPkgs {
		pkgMap[p.ID] = true
		allPkgs = append(allPkgs, p)
		allPkgIDs = append(allPkgIDs, p.ID)
	}
	if builderPkgID != "" && !pkgMap[builderPkgID] {
		allPkgs = append(allPkgs, DiscoveredPackage{
			ID:      builderPkgID,
			Version: "1",
			Owner:   "local .env",
		})
		allPkgIDs = append(allPkgIDs, builderPkgID)
	}

	info.DiscoveredPkgs = allPkgs

	extensions, errE := DiscoverExtensions(gqlURL, allPkgIDs)
	if errE != nil {
		if info.DiscoveryErr != "" {
			info.DiscoveryErr += "; "
		}
		info.DiscoveryErr += fmt.Sprintf("Extensions: %v", errE)
	}
	info.Extensions = extensions

	return info
}

func extractEnvVars(workspace string) map[string]string {
	result := make(map[string]string)
	envPath := filepath.Join(workspace, "world-contracts", ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		// Fallback for test environments where contracts might be in a subfolder
		envPath = filepath.Join(workspace, "test-env", "world-contracts", ".env")
	}

	data, err := os.ReadFile(envPath) // #nosec G304 -- path is workspace-relative by design
	if err != nil {
		return result
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

func extractAddresses(envVars map[string]string) map[string]string {
	addresses := make(map[string]string)

	// Well-known mappings
	roleMap := map[string]string{
		"ADMIN_ADDRESS":    "Admin",
		"SPONSOR_ADDRESS":  "Sponsor",
		"PLAYER_A_ADDRESS": "Player A",
		"PLAYER_B_ADDRESS": "Player B",
	}

	// First pass: try to get explicit addresses
	for envKey, role := range roleMap {
		if addr, ok := envVars[envKey]; ok && addr != "" {
			addresses[role] = addr
		}
	}

	// Second pass: derive from private keys if missing
	keyToRole := map[string]string{
		"ADMIN_PRIVATE_KEY":    "Admin",
		"PLAYER_A_PRIVATE_KEY": "Player A",
		"PLAYER_B_PRIVATE_KEY": "Player B",
	}
	for keyVar, role := range keyToRole {
		if _, exists := addresses[role]; !exists {
			if privKey, ok := envVars[keyVar]; ok && privKey != "" {
				if addr, err := sui.DeriveAddressFromPrivateKey(privKey); err == nil {
					addresses[role] = addr
				}
			}
		}
	}

	// Final pass: catch any other _ADDRESS variables
	keys := make([]string, 0, len(envVars))
	for key := range envVars {
		if strings.HasSuffix(key, "_ADDRESS") {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		// Only add if not already mapped to a role
		isMapped := false
		for envKey := range roleMap {
			if key == envKey {
				isMapped = true
				break
			}
		}
		if !isMapped {
			addresses[key] = envVars[key]
		}
	}

	return addresses
}

func extractWorldObjects(workspace string) (map[string]string, string) {
	objs := make(map[string]string)
	filePath := filepath.Join(workspace, "world-contracts", "deployments", "localnet", "extracted-object-ids.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		filePath = filepath.Join(workspace, "test-env", "world-contracts", "deployments", "localnet", "extracted-object-ids.json")
	}

	data, err := os.ReadFile(filePath) // #nosec G304 -- path is workspace-relative by design
	if err != nil {
		return objs, ""
	}

	var top map[string]interface{}
	if err := json.Unmarshal(data, &top); err != nil {
		return objs, ""
	}

	world, ok := top["world"].(map[string]interface{})
	if !ok {
		return objs, ""
	}

	pkgID := ""
	for key, value := range world {
		strValue, ok := value.(string)
		if !ok {
			continue
		}
		if key == "packageId" {
			pkgID = strValue
			continue
		}
		objs[key] = strValue
	}
	return objs, pkgID
}
func extractBuilderPackageID(workspace string) string {
	// 1. Try builder-scaffold/.env
	builderEnvPath := filepath.Join(workspace, "builder-scaffold", ".env")
	if id := extractIDFromEnv(builderEnvPath, "BUILDER_PACKAGE_ID"); id != "" {
		return id
	}

	// 2. Try world-contracts/.env
	worldEnvPath := filepath.Join(workspace, "world-contracts", ".env")
	if id := extractIDFromEnv(worldEnvPath, "BUILDER_PACKAGE_ID"); id != "" {
		return id
	}

	// 3. Try Pub.extension.toml in builder-scaffold deployments
	// We check both localnet and testnet based on what's available
	for _, network := range []string{"localnet", "testnet"} {
		pubPath := filepath.Join(workspace, "builder-scaffold", "deployments", network, "Pub.extension.toml")
		if data, err := os.ReadFile(pubPath); err == nil { // #nosec G304 -- path constructed from known workspace prefix
			re := regexp.MustCompile(`(?m)published-at\s*=\s*"(0x[a-fA-F0-9]+)"`)
			if matches := re.FindStringSubmatch(string(data)); len(matches) > 1 {
				return matches[1]
			}
		}
	}

	return ""
}

func extractIDFromEnv(path, key string) string {
	data, err := os.ReadFile(path) // #nosec G304 -- path passed from internal discovery logic
	if err != nil {
		return ""
	}
	// Flexible regex: handle spaces, quotes, and optional prefix
	pattern := fmt.Sprintf(`(?m)^\s*%s\s*=\s*["']?(0x[a-fA-F0-9]+)["']?`, key)
	re := regexp.MustCompile(pattern)
	if matches := re.FindStringSubmatch(string(data)); len(matches) > 1 {
		return matches[1]
	}
	return ""
}
