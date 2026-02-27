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
	Digest string
	Status string
	Kind   string
	Age    string
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
	Sui     containerStat
	Pg      containerStat
	Chain   chainStat
	Objects []string
	Admin   string
	EnvVars map[string]string
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
							Transaction struct {
								Kind string `json:"kind"`
							} `json:"transaction"`
						} `json:"data"`
					} `json:"transaction"`
					Effects struct {
						Status struct {
							Status string `json:"status"`
						} `json:"status"`
					} `json:"effects"`
				} `json:"data"`
			} `json:"result"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&res)
		for _, tx := range res.Result.Data {
			d := tx.Digest
			if len(d) > 16 {
				d = d[:8] + "..." + d[len(d)-4:]
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
			info.RecentTxs = append(info.RecentTxs, recentTx{
				Digest: d,
				Status: status,
				Kind:   kind,
				Age:    age,
			})
		}
		_ = resp.Body.Close()
	}

	return info
}

func fetchStats(engine string, workspace string) StatsMsg {
	msg := StatsMsg{
		Sui: containerStat{Status: "Stopped", CPU: "-", Mem: "-"},
		Pg:  containerStat{Status: "Stopped", CPU: "-", Mem: "-"},
	}

	// fetch container stats
	out, err := exec.Command(engine, "stats", "--no-stream", "--format", "{{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}").Output() // #nosec G204
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, l := range lines {
			parts := strings.Split(l, "\t")
			if len(parts) >= 3 {
				name := strings.TrimSpace(parts[0])
				if name == container.ContainerSuiPlayground {
					msg.Sui.Status = "Running"
					msg.Sui.CPU = parts[1]
					msg.Sui.Mem = parts[2]
				}
				if name == container.ContainerPostgres {
					msg.Pg.Status = "Running"
					msg.Pg.CPU = parts[1]
					msg.Pg.Mem = parts[2]
				}
			}
		}
	}

	client := &http.Client{Timeout: 1 * time.Second}
	msg.Chain = fetchChainInfo(client)

	// Check extracted objects
	extractFile := filepath.Join(workspace, "world-contracts", "deployments", "localnet", "extracted-object-ids.json")
	if data, err := os.ReadFile(extractFile); err == nil { // #nosec G304 -- path constructed from known workspace prefix
		var objMap map[string]interface{}
		if json.Unmarshal(data, &objMap) == nil {
			for k, v := range objMap {
				if str, ok := v.(string); ok {
					// Only keep first few chars or handle them nicely
					shortObj := str
					if len(str) > 10 {
						shortObj = str[:6] + "..." + str[len(str)-4:]
					}
					msg.Objects = append(msg.Objects, fmt.Sprintf("%-20s %s", k, shortObj))
				}
			}
		}
	}

	// Admin
	msg.Admin = extractAdmin(workspace)

	// Environment variables
	msg.EnvVars = extractEnvVars(workspace)

	return msg
}

func collectLogs(ctx context.Context, p *tea.Program, engine, workspace string) {
	// 1. Container logs
	cmd := exec.CommandContext(ctx, engine, "logs", "-f", "--tail", "20", container.ContainerSuiPlayground) // #nosec G204
	stdout, err := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout
	if err == nil {
		if err := cmd.Start(); err == nil {
			go func() {
				scanner := bufio.NewScanner(stdout)
				for scanner.Scan() {
					p.Send(LogMsg(fmt.Sprintf("[docker] %s", scanner.Text())))
				}
				_ = cmd.Wait() // Reclaim process
			}()
		}
	}

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
	logLines := padLines(renderToLines(strings.Join(m.logs[startIdx:], "\n"), logInner), botRows, logInner)

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
	b.WriteString(fmt.Sprintf("   Status: %-10s  CPU: %-8s Mem: %s\n\n", valueStyle.Render(m.suiStat.Status), m.suiStat.CPU, m.suiStat.Mem))
	b.WriteString(" database\n")
	b.WriteString(fmt.Sprintf("   Status: %-10s  CPU: %-8s Mem: %s\n", valueStyle.Render(m.pgStat.Status), m.pgStat.CPU, m.pgStat.Mem))
	return b.String()
}

func (m model) renderEnvContent() string {
	var b bytes.Buffer

	// Shorten an address/value for display
	shorten := func(s string, maxLen int) string {
		if len(s) > maxLen {
			return s[:maxLen-3] + "..."
		}
		return s
	}

	adminDisp := shorten(m.adminAddr, 42)
	b.WriteString("\n")
	b.WriteString(labelStyle.Render(" ADMIN_ADDRESS:  ") + " " + adminDisp + "\n")

	if v, ok := m.envVars["SPONSOR_ADDRESS"]; ok {
		b.WriteString(labelStyle.Render(" SPONSOR_ADDR:   ") + " " + shorten(v, 42) + "\n")
	}

	network := "localnet"
	if v, ok := m.envVars["SUI_NETWORK"]; ok {
		network = v
	}
	b.WriteString(labelStyle.Render(" SUI_NETWORK:    ") + " " + network + "\n")
	b.WriteString(labelStyle.Render(" RPC URL:        ") + " http://localhost:9000\n")

	if v, ok := m.envVars["WORLD_PACKAGE_ID"]; ok {
		b.WriteString(labelStyle.Render(" WORLD_PKG:      ") + " " + shorten(v, 42) + "\n")
	}
	if v, ok := m.envVars["BUILDER_PACKAGE_ID"]; ok {
		b.WriteString(labelStyle.Render(" BUILDER_PKG:    ") + " " + shorten(v, 42) + "\n")
	}
	if v, ok := m.envVars["TENANT"]; ok {
		b.WriteString(labelStyle.Render(" TENANT:         ") + " " + v + "\n")
	}

	return b.String()
}

func (m model) renderRightContent(topRows int) string {
	var b bytes.Buffer
	b.WriteString("\n")
	b.WriteString(labelStyle.Render(" Checkpoint:   ") + valueStyle.Render(m.chainInfo.Checkpoint) + "\n")
	b.WriteString(labelStyle.Render(" Transactions: ") + valueStyle.Render(m.chainInfo.TxCount) + "\n")
	b.WriteString(labelStyle.Render(" Epoch:        ") + valueStyle.Render(m.chainInfo.Epoch) + "\n\n")

	b.WriteString("Object Tracker\n\n")

	if len(m.objectTrackers) == 0 {
		b.WriteString(grayStyle.Render("  No objects tracked yet.\n"))
	} else {
		for _, obj := range m.objectTrackers {
			b.WriteString(fmt.Sprintf("  %s\n", obj))
		}
	}

	// Recent transactions -- adaptive: fill remaining rows
	// Fixed lines above: 1 blank + 3 stats + 1 blank + 1 header + 1 blank + object lines + 1 blank
	fixedLines := 8
	objLines := len(m.objectTrackers)
	if objLines == 0 {
		objLines = 1 // "No objects tracked yet."
	}
	usedLines := fixedLines + objLines
	availForTx := topRows - usedLines - 2 // 2 = header line + blank line before txs
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
				statusStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444")).Render("FAIL")
			}
			kindStr := grayStyle.Render(fmt.Sprintf("%-16s", tx.Kind))
			digestStr := grayStyle.Render(tx.Digest)
			ageStr := grayStyle.Render(tx.Age)
			b.WriteString(fmt.Sprintf("  %-6s %s %s %s\n", statusStr, kindStr, digestStr, ageStr))
		}
	}
	return b.String()
}
