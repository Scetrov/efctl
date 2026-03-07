package ui

import (
	"github.com/pterm/pterm"
)

// DebugEnabled controls whether Debug.Println and friends produce output.
// Set to true via the global --debug flag.
var DebugEnabled bool

// ProgressEnabled controls whether spinners are displayed.
// Set to false via the global --no-progress flag or CI env var.
var ProgressEnabled = true

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
	Info    = SpacedPrinter{pterm.PrefixPrinter{Prefix: pterm.Prefix{Text: "  INF  ", Style: pterm.NewStyle(pterm.FgBlack, pterm.BgCyan)}, MessageStyle: pterm.NewStyle(pterm.FgDefault)}}
	Success = SpacedPrinter{pterm.PrefixPrinter{Prefix: pterm.Prefix{Text: "SUCCESS", Style: pterm.NewStyle(pterm.FgBlack, pterm.BgGreen)}, MessageStyle: pterm.NewStyle(pterm.FgDefault)}}
	Warn    = SpacedPrinter{pterm.PrefixPrinter{Prefix: pterm.Prefix{Text: "WARNING", Style: pterm.NewStyle(pterm.FgBlack, pterm.BgYellow)}, MessageStyle: pterm.NewStyle(pterm.FgDefault)}}
	Error   = SpacedPrinter{pterm.PrefixPrinter{Prefix: pterm.Prefix{Text: " ERROR ", Style: pterm.NewStyle(pterm.FgBlack, pterm.BgRed)}, MessageStyle: pterm.NewStyle(pterm.FgDefault)}}

	// Debug uses a distinct prefix; output is suppressed unless DebugEnabled is set.
	Debug = DebugPrinter{SpacedPrinter{pterm.PrefixPrinter{Prefix: pterm.Prefix{Text: " DEBUG ", Style: pterm.NewStyle(pterm.FgBlack, pterm.BgMagenta)}, MessageStyle: pterm.NewStyle(pterm.FgGray)}}}
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

// DebugPrinter wraps SpacedPrinter and only emits output when DebugEnabled is true.
type DebugPrinter struct {
	SpacedPrinter
}

func (d DebugPrinter) Print(a ...any) {
	if !DebugEnabled {
		return
	}
	d.SpacedPrinter.Print(a...)
}

func (d DebugPrinter) Println(a ...any) {
	if !DebugEnabled {
		return
	}
	d.SpacedPrinter.Println(a...)
}

func (d DebugPrinter) Printf(format string, a ...any) {
	if !DebugEnabled {
		return
	}
	d.SpacedPrinter.Printf(format, a...)
}

func (d DebugPrinter) Printfln(format string, a ...any) {
	if !DebugEnabled {
		return
	}
	d.SpacedPrinter.Printfln(format, a...)
}

func init() {
	pterm.EnableColor()
}

// Ensure our custom spacing applies to spinners manually without relying on pterm's implicit newlines.
type SpacedSpinner struct {
	*pterm.SpinnerPrinter
}

// Success displays the success printer with trailing newline spacing
func (s SpacedSpinner) Success(message ...any) {
	wasActive := s.IsActive && !pterm.RawOutput
	s.SpinnerPrinter.Success(message...)
	if !wasActive {
		pterm.Println()
	}
}

// Fail displays the fail printer with trailing newline spacing
func (s SpacedSpinner) Fail(message ...any) {
	wasActive := s.IsActive && !pterm.RawOutput
	s.SpinnerPrinter.Fail(message...)
	if !wasActive {
		pterm.Println()
	}
}

// Warning displays the warning printer with trailing newline spacing
func (s SpacedSpinner) Warning(message ...any) {
	wasActive := s.IsActive && !pterm.RawOutput
	s.SpinnerPrinter.Warning(message...)
	if !wasActive {
		pterm.Println()
	}
}

// Info displays the info printer with trailing newline spacing
func (s SpacedSpinner) Info(message ...any) {
	wasActive := s.IsActive && !pterm.RawOutput
	s.SpinnerPrinter.Info(message...)
	if !wasActive {
		pterm.Println()
	}
}

// Spin configures and returns a spaced spinner
func Spin(text string) (*SpacedSpinner, error) {
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

	s := pterm.DefaultSpinner.WithText(text)
	if !ProgressEnabled {
		return &SpacedSpinner{s}, nil
	}
	s, err := s.Start()
	return &SpacedSpinner{s}, err
}

// Confirm asks the user for permission
func Confirm(message string) bool {
	result, _ := pterm.DefaultInteractiveConfirm.WithDefaultText(message).Show()
	pterm.Println()
	return result
}
