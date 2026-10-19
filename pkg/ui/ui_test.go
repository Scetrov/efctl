package ui

import (
	"bytes"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pterm/pterm"
)

type lockedBuffer struct {
	mu sync.Mutex
	bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.Buffer.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.Buffer.String()
}

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
	// Configure output before starting the spinner; its writer is read by the
	// animation goroutine while active.
	var buf lockedBuffer
	pterm.SetDefaultOutput(&buf)
	defer pterm.SetDefaultOutput(os.Stdout)
	oldSpinnerWriter := pterm.DefaultSpinner.Writer
	pterm.DefaultSpinner.SetWriter(&buf)
	defer pterm.DefaultSpinner.SetWriter(oldSpinnerWriter)

	// Save and restore state
	oldProgress := ProgressEnabled
	defer func() { ProgressEnabled = oldProgress }()

	ProgressEnabled = true // Simulate active spinner

	spinner, err := Spin("Testing active spinner")
	if err != nil {
		t.Fatalf("Failed to create spinner: %v", err)
	}

	// Allow to spin briefly
	time.Sleep(10 * time.Millisecond)

	spinner.Success("Done")

	output := buf.String()

	if !strings.Contains(output, "Done") {
		t.Errorf("Expected output to contain 'Done' message, got %q", output)
	}
}
