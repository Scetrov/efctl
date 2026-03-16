package ui

import "strings"

// ShortenAddress abbreviates a Sui address or object ID for display.
// Format: 0x1234...5678 (6 prefix chars including 0x, 4 suffix chars).
func ShortenAddress(s string) string {
	if len(s) > 16 && strings.HasPrefix(s, "0x") {
		return s[:6] + "..." + s[len(s)-4:]
	}
	return s
}
