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
	Info    = SpacedPrinter{pterm.PrefixPrinter{Prefix: pterm.Prefix{Text: "  INF  ", Style: pterm.NewStyle(pterm.FgBlack, pterm.BgCyan, pterm.Bold)}, MessageStyle: pterm.NewStyle(pterm.FgDefault)}}
	Success = SpacedPrinter{pterm.PrefixPrinter{Prefix: pterm.Prefix{Text: "SUCCESS", Style: pterm.NewStyle(pterm.FgBlack, pterm.BgGreen, pterm.Bold)}, MessageStyle: pterm.NewStyle(pterm.FgDefault)}}
	Warn    = SpacedPrinter{pterm.PrefixPrinter{Prefix: pterm.Prefix{Text: "WARNING", Style: pterm.NewStyle(pterm.FgBlack, pterm.BgYellow, pterm.Bold)}, MessageStyle: pterm.NewStyle(pterm.FgDefault)}}
	Error   = SpacedPrinter{pterm.PrefixPrinter{Prefix: pterm.Prefix{Text: " ERROR ", Style: pterm.NewStyle(pterm.FgBlack, pterm.BgRed, pterm.Bold)}, MessageStyle: pterm.NewStyle(pterm.FgDefault)}}
)

type SpacedPrinter struct {
	pterm.PrefixPrinter
}

func (s SpacedPrinter) Print(a ...any) *pterm.TextPrinter {
	p := s.PrefixPrinter.Print(a...)
	pterm.Println()
	return p
}

func (s SpacedPrinter) Println(a ...any) *pterm.TextPrinter {
	p := s.PrefixPrinter.Println(a...)
	pterm.Println()
	return p
}

func (s SpacedPrinter) Printf(format string, a ...any) *pterm.TextPrinter {
	p := s.PrefixPrinter.Printf(format, a...)
	pterm.Println()
	return p
}

func (s SpacedPrinter) Printfln(format string, a ...any) *pterm.TextPrinter {
	p := s.PrefixPrinter.Printfln(format, a...)
	pterm.Println()
	return p
}

func (s SpacedPrinter) Sprint(a ...any) string {
	return s.PrefixPrinter.Sprint(a...) + "\n"
}

func (s SpacedPrinter) Sprintln(a ...any) string {
	return s.PrefixPrinter.Sprintln(a...) + "\n"
}

func (s SpacedPrinter) Sprintf(format string, a ...any) string {
	return s.PrefixPrinter.Sprintf(format, a...) + "\n"
}

func (s SpacedPrinter) Sprintfln(format string, a ...any) string {
	return s.PrefixPrinter.Sprintfln(format, a...) + "\n"
}

func init() {
	pterm.EnableColor()
}

// Spin configures and returns a spinner
func Spin(text string) (*pterm.SpinnerPrinter, error) {
	chars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	seqLen := 40
	var gradientSeq []string

	orange := pterm.NewRGB(255, 116, 0)
	complementary := pterm.NewRGB(0, 139, 255) // Blue complementary color

	for i := 0; i < seqLen; i++ {
		char := chars[i%len(chars)]
		var c pterm.RGB
		half := seqLen / 2
		if i < half {
			c = orange.Fade(0, float32(half-1), float32(i), complementary)
		} else {
			c = complementary.Fade(float32(half), float32(seqLen-1), float32(i), orange)
		}
		gradientSeq = append(gradientSeq, c.Sprint(char))
	}

	pterm.DefaultSpinner.Sequence = gradientSeq
	pterm.DefaultSpinner.SuccessPrinter = &Success
	pterm.DefaultSpinner.FailPrinter = &Error
	pterm.DefaultSpinner.WarningPrinter = &Warn
	pterm.DefaultSpinner.InfoPrinter = &Info
	return pterm.DefaultSpinner.WithText(text).Start()
}

// Confirm asks the user for permission
func Confirm(message string) bool {
	result, _ := pterm.DefaultInteractiveConfirm.WithDefaultText(message).Show()
	pterm.Println()
	return result
}
