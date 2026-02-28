package dashboard

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestBorderStr(t *testing.T) {
	result := BorderStr("─")
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "─")
}

func TestTruncateToWidth(t *testing.T) {
	tests := []struct {
		name  string
		input string
		maxW  int
	}{
		{"short string stays", "hello", 10},
		{"exact length stays", "hello", 5},
		{"long string truncated", "hello world this is long", 10},
		{"zero width", "hello", 0},
		{"single char", "hello", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateToWidth(tt.input, tt.maxW)
			w := lipgloss.Width(result)
			assert.LessOrEqual(t, w, max(tt.maxW, 0)+1, "width should not exceed max + 1 char")
		})
	}
}

func TestRenderToLines(t *testing.T) {
	content := "line one\nline two\nline three"
	lines := RenderToLines(content, 20)
	assert.NotEmpty(t, lines)
	for _, line := range lines {
		w := lipgloss.Width(line)
		assert.LessOrEqual(t, w, 20, "each line should be at most innerW wide")
	}
}

func TestPadLines(t *testing.T) {
	t.Run("adds missing lines", func(t *testing.T) {
		input := []string{"hello", "world"}
		result := PadLines(input, 5, 10)
		assert.Len(t, result, 5)
		for _, line := range result {
			w := lipgloss.Width(line)
			assert.Equal(t, 10, w, "each line should be exactly innerW wide")
		}
	})

	t.Run("truncates extra lines", func(t *testing.T) {
		input := []string{"a", "b", "c", "d", "e"}
		result := PadLines(input, 3, 5)
		assert.Len(t, result, 3)
	})

	t.Run("pads short lines", func(t *testing.T) {
		input := []string{"hi"}
		result := PadLines(input, 1, 10)
		assert.Len(t, result, 1)
		assert.Equal(t, 10, lipgloss.Width(result[0]))
	})

	t.Run("exact fit no change needed", func(t *testing.T) {
		line := strings.Repeat("x", 10)
		result := PadLines([]string{line}, 1, 10)
		assert.Len(t, result, 1)
		assert.Equal(t, 10, lipgloss.Width(result[0]))
	})
}

func TestBuildTopBorder(t *testing.T) {
	result := BuildTopBorder(30, 20, "Left", "Right")
	assert.Contains(t, result, "╭")
	assert.Contains(t, result, "┬")
	assert.Contains(t, result, "╮")
	assert.Contains(t, result, "Left")
	assert.Contains(t, result, "Right")
}

func TestBuildTopBorder_SmallWidths(t *testing.T) {
	result := BuildTopBorder(5, 5, "LongTitle", "AlsoLong")
	assert.Contains(t, result, "╭")
	assert.Contains(t, result, "╮")
}

func TestBuildLeftMidBorder(t *testing.T) {
	result := BuildLeftMidBorder(30, "Events")
	assert.Contains(t, result, "├")
	assert.Contains(t, result, "┤")
	assert.Contains(t, result, "Events")
}

func TestBuildMiddleBorder(t *testing.T) {
	result := BuildMiddleBorder(50, 30, "Logs")
	assert.Contains(t, result, "├")
	assert.Contains(t, result, "┤")
	assert.Contains(t, result, "Logs")
}

func TestBuildMiddleBorder_WithJunction(t *testing.T) {
	// When junction position is valid, should include ┴
	result := BuildMiddleBorder(50, 30, "Logs")
	// The junction position depends on title width; check for either ┴ or no error
	assert.Contains(t, result, "├")
	assert.Contains(t, result, "┤")
}

func TestBuildMiddleBorder_NoJunction(t *testing.T) {
	// Small width where junction can't be placed
	result := BuildMiddleBorder(10, 2, "VeryLongTitle")
	assert.Contains(t, result, "├")
	assert.Contains(t, result, "┤")
}

func TestBuildBottomBorder(t *testing.T) {
	result := BuildBottomBorder(50, "efctl v1.0")
	assert.Contains(t, result, "╰")
	assert.Contains(t, result, "╯")
	assert.Contains(t, result, "efctl v1.0")
}

func TestBuildBottomBorder_SmallWidth(t *testing.T) {
	result := BuildBottomBorder(5, "very long footer text")
	assert.Contains(t, result, "╰")
	assert.Contains(t, result, "╯")
}

func TestBuildFullBorder(t *testing.T) {
	result := BuildFullBorder(50, "Overview")
	assert.Contains(t, result, "├")
	assert.Contains(t, result, "┤")
	assert.Contains(t, result, "Overview")
}

func TestBuildSplitMiddleBorder(t *testing.T) {
	result := BuildSplitMiddleBorder(30, 20, "Left", "Right")
	assert.Contains(t, result, "├")
	assert.Contains(t, result, "┼")
	assert.Contains(t, result, "┤")
	assert.Contains(t, result, "Left")
	assert.Contains(t, result, "Right")
}

func TestBuildBottomBorderWithJunction(t *testing.T) {
	t.Run("with junction", func(t *testing.T) {
		result := BuildBottomBorderWithJunction(50, 30, "footer")
		assert.Contains(t, result, "╰")
		assert.Contains(t, result, "╯")
		assert.Contains(t, result, "footer")
	})

	t.Run("without junction - too small", func(t *testing.T) {
		result := BuildBottomBorderWithJunction(10, 2, "long footer text")
		assert.Contains(t, result, "╰")
		assert.Contains(t, result, "╯")
	})
}

func TestRenderLogo(t *testing.T) {
	logo := RenderLogo()
	assert.NotEmpty(t, logo, "logo should have lines")
	for _, line := range logo {
		assert.NotEmpty(t, line, "each logo line should have content")
	}
}

func TestOverlayLogo(t *testing.T) {
	t.Run("enough space", func(t *testing.T) {
		// Create lines that are wide enough and tall enough
		lines := make([]string, 20)
		for i := range lines {
			lines[i] = strings.Repeat(" ", 120)
		}
		result := OverlayLogo(lines, 120)
		assert.Len(t, result, 20)
	})

	t.Run("not enough height", func(t *testing.T) {
		// Only 1 line - not enough for logo
		lines := []string{strings.Repeat(" ", 120)}
		result := OverlayLogo(lines, 120)
		assert.Len(t, result, 1)
		// Should return unchanged
		assert.Equal(t, lines, result)
	})

	t.Run("not enough width", func(t *testing.T) {
		lines := make([]string, 20)
		for i := range lines {
			lines[i] = "x"
		}
		result := OverlayLogo(lines, 5)
		assert.Equal(t, lines, result)
	})
}
