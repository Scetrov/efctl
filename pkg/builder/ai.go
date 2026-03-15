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
	agentName = strings.ToLower(agentName)

	// 1. Setup AGENTS.md (Constitutional & efctl info)
	if err := setupAgentsMd(workspace); err != nil {
		return fmt.Errorf("failed to setup AGENTS.md: %w", err)
	}

	// 2. Setup Agent-specific rules
	var targetFile string
	var instructions string

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

	return updateFileIdempotently(workspace, targetFile, instructions)
}

func setupAgentsMd(workspace string) error {
	const agentsFile = "AGENTS.md"
	content := getAgentsMdTemplate()
	return updateFileIdempotently(workspace, agentsFile, content)
}

func updateFileIdempotently(workspace, targetFile, instructions string) error {
	absPath := filepath.Join(workspace, targetFile)
	targetDir := filepath.Dir(absPath)
	if err := os.MkdirAll(targetDir, 0750); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", targetFile, err)
	}

	wrappedContent := fmt.Sprintf("\n%s\n%s\n%s\n", MarkerStart, strings.TrimSpace(instructions), MarkerEnd)

	existing, err := os.ReadFile(absPath) // #nosec G304
	if err != nil {
		if os.IsNotExist(err) {
			return os.WriteFile(absPath, []byte(strings.TrimSpace(wrappedContent)+"\n"), 0600) // #nosec G306 G703
		}
		return err
	}

	existingStr := string(existing)
	startIdx := strings.Index(existingStr, MarkerStart)
	endIdx := strings.Index(existingStr, MarkerEnd)

	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		// Replace existing block
		newContent := existingStr[:startIdx] + strings.TrimSpace(wrappedContent) + existingStr[endIdx+len(MarkerEnd):]
		return os.WriteFile(absPath, []byte(newContent), 0600) // #nosec G306 G703
	}

	// Double check if it already exists without markers (for AGENTS.md or legacy files)
	if strings.Contains(existingStr, "efctl Context") || strings.Contains(existingStr, "Development Cheat Sheet") {
		// If it's already there but no markers, we might want to be careful.
		// For now, let's just append if markers are missing but content seems related.
	}

	// Append to file
	newContent := strings.TrimRight(existingStr, "\n") + "\n" + strings.TrimSpace(wrappedContent) + "\n"
	return os.WriteFile(absPath, []byte(newContent), 0600) // #nosec G306 G703
}

func getAgentsMdTemplate() string {
	return `
# Agent Constitution

This document defines the core principles and operational constraints for any AI Agent working on this repository.

## 1. Test-First Development
All features and bug fixes must follow a test-driven approach.

## 2. Testing Pyramid Adherence
- **Unit Tests:** High volume, isolation.
- **Integration Tests:** Interaction between modules.
- **E2E Tests:** Full user flows.

## 3. Clean Code & Quality Gates
- Write clean, maintainable, and idiomatic Go code.
- Always run pre-commit hooks.

## 4. Security-First
Prioritize security in every change. Avoid hardcoding credentials.

## 5. Independent Operation
The Agent is authorized to operate independently.

## 6. Environmental Isolation
- Use ./tmp for temporary files.
- Do not write to system-level directories.

## 7. Context-Mode Routing
- Prefer context-mode MCP tools for large analysis.

## 8. Development Cheat Sheet

Quick reference for common efctl operations:

- **Initialize configuration**: efctl init (or efctl init --ai [agent])
- **Environment Lifecycle**:
  - Up: efctl env up
  - Down: efctl env down
- **Status Check**: efctl env status
- **Deploy Extension**: efctl env extension publish [contract-path] (path defaults to ./my-extension)
- **Query World**: efctl world query [object_id] (queries the Sui GraphQL RPC)
`
}

func getSharedInstructions() string {
	return `
# efctl Context
You are working on a project using 'efctl', the EVE Frontier CLI.
Reference AGENTS.md for core principles and a detailed command cheat sheet.

## Key Instructions
- Always verify environment status with 'efctl env status' before operations.
- Use 'efctl env up' to start the local development environment.
- Use 'efctl env extension publish [contract-path]' to deploy your modifications.
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
