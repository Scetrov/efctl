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

type chainStat struct {
	Checkpoint string
	Epoch      string
	TxCount    string
}

type TickMsg time.Time
type LogMsg string
type StatsMsg struct {
	Sui     containerStat
	Pg      containerStat
	Chain   chainStat
	Objects []string
	Admin   string
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

	// fetch chain info via JSON RPC
	rpcPayload := `{"jsonrpc":"2.0","id":1,"method":"sui_getLatestCheckpointSequenceNumber","params":[]}`
	rpcReq, _ := http.NewRequest("POST", "http://localhost:9000", strings.NewReader(rpcPayload))
	rpcReq.Header.Set("Content-Type", "application/json")
	if resp, err := client.Do(rpcReq); err == nil { // #nosec G704 -- hardcoded localhost URL
		var res struct {
			Result string `json:"result"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&res)
		msg.Chain.Checkpoint = res.Result
		_ = resp.Body.Close()
	} else {
		msg.Chain.Checkpoint = "Offline"
	}

	// Check total txs
	rpcPayloadTx := `{"jsonrpc":"2.0","id":1,"method":"sui_getTotalTransactionBlocks","params":[]}`
	rpcReqTx, _ := http.NewRequest("POST", "http://localhost:9000", strings.NewReader(rpcPayloadTx))
	rpcReqTx.Header.Set("Content-Type", "application/json")
	if resp, err := client.Do(rpcReqTx); err == nil { // #nosec G704 -- hardcoded localhost URL
		var res struct {
			Result string `json:"result"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&res)
		msg.Chain.TxCount = res.Result
		_ = resp.Body.Close()
	} else {
		msg.Chain.TxCount = "-"
	}

	// Check epoch
	rpcPayloadEpoch := `{"jsonrpc":"2.0","id":1,"method":"sui_getLatestSuiSystemState","params":[]}`
	rpcReqEpoch, _ := http.NewRequest("POST", "http://localhost:9000", strings.NewReader(rpcPayloadEpoch))
	rpcReqEpoch.Header.Set("Content-Type", "application/json")
	if resp, err := client.Do(rpcReqEpoch); err == nil { // #nosec G704 -- hardcoded localhost URL
		var res struct {
			Result map[string]interface{} `json:"result"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&res)
		if ep, ok := res.Result["epoch"].(string); ok {
			msg.Chain.Epoch = ep
		}
		_ = resp.Body.Close()
	} else {
		msg.Chain.Epoch = "-"
	}

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
	objectTrackers []string
	adminAddr      string
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
		m.objectTrackers = msg.Objects
		m.adminAddr = msg.Admin

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
	//   ╭─ Left ────┬─ Right ───╮
	//   │           │           │
	//   ├─ Logs ─┴──────────────┤
	//   │                       │
	//   ╰─ keybindings ─────────╯
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

	// Vertical budget: headerH + 1(top border) + topRows + 1(mid border) + botRows + 1(bot border)
	available := m.height - headerH
	topRows := (available * 2) / 5
	if topRows < 5 {
		topRows = 5
	}
	botRows := available - topRows - 3
	if botRows < 3 {
		botRows = 3
	}

	// ── Render panel content ──
	leftLines := padLines(renderToLines(m.renderLeftContent(), leftInner), topRows, leftInner)
	rightLines := padLines(renderToLines(m.renderRightContent(), rightInner), topRows, rightInner)

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
	leftTitle := "Container Status"
	rightTitle := "Chain Info"
	out.WriteString(buildTopBorder(leftInner, rightInner, leftTitle, rightTitle))
	out.WriteByte('\n')

	// Top panel rows
	for i := 0; i < topRows; i++ {
		out.WriteString(borderStr("│") + leftLines[i] + borderStr("│") + rightLines[i] + borderStr("│"))
		out.WriteByte('\n')
	}

	// Middle border
	logTitle := "Combined Log View"
	out.WriteString(buildMiddleBorder(m.width, leftInner, logTitle))
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

func (m model) renderLeftContent() string {
	var b bytes.Buffer
	b.WriteString("\n sui-playground\n")
	b.WriteString(fmt.Sprintf("   Status: %-10s  CPU: %-8s Mem: %s\n\n", valueStyle.Render(m.suiStat.Status), m.suiStat.CPU, m.suiStat.Mem))

	b.WriteString(" database\n")
	b.WriteString(fmt.Sprintf("   Status: %-10s  CPU: %-8s Mem: %s\n\n", valueStyle.Render(m.pgStat.Status), m.pgStat.CPU, m.pgStat.Mem))

	b.WriteString("Environment Info\n\n")

	adminDisp := m.adminAddr
	if len(adminDisp) > 40 {
		adminDisp = adminDisp[:37] + "..."
	}

	b.WriteString(labelStyle.Render(" ADMIN_ADDRESS: ") + " " + adminDisp + "\n")
	b.WriteString(labelStyle.Render(" SUI_NETWORK:   ") + " localnet\n")
	b.WriteString(labelStyle.Render(" RPC URL:       ") + " http://localhost:9000\n")
	return b.String()
}

func (m model) renderRightContent() string {
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
	return b.String()
}
