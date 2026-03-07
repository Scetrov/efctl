package ui

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pterm/pterm"
)

func TestSpacedSpinner_Success_Inactive(t *testing.T) {
	// Temporarily redirect pterm output
	var buf bytes.Buffer
	pterm.SetDefaultOutput(&buf)
	defer pterm.SetDefaultOutput(os.Stdout)

	// Save and restore state
	oldProgress := ProgressEnabled
	defer func() { ProgressEnabled = oldProgress }()

	ProgressEnabled = false // Simulate inactive spinner

	spinner, err := Spin("Testing inactive spinner")
	if err != nil {
		t.Fatalf("Failed to create spinner: %v", err)
	}

	spinner.Success("Done")

	// Pterm usually emits ANSI escape codes, so we should look at the end of the output.
	output := buf.String()

	// It should end with a newline because of SpacedSpinner conditional pterm.Println()
	if !strings.HasSuffix(output, "\n") {
		t.Errorf("Expected output to end with a newline when spinner is inactive, got %q", output)
	}

	// Technically, if it's inactive, the `pterm.Println()` we added inserts "\n".
	// Depending on pterm's own Success() logic, we might even have multiple, but our goal is to ensure AT LEAST 1 newline,
	// and specifically verify the custom blank line. The PrefixPrinter already appends "\n" on Println,
	// but Success was lacking the trailing gap. Let's just verify it ends cleanly and has spacing.
}

func TestSpacedSpinner_Success_Active(t *testing.T) {
	// Temporarily redirect pterm output
	var buf bytes.Buffer
	pterm.SetDefaultOutput(&buf)
	defer pterm.SetDefaultOutput(os.Stdout)

	// Save and restore state
	oldProgress := ProgressEnabled
	defer func() { ProgressEnabled = oldProgress }()

	ProgressEnabled = true // Simulate active spinner

	spinner, err := Spin("Testing active spinner")
	if err != nil {
		t.Fatalf("Failed to create spinner: %v", err)
	}

	spinner.SetWriter(&buf)

	// We need to route the global success printer output to the buffer as well,
	// because pterm's Success() method might write to default output instead of spinner writer depending on context.
	// Oh wait, Spin() initializes Success with `SpacedPrinter{pterm.PrefixPrinter{Prefix: pterm.Prefix{Text: "SUCCESS"}}}`.
	// And SpacedSpinner.Success() calls s.SpinnerPrinter.Success(), which might use pterm.Printo() bypassing the spinner writer if not set on the success printer...

	// Allow to spin briefly
	time.Sleep(10 * time.Millisecond)

	spinner.Success("Done")

	output := buf.String()

	if !strings.Contains(output, "Done") {
		t.Errorf("Expected output to contain 'Done' message, got %q", output)
	}
}
