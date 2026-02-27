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
		p := tea.NewProgram(m, tea.WithAltScreen())

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

type TickMsg time.Time
type LogMsg string
type StatsMsg struct {
	Sui        containerStat
	Pg         containerStat
	Chain      chainStat
	Objects    []string
	Admin      string
	EnvVars    map[string]string
	WorldObjs  map[string]string // component → object ID from extracted-object-ids.json
	Addresses  map[string]string // role → address (Admin, Player A, Player B, Sponsor)
	WorldPkgID string
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

// parseContainerStats parses docker stats output into sui and postgres container stats.
func parseContainerStats(engine string) (sui, pg containerStat) {
	sui = containerStat{Status: "Stopped", CPU: "-", Mem: "-"}
	pg = containerStat{Status: "Stopped", CPU: "-", Mem: "-"}
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
	msg.Sui, msg.Pg = parseContainerStats(engine)

	client := &http.Client{Timeout: 1 * time.Second}
	msg.Chain = fetchChainInfo(client)

	msg.WorldObjs, msg.WorldPkgID = extractWorldObjects(workspace)
	msg.Admin = extractAdmin(workspace)
	msg.EnvVars = extractEnvVars(workspace)
	msg.Addresses = buildAddresses(msg.Admin, msg.EnvVars)

	return msg
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

	// 2. Deploy logs
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
	chainInfo      chainStat
	recentTxs      []recentTx
	objectTrackers []string
	adminAddr      string
	envVars        map[string]string
	worldObjs      map[string]string
	addresses      map[string]string
	worldPkgID     string
	logs           []string
}

func initialModel(engine string, workspace string) model {
	return model{
		engine:    engine,
		workspace: workspace,
		startTime: time.Now(),
		suiStat:   containerStat{Status: "Checking...", CPU: "-", Mem: "-"},
		pgStat:    containerStat{Status: "Checking...", CPU: "-", Mem: "-"},
		adminAddr: "Checking...",
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		func() tea.Msg { return fetchStats(m.engine, m.workspace) },
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			// run up
			c := exec.Command("efctl", "env", "up", "-w", m.workspace) // #nosec G204 -- workspace path from trusted CLI flag
			return m, tea.ExecProcess(c, func(err error) tea.Msg {
				if err != nil {
					return LogMsg("Error running env up: " + err.Error())
				}
				return LogMsg("Env UP completed.")
			})
		case "d":
			// run down
			c := exec.Command("efctl", "env", "down", "-w", m.workspace) // #nosec G204 -- workspace path from trusted CLI flag
			return m, tea.ExecProcess(c, func(err error) tea.Msg {
				if err != nil {
					return LogMsg("Error running env down: " + err.Error())
				}
				return LogMsg("Env DOWN completed.")
			})
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case TickMsg:
		return m, tea.Batch(
			tickCmd(),
			func() tea.Msg { return fetchStats(m.engine, m.workspace) },
		)

	case StatsMsg:
		m.suiStat = msg.Sui
		m.pgStat = msg.Pg
		m.chainInfo = msg.Chain
		m.recentTxs = msg.Chain.RecentTxs
		m.objectTrackers = msg.Objects
		m.adminAddr = msg.Admin
		m.envVars = msg.EnvVars
		m.worldObjs = msg.WorldObjs
		m.addresses = msg.Addresses
		m.worldPkgID = msg.WorldPkgID

	case LogMsg:
		m.logs = append(m.logs, string(msg))
		// Keep only the last 100 log lines to avoid unbounded memory growth
		if len(m.logs) > 100 {
			m.logs = m.logs[len(m.logs)-100:]
		}
	}

	return m, nil
}

// borderStr renders s in the border (cyan) colour.
func borderStr(s string) string {
	return lipgloss.NewStyle().Foreground(cyan).Render(s)
}

// efctlLogoLines holds the raw (uncolored) pterm BigText for "EFCTL".
var efctlLogoLines []string

func init() {
	raw, _ := pterm.DefaultBigText.WithLetters(putils.LettersFromString("EFCTL")).Srender()
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
		var out strings.Builder
		for j, r := range runes {
			// Gradient from cyan (#00FFFF) to orange (#FF7400)
			t := float64(j) / float64(max(len(runes)-1, 1))
			red := int(0 + t*255)
			grn := int(255 - t*(255-116))
			blu := int(255 - t*255)
			c := lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", red, grn, blu))
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

func (m model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	// ── Header bar ──
	uptime := time.Since(m.startTime).Round(time.Second)
	headerTitle := fmt.Sprintf(" efctl dashboard | Uptime: %v ", uptime)
	padLen := m.width - lipgloss.Width(headerTitle)
	if padLen < 0 {
		padLen = 0
	}
	header := headerStyle.Width(m.width).Render(headerTitle + strings.Repeat(" ", padLen))
	headerH := lipgloss.Height(header)

	// ── Layout geometry ──
	// Frame uses shared borders for a masonry look:
	//   ╭─ Container ─┬─ Chain ──────╮
	//   │             │              │
	//   ├─ Env Info ──┤              │
	//   │             │              │
	//   ├─ Logs ──────┴──────────────┤
	//   │                            │
	//   ╰─ keybindings ─────────────╯
	//
	// Horizontal: 1(│) + leftInner + 1(│) + rightInner + 1(│) = width
	leftInner := (m.width - 3) / 2
	rightInner := m.width - 3 - leftInner
	logInner := m.width - 2

	if leftInner < 1 {
		leftInner = 1
	}
	if rightInner < 1 {
		rightInner = 1
	}
	if logInner < 1 {
		logInner = 1
	}

	// Vertical budget:
	// headerH + topBorder(1) + containerRows + leftMidBorder(1) + envRows + midBorder(1) + logRows + botBorder(1)
	available := m.height - headerH
	topRows := (available * 2) / 5
	if topRows < 8 {
		topRows = 8
	}
	// Split top rows: container panel gets ~40%, env panel gets ~60%
	containerRows := topRows * 2 / 5
	if containerRows < 3 {
		containerRows = 3
	}
	envRows := topRows - containerRows - 1 // -1 for the left-mid border
	if envRows < 3 {
		envRows = 3
	}
	rightRows := containerRows + envRows + 1 // right panel spans full height including left-mid border line
	botRows := available - topRows - 3
	if botRows < 3 {
		botRows = 3
	}

	// ── Render panel content ──
	containerLines := padLines(renderToLines(m.renderContainerContent(), leftInner), containerRows, leftInner)
	envLines := padLines(renderToLines(m.renderEnvContent(), leftInner), envRows, leftInner)
	rightLines := padLines(renderToLines(m.renderRightContent(rightRows), rightInner), rightRows, rightInner)

	maxLogs := botRows
	startIdx := len(m.logs) - maxLogs
	if startIdx < 0 {
		startIdx = 0
	}
	coloredLogs := make([]string, len(m.logs[startIdx:]))
	for i, line := range m.logs[startIdx:] {
		coloredLogs[i] = colorizeLogLine(line)
	}
	logLines := padLines(renderToLines(strings.Join(coloredLogs, "\n"), logInner), botRows, logInner)
	logLines = overlayLogo(logLines, logInner)

	// ── Assemble frame ──
	var out strings.Builder
	out.WriteString(header)
	out.WriteByte('\n')

	// Top border
	out.WriteString(buildTopBorder(leftInner, rightInner, "Container Status", "Chain Info"))
	out.WriteByte('\n')

	// Container rows (top-left) + right panel rows
	rightIdx := 0
	for i := 0; i < containerRows; i++ {
		out.WriteString(borderStr("│") + containerLines[i] + borderStr("│") + rightLines[rightIdx] + borderStr("│"))
		out.WriteByte('\n')
		rightIdx++
	}

	// Left-mid border row (Environment Info title) + right panel continues
	leftMidBorder := buildLeftMidBorder(leftInner, "Environment Info")
	out.WriteString(leftMidBorder + rightLines[rightIdx] + borderStr("│"))
	out.WriteByte('\n')
	rightIdx++

	// Environment rows (bottom-left) + right panel rows
	for i := 0; i < envRows; i++ {
		out.WriteString(borderStr("│") + envLines[i] + borderStr("│") + rightLines[rightIdx] + borderStr("│"))
		out.WriteByte('\n')
		rightIdx++
	}

	// Middle border (logs)
	out.WriteString(buildMiddleBorder(m.width, leftInner, "Combined Log View"))
	out.WriteByte('\n')

	// Log rows
	for i := 0; i < botRows; i++ {
		out.WriteString(borderStr("│") + logLines[i] + borderStr("│"))
		out.WriteByte('\n')
	}

	// Bottom border with keybindings
	out.WriteString(buildBottomBorder(m.width, "[r] env up | [d] env down | [q] quit"))

	return out.String()
}

func (m model) renderContainerContent() string {
	var b bytes.Buffer
	b.WriteString("\n sui-playground\n")
	b.WriteString(fmt.Sprintf("   Status: %-10s  CPU: %-8s Mem: %s\n\n", renderStatus(m.suiStat.Status), m.suiStat.CPU, m.suiStat.Mem))
	b.WriteString(" database\n")
	b.WriteString(fmt.Sprintf("   Status: %-10s  CPU: %-8s Mem: %s\n", renderStatus(m.pgStat.Status), m.pgStat.CPU, m.pgStat.Mem))
	return b.String()
}

func (m model) renderEnvContent() string {
	var b bytes.Buffer

	// Compute max display length for values based on panel width.
	// leftInner = (width-3)/2; content area = leftInner - 2 (padding); label prefix ~18 chars.
	maxVal := (m.width-3)/2 - 20
	if maxVal < 10 {
		maxVal = 10
	}

	// Shorten a value only when it exceeds the available width.
	shorten := func(s string) string {
		if len(s) > maxVal {
			return s[:maxVal-3] + "..."
		}
		return s
	}

	b.WriteString("\n")

	// ── Configuration ──
	network := "localnet"
	if v, ok := m.envVars["SUI_NETWORK"]; ok {
		network = v
	}
	b.WriteString(labelStyle.Render(" Network:  ") + " " + network)
	b.WriteString("   " + labelStyle.Render("RPC: ") + " http://localhost:9000")
	if v, ok := m.envVars["TENANT"]; ok {
		b.WriteString("   " + labelStyle.Render("Tenant: ") + " " + v)
	}
	b.WriteString("\n")

	// ── Package ──
	if m.worldPkgID != "" {
		b.WriteString(labelStyle.Render(" World Pkg:") + " " + shorten(m.worldPkgID) + "\n")
	}

	// ── Addresses ──
	if len(m.addresses) > 0 {
		b.WriteString("\n " + labelStyle.Render("Addresses") + "\n")
		// Ordered display
		for _, role := range []string{"Admin", "Sponsor", "Player A", "Player B"} {
			if addr, ok := m.addresses[role]; ok {
				b.WriteString(fmt.Sprintf("  %-9s %s\n", labelStyle.Render(role), shorten(addr)))
			}
		}
	}

	// ── World Objects ──
	if len(m.worldObjs) > 0 {
		b.WriteString("\n " + labelStyle.Render("Objects") + "\n")
		for _, key := range []string{"governorCap", "adminAcl", "objectRegistry", "serverAddressRegistry", "energyConfig", "fuelConfig", "gateConfig"} {
			if v, ok := m.worldObjs[key]; ok {
				label := humanizeCamelCase(key)
				b.WriteString(fmt.Sprintf("  %-20s %s\n", labelStyle.Render(label), grayStyle.Render(shorten(v))))
			}
		}
		// Any remaining keys not in the ordered list
		for k, v := range m.worldObjs {
			switch k {
			case "governorCap", "adminAcl", "objectRegistry", "serverAddressRegistry", "energyConfig", "fuelConfig", "gateConfig":
				continue
			}
			label := humanizeCamelCase(k)
			b.WriteString(fmt.Sprintf("  %-20s %s\n", labelStyle.Render(label), grayStyle.Render(shorten(v))))
		}
	}

	return b.String()
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
	b.WriteString(labelStyle.Render(" Checkpoint:   ") + valueStyle.Render(formatWithCommas(m.chainInfo.Checkpoint)) + "\n")
	b.WriteString(labelStyle.Render(" Transactions: ") + valueStyle.Render(formatWithCommas(m.chainInfo.TxCount)) + "\n")
	b.WriteString(labelStyle.Render(" Epoch:        ") + valueStyle.Render(m.chainInfo.Epoch) + "\n")

	// Recent transactions -- adaptive: fill remaining rows
	// Fixed lines above: 1 blank + 3 stats = 4
	fixedLines := 4
	availForTx := topRows - fixedLines - 2 // 2 = header line + blank line before txs
	if availForTx > 0 && len(m.recentTxs) > 0 {
		b.WriteString("\nRecent Transactions\n\n")
		showCount := availForTx
		if showCount > len(m.recentTxs) {
			showCount = len(m.recentTxs)
		}
		for i := 0; i < showCount; i++ {
			tx := m.recentTxs[i]
			statusStr := grayStyle.Render(tx.Status)
			if tx.Status == "success" {
				statusStr = valueStyle.Render("OK")
			} else if tx.Status == "failure" {
				statusStr = lipgloss.NewStyle().Foreground(red).Render("FAIL")
			}
			senderStr := grayStyle.Render(tx.Sender)
			kindStr := grayStyle.Render(fmt.Sprintf("%-9s", tx.Kind))
			gasStr := grayStyle.Render(fmt.Sprintf("%8s", tx.GasUsed))
			ageStr := grayStyle.Render(fmt.Sprintf("%4s", tx.Age))
			b.WriteString(fmt.Sprintf("  %-6s %s  %s %s %s\n", statusStr, senderStr, kindStr, gasStr, ageStr))
		}
	}
	return b.String()
}
