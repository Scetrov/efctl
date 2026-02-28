package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
)

var labelStyle = lipgloss.NewStyle().Foreground(Orange).Bold(true)
var grayStyle = lipgloss.NewStyle().Foreground(Gray)

// BorderStr renders s in the border (cyan) colour.
func BorderStr(s string) string {
	return lipgloss.NewStyle().Foreground(Cyan).Render(s)
}

// TruncateToWidth truncates a styled string to at most maxW visible characters.
func TruncateToWidth(s string, maxW int) string {
	if lipgloss.Width(s) <= maxW {
		return s
	}
	var out strings.Builder
	for _, r := range s {
		out.WriteRune(r)
		if lipgloss.Width(out.String()) >= maxW {
			break
		}
	}
	result := out.String()
	w := lipgloss.Width(result)
	if w < maxW {
		result += strings.Repeat(" ", maxW-w)
	}
	return result
}

// RenderToLines renders content with 1-char horizontal padding into a
// slice of lines, each exactly innerW visual characters wide.
func RenderToLines(content string, innerW int) []string {
	rendered := lipgloss.NewStyle().Padding(0, 1).Width(innerW).Render(content)
	return strings.Split(rendered, "\n")
}

// PadLines ensures exactly targetRows lines, each innerW visual chars wide.
func PadLines(lines []string, targetRows, innerW int) []string {
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

// BuildTopBorder builds: ╭─ LeftTitle ──┬─ RightTitle ──╮
func BuildTopBorder(leftW, rightW int, leftTitle, rightTitle string) string {
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
	return BorderStr("╭─") + " " + labelStyle.Render(leftTitle) + " " +
		BorderStr(strings.Repeat("─", ld)+"┬─") + " " + labelStyle.Render(rightTitle) + " " +
		BorderStr(strings.Repeat("─", rd)+"╮")
}

// BuildLeftMidBorder builds: ├─ Title ──────────┤ (left-side only, with ┤ connecting to │)
func BuildLeftMidBorder(leftW int, title string) string {
	tw := lipgloss.Width(title)
	d := leftW - 3 - tw
	if d < 0 {
		d = 0
	}
	return BorderStr("├─") + " " + labelStyle.Render(title) + " " +
		BorderStr(strings.Repeat("─", d)+"┤")
}

// BuildMiddleBorder builds: ├─ Title ──┴────────┤
// The ┴ character is placed where the top-section vertical divider was.
func BuildMiddleBorder(totalW, leftW int, title string) string {
	tw := lipgloss.Width(title)
	totalDashes := totalW - 5 - tw
	if totalDashes < 0 {
		totalDashes = 0
	}
	junction := leftW - 3 - tw
	if junction >= 0 && junction < totalDashes {
		return BorderStr("├─") + " " + labelStyle.Render(title) + " " +
			BorderStr(strings.Repeat("─", junction)+"┴"+strings.Repeat("─", totalDashes-junction-1)+"┤")
	}
	return BorderStr("├─") + " " + labelStyle.Render(title) + " " +
		BorderStr(strings.Repeat("─", totalDashes)+"┤")
}

// BuildBottomBorder builds: ╰─ footer ──────╯
func BuildBottomBorder(totalW int, footer string) string {
	fw := lipgloss.Width(footer)
	d := totalW - 5 - fw
	if d < 0 {
		d = 0
	}
	return BorderStr("╰─") + " " + grayStyle.Render(footer) + " " +
		BorderStr(strings.Repeat("─", d)+"╯")
}

// BuildFullBorder builds: ├─ Title ──────────────┤ (full width, no junction)
func BuildFullBorder(totalW int, title string) string {
	tw := lipgloss.Width(title)
	d := totalW - 5 - tw
	if d < 0 {
		d = 0
	}
	return BorderStr("├─") + " " + labelStyle.Render(title) + " " +
		BorderStr(strings.Repeat("─", d)+"┤")
}

// BuildSplitMiddleBorder builds: ├─ LeftTitle ──┼─ RightTitle ──┤
// Used when a vertical divider continues through top and bottom sections.
func BuildSplitMiddleBorder(leftW, rightW int, leftTitle, rightTitle string) string {
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
	return BorderStr("├─") + " " + labelStyle.Render(leftTitle) + " " +
		BorderStr(strings.Repeat("─", ld)+"┼─") + " " + labelStyle.Render(rightTitle) + " " +
		BorderStr(strings.Repeat("─", rd)+"┤")
}

// BuildBottomBorderWithJunction builds: ╰─ footer ──┴──────╯
// Places a ┴ junction at the vertical divider position.
func BuildBottomBorderWithJunction(totalW, leftW int, footer string) string {
	fw := lipgloss.Width(footer)
	totalDashes := totalW - 5 - fw
	if totalDashes < 0 {
		totalDashes = 0
	}
	junction := leftW - 3 - fw
	if junction >= 0 && junction < totalDashes {
		return BorderStr("╰─") + " " + grayStyle.Render(footer) + " " +
			BorderStr(strings.Repeat("─", junction)+"┴"+strings.Repeat("─", totalDashes-junction-1)+"╯")
	}
	return BorderStr("╰─") + " " + grayStyle.Render(footer) + " " +
		BorderStr(strings.Repeat("─", totalDashes)+"╯")
}

// efctlLogoLines holds the raw (uncolored) pterm BigText for "> EFCTL".
var efctlLogoLines []string

func init() {
	raw, _ := pterm.DefaultBigText.WithLetters(putils.LettersFromString("> EFCTL")).Srender()
	for _, line := range strings.Split(raw, "\n") {
		clean := pterm.RemoveColorFromString(line)
		if strings.TrimSpace(clean) != "" {
			efctlLogoLines = append(efctlLogoLines, clean)
		}
	}
}

// RenderLogo renders the EFCTL logo with a horizontal gradient from cyan to orange.
func RenderLogo() []string {
	result := make([]string, len(efctlLogoLines))
	for i, line := range efctlLogoLines {
		runes := []rune(line)
		numCols := float64(max(len(runes)-1, 1))
		var out strings.Builder
		for j, r := range runes {
			t := float64(j) / numCols
			rd := int(t * 255)
			gn := int(255 - t*(255-116))
			bl := int(255 - t*255)
			c := lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", rd, gn, bl))
			out.WriteString(lipgloss.NewStyle().Foreground(c).Render(string(r)))
		}
		result[i] = out.String()
	}
	return result
}

// OverlayLogo places the gradient logo on the bottom-right of the log lines.
func OverlayLogo(logLines []string, innerW int) []string {
	logo := RenderLogo()
	logoW := lipgloss.Width(efctlLogoLines[0])
	logoH := len(logo)
	if len(logLines) < logoH || innerW < logoW+4 {
		return logLines // not enough space
	}
	for i := 0; i < logoH; i++ {
		row := len(logLines) - logoH + i
		line := logLines[row]
		padLeft := innerW - logoW - 2
		if padLeft < 0 {
			padLeft = 0
		}
		leftPart := strings.Repeat(" ", padLeft)
		if lipgloss.Width(line) >= padLeft {
			leftPart = TruncateToWidth(line, padLeft)
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

// BuildAddresses assembles the role→address map from env vars and derived keys.
// deriveAddr is an injected function that converts a private key to an address.
func BuildAddresses(admin string, envVars map[string]string, deriveAddr func(string) string) map[string]string {
	addrs := make(map[string]string)
	if admin != "" && admin != "Unknown" && admin != "Not Found" {
		addrs["Admin"] = admin
	}
	if v, ok := envVars["SPONSOR_ADDRESS"]; ok {
		addrs["Sponsor"] = v
	}
	if pk, ok := envVars["PLAYER_A_PRIVATE_KEY"]; ok {
		if addr := deriveAddr(pk); addr != "" {
			addrs["Player A"] = addr
		}
	}
	if pk, ok := envVars["PLAYER_B_PRIVATE_KEY"]; ok {
		if addr := deriveAddr(pk); addr != "" {
			addrs["Player B"] = addr
		}
	}
	return addrs
}
