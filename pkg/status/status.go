package status

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"efctl/pkg/container"
	"efctl/pkg/env"
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

type WorldInfo struct {
	PackageID string
	Objects   map[string]string
	Addresses map[string]string
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
		World: GatherWorldInfo(workspace),
	}
}

func GatherContainerStats(engine string) []ContainerStat {
	sui := ContainerStat{Name: container.ContainerSuiPlayground, Status: "Stopped", CPU: "-", Mem: "-"}
	pg := ContainerStat{Name: container.ContainerPostgres, Status: "Stopped", CPU: "-", Mem: "-"}
	fe := ContainerStat{Name: container.ContainerFrontend, Status: "Stopped", CPU: "-", Mem: "-"}

	if engine == "" {
		return []ContainerStat{sui, pg, fe}
	}

	out, err := exec.Command(engine, "stats", "--no-stream", "--format", "{{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}").Output() // #nosec G204
	if err == nil {
		sui, pg, fe = parseStatsOutput(string(out), sui, pg, fe)
	}

	if sui.Status == "Stopped" && containerRunning(engine, container.ContainerSuiPlayground) {
		sui.Status = "Running"
	}
	if pg.Status == "Stopped" && (containerRunning(engine, container.ContainerPostgres) || containerRunning(engine, container.ContainerPostgresOld)) {
		pg.Status = "Running"
	}
	if fe.Status == "Stopped" && (containerRunning(engine, container.ContainerFrontend) || containerRunning(engine, container.ContainerFrontendOld)) {
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
		case container.ContainerPostgres, container.ContainerPostgresOld:
			pg.Status = "Running"
			pg.CPU = cpu
			pg.Mem = mem
		case container.ContainerFrontend, container.ContainerFrontendOld:
			fe.Status = "Running"
			fe.CPU = cpu
			fe.Mem = mem
		}
	}
	return sui, pg, fe
}

func containerRunning(engine, name string) bool {
	out, err := exec.Command(engine, "inspect", "--format", "{{.State.Running}}", name).Output() // #nosec G204
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

func GatherWorldInfo(workspace string) WorldInfo {
	envVars := extractEnvVars(workspace)
	addresses := extractAddresses(envVars)
	objs, pkgID := extractWorldObjects(workspace)
	return WorldInfo{
		PackageID: pkgID,
		Objects:   objs,
		Addresses: addresses,
	}
}

func extractEnvVars(workspace string) map[string]string {
	result := make(map[string]string)
	envPath := filepath.Join(workspace, "world-contracts", ".env")
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
	keys := make([]string, 0, len(envVars))
	for key := range envVars {
		if strings.HasSuffix(key, "_ADDRESS") {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		addresses[key] = envVars[key]
	}
	return addresses
}

func extractWorldObjects(workspace string) (map[string]string, string) {
	objs := make(map[string]string)
	filePath := filepath.Join(workspace, "world-contracts", "deployments", "localnet", "extracted-object-ids.json")
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
