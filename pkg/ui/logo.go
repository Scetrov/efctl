package ui

import (
	"strings"

	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
)

func PrintBanner() {
	cyan := pterm.NewRGB(0, 255, 255)
	orange := pterm.NewRGB(255, 116, 0)

	raw, _ := pterm.DefaultBigText.WithLetters(putils.LettersFromString("$> EFCTL")).Srender()

	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		clean := pterm.RemoveColorFromString(line)
		var gradStr string
		runes := []rune(clean)
		for i, r := range runes {
			// Prevent divide by zero or out of bounds fading
			maxLen := len(runes)
			if maxLen == 0 {
				maxLen = 1
			}
			c := cyan.Fade(0, float32(maxLen), float32(i), orange)
			gradStr += c.Sprint(string(r))
		}
		pterm.Println(gradStr)
	}
	pterm.Println()
}
