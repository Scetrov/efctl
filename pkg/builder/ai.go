package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	MarkerStart = "<!-- EFCTL_INSTRUCTIONS_START -->"
	MarkerEnd   = "<!-- EFCTL_INSTRUCTIONS_END -->"
)

// SetupAIInstructions configures instructions for various AI agents idempotently.
func SetupAIInstructions(agentName string, workspace string) error {
	var targetFile string
	var instructions string

	agentName = strings.ToLower(agentName)

	switch agentName {
	case "copilot":
		targetFile = filepath.Join(".github", "copilot-instructions.md")
		instructions = getCopilotInstructions()
	case "cursor":
		targetFile = filepath.Join(".cursor", "rules", "efctl.md")
		instructions = getCursorInstructions()
	case "claude":
		targetFile = "CLAUDE.md"
		instructions = getClaudeInstructions()
	case "gemini":
		targetFile = filepath.Join(".agents", "instructions.md")
		instructions = getGeminiInstructions()
	default:
		return fmt.Errorf("unsupported agent: %s; supported agents are: copilot, cursor, claude, gemini", agentName)
	}

	absPath := filepath.Join(workspace, targetFile)
	targetDir := filepath.Dir(absPath)
	if err := os.MkdirAll(targetDir, 0750); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", targetFile, err)
	}

	content := fmt.Sprintf("\n%s\n%s\n%s\n", MarkerStart, strings.TrimSpace(instructions), MarkerEnd)

	existing, err := os.ReadFile(absPath) // #nosec G304
	if err != nil {
		if os.IsNotExist(err) {
			return os.WriteFile(absPath, []byte(strings.TrimSpace(content)+"\n"), 0600) // #nosec G306 G703
		}
		return err
	}

	existingStr := string(existing)
	startIdx := strings.Index(existingStr, MarkerStart)
	endIdx := strings.Index(existingStr, MarkerEnd)

	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		// Replace existing block
		newContent := existingStr[:startIdx] + strings.TrimSpace(content) + existingStr[endIdx+len(MarkerEnd):]
		return os.WriteFile(absPath, []byte(newContent), 0600) // #nosec G306 G703
	}

	// Append to file
	newContent := strings.TrimRight(existingStr, "\n") + "\n" + strings.TrimSpace(content) + "\n"
	return os.WriteFile(absPath, []byte(newContent), 0600) // #nosec G306 G703
}

func getSharedInstructions() string {
	return `
# efctl Context
You are working on a project using 'efctl', the EVE Frontier CLI.
Reference AGENTS.md for core principles.

## efctl Commands
- Init: efctl init
- Up: efctl env up
- Down: efctl env down
- Status: efctl env status
- Publish Extension: efctl env extension publish [contract-path]
- Query World: efctl world query [object_id]
`
}

func getCopilotInstructions() string {
	return getSharedInstructions()
}

func getCursorInstructions() string {
	return `---
description: efctl usage instructions
globs: **/*
---
` + getSharedInstructions()
}

func getClaudeInstructions() string {
	return getSharedInstructions()
}

func getGeminiInstructions() string {
	return getSharedInstructions()
}
