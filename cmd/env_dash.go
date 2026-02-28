package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"efctl/pkg/container"
	"efctl/pkg/env"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
	"github.com/spf13/cobra"
)

var envDashCmd = &cobra.Command{
	Use:   "dash",
	Short: "Launch the environment dashboard",
	Long:  `Launches an interactive, responsive terminal dashboard for the EVE Frontier local development environment.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		res := env.CheckPrerequisites()
		engine, _ := res.Engine()
		if engine == "" {
			engine = "docker" // Default fallback if not found
		}

		m := initialModel(engine, workspacePath)
		f, _ := tea.LogToFile("/tmp/tea.log", "debug")
		defer f.Close()
		p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

		// Start log collection
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go collectLogs(ctx, p, engine, workspacePath)

		if _, err := p.Run(); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	envCmd.AddCommand(envDashCmd)
}

// styling
var (
	cyan   = lipgloss.Color("#00FFFF")
	orange = lipgloss.Color("#FF7400")
	dark   = lipgloss.Color("#111111")
	white  = lipgloss.Color("#FFFFFF")
	gray   = lipgloss.Color("#666666")
	red    = lipgloss.Color("#FF4444")
	green  = lipgloss.Color("#00CC66")
	yellow = lipgloss.Color("#FFAA00")

	headerStyle = lipgloss.NewStyle().
			Foreground(dark).
			Background(orange). // Approximate for gradient
			Padding(0, 1).
			Bold(true)

	labelStyle = lipgloss.NewStyle().Foreground(orange).Bold(true)
	valueStyle = lipgloss.NewStyle().Foreground(cyan)
	grayStyle  = lipgloss.NewStyle().Foreground(gray)
)

type containerStat struct {
	Name   string
	Status string
	CPU    string
	Mem    string
}

type recentTx struct {
	Digest  string
	Status  string
	Kind    string
	Age     string
	Sender  string
	GasUsed string
}

type chainStat struct {
	Checkpoint string
	Epoch      string
	TxCount    string
	RecentTxs  []recentTx
}

type worldEvent struct {
	EventType  string
	Module     string
	Sender     string
	Age        string
	ParsedJSON map[string]interface{}
}

type TickMsg time.Time
type LogMsg string

// restartUpMsg is sent after a successful env down to chain into env up during restart.
type restartUpMsg struct {
	upCmd *exec.Cmd
}

type StatsMsg struct {
	Sui        containerStat
	Pg         containerStat
	Fe         containerStat
	Chain      chainStat
	Objects    []string
	Admin      string
	EnvVars    map[string]string
	WorldObjs  map[string]string // component → object ID from extracted-object-ids.json
	Addresses  map[string]string // role → address (Admin, Player A, Player B, Sponsor)
	WorldPkgID string
	Events     []worldEvent
}

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// extractAdmin extracts ADMIN_ADDRESS from the .env file
func extractAdmin(workspace string) string {
	envPath := filepath.Join(workspace, "world-contracts", ".env")
	data, err := os.ReadFile(envPath) // #nosec G304 -- path constructed from known workspace prefix
	if err != nil {
		return "Unknown"
	}
	re := regexp.MustCompile(`(?m)^ADMIN_ADDRESS=(0x[a-fA-F0-9]+)`)
	matches := re.FindStringSubmatch(string(data))
	if len(matches) > 1 {
		return matches[1]
	}
	return "Not Found"
}

// extractEnvVars reads key=value pairs from the world-contracts .env file.
func extractEnvVars(workspace string) map[string]string {
	result := make(map[string]string)
	envPath := filepath.Join(workspace, "world-contracts", ".env")
	data, err := os.ReadFile(envPath) // #nosec G304 -- path constructed from known workspace prefix
	if err != nil {
		return result
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 && parts[1] != "" {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

// formatAge converts a duration into a short human-readable string.
func formatAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// renderStatus returns a styled status string, using red for Stopped.
func renderStatus(s string) string {
	if s == "Stopped" {
		return lipgloss.NewStyle().Foreground(red).Render(s)
	}
	return valueStyle.Render(s)
}

// formatWithCommas adds thousand separators to a numeric string.
func formatWithCommas(s string) string {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return s
	}
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	digits := strconv.FormatInt(n, 10)
	var buf []byte
	for i, c := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			buf = append(buf, ',')
		}
		buf = append(buf, byte(c)) // #nosec G115 -- digits are ASCII 0-9
	}
	return sign + string(buf)
}

// shortKind abbreviates common Sui transaction kind names.
func shortKind(kind string) string {
	switch kind {
	case "ProgrammableTransaction":
		return "PrgTx"
	case "ConsensusCommitPrologue", "ConsensusCommitPrologueV2", "ConsensusCommitPrologueV3":
		return "Consensus"
	case "AuthenticatorStateUpdate", "AuthenticatorStateUpdateV2":
		return "AuthState"
	case "RandomnessStateUpdate":
		return "Randomness"
	case "EndOfEpochTransaction":
		return "EndEpoch"
	case "ChangeEpoch":
		return "Epoch"
	case "Genesis":
		return "Genesis"
	default:
		if len(kind) > 10 {
			return kind[:10]
		}
		return kind
	}
}

// formatGas computes net gas used from Sui gas fields and returns a compact string.
func formatGas(computation, storage, rebate string) string {
	comp, _ := strconv.ParseInt(computation, 10, 64)
	stor, _ := strconv.ParseInt(storage, 10, 64)
	reb, _ := strconv.ParseInt(rebate, 10, 64)
	total := comp + stor - reb
	if total <= 0 {
		return "-"
	}
	return formatWithCommas(strconv.FormatInt(total, 10))
}

// colorizeLogLine applies colour to log line prefixes.
func colorizeLogLine(line string) string {
	if strings.HasPrefix(line, "[docker]") {
		return lipgloss.NewStyle().Foreground(cyan).Render("[docker]") + line[8:]
	}
	if strings.HasPrefix(line, "[db]") {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#CC88FF")).Render("[db]") + line[4:]
	}
	if strings.HasPrefix(line, "[deploy]") {
		return lipgloss.NewStyle().Foreground(green).Render("[deploy]") + line[8:]
	}
	if strings.HasPrefix(line, "[frontend]") {
		return lipgloss.NewStyle().Foreground(yellow).Render("[frontend]") + line[10:]
	}
	return line
}

func fetchChainInfo(client *http.Client) chainStat {
	info := chainStat{Checkpoint: "Offline", TxCount: "-", Epoch: "-"}

	// Checkpoint
	rpcPayload := `{"jsonrpc":"2.0","id":1,"method":"sui_getLatestCheckpointSequenceNumber","params":[]}`
	rpcReq, _ := http.NewRequest("POST", "http://localhost:9000", strings.NewReader(rpcPayload))
	rpcReq.Header.Set("Content-Type", "application/json")
	if resp, err := client.Do(rpcReq); err == nil { // #nosec G704 -- hardcoded localhost URL
		var res struct {
			Result string `json:"result"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&res)
		info.Checkpoint = res.Result
		_ = resp.Body.Close()
	}

	// Total transactions
	rpcPayloadTx := `{"jsonrpc":"2.0","id":1,"method":"sui_getTotalTransactionBlocks","params":[]}`
	rpcReqTx, _ := http.NewRequest("POST", "http://localhost:9000", strings.NewReader(rpcPayloadTx))
	rpcReqTx.Header.Set("Content-Type", "application/json")
	if resp, err := client.Do(rpcReqTx); err == nil { // #nosec G704 -- hardcoded localhost URL
		var res struct {
			Result string `json:"result"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&res)
		info.TxCount = res.Result
		_ = resp.Body.Close()
	}

	// Epoch
	rpcPayloadEpoch := `{"jsonrpc":"2.0","id":1,"method":"sui_getLatestSuiSystemState","params":[]}`
	rpcReqEpoch, _ := http.NewRequest("POST", "http://localhost:9000", strings.NewReader(rpcPayloadEpoch))
	rpcReqEpoch.Header.Set("Content-Type", "application/json")
	if resp, err := client.Do(rpcReqEpoch); err == nil { // #nosec G704 -- hardcoded localhost URL
		var res struct {
			Result map[string]interface{} `json:"result"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&res)
		if ep, ok := res.Result["epoch"].(string); ok {
			info.Epoch = ep
		}
		_ = resp.Body.Close()
	}

	// Recent transactions (descending order, up to 20)
	rpcPayloadRecent := `{"jsonrpc":"2.0","id":1,"method":"suix_queryTransactionBlocks","params":[{"options":{"showInput":true,"showEffects":true}},null,20,true]}`
	rpcReqRecent, _ := http.NewRequest("POST", "http://localhost:9000", strings.NewReader(rpcPayloadRecent))
	rpcReqRecent.Header.Set("Content-Type", "application/json")
	if resp, err := client.Do(rpcReqRecent); err == nil { // #nosec G704 -- hardcoded localhost URL
		var res struct {
			Result struct {
				Data []struct {
					Digest      string `json:"digest"`
					TimestampMs string `json:"timestampMs"`
					Transaction struct {
						Data struct {
							Sender      string `json:"sender"`
							Transaction struct {
								Kind string `json:"kind"`
							} `json:"transaction"`
						} `json:"data"`
					} `json:"transaction"`
					Effects struct {
						Status struct {
							Status string `json:"status"`
						} `json:"status"`
						GasUsed struct {
							ComputationCost string `json:"computationCost"`
							StorageCost     string `json:"storageCost"`
							StorageRebate   string `json:"storageRebate"`
						} `json:"gasUsed"`
					} `json:"effects"`
				} `json:"data"`
			} `json:"result"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&res)
		for _, tx := range res.Result.Data {
			d := tx.Digest
			if len(d) > 16 {
				d = d[:8] + ".." + d[len(d)-4:]
			}
			age := "-"
			if ms, err := strconv.ParseInt(tx.TimestampMs, 10, 64); err == nil {
				age = formatAge(time.Since(time.UnixMilli(ms)))
			}
			status := tx.Effects.Status.Status
			if status == "" {
				status = "?"
			}
			kind := tx.Transaction.Data.Transaction.Kind
			if kind == "" {
				kind = "tx"
			}
			sender := tx.Transaction.Data.Sender
			if len(sender) > 14 {
				sender = sender[:6] + ".." + sender[len(sender)-4:]
			}
			gas := formatGas(
				tx.Effects.GasUsed.ComputationCost,
				tx.Effects.GasUsed.StorageCost,
				tx.Effects.GasUsed.StorageRebate,
			)
			info.RecentTxs = append(info.RecentTxs, recentTx{
				Digest:  d,
				Status:  status,
				Kind:    shortKind(kind),
				Age:     age,
				Sender:  sender,
				GasUsed: gas,
			})
		}
		_ = resp.Body.Close()
	}

	return info
}

// parseContainerStats parses docker stats output into sui, postgres, and frontend container stats.
func parseContainerStats(engine string) (sui, pg, fe containerStat) {
	sui = containerStat{Status: "Stopped", CPU: "-", Mem: "-"}
	pg = containerStat{Status: "Stopped", CPU: "-", Mem: "-"}
	fe = containerStat{Status: "Stopped", CPU: "-", Mem: "-"}
	out, err := exec.Command(engine, "stats", "--no-stream", "--format", "{{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}").Output() // #nosec G204
	if err != nil {
		return
	}
	for _, l := range strings.Split(string(out), "\n") {
		parts := strings.Split(l, "\t")
		if len(parts) < 3 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		if name == container.ContainerSuiPlayground {
			sui = containerStat{Status: "Running", CPU: parts[1], Mem: parts[2]}
		}
		if name == container.ContainerPostgres || name == container.ContainerPostgresOld {
			pg = containerStat{Status: "Running", CPU: parts[1], Mem: parts[2]}
		}
		if name == container.ContainerFrontend || name == container.ContainerFrontendOld {
			fe = containerStat{Status: "Running", CPU: parts[1], Mem: parts[2]}
		}
	}
	return
}

// extractWorldObjects reads the extracted-object-ids.json and returns world objects and package ID.
func extractWorldObjects(workspace string) (objs map[string]string, pkgID string) {
	objs = make(map[string]string)
	extractFile := filepath.Join(workspace, "world-contracts", "deployments", "localnet", "extracted-object-ids.json")
	data, err := os.ReadFile(extractFile) // #nosec G304 -- path constructed from known workspace prefix
	if err != nil {
		return
	}
	var objMap map[string]interface{}
	if json.Unmarshal(data, &objMap) != nil {
		return
	}
	world, ok := objMap["world"].(map[string]interface{})
	if !ok {
		return
	}
	for k, v := range world {
		if str, ok := v.(string); ok {
			if k == "packageId" {
				pkgID = str
			} else {
				objs[k] = str
			}
		}
	}
	return
}

// buildAddresses assembles the role→address map from env vars and derived keys.
func buildAddresses(admin string, envVars map[string]string) map[string]string {
	addrs := make(map[string]string)
	if admin != "" && admin != "Unknown" && admin != "Not Found" {
		addrs["Admin"] = admin
	}
	if v, ok := envVars["SPONSOR_ADDRESS"]; ok {
		addrs["Sponsor"] = v
	}
	if pk, ok := envVars["PLAYER_A_PRIVATE_KEY"]; ok {
		if addr := deriveAddress(pk); addr != "" {
			addrs["Player A"] = addr
		}
	}
	if pk, ok := envVars["PLAYER_B_PRIVATE_KEY"]; ok {
		if addr := deriveAddress(pk); addr != "" {
			addrs["Player B"] = addr
		}
	}
	return addrs
}

func fetchStats(engine string, workspace string) StatsMsg {
	msg := StatsMsg{}
	msg.Sui, msg.Pg, msg.Fe = parseContainerStats(engine)

	client := &http.Client{Timeout: 1 * time.Second}
	msg.Chain = fetchChainInfo(client)

	msg.WorldObjs, msg.WorldPkgID = extractWorldObjects(workspace)
	msg.Admin = extractAdmin(workspace)
	msg.EnvVars = extractEnvVars(workspace)
	msg.Addresses = buildAddresses(msg.Admin, msg.EnvVars)

	if msg.WorldPkgID != "" && msg.Admin != "" && msg.Admin != "Unknown" && msg.Admin != "Not Found" {
		msg.Events = fetchWorldEvents(client, msg.WorldPkgID, msg.Admin)
	}

	return msg
}

// fetchWorldEvents queries recent events emitted by the world package.
// It queries events by Sender (admin) and filters to those matching the world package ID.
func fetchWorldEvents(client *http.Client, pkgID string, admin string) []worldEvent {
	var events []worldEvent

	// Query events by sender (admin deploys and interacts with world contracts)
	payload := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"suix_queryEvents","params":[{"Sender":"%s"},null,20,true]}`, admin)
	req, _ := http.NewRequest("POST", "http://localhost:9000", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req) // #nosec G704 -- hardcoded localhost URL
	if err != nil {
		return events
	}
	var res struct {
		Result struct {
			Data []struct {
				ID struct {
					TxDigest string `json:"txDigest"`
				} `json:"id"`
				PackageID   string                 `json:"packageId"`
				Module      string                 `json:"transactionModule"`
				Sender      string                 `json:"sender"`
				Type        string                 `json:"type"`
				TimestampMs string                 `json:"timestampMs"`
				ParsedJSON  map[string]interface{} `json:"parsedJson"`
			} `json:"data"`
		} `json:"result"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&res)
	_ = resp.Body.Close()

	for _, ev := range res.Result.Data {
		// Only include events from the world package
		if ev.PackageID != pkgID {
			continue
		}
		age := "-"
		if ms, err := strconv.ParseInt(ev.TimestampMs, 10, 64); err == nil {
			age = formatAge(time.Since(time.UnixMilli(ms)))
		}
		sender := ev.Sender
		if len(sender) > 14 {
			sender = sender[:6] + ".." + sender[len(sender)-4:]
		}
		eventType := ev.Type
		if idx := strings.LastIndex(eventType, "::"); idx >= 0 {
			eventType = eventType[idx+2:]
		}
		if idx := strings.Index(eventType, "<"); idx >= 0 {
			eventType = eventType[:idx]
		}
		events = append(events, worldEvent{
			EventType:  eventType,
			Module:     ev.Module,
			Sender:     sender,
			Age:        age,
			ParsedJSON: ev.ParsedJSON,
		})
	}

	return events
}

// deriveAddress uses sui keytool to derive an address from a bech32 private key.
func deriveAddress(privkey string) string {
	out, err := exec.Command("sui", "keytool", "import", privkey, "ed25519", "--json").Output() // #nosec G204 -- privkey from trusted .env file
	if err != nil {
		return ""
	}
	var res struct {
		SuiAddress string `json:"suiAddress"`
	}
	if json.Unmarshal(out, &res) == nil {
		return res.SuiAddress
	}
	return ""
}

// streamContainerLogs starts tailing a container's logs and sends lines with the given prefix.
func streamContainerLogs(ctx context.Context, p *tea.Program, engine, containerName, prefix string) {
	cmd := exec.CommandContext(ctx, engine, "logs", "-f", "--tail", "20", containerName) // #nosec G204
	stdout, err := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout
	if err == nil {
		if err := cmd.Start(); err == nil {
			go func() {
				scanner := bufio.NewScanner(stdout)
				for scanner.Scan() {
					p.Send(LogMsg(fmt.Sprintf("%s %s", prefix, scanner.Text())))
				}
				_ = cmd.Wait() // Reclaim process
			}()
		}
	}
}

func collectLogs(ctx context.Context, p *tea.Program, engine, workspace string) {
	// 1. Sui container logs
	streamContainerLogs(ctx, p, engine, container.ContainerSuiPlayground, "[docker]")

	// 2. Database container logs (try both naming conventions)
	dbContainer := container.ContainerPostgres
	// Check which name is running
	if out, err := exec.Command(engine, "ps", "--format", "{{.Names}}").Output(); err == nil { // #nosec G204
		names := string(out)
		if strings.Contains(names, container.ContainerPostgresOld) {
			dbContainer = container.ContainerPostgresOld
		}
	}
	streamContainerLogs(ctx, p, engine, dbContainer, "[db]")

	// 3. Frontend container logs (try both naming conventions)
	feContainer := container.ContainerFrontend
	if out, err := exec.Command(engine, "ps", "--format", "{{.Names}}").Output(); err == nil { // #nosec G204
		names := string(out)
		if strings.Contains(names, container.ContainerFrontendOld) {
			feContainer = container.ContainerFrontendOld
		}
	}
	streamContainerLogs(ctx, p, engine, feContainer, "[frontend]")

	// 4. Deploy logs
	deployLogPath := filepath.Join(workspace, "world-contracts", "deployments", "localnet", "deploy.log")
	// Try to start immediately or retry if missing (during env up)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if _, err := os.Stat(deployLogPath); err == nil {
					file, err := os.Open(deployLogPath) // #nosec G304 -- path constructed from known workspace prefix
					if err == nil {
						// Seek to near the end
						fileInfo, _ := file.Stat()
						if fileInfo.Size() > 2048 {
							_, _ = file.Seek(-2048, 2)
						}

						reader := bufio.NewReader(file)
						for {
							select {
							case <-ctx.Done():
								_ = file.Close()
								return
							default:
								line, err := reader.ReadString('\n')
								if err != nil {
									time.Sleep(500 * time.Millisecond) // wait for more
									continue
								}
								p.Send(LogMsg(fmt.Sprintf("[deploy] %s", strings.TrimSpace(line))))
							}
						}
					}
				}
				time.Sleep(2 * time.Second)
			}
		}
	}()
}

type model struct {
	engine         string
	workspace      string
	width          int
	height         int
	startTime      time.Time
	suiStat        containerStat
	pgStat         containerStat
	feStat         containerStat
	chainInfo      chainStat
	recentTxs      []recentTx
	objectTrackers []string
	adminAddr      string
	envVars        map[string]string
	worldObjs      map[string]string
	addresses      map[string]string
	worldPkgID     string
	logs           []string
	logScroll      int          // lines scrolled up from the bottom (0 = tailing)
	graphqlOn      bool         // whether GraphQL/Indexer is currently enabled
	frontendOn     bool         // whether the frontend dApp container is enabled
	worldEvents    []worldEvent // recent events from the world package
}

func initialModel(engine string, workspace string) model {
	// Detect whether GraphQL is currently enabled by checking the override file
	gqlOn := false
	feOn := false
	overridePath := filepath.Join(workspace, "builder-scaffold", "docker", "docker-compose.override.yml")
	if data, err := os.ReadFile(overridePath); err == nil { // #nosec G304 -- path constructed from known workspace prefix
		content := string(data)
		if strings.Contains(content, "postgres:") || strings.Contains(content, "SUI_GRAPHQL_ENABLED") {
			gqlOn = true
		}
		if strings.Contains(content, "frontend:") {
			feOn = true
		}
	}
	return model{
		engine:     engine,
		workspace:  workspace,
		startTime:  time.Now(),
		suiStat:    containerStat{Status: "Checking...", CPU: "-", Mem: "-"},
		pgStat:     containerStat{Status: "Checking...", CPU: "-", Mem: "-"},
		feStat:     containerStat{Status: "Checking...", CPU: "-", Mem: "-"},
		adminAddr:  "Checking...",
		graphqlOn:  gqlOn,
		frontendOn: feOn,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		func() tea.Msg { return fetchStats(m.engine, m.workspace) },
		tea.SetWindowTitle("efctl dashboard"),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		m.handleMouseScroll(msg)
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case TickMsg:
		return m, tea.Batch(
			tickCmd(),
			func() tea.Msg { return fetchStats(m.engine, m.workspace) },
		)
	case StatsMsg:
		m.applyStats(msg)
	case restartUpMsg:
		return m, tea.ExecProcess(msg.upCmd, func(err error) tea.Msg {
			if err != nil {
				return LogMsg("Error during restart (up): " + err.Error())
			}
			return LogMsg("Environment restarted successfully.")
		})
	case LogMsg:
		m.logs = append(m.logs, string(msg))
		if len(m.logs) > 500 {
			m.logs = m.logs[len(m.logs)-500:]
		}
	}
	return m, nil
}

// handleMouseScroll adjusts logScroll for mouse wheel events.
func (m *model) handleMouseScroll(msg tea.MouseMsg) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.logScroll += 3
		if ms := m.maxLogScroll(); m.logScroll > ms {
			m.logScroll = ms
		}
	case tea.MouseButtonWheelDown:
		m.logScroll -= 3
		if m.logScroll < 0 {
			m.logScroll = 0
		}
	}
}

// clampScroll clamps logScroll within [0, maxLogScroll].
func (m *model) clampScroll() {
	if ms := m.maxLogScroll(); m.logScroll > ms {
		m.logScroll = ms
	}
	if m.logScroll < 0 {
		m.logScroll = 0
	}
}

// handleKeyMsg processes keyboard input and returns the updated model and command.
func (m model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		m.logScroll++
		m.clampScroll()
	case "down", "j":
		m.logScroll--
		m.clampScroll()
	case "home":
		m.logScroll = m.maxLogScroll()
	case "end":
		m.logScroll = 0
	case "pgup":
		m.logScroll += 20
		m.clampScroll()
	case "pgdown":
		m.logScroll -= 20
		m.clampScroll()
	case "r":
		return m.handleRestart()
	case "d":
		return m.handleEnvDown()
	case "g":
		return m.handleEnableGraphQL()
	case "f":
		return m.handleEnableFrontend()
	}
	return m, nil
}

// handleRestart runs env down then env up, preserving --with-graphql and --with-frontend if enabled.
func (m model) handleRestart() (tea.Model, tea.Cmd) {
	args := []string{"env", "up", "-w", m.workspace}
	if m.graphqlOn {
		args = append(args, "--with-graphql")
	}
	if m.frontendOn {
		args = append(args, "--with-frontend")
	}
	downCmd := exec.Command("efctl", "env", "down", "-w", m.workspace) // #nosec G204
	upArgs := args
	return m, tea.ExecProcess(downCmd, func(err error) tea.Msg {
		if err != nil {
			return LogMsg("Error during restart (down): " + err.Error())
		}
		upCmd := exec.Command("efctl", upArgs...) // #nosec G204
		return restartUpMsg{upCmd: upCmd}
	})
}

// handleEnvDown runs efctl env down.
func (m model) handleEnvDown() (tea.Model, tea.Cmd) {
	c := exec.Command("efctl", "env", "down", "-w", m.workspace) // #nosec G204
	return m, tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return LogMsg("Error running env down: " + err.Error())
		}
		return LogMsg("Env DOWN completed.")
	})
}

// handleEnableGraphQL enables GraphQL if not already on.
func (m model) handleEnableGraphQL() (tea.Model, tea.Cmd) {
	if !m.graphqlOn {
		args := []string{"env", "up", "-w", m.workspace, "--with-graphql"}
		if m.frontendOn {
			args = append(args, "--with-frontend")
		}
		c := exec.Command("efctl", args...) // #nosec G204
		return m, tea.ExecProcess(c, func(err error) tea.Msg {
			if err != nil {
				return LogMsg("Error enabling GraphQL: " + err.Error())
			}
			return LogMsg("GraphQL enabled successfully.")
		})
	}
	return m, nil
}

// handleEnableFrontend enables the frontend dApp if not already on.
func (m model) handleEnableFrontend() (tea.Model, tea.Cmd) {
	if !m.frontendOn {
		args := []string{"env", "up", "-w", m.workspace, "--with-frontend"}
		if m.graphqlOn {
			args = append(args, "--with-graphql")
		}
		c := exec.Command("efctl", args...) // #nosec G204
		return m, tea.ExecProcess(c, func(err error) tea.Msg {
			if err != nil {
				return LogMsg("Error enabling frontend: " + err.Error())
			}
			return LogMsg("Frontend enabled successfully.")
		})
	}
	return m, nil
}

// applyStats updates the model with fresh stats data.
func (m *model) applyStats(msg StatsMsg) {
	m.suiStat = msg.Sui
	m.pgStat = msg.Pg
	m.feStat = msg.Fe
	m.chainInfo = msg.Chain
	m.recentTxs = msg.Chain.RecentTxs
	m.objectTrackers = msg.Objects
	m.adminAddr = msg.Admin
	m.envVars = msg.EnvVars
	m.worldObjs = msg.WorldObjs
	m.addresses = msg.Addresses
	m.worldPkgID = msg.WorldPkgID
	m.worldEvents = msg.Events
	overridePath := filepath.Join(m.workspace, "builder-scaffold", "docker", "docker-compose.override.yml")
	if data, err := os.ReadFile(overridePath); err == nil { // #nosec G304
		content := string(data)
		m.graphqlOn = strings.Contains(content, "postgres:") || strings.Contains(content, "SUI_GRAPHQL_ENABLED")
		m.frontendOn = strings.Contains(content, "frontend:")
	} else {
		m.graphqlOn = false
		m.frontendOn = false
	}
}

// fileExists returns true if the path exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// borderStr renders s in the border (cyan) colour.
func borderStr(s string) string {
	return lipgloss.NewStyle().Foreground(cyan).Render(s)
}

// logViewportRows returns the number of log lines visible in the log panel
// given the current terminal height and number of world events.
func logViewportRows(height, numEvents int) int {
	headerH := 1
	available := height - headerH
	topRows := (available * 30) / 100
	if topRows < 8 {
		topRows = 8
	}
	// Events are now side-by-side with logs, so they don't consume vertical space
	botRows := available - topRows - 3 // 3 = top/mid/bottom borders
	if botRows < 3 {
		botRows = 3
	}
	return botRows
}

// maxLogScroll returns the maximum logScroll value so the viewport
// never extends beyond the first log line.
func (m model) maxLogScroll() int {
	viewport := logViewportRows(m.height, len(m.worldEvents))
	max := len(m.logs) - viewport
	if max < 0 {
		return 0
	}
	return max
}

// efctlLogoLines holds the raw (uncolored) pterm BigText for "> EFCTL".
var efctlLogoLines []string

func init() {
	raw, _ := pterm.DefaultBigText.WithLetters(putils.LettersFromString("> EFCTL")).Srender()
	for _, line := range strings.Split(raw, "\n") {
		clean := pterm.RemoveColorFromString(line)
		if strings.TrimSpace(clean) != "" {
			efctlLogoLines = append(efctlLogoLines, clean)
		}
	}
}

func renderLogo() []string {
	result := make([]string, len(efctlLogoLines))
	for i, line := range efctlLogoLines {
		runes := []rune(line)
		numCols := float64(max(len(runes)-1, 1))
		var out strings.Builder
		for j, r := range runes {
			// Linear horizontal gradient from cyan (#00FFFF) to orange (#FF7400)
			t := float64(j) / numCols
			rd := int(t * 255)
			gn := int(255 - t*(255-116))
			bl := int(255 - t*255)
			c := lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", rd, gn, bl))
			out.WriteString(lipgloss.NewStyle().Foreground(c).Render(string(r)))
		}
		result[i] = out.String()
	}
	return result
}

// overlayLogo places the gradient logo on the bottom-right of the log lines.
func overlayLogo(logLines []string, innerW int) []string {
	logo := renderLogo()
	logoW := lipgloss.Width(efctlLogoLines[0])
	logoH := len(logo)
	if len(logLines) < logoH || innerW < logoW+4 {
		return logLines // not enough space
	}
	for i := 0; i < logoH; i++ {
		row := len(logLines) - logoH + i
		line := logLines[row]
		// Place logo with 2-char right margin
		padLeft := innerW - logoW - 2
		if padLeft < 0 {
			padLeft = 0
		}
		// Take existing content up to padLeft, then logo, then fill
		var runes []rune
		for _, r := range line {
			runes = append(runes, r)
		}
		leftPart := strings.Repeat(" ", padLeft)
		if lipgloss.Width(line) >= padLeft {
			// truncate visible content to padLeft width
			leftPart = truncateToWidth(line, padLeft)
		}
		combined := leftPart + logo[i]
		cw := lipgloss.Width(combined)
		if cw < innerW {
			combined += strings.Repeat(" ", innerW-cw)
		}
		logLines[row] = combined
	}
	return logLines
}

// truncateToWidth truncates a styled string to at most maxW visible characters.
func truncateToWidth(s string, maxW int) string {
	if lipgloss.Width(s) <= maxW {
		return s
	}
	// Byte-by-byte approach: take characters until width reached
	var out strings.Builder
	for _, r := range s {
		out.WriteRune(r)
		if lipgloss.Width(out.String()) >= maxW {
			break
		}
	}
	// Pad if needed
	result := out.String()
	w := lipgloss.Width(result)
	if w < maxW {
		result += strings.Repeat(" ", maxW-w)
	}
	return result
}

// renderToLines renders content with 1-char horizontal padding into a
// slice of lines, each exactly innerW visual characters wide.
func renderToLines(content string, innerW int) []string {
	rendered := lipgloss.NewStyle().Padding(0, 1).Width(innerW).Render(content)
	return strings.Split(rendered, "\n")
}

// padLines ensures exactly targetRows lines, each innerW visual chars wide.
func padLines(lines []string, targetRows, innerW int) []string {
	if len(lines) > targetRows {
		lines = lines[:targetRows]
	}
	emptyLine := strings.Repeat(" ", innerW)
	for len(lines) < targetRows {
		lines = append(lines, emptyLine)
	}
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w < innerW {
			lines[i] = line + strings.Repeat(" ", innerW-w)
		}
	}
	return lines
}

// buildTopBorder builds: ╭─ LeftTitle ──┬─ RightTitle ──╮
func buildTopBorder(leftW, rightW int, leftTitle, rightTitle string) string {
	ltw := lipgloss.Width(leftTitle)
	rtw := lipgloss.Width(rightTitle)
	ld := leftW - 3 - ltw
	if ld < 0 {
		ld = 0
	}
	rd := rightW - 3 - rtw
	if rd < 0 {
		rd = 0
	}

	return borderStr("╭─") + " " + labelStyle.Render(leftTitle) + " " +
		borderStr(strings.Repeat("─", ld)+"┬─") + " " + labelStyle.Render(rightTitle) + " " +
		borderStr(strings.Repeat("─", rd)+"╮")
}

// buildLeftMidBorder builds: ├─ Title ──────────┤ (left-side only, with ┤ connecting to │)
func buildLeftMidBorder(leftW int, title string) string {
	tw := lipgloss.Width(title)
	d := leftW - 3 - tw
	if d < 0 {
		d = 0
	}
	return borderStr("├─") + " " + labelStyle.Render(title) + " " +
		borderStr(strings.Repeat("─", d)+"┤")
}

// buildMiddleBorder builds: ├─ Title ──┴────────┤
// The ┴ character is placed where the top-section vertical divider was.
func buildMiddleBorder(totalW, leftW int, title string) string {
	tw := lipgloss.Width(title)
	totalDashes := totalW - 5 - tw
	if totalDashes < 0 {
		totalDashes = 0
	}

	junction := leftW - 3 - tw
	if junction >= 0 && junction < totalDashes {
		return borderStr("├─") + " " + labelStyle.Render(title) + " " +
			borderStr(strings.Repeat("─", junction)+"┴"+strings.Repeat("─", totalDashes-junction-1)+"┤")
	}
	return borderStr("├─") + " " + labelStyle.Render(title) + " " +
		borderStr(strings.Repeat("─", totalDashes)+"┤")
}

// buildBottomBorder builds: ╰─ footer ──────╯
func buildBottomBorder(totalW int, footer string) string {
	fw := lipgloss.Width(footer)
	d := totalW - 5 - fw
	if d < 0 {
		d = 0
	}
	return borderStr("╰─") + " " + grayStyle.Render(footer) + " " +
		borderStr(strings.Repeat("─", d)+"╯")
}

// buildFullBorder builds: ├─ Title ──────────────┤ (full width, no junction)
func buildFullBorder(totalW int, title string) string {
	tw := lipgloss.Width(title)
	d := totalW - 5 - tw
	if d < 0 {
		d = 0
	}
	return borderStr("├─") + " " + labelStyle.Render(title) + " " +
		borderStr(strings.Repeat("─", d)+"┤")
}

// buildSplitMiddleBorder builds: ├─ LeftTitle ──┼─ RightTitle ──┤
// Used when a vertical divider continues through top and bottom sections.
func buildSplitMiddleBorder(leftW, rightW int, leftTitle, rightTitle string) string {
	ltw := lipgloss.Width(leftTitle)
	rtw := lipgloss.Width(rightTitle)
	ld := leftW - 3 - ltw
	if ld < 0 {
		ld = 0
	}
	rd := rightW - 3 - rtw
	if rd < 0 {
		rd = 0
	}
	return borderStr("├─") + " " + labelStyle.Render(leftTitle) + " " +
		borderStr(strings.Repeat("─", ld)+"┼─") + " " + labelStyle.Render(rightTitle) + " " +
		borderStr(strings.Repeat("─", rd)+"┤")
}

// buildBottomBorderWithJunction builds: ╰─ footer ──┴──────╯
// Places a ┴ junction at the vertical divider position.
func buildBottomBorderWithJunction(totalW, leftW int, footer string) string {
	fw := lipgloss.Width(footer)
	totalDashes := totalW - 5 - fw
	if totalDashes < 0 {
		totalDashes = 0
	}
	junction := leftW - 3 - fw
	if junction >= 0 && junction < totalDashes {
		return borderStr("╰─") + " " + grayStyle.Render(footer) + " " +
			borderStr(strings.Repeat("─", junction)+"┴"+strings.Repeat("─", totalDashes-junction-1)+"╯")
	}
	// Junction falls outside dashes — skip it
	return borderStr("╰─") + " " + grayStyle.Render(footer) + " " +
		borderStr(strings.Repeat("─", totalDashes)+"╯")
}

func (m model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	header := m.renderHeader()
	headerH := lipgloss.Height(header)

	// ── Layout geometry ──
	leftInner, rightInner, logInner := m.panelWidths()
	hasEvents := len(m.worldEvents) > 0

	// Vertical budget
	available := m.height - headerH
	containerRows := 4
	topRows := max((available*30)/100, 8)
	envRows := max(topRows-containerRows-1, 3)
	rightRows := containerRows + envRows + 1
	botRows := max(available-topRows-3, 3)

	// ── Render panel content ──
	containerLines := padLines(renderToLines(m.renderContainerContent(), leftInner), containerRows, leftInner)
	envLines := padLines(renderToLines(m.renderEnvContent(), leftInner), envRows, leftInner)
	rightLines := padLines(renderToLines(m.renderRightContent(rightRows), rightInner), rightRows, rightInner)

	logW, logLines, eventLines := m.renderBottomPanels(hasEvents, botRows, leftInner, rightInner, logInner)

	// ── Assemble frame ──
	var out strings.Builder
	out.WriteString(header)
	out.WriteByte('\n')
	m.writeTopSection(&out, leftInner, rightInner, containerRows, envRows, containerLines, envLines, rightLines)
	m.writeBottomSection(&out, hasEvents, botRows, leftInner, rightInner, logW, logLines, eventLines)

	return out.String()
}

// renderHeader builds the header bar showing service status and uptime.
func (m model) renderHeader() string {
	uptime := time.Since(m.startTime).Round(time.Second)
	suiUp := "UP"
	if m.suiStat.Status != "Running" {
		suiUp = "DOWN"
	}
	dbUp := "UP"
	if m.pgStat.Status != "Running" {
		dbUp = "DOWN"
	}
	gqlStatus := "gql:OFF"
	if m.graphqlOn {
		gqlStatus = "gql:ON"
	}
	feStatus := "fe:OFF"
	if m.frontendOn {
		feStatus = "fe:ON"
	}
	headerTitle := fmt.Sprintf(" efctl dashboard │ sui:%s  db:%s  %s  %s │ Uptime: %v ", suiUp, dbUp, gqlStatus, feStatus, uptime)
	padLen := m.width - lipgloss.Width(headerTitle)
	if padLen < 0 {
		padLen = 0
	}
	return headerStyle.Width(m.width).Render(headerTitle + strings.Repeat(" ", padLen))
}

// panelWidths returns the inner widths for left, right, and full-width panels.
func (m model) panelWidths() (leftInner, rightInner, logInner int) {
	leftInner = max((m.width-3)/2, 1)
	rightInner = max(m.width-3-leftInner, 1)
	logInner = max(m.width-2, 1)
	return
}

// renderBottomPanels prepares the events and log line arrays for the bottom section.
func (m model) renderBottomPanels(hasEvents bool, botRows, leftInner, rightInner, logInner int) (int, []string, []string) {
	var eventLines []string
	logW := logInner
	if hasEvents {
		eventLines = padLines(renderToLines(m.renderEventsContent(botRows, leftInner), leftInner), botRows, leftInner)
		logW = rightInner
	}

	endIdx := len(m.logs) - m.logScroll
	if endIdx < 0 {
		endIdx = 0
	}
	startIdx := endIdx - botRows
	if startIdx < 0 {
		startIdx = 0
	}
	visibleLogs := m.logs[startIdx:endIdx]
	coloredLogs := make([]string, len(visibleLogs))
	for i, line := range visibleLogs {
		coloredLogs[i] = colorizeLogLine(line)
	}
	logLines := padLines(renderToLines(strings.Join(coloredLogs, "\n"), logW), botRows, logW)
	logLines = overlayLogo(logLines, logW)
	return logW, logLines, eventLines
}

// writeTopSection writes the Services/Environment + Chain rows to the output.
func (m model) writeTopSection(out *strings.Builder, leftInner, rightInner, containerRows, envRows int, containerLines, envLines, rightLines []string) {
	out.WriteString(buildTopBorder(leftInner, rightInner, "Services", "Chain"))
	out.WriteByte('\n')

	rightIdx := 0
	for i := 0; i < containerRows; i++ {
		out.WriteString(borderStr("│") + containerLines[i] + borderStr("│") + rightLines[rightIdx] + borderStr("│"))
		out.WriteByte('\n')
		rightIdx++
	}

	out.WriteString(buildLeftMidBorder(leftInner, "Environment") + rightLines[rightIdx] + borderStr("│"))
	out.WriteByte('\n')
	rightIdx++

	for i := 0; i < envRows; i++ {
		out.WriteString(borderStr("│") + envLines[i] + borderStr("│") + rightLines[rightIdx] + borderStr("│"))
		out.WriteByte('\n')
		rightIdx++
	}
}

// writeBottomSection writes the Events+Logs (split) or Logs (full-width) rows and footer.
func (m model) writeBottomSection(out *strings.Builder, hasEvents bool, botRows, leftInner, rightInner, logW int, logLines, eventLines []string) {
	logTitle := "Logs ● LIVE"
	if m.logScroll > 0 {
		logTitle = fmt.Sprintf("Logs ‖ PAUSED (↑%d lines)", m.logScroll)
	}

	if hasEvents {
		eventsTitle := fmt.Sprintf("World Events (%d)", len(m.worldEvents))
		if m.logScroll > 0 {
			logTitle = fmt.Sprintf("Logs ‖ PAUSED (↑%d)", m.logScroll)
		}
		out.WriteString(buildSplitMiddleBorder(leftInner, rightInner, eventsTitle, logTitle))
		out.WriteByte('\n')
		for i := 0; i < botRows; i++ {
			out.WriteString(borderStr("│") + eventLines[i] + borderStr("│") + logLines[i] + borderStr("│"))
			out.WriteByte('\n')
		}
	} else {
		out.WriteString(buildMiddleBorder(m.width, leftInner, logTitle))
		out.WriteByte('\n')
		for i := 0; i < botRows; i++ {
			out.WriteString(borderStr("│") + logLines[i] + borderStr("│"))
			out.WriteByte('\n')
		}
	}

	footerKeys := "[r] restart  [d] env down  [↑↓/PgUp/PgDn] scroll  [Home/End] jump  [q] quit"
	if !m.graphqlOn || !m.frontendOn {
		extras := ""
		if !m.graphqlOn {
			extras += "  [g] enable graphql"
		}
		if !m.frontendOn {
			extras += "  [f] enable frontend"
		}
		footerKeys = "[r] restart  [d] env down" + extras + "  [↑↓/PgUp/PgDn] scroll  [Home/End] jump  [q] quit"
	}
	if hasEvents {
		out.WriteString(buildBottomBorderWithJunction(m.width, leftInner, footerKeys))
	} else {
		out.WriteString(buildBottomBorder(m.width, footerKeys))
	}
}

func (m model) renderContainerContent() string {
	var b bytes.Buffer
	renderRow := func(name string, stat containerStat) {
		dot := lipgloss.NewStyle().Foreground(green).Bold(true).Render("●")
		if stat.Status == "Stopped" {
			dot = lipgloss.NewStyle().Foreground(red).Bold(true).Render("●")
		} else if stat.Status != "Running" {
			dot = lipgloss.NewStyle().Foreground(yellow).Bold(true).Render("●")
		}
		b.WriteString(fmt.Sprintf(" %s %-18s %s %-7s  %s %s\n",
			dot, name,
			grayStyle.Render("CPU"), valueStyle.Render(stat.CPU),
			grayStyle.Render("Mem"), valueStyle.Render(stat.Mem)))
	}
	b.WriteString("\n")
	renderRow("sui-playground", m.suiStat)
	renderRow("database", m.pgStat)
	if m.frontendOn {
		renderRow("frontend", m.feStat)
	}
	b.WriteString("\n")
	return b.String()
}

func (m model) renderEnvContent() string {
	var b bytes.Buffer
	shorten := m.hexShortener()

	b.WriteString("\n")
	m.writeEnvConfig(&b, shorten)
	m.writeEnvAddresses(&b, shorten)
	m.writeEnvObjects(&b, shorten)

	return b.String()
}

// hexShortener returns a function that abbreviates hex values for readability.
func (m model) hexShortener() func(string) string {
	maxVal := (m.width-3)/2 - 20
	if maxVal < 10 {
		maxVal = 10
	}
	return func(s string) string {
		if len(s) <= maxVal {
			return s
		}
		if strings.HasPrefix(s, "0x") && len(s) > 16 {
			return s[:10] + "…" + s[len(s)-4:]
		}
		return s[:maxVal-1] + "…"
	}
}

// writeEnvConfig writes network/RPC/tenant config lines.
func (m model) writeEnvConfig(b *bytes.Buffer, shorten func(string) string) {
	network := "localnet"
	if v, ok := m.envVars["SUI_NETWORK"]; ok {
		network = v
	}
	b.WriteString(labelStyle.Render(" Network:") + " " + valueStyle.Render(network))
	b.WriteString("   " + labelStyle.Render("RPC:") + " " + valueStyle.Render("http://localhost:9000"))
	if v, ok := m.envVars["TENANT"]; ok {
		b.WriteString("   " + labelStyle.Render("Tenant:") + " " + valueStyle.Render(v))
	}
	b.WriteString("\n")
	if m.worldPkgID != "" {
		b.WriteString(labelStyle.Render(" World Pkg:") + " " + valueStyle.Render(shorten(m.worldPkgID)) + "\n")
	}
	if m.frontendOn {
		b.WriteString(labelStyle.Render(" Frontend:") + " " + valueStyle.Render("http://localhost:5173") + "\n")
	}
}

// writeEnvAddresses writes the addresses section.
func (m model) writeEnvAddresses(b *bytes.Buffer, shorten func(string) string) {
	if len(m.addresses) == 0 {
		return
	}
	b.WriteString("\n " + labelStyle.Render("Addresses") + "\n")
	for _, role := range []string{"Admin", "Sponsor", "Player A", "Player B"} {
		if addr, ok := m.addresses[role]; ok {
			b.WriteString(fmt.Sprintf("  %-10s %s\n", labelStyle.Render(role), valueStyle.Render(shorten(addr))))
		}
	}
}

// writeEnvObjects writes the world objects section.
func (m model) writeEnvObjects(b *bytes.Buffer, shorten func(string) string) {
	if len(m.worldObjs) == 0 {
		return
	}
	b.WriteString(fmt.Sprintf("\n "+labelStyle.Render("Objects")+" %s\n", grayStyle.Render(fmt.Sprintf("(%d)", len(m.worldObjs)))))
	knownKeys := []string{"governorCap", "adminAcl", "objectRegistry", "serverAddressRegistry", "energyConfig", "fuelConfig", "gateConfig"}
	for _, key := range knownKeys {
		if v, ok := m.worldObjs[key]; ok {
			b.WriteString(fmt.Sprintf("  %-22s %s\n", labelStyle.Render(humanizeCamelCase(key)), grayStyle.Render(shorten(v))))
		}
	}
	knownSet := make(map[string]bool, len(knownKeys))
	for _, k := range knownKeys {
		knownSet[k] = true
	}
	for k, v := range m.worldObjs {
		if knownSet[k] {
			continue
		}
		b.WriteString(fmt.Sprintf("  %-22s %s\n", labelStyle.Render(humanizeCamelCase(k)), grayStyle.Render(shorten(v))))
	}
}

// humanizeCamelCase converts "governorCap" to "Governor Cap", etc.
func humanizeCamelCase(s string) string {
	var words []string
	start := 0
	for i := 1; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			words = append(words, s[start:i])
			start = i
		}
	}
	words = append(words, s[start:])
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

func (m model) renderRightContent(topRows int) string {
	var b bytes.Buffer
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(" %s %s     %s %s\n",
		labelStyle.Render("Checkpoint:"),
		valueStyle.Render(formatWithCommas(m.chainInfo.Checkpoint)),
		labelStyle.Render("Epoch:"),
		valueStyle.Render(m.chainInfo.Epoch)))
	b.WriteString(fmt.Sprintf(" %s %s\n",
		labelStyle.Render("Transactions:"),
		valueStyle.Render(formatWithCommas(m.chainInfo.TxCount))))

	// Recent transactions with column headers — adaptive to available rows
	fixedLines := 3                        // blank + 2 stat lines
	availForTx := topRows - fixedLines - 3 // 3 = blank + title + column header
	if availForTx > 0 && len(m.recentTxs) > 0 {
		b.WriteString("\n " + labelStyle.Render("Recent Transactions") + "\n")
		b.WriteString(grayStyle.Render("  ST  SENDER          TYPE        GAS       AGE") + "\n")
		showCount := availForTx
		if showCount > len(m.recentTxs) {
			showCount = len(m.recentTxs)
		}
		for i := 0; i < showCount; i++ {
			tx := m.recentTxs[i]
			statusIcon := grayStyle.Render(" ?")
			if tx.Status == "success" {
				statusIcon = lipgloss.NewStyle().Foreground(green).Render(" ✓")
			} else if tx.Status == "failure" {
				statusIcon = lipgloss.NewStyle().Foreground(red).Render(" ✗")
			}
			senderStr := grayStyle.Render(fmt.Sprintf("%-14s", tx.Sender))
			kindStr := grayStyle.Render(fmt.Sprintf("%-10s", tx.Kind))
			gasStr := grayStyle.Render(fmt.Sprintf("%9s", tx.GasUsed))
			ageStr := grayStyle.Render(fmt.Sprintf("%5s", tx.Age))
			b.WriteString(fmt.Sprintf("  %s %s  %s %s %s\n", statusIcon, senderStr, kindStr, gasStr, ageStr))
		}
	}
	return b.String()
}

func (m model) renderEventsContent(maxRows int, panelW int) string {
	var b bytes.Buffer
	b.WriteString("\n")

	// Adaptive columns based on panel width
	// Narrow panel: EVENT + MODULE + AGE
	// Wide panel (≥70): add SENDER column
	wide := panelW >= 70
	if wide {
		b.WriteString(grayStyle.Render("  EVENT                        MODULE                SENDER          AGE") + "\n")
	} else {
		b.WriteString(grayStyle.Render("  EVENT                    MODULE          AGE") + "\n")
	}
	showCount := maxRows - 2 // subtract blank + column header line
	if showCount > len(m.worldEvents) {
		showCount = len(m.worldEvents)
	}
	for i := 0; i < showCount; i++ {
		ev := m.worldEvents[i]
		if wide {
			eventStr := valueStyle.Render(fmt.Sprintf("%-28s", ev.EventType))
			moduleStr := grayStyle.Render(fmt.Sprintf("%-20s", ev.Module))
			senderStr := grayStyle.Render(fmt.Sprintf("%-14s", ev.Sender))
			ageStr := grayStyle.Render(fmt.Sprintf("%5s", ev.Age))
			b.WriteString(fmt.Sprintf("  %s  %s  %s  %s\n", eventStr, moduleStr, senderStr, ageStr))
		} else {
			eventStr := valueStyle.Render(fmt.Sprintf("%-24s", ev.EventType))
			moduleStr := grayStyle.Render(fmt.Sprintf("%-14s", ev.Module))
			ageStr := grayStyle.Render(fmt.Sprintf("%5s", ev.Age))
			b.WriteString(fmt.Sprintf("  %s  %s  %s\n", eventStr, moduleStr, ageStr))
		}
	}
	return b.String()
}
