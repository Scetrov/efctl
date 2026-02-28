// Package dashboard provides pure data-transformation and formatting utilities
// used by the efctl TUI dashboard. These functions are extracted from cmd/env_dash.go
// to enable unit testing without TUI dependencies.
package dashboard

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Colours used by the dashboard.
var (
	Cyan   = lipgloss.Color("#00FFFF")
	Orange = lipgloss.Color("#FF7400")
	Green  = lipgloss.Color("#00CC66")
	Yellow = lipgloss.Color("#FFAA00")
	Red    = lipgloss.Color("#FF4444")
	Gray   = lipgloss.Color("#666666")
)

// FormatAge converts a duration into a short human-readable string.
func FormatAge(d time.Duration) string {
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

// RenderStatus returns a styled status string, using red for Stopped.
func RenderStatus(s string) string {
	if s == "Stopped" {
		return lipgloss.NewStyle().Foreground(Red).Render(s)
	}
	return lipgloss.NewStyle().Foreground(Cyan).Render(s)
}

// FormatWithCommas adds thousand separators to a numeric string.
func FormatWithCommas(s string) string {
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

// ShortKind abbreviates common Sui transaction kind names.
func ShortKind(kind string) string {
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

// FormatGas computes net gas used from Sui gas fields and returns a compact string.
func FormatGas(computation, storage, rebate string) string {
	comp, _ := strconv.ParseInt(computation, 10, 64)
	stor, _ := strconv.ParseInt(storage, 10, 64)
	reb, _ := strconv.ParseInt(rebate, 10, 64)
	total := comp + stor - reb
	if total <= 0 {
		return "-"
	}
	return FormatWithCommas(strconv.FormatInt(total, 10))
}

// ColorizeLogLine applies colour to log line prefixes.
func ColorizeLogLine(line string) string {
	if strings.HasPrefix(line, "[docker]") {
		return lipgloss.NewStyle().Foreground(Cyan).Render("[docker]") + line[8:]
	}
	if strings.HasPrefix(line, "[db]") {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#CC88FF")).Render("[db]") + line[4:]
	}
	if strings.HasPrefix(line, "[deploy]") {
		return lipgloss.NewStyle().Foreground(Green).Render("[deploy]") + line[8:]
	}
	if strings.HasPrefix(line, "[frontend]") {
		return lipgloss.NewStyle().Foreground(Yellow).Render("[frontend]") + line[10:]
	}
	return line
}

// HumanizeCamelCase converts "governorCap" to "Governor Cap", etc.
func HumanizeCamelCase(s string) string {
	if s == "" {
		return ""
	}
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

// LogViewportRows returns the number of log lines visible in the log panel
// given the current terminal height and number of world events.
func LogViewportRows(height, numEvents int) int {
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
