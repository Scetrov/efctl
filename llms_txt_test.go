package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// llmsTxtPath returns the absolute path to LLMS.txt at the repository root,
// located relative to this test file's directory via runtime.Caller.
func llmsTxtPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller: cannot determine test source location")
	}
	return filepath.Join(filepath.Dir(file), "LLMS.txt")
}

// readLLMS reads and returns LLMS.txt content, failing the test if missing.
func readLLMS(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(llmsTxtPath(t))
	if err != nil {
		t.Fatalf("LLMS.txt: unable to read (expected file at %s): %v", llmsTxtPath(t), err)
	}
	return string(data)
}

// headingLevel returns the heading level (1-6) for a Markdown heading line,
// or 0 if the line is not a heading.
func headingLevel(line string) int {
	level := 0
	for _, ch := range line {
		if ch == '#' {
			level++
		} else {
			break
		}
	}
	if level == 0 || level > 6 {
		return 0
	}
	rest := strings.TrimSpace(line[level:])
	if len(rest) == 0 {
		return 0
	}
	return level
}

const maxLLMSScanTokenSize = 1 << 20

var (
	linkRe     = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	listItemRe = regexp.MustCompile(`^(?:[-+*]|\d+[.)])\s+`)
)

// structure holds parsed results from an LLMS.txt scan.
type llmsStructure struct {
	h1Count      int
	h2Count      int
	summaryFound bool
	linkEntries  int
	errors       []string
}

func newLLMSScanner(content string) *bufio.Scanner {
	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 64*1024), maxLLMSScanTokenSize)
	return scanner
}

func missingH2LinkError(line, links int) string {
	if line != 0 && links == 0 {
		return fmt.Sprintf("line %d: H2 section has no descriptive Markdown link entry", line)
	}
	return ""
}

func isDescriptiveLinkEntry(line string) bool {
	trimmed := strings.TrimSpace(line)
	prefix := listItemRe.FindString(trimmed)
	if prefix == "" || !linkRe.MatchString(trimmed) {
		return false
	}

	description := linkRe.ReplaceAllString(strings.TrimSpace(strings.TrimPrefix(trimmed, prefix)), "")
	description = strings.Trim(description, " \t—–-:;|")
	return description != ""
}

func scanH2LinkEntry(s *llmsStructure, inH2 bool, line string, lineno int) int {
	if !inH2 || !linkRe.MatchString(line) {
		return 0
	}
	if !isDescriptiveLinkEntry(line) {
		s.errors = append(s.errors,
			fmt.Sprintf("line %d: H2 link entry must be a Markdown list item with a non-empty description", lineno))
		return 0
	}
	s.linkEntries++
	return 1
}

// scanLLMSStructure performs a single pass over the file content, collecting
// heading counts and verifying that every H2-delimited section has descriptive links.
func scanLLMSStructure(content string) llmsStructure {
	var s llmsStructure
	scanner := newLLMSScanner(content)

	h1Found := false
	expectingSummary := false
	currentH2Line := 0
	currentH2Links := 0
	lineno := 0

	finishH2 := func() {
		if errMessage := missingH2LinkError(currentH2Line, currentH2Links); errMessage != "" {
			s.errors = append(s.errors, errMessage)
		}
	}

	for scanner.Scan() {
		lineno++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if expectingSummary && trimmed != "" {
			if strings.HasPrefix(trimmed, ">") {
				s.summaryFound = true
			} else {
				s.errors = append(s.errors,
					fmt.Sprintf("line %d: expected blockquote summary immediately after H1", lineno))
			}
			expectingSummary = false
		}

		if level := headingLevel(line); level > 0 {
			switch level {
			case 1:
				if h1Found {
					s.errors = append(s.errors,
						fmt.Sprintf("line %d: multiple H1 headings found; exactly one is required", lineno))
				}
				h1Found = true
				s.h1Count++
				expectingSummary = true
			case 2:
				if !h1Found {
					s.errors = append(s.errors,
						fmt.Sprintf("line %d: H2 heading found before H1", lineno))
				}
				finishH2()
				s.h2Count++
				currentH2Line = lineno
				currentH2Links = 0
			default:
				if h1Found {
					s.errors = append(s.errors,
						fmt.Sprintf("line %d: H%d heading not allowed; sections must contain only H2 with link lists", lineno, level))
				}
			}
			continue
		}

		currentH2Links += scanH2LinkEntry(&s, currentH2Line != 0, line, lineno)
	}
	if err := scanner.Err(); err != nil {
		s.errors = append(s.errors, fmt.Sprintf("scan LLMS.txt structure: %v", err))
	}
	finishH2()

	if expectingSummary {
		s.errors = append(s.errors, "H1 is not followed by a blockquote summary")
	}
	return s
}

