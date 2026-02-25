package ui

import (
	"github.com/pterm/pterm"
)

var (
	// Emojis
	SuccessEmoji = "✅"
	ErrorEmoji   = "❌"
	InfoEmoji    = "ℹ️ "
	DockerEmoji  = "🐳"
	PodmanEmoji  = "🦭"
	GitEmoji     = "📦"
	CleanEmoji   = "🧹"
	PlayEmoji    = "▶️ "
	GlobeEmoji   = "🌍"

	// Printers
	Info    = pterm.PrefixPrinter{Prefix: pterm.Prefix{Text: InfoEmoji, Style: pterm.NewStyle(pterm.FgCyan)}, MessageStyle: pterm.NewStyle(pterm.FgDefault)}
	Success = pterm.PrefixPrinter{Prefix: pterm.Prefix{Text: SuccessEmoji, Style: pterm.NewStyle(pterm.FgGreen)}, MessageStyle: pterm.NewStyle(pterm.FgDefault)}
	Warn    = pterm.PrefixPrinter{Prefix: pterm.Prefix{Text: "⚠️ ", Style: pterm.NewStyle(pterm.FgYellow)}, MessageStyle: pterm.NewStyle(pterm.FgDefault)}
	Error   = pterm.PrefixPrinter{Prefix: pterm.Prefix{Text: ErrorEmoji, Style: pterm.NewStyle(pterm.FgRed)}, MessageStyle: pterm.NewStyle(pterm.FgDefault)}
)

func init() {
	pterm.EnableColor()
}

// Spin configures and returns a spinner
func Spin(text string) (*pterm.SpinnerPrinter, error) {
	pterm.DefaultSpinner.Sequence = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	return pterm.DefaultSpinner.WithText(text).Start()
}