func TestScanLLMSStructure(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantError bool
	}{
		{
			name:    "descriptive list entry follows H1 after blank lines",
			content: "# efctl\n\n> summary\n\n## References\n\n- [Guide](README.md) — project overview\n",
		},
		{
			name:      "summary is not delayed past prose",
			content:   "# efctl\nPreamble\n> summary\n\n## References\n\n- [Guide](README.md) — project overview\n",
			wantError: true,
		},
		{
			name:      "H2 requires link entry",
			content:   "# efctl\n> summary\n\n## References\n\nNo links.\n",
			wantError: true,
		},
		{
			name:      "link entry requires description",
			content:   "# efctl\n> summary\n\n## References\n\n- [Guide](README.md)\n",
			wantError: true,
		},
		{
			name:      "link entry requires Markdown list item",
			content:   "# efctl\n> summary\n\n## References\n\n[Guide](README.md) — project overview\n",
			wantError: true,
		},
		{
			name:      "oversized line reports scanner error",
			content:   "# efctl\n> summary\n\n## References\n\n" + strings.Repeat("x", maxLLMSScanTokenSize+1),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := scanLLMSStructure(tt.content)
			if (len(s.errors) > 0) != tt.wantError {
				t.Fatalf("scanLLMSStructure() errors = %v, wantError %t", s.errors, tt.wantError)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 1.1  Structure validation
// ---------------------------------------------------------------------------

// TestLLMSTxtStructure validates the sole H1, immediate blockquote summary,
// H2 link-list sections, and descriptive link entries using only the standard library.
func TestLLMSTxtStructure(t *testing.T) {
	content := readLLMS(t)
	s := scanLLMSStructure(content)

	for _, e := range s.errors {
		t.Errorf("LLMS.txt: %s", e)
	}
	if s.h1Count != 1 {
		t.Errorf("LLMS.txt: expected exactly 1 H1 heading, found %d", s.h1Count)
	}
	if !s.summaryFound {
		t.Error("LLMS.txt: no blockquote summary found after H1; expected a '> ...' line immediately after the heading")
	}
	if s.h2Count == 0 {
		t.Error("LLMS.txt: no H2 sections found; expected at least one H2 with link entries")
	}
	if s.linkEntries == 0 {
		t.Error("LLMS.txt: no descriptive link entries found in H2 sections")
	}
}

// ---------------------------------------------------------------------------
// 1.2  Acceptance matrix: links, coverage, endpoints, interaction, safety
// ---------------------------------------------------------------------------

// validateLLMSLinks resolves every repository-local Markdown link and reports
// malformed targets or scan failures.
func validateLLMSLinks(content, repoRoot string) []string {
	var errors []string
	scanner := newLLMSScanner(content)
	lineno := 0

	for scanner.Scan() {
		lineno++
		for _, m := range linkRe.FindAllStringSubmatch(scanner.Text(), -1) {
			if isExternalLink(m[2]) {
				continue
			}
			if err := checkLink(repoRoot, m[2]); err != nil {
				errors = append(errors, fmt.Sprintf("line %d: invalid local link target %q: %v", lineno, m[2], err))
			}
		}
	}
	if err := scanner.Err(); err != nil {
		errors = append(errors, fmt.Sprintf("scan LLMS.txt links: %v", err))
	}
	return errors
}

// TestLLMSTxtLinks resolves every repository-local Markdown link in LLMS.txt
// and verifies that the destination exists.
func TestLLMSTxtLinks(t *testing.T) {
	for _, errMessage := range validateLLMSLinks(readLLMS(t), filepath.Dir(llmsTxtPath(t))) {
		t.Errorf("LLMS.txt: %s", errMessage)
	}
}

func TestLLMSLinkScanReportsOversizedLine(t *testing.T) {
	content := strings.Join([]string{
		"# efctl",
		"> summary",
		strings.Repeat("x", maxLLMSScanTokenSize+1),
		"- [Broken](missing-guide.md) — must not be silently skipped",
	}, "\n")

	errors := validateLLMSLinks(content, filepath.Dir(llmsTxtPath(t)))
	if len(errors) != 1 || !strings.Contains(errors[0], "scan LLMS.txt links") {
		t.Fatalf("validateLLMSLinks() errors = %v, want scanner error", errors)
	}
}

// isExternalLink returns true for http(s) URLs and mailto links.
func isExternalLink(target string) bool {
	return strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") || strings.HasPrefix(target, "mailto:")
}

func pathWithinRoot(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func checkConcreteLink(resolvedRoot, target string) error {
	if _, err := os.Stat(target); err != nil {
		return fmt.Errorf("stat target: %w", err)
	}
	resolvedTarget, err := filepath.EvalSymlinks(target)
	if err != nil {
		return fmt.Errorf("resolve target symlinks: %w", err)
	}
	if !pathWithinRoot(resolvedRoot, resolvedTarget) {
		return fmt.Errorf("resolved target escapes repository root")
	}
	return nil
}

// checkLink resolves a repository-local target, rejecting lexical paths and
// symlink destinations outside repoRoot.
func checkLink(repoRoot, target string) error {
	pathPart := target
	if idx := strings.IndexByte(target, '#'); idx >= 0 {
		pathPart = target[:idx]
	}
	if filepath.IsAbs(pathPart) {
		return fmt.Errorf("absolute path is not allowed")
	}

	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return fmt.Errorf("resolve repository root symlinks: %w", err)
	}
	absTarget := filepath.Clean(filepath.Join(absRoot, pathPart))
	if !pathWithinRoot(absRoot, absTarget) {
		return fmt.Errorf("target escapes repository root")
	}

	// Glob patterns must be valid, match at least one local path, and resolve cleanly.
	if strings.ContainsAny(pathPart, "*?[") {
		matched, err := filepath.Glob(absTarget)
		if err != nil {
			return fmt.Errorf("invalid glob: %w", err)
		}
		if len(matched) == 0 {
			return fmt.Errorf("glob matched no files")
		}
		for _, match := range matched {
			if !pathWithinRoot(absRoot, match) {
				return fmt.Errorf("glob match escapes repository root")
			}
			if err := checkConcreteLink(resolvedRoot, match); err != nil {
				return err
			}
		}
		return nil
	}

	return checkConcreteLink(resolvedRoot, absTarget)
}

func TestCheckLink(t *testing.T) {
	repoRoot := filepath.Dir(llmsTxtPath(t))
	tests := []struct {
		name    string
		target  string
		wantErr bool
	}{
		{name: "valid local target", target: "docs/efctl.md"},
		{name: "traversal", target: "../outside.md", wantErr: true},
		{name: "absolute target", target: filepath.Join(string(filepath.Separator), "etc", "passwd"), wantErr: true},
		{name: "missing target", target: "missing-guide.md", wantErr: true},
		{name: "stat error", target: "LLMS.txt/child", wantErr: true},
		{name: "invalid glob", target: "[", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkLink(repoRoot, tt.target)
			if (err != nil) != tt.wantErr {
				t.Fatalf("checkLink(%q) error = %v, wantErr %t", tt.target, err, tt.wantErr)
			}
		})
	}
}

func TestCheckLinkRejectsOutsideSymlink(t *testing.T) {
	repoRoot := filepath.Dir(llmsTxtPath(t))
	tmpRoot := filepath.Join(repoRoot, "tmp")
	_, err := os.Stat(tmpRoot)
	tmpRootDidNotExist := os.IsNotExist(err)
	if err := os.MkdirAll(tmpRoot, 0o755); err != nil {
		t.Fatalf("create test scratch root: %v", err)
	}
	if tmpRootDidNotExist {
		t.Cleanup(func() { _ = os.Remove(tmpRoot) })
	}

	testDir, err := os.MkdirTemp(tmpRoot, "llms-link-")
	if err != nil {
		t.Fatalf("create test scratch directory: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(testDir) })

	escapingLink := filepath.Join(testDir, "outside")
	if err := os.Symlink(filepath.Dir(repoRoot), escapingLink); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	relDir, err := filepath.Rel(repoRoot, testDir)
	if err != nil {
		t.Fatalf("make test target relative: %v", err)
	}

	for _, target := range []string{filepath.Join(relDir, "outside"), filepath.Join(relDir, "*")} {
		if err := checkLink(repoRoot, target); err == nil {
			t.Errorf("checkLink(%q) accepted symlink destination outside repository", target)
		}
	}
}

// acceptanceToken defines a named acceptance concern for the coverage matrix.
type acceptanceToken struct {
	token    string
	category string
}

var acceptedTokens = []acceptanceToken{
	// --- Representative commands from every capability family ---
	{token: "efctl init", category: "command: init"},
	{token: "efctl env up", category: "command: env up"},
	{token: "efctl env down", category: "command: env down"},
	{token: "efctl env status", category: "command: env status"},
	{token: "efctl env run", category: "command: env run"},
	{token: "efctl env shell", category: "command: env shell"},
	{token: "efctl env faucet", category: "command: env faucet"},
	{token: "env extension", category: "command: extension"},
	{token: "env extension publish", category: "command: extension publish"},
	{token: "env assembly", category: "command: assembly"},
	{token: "sui install", category: "command: sui install"},
	{token: "efctl graphql", category: "command: graphql"},
	{token: "world query", category: "command: world query"},
	{token: "efctl update", category: "command: update"},
	{token: "efctl version", category: "command: version"},
	{token: "efctl doctor", category: "command: doctor"},
	{token: "efctl completion", category: "command: completion"},

	// --- Configuration identifiers ---
	{token: "efctl.yaml", category: "config: filename"},
	{token: "efctl.yml", category: "config: filename alias"},
	{token: "--config-file", category: "config: explicit override"},
	{token: "--debug", category: "config: debug flag"},
	{token: "--no-progress", category: "config: non-interactive output"},
	{token: "--with-graphql", category: "config: feature flag override"},
	{token: "--with-frontend", category: "config: feature flag override"},
	{token: "with-frontend", category: "config: frontend key"},
	{token: "with-graphql", category: "config: graphql key"},
	{token: "world-contracts-url", category: "config: world contracts URL"},
	{token: "world-contracts-ref", category: "config: world contracts ref"},
	{token: "world-contracts-branch", category: "config: world contracts deprecated branch alias"},
	{token: "builder-scaffold-url", category: "config: builder scaffold URL"},
	{token: "builder-scaffold-ref", category: "config: builder scaffold ref"},
	{token: "builder-scaffold-branch", category: "config: builder scaffold deprecated branch alias"},
	{token: "git-autocrlf", category: "config: Git autocrlf key"},
	{token: "container-engine", category: "config: engine key"},
	{token: "host", category: "config: host key"},
	{token: "expose-postgres", category: "config: PostgreSQL exposure key"},
	{token: "additional-bind-mounts", category: "config: bind mounts key"},

	// --- Environment identifiers ---
	{token: "EFCTL_ENGINE", category: "env: engine selection"},
	{token: "DOCKER_HOST", category: "env: daemon"},
	{token: "EFCTL_STARTUP_TIMEOUT_SECONDS", category: "env: timeout"},
	{token: "EFCTL_PG_PASSWORD", category: "env: pg password"},
	{token: "CI=", category: "env: ci mode"},

	// --- Host endpoint mappings ---
	{token: "9000", category: "endpoint: sui rpc"},
	{token: "9123", category: "endpoint: faucet"},
	{token: "9125", category: "endpoint: graphql"},
	{token: "5173", category: "endpoint: frontend"},
	{token: "5432", category: "endpoint: postgresql"},
	{token: "8000", category: "endpoint: graphql preflight"},

	// --- Interaction-mode warnings ---
	{token: "CI=true", category: "interaction: ci mode"},
	{token: "interactive", category: "interaction: tty warning"},

	// --- Approval markers for mutating families ---
	{token: "approval", category: "approval: marker"},
	{token: "human", category: "approval: human required"},
	{token: "destructive", category: "safety: destructive marker"},

	// --- Recovery guidance ---
	{token: "recovery", category: "recovery: guidance"},
}

// TestLLMSTxtAcceptanceMatrix verifies that LLMS.txt covers every named
// acceptance token across commands, config, endpoints, interaction modes,
// approval markers, and recovery guidance.
func TestLLMSTxtAcceptanceMatrix(t *testing.T) {
	content := strings.ToLower(readLLMS(t))
	var missing []string

	for _, tok := range acceptedTokens {
		if !strings.Contains(content, strings.ToLower(tok.token)) {
			missing = append(missing, fmt.Sprintf("[%s] %q", tok.category, tok.token))
		}
	}

	if len(missing) > 0 {
		t.Errorf("LLMS.txt: acceptance matrix has %d missing token(s):\n  %s",
			len(missing), strings.Join(missing, "\n  "))
	}
}

// TestLLMSTxtNoCredentialLeaks rejects credential-like example material.
func TestLLMSTxtNoCredentialLeaks(t *testing.T) {
	content := readLLMS(t)

	// Patterns that match concrete credential data, not instructional mentions
	// of credential types. Each pattern is designed to detect actual values.
	credentialPatterns := []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{
			// BIP39 seed phrases: 6+ consecutive BIP39 words (strong signal of an actual phrase).
			name:    "mnemonic seed phrase (6+ BIP39 words)",
			pattern: regexp.MustCompile(`(?i)\b(?:abandon|ability|able|about|above|absent|absorb|abstract|absurd|abuse|access|accident|account|accuse|achieve|acid|acoustic|acquire|across|act|action|actor)\s+(?:abandon|ability|able|about|above|absent|absorb|abstract|absurd|abuse|access|accident|account|accuse|achieve|acid|acoustic|acquire|across|act|action|actor)\s+(?:abandon|ability|able|about|above|absent|absorb|abstract|absurd|abuse|access|accident|account|accuse|achieve|acid|acoustic|acquire|across|act|action|actor)\s+(?:abandon|ability|able|about|above|absent|absorb|abstract|absurd|abuse|access|accident|account|accuse|achieve|acid|acoustic|acquire|across|act|action|actor)\s+(?:abandon|ability|able|about|above|absent|absorb|abstract|absurd|abuse|access|accident|account|accuse|achieve|acid|acoustic|acquire|across|act|action|actor)\s+(?:abandon|ability|able|about|above|absent|absorb|abstract|absurd|abuse|access|accident|account|accuse|achieve|acid|acoustic|acquire|across|act|action|actor)\b`),
		},
		{
			// 0x-prefixed 64-character hex strings (actual key values).
			name:    "private key hex value (0x + 64 hex chars)",
			pattern: regexp.MustCompile(`(?i)0x[a-f0-9]{64}`),
		},
		{
			// password = <actual value> with 6+ non-whitespace characters.
			name:    "hardcoded password assignment",
			pattern: regexp.MustCompile(`(?i)(?:password|passwd)\s*[=:]\s*\S{6,}`),
		},
		{
			// Actual seed/recovery words listed (5+ consecutive BIP39 words starting with rare set).
			name:    "recovery word sequence (5+ words)",
			pattern: regexp.MustCompile(`(?i)\b(?:zoo|wrong|zero|zone|zebra|zealous)\s+(?:zoo|wrong|zero|zone|zebra|zealous)\s+(?:zoo|wrong|zero|zone|zebra|zealous)\s+(?:zoo|wrong|zero|zone|zebra|zealous)\s+(?:zoo|wrong|zero|zone|zebra|zealous)\b`),
		},
	}

	for _, cp := range credentialPatterns {
		if cp.pattern.MatchString(content) {
			t.Errorf("LLMS.txt: prohibited credential-like material found (%s)", cp.name)
		}
	}
}
