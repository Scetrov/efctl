package doctor

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"efctl/pkg/config"
	"efctl/pkg/container"
	"efctl/pkg/env"
)

// ── Public types ───────────────────────────────────────────────────

// Options carries all inputs required to produce a Report without creating
// dependencies on the cmd package.
type Options struct {
	Workspace string

	// Version fields from cmd.Version / cmd.CommitSHA / cmd.BuildDate.
	Version   string
	CommitSHA string
	BuildDate string

	// Prereqs must be populated by the caller via env.CheckPrerequisites().
	Prereqs *env.CheckResult

	// Config state resolved by the root PersistentPreRun.
	ConfigLoaded bool
	ConfigPath   string
	Config       *config.Config
}

// EfctlInfo holds the version/build identity of the running efctl binary.
type EfctlInfo struct {
	Version   string
	CommitSHA string
	BuildDate string
	GOOS      string
	GOARCH    string
}

// SystemInfo holds host operating-system information.
type SystemInfo struct {
	// OS is the human-readable distribution name, e.g. "Ubuntu 22.04.3 LTS".
	OS string
	// Platform is runtime.GOOS/runtime.GOARCH, e.g. "linux/amd64".
	Platform string
	// GoVersion is the Go runtime embedded in the binary, e.g. "go1.26.1".
	GoVersion string
	// IsWSL is true when running inside Windows Subsystem for Linux.
	IsWSL bool
}

// ContainerRuntimeInfo holds the detected container engine details.
type ContainerRuntimeInfo struct {
	Engine  string // "docker" or "podman"
	Version string // e.g. "4.9.0"
	Path    string // absolute path to the binary
	Found   bool

	// Podman specific configuration from containers.conf
	PodmanNetns          string
	PodmanRuntime        string
	PodmanFirewallDriver string
}

// NodeInfo holds the detected Node.js installation details.
type NodeInfo struct {
	Version string // e.g. "v24.11.0"
	Path    string // absolute path to the binary
	Found   bool
}

// GitInfo holds the detected git installation details.
type GitInfo struct {
	Version string // e.g. "2.43.0"
	Path    string // absolute path to the binary
	Found   bool
}

// EnvironmentInfo holds the current state of the managed containers.
type EnvironmentInfo struct {
	Running int
	Total   int
	// State is "up", "down", or "unknown".
	State string
	Error string
	Logs  []ContainerLogInfo
}

// ContainerLogInfo holds the last few log lines for a running container.
type ContainerLogInfo struct {
	Name string
	Tail string
}

// PortInfo holds the availability of a single TCP port.
type PortInfo struct {
	Port      int
	Available bool
}

// RepoInfo holds the git ref state of a cloned sub-repository.
type RepoInfo struct {
	Name    string // e.g. "builder-scaffold"
	Path    string // absolute path on disk
	Remote  string // git remote URL
	Commit  string // HEAD commit SHA (full)
	Branch  string // current branch, or empty if detached
	IsDirty bool   // true if there are uncommitted changes
	Found   bool
	Error   string
}

// SuiClientInfo holds the detected Sui client configuration details.
type SuiClientInfo struct {
	Found              bool
	ActiveEnv          string
	ActiveAddress      string
	ActiveEnvRpcUrl    string
	ActiveEnvFaucetUrl string
	Error              string
}

// ConfigInfo holds information about the loaded efctl config file.
type ConfigInfo struct {
	Loaded   bool
	FilePath string
	Entries  []ConfigEntry
}

// ConfigEntry holds one explicit key-value pair loaded from the config file.
type ConfigEntry struct {
	Key   string
	Value string
}

// Report is the complete diagnostic report produced by Gather.
type Report struct {
	Efctl     EfctlInfo
	System    SystemInfo
	Container ContainerRuntimeInfo
	Node      NodeInfo
	Git       GitInfo
	Env       EnvironmentInfo
	Ports     []PortInfo
	Repos     []RepoInfo
	Sui       SuiClientInfo
	Config    ConfigInfo
}

// ── Entry point ────────────────────────────────────────────────────

// Gather collects all diagnostic information and returns a populated Report.
// It never panics; individual sub-gatherers capture errors into their fields.
func Gather(opts Options) *Report {
	prereqs := opts.Prereqs
	if prereqs == nil {
		prereqs = env.CheckPrerequisites()
	}

	r := &Report{}

	r.Efctl = EfctlInfo{
		Version:   opts.Version,
		CommitSHA: opts.CommitSHA,
		BuildDate: opts.BuildDate,
		GOOS:      runtime.GOOS,
		GOARCH:    runtime.GOARCH,
	}

	r.System = gatherSystem()
	r.Container = gatherContainerRuntime(prereqs)
	r.Node = gatherNode(prereqs)
	r.Git = gatherGit()
	r.Env = gatherEnvironment(opts.Workspace)
	r.Ports = gatherPorts()
	r.Repos = gatherRepos(opts.Workspace)
	r.Sui = gatherSuiClient()
	r.Config = gatherConfig(opts.Config, opts.ConfigLoaded, opts.ConfigPath)

	return r
}

// ── Sub-gatherers ──────────────────────────────────────────────────

func gatherSystem() SystemInfo {
	info := SystemInfo{
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
		GoVersion: runtime.Version(),
		IsWSL:     detectWSL(),
	}

	switch runtime.GOOS {
	case "linux":
		info.OS = linuxOSName()
	case "darwin":
		if out, err := exec.Command("sw_vers", "-productName").Output(); err == nil {
			name := strings.TrimSpace(string(out))
			if ver, err2 := exec.Command("sw_vers", "-productVersion").Output(); err2 == nil {
				info.OS = name + " " + strings.TrimSpace(string(ver))
			} else {
				info.OS = name
			}
		} else {
			info.OS = "macOS"
		}
	case "windows":
		info.OS = "Windows"
	default:
		info.OS = runtime.GOOS
	}

	return info
}

func detectWSL() bool {
	if os.Getenv("WSL_DISTRO_NAME") != "" || os.Getenv("WSL_INTEROP") != "" {
		return true
	}

	data, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return false
	}

	return detectWSLFrom(string(data))
}

func detectWSLFrom(values ...string) bool {
	for _, value := range values {
		lower := strings.ToLower(strings.TrimSpace(value))
		if strings.Contains(lower, "microsoft") || strings.Contains(lower, "wsl") {
			return true
		}
	}
	return false
}

// linuxOSName reads /etc/os-release for the PRETTY_NAME field, e.g.
// "Ubuntu 22.04.3 LTS". Falls back to "Linux" on any error.
func linuxOSName() string {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return "Linux"
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			val := strings.TrimPrefix(line, "PRETTY_NAME=")
			val = strings.Trim(val, `"'`)
			return val
		}
	}
	return "Linux"
}

func gatherContainerRuntime(prereqs *env.CheckResult) ContainerRuntimeInfo {
	engine, err := prereqs.Engine()
	if err != nil {
		return ContainerRuntimeInfo{Found: false}
	}

	info := ContainerRuntimeInfo{Engine: engine, Found: true}

	if path, err := exec.LookPath(engine); err == nil {
		info.Path = path
	}

	if out, err := exec.Command(engine, "--version").Output(); err == nil { // #nosec G204 -- engine is validated by prereqs.Engine() to be "docker" or "podman"
		info.Version = parseContainerVersion(engine, strings.TrimSpace(string(out)))
	}

	if engine == "podman" {
		gatherPodmanConfig(&info)
	}

	return info
}

func gatherPodmanConfig(info *ContainerRuntimeInfo) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	configPath := filepath.Join(home, ".config/containers/containers.conf")
	data, err := os.ReadFile(configPath) // #nosec G304
	if err != nil {
		// Try system-wide as fallback
		data, err = os.ReadFile("/etc/containers/containers.conf")
		if err != nil {
			return
		}
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	var currentSection string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.Trim(line, "[]")
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)

		switch currentSection {
		case "containers":
			if key == "netns" {
				info.PodmanNetns = val
			} else if key == "runtime" {
				info.PodmanRuntime = val
			}
		case "network":
			if key == "firewall_driver" {
				info.PodmanFirewallDriver = val
			}
		}
	}
}

// parseContainerVersion extracts a bare version number from the first line of
// `docker --version` / `podman --version` output.
//
// Examples:
//
//	"Docker version 28.5.2, build 123abc"  → "28.5.2"
//	"podman version 4.9.0"                 → "4.9.0"
func parseContainerVersion(engine, raw string) string {
	// Take only the first line in case of multi-line output.
	first := strings.SplitN(raw, "\n", 2)[0]
	// Some outputs wrap with extra info after a comma.
	first = strings.SplitN(first, ",", 2)[0]

	lower := strings.ToLower(first)
	needle := engine + " version "
	if idx := strings.Index(lower, needle); idx >= 0 {
		return strings.TrimSpace(first[idx+len(needle):])
	}
	// Fallback: last whitespace-separated token.
	parts := strings.Fields(first)
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return first
}

func gatherNode(prereqs *env.CheckResult) NodeInfo {
	if !prereqs.HasNode {
		return NodeInfo{Found: false}
	}
	info := NodeInfo{Version: prereqs.NodeVer, Found: true}
	if path, err := exec.LookPath("node"); err == nil {
		info.Path = path
	}
	return info
}

func gatherGit() GitInfo {
	path, err := exec.LookPath("git")
	if err != nil {
		return GitInfo{Found: false}
	}

	info := GitInfo{Path: path, Found: true}

	if out, err := exec.Command("git", "--version").Output(); err == nil {
		// "git version 2.43.0" → "2.43.0"
		raw := strings.TrimSpace(string(out))
		raw = strings.TrimPrefix(raw, "git version ")
		info.Version = raw
	}

	return info
}

func gatherEnvironment(workspace string) EnvironmentInfo {
	names := []string{
		container.ContainerSuiPlayground,
		container.ContainerPostgres,
		container.ContainerFrontend,
	}
	total := len(names)

	client, err := container.NewClient()
	if err != nil {
		return EnvironmentInfo{
			Total: total,
			State: "unknown",
			Error: fmt.Sprintf("no container engine: %v", err),
		}
	}

	running := 0
	for _, name := range names {
		if client.ContainerRunning(name) {
			running++
		}
	}

	state := "down"
	if running == total {
		state = "up"
	} else if running > 0 {
		state = "partial"
	}

	logs := make([]ContainerLogInfo, 0, running)
	if running > 0 {
		for _, name := range names {
			if !client.ContainerRunning(name) {
				continue
			}
			logs = append(logs, ContainerLogInfo{
				Name: name,
				Tail: client.ContainerLogs(name, 10),
			})
		}
	}

	return EnvironmentInfo{
		Running: running,
		Total:   total,
		State:   state,
		Logs:    logs,
	}
}

func gatherPorts() []PortInfo {
	ports := []int{9000, 9123, 9125, 5432, 5173}
	result := make([]PortInfo, 0, len(ports))
	for _, p := range ports {
		result = append(result, PortInfo{
			Port:      p,
			Available: env.IsPortAvailable(p),
		})
	}
	return result
}

func gatherRepos(workspace string) []RepoInfo {
	repoNames := []string{"builder-scaffold", "world-contracts"}
	result := make([]RepoInfo, 0, len(repoNames))

	for _, name := range repoNames {
		// First check in the workspace root
		fullPath := filepath.Join(workspace, name)
		info := gatherRepo(name, fullPath)

		// If not found in root, check in test-env/
		if !info.Found {
			testEnvPath := filepath.Join(workspace, "test-env", name)
			info = gatherRepo(name, testEnvPath)
		}

		result = append(result, info)
	}
	return result
}

func gatherRepo(name, path string) RepoInfo {
	info := RepoInfo{Name: name, Path: path}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		info.Found = false
		return info
	}

	// Check it is actually a git repository.
	if err := exec.Command("git", "-C", path, "rev-parse", "--git-dir").Run(); err != nil { // #nosec G204 -- path is a directory argument to git -C, not a shell command
		info.Found = false
		info.Error = "not a git repository"
		return info
	}
	info.Found = true

	// HEAD commit SHA.
	if out, err := exec.Command("git", "-C", path, "rev-parse", "HEAD").Output(); err == nil { // #nosec G204 -- path is a directory argument to git -C, not a shell command
		info.Commit = strings.TrimSpace(string(out))
	}

	// Current branch (empty / error means detached HEAD).
	if out, err := exec.Command("git", "-C", path, "symbolic-ref", "--short", "HEAD").Output(); err == nil { // #nosec G204 -- path is a directory argument to git -C, not a shell command
		info.Branch = strings.TrimSpace(string(out))
	}

	// Dirty status: any output means uncommitted changes exist.
	if out, err := exec.Command("git", "-C", path, "status", "--porcelain").Output(); err == nil { // #nosec G204
		info.IsDirty = len(strings.TrimSpace(string(out))) > 0
	}

	// Remote URL.
	if out, err := exec.Command("git", "-C", path, "remote", "get-url", "origin").Output(); err == nil { // #nosec G204
		info.Remote = strings.TrimSpace(string(out))
	}

	return info
}

func gatherConfig(cfg *config.Config, loaded bool, path string) ConfigInfo {
	return ConfigInfo{
		Loaded:   loaded,
		FilePath: path,
		Entries:  gatherConfigEntries(cfg),
	}
}

func gatherConfigEntries(cfg *config.Config) []ConfigEntry {
	if cfg == nil {
		return nil
	}

	entries := make([]ConfigEntry, 0, 8)
	if cfg.WithFrontend != nil {
		entries = append(entries, ConfigEntry{Key: "with-frontend", Value: fmt.Sprintf("%t", *cfg.WithFrontend)})
	}
	if cfg.WithGraphql != nil {
		entries = append(entries, ConfigEntry{Key: "with-graphql", Value: fmt.Sprintf("%t", *cfg.WithGraphql)})
	}
	if cfg.WorldContractsURL != "" {
		entries = append(entries, ConfigEntry{Key: "world-contracts-url", Value: cfg.WorldContractsURL})
	}
	if cfg.WorldContractsRef != "" {
		entries = append(entries, ConfigEntry{Key: "world-contracts-ref", Value: cfg.WorldContractsRef})
	}
	if cfg.WorldContractsBranch != "" {
		entries = append(entries, ConfigEntry{Key: "world-contracts-branch", Value: cfg.WorldContractsBranch})
	}
	if cfg.BuilderScaffoldURL != "" {
		entries = append(entries, ConfigEntry{Key: "builder-scaffold-url", Value: cfg.BuilderScaffoldURL})
	}
	if cfg.BuilderScaffoldRef != "" {
		entries = append(entries, ConfigEntry{Key: "builder-scaffold-ref", Value: cfg.BuilderScaffoldRef})
	}
	if cfg.BuilderScaffoldBranch != "" {
		entries = append(entries, ConfigEntry{Key: "builder-scaffold-branch", Value: cfg.BuilderScaffoldBranch})
	}

	return entries
}
func gatherSuiClient() SuiClientInfo {
	info := SuiClientInfo{}
	if _, err := exec.LookPath("sui"); err != nil {
		info.Found = false
		return info
	}
	info.Found = true

	// Gather active environment
	if out, err := exec.Command("sui", "client", "active-env").Output(); err == nil {
		info.ActiveEnv = strings.TrimSpace(string(out))
	}

	// Gather active address
	if out, err := exec.Command("sui", "client", "active-address").Output(); err == nil {
		info.ActiveAddress = strings.TrimSpace(string(out))
	}

	// Gather envs to find RPC and Faucet URLs
	// We use --json for robust parsing if available, but keep a fallback
	if out, err := exec.Command("sui", "client", "envs", "--json").Output(); err == nil {
		info.ActiveEnvRpcUrl, info.ActiveEnvFaucetUrl = parseSuiEnvsJSON(string(out), info.ActiveEnv)
	}

	// Fallback/Legacy parsing for older Sui versions or if --json failed
	if info.ActiveEnvRpcUrl == "" {
		if out, err := exec.Command("sui", "client", "envs").Output(); err == nil {
			info.ActiveEnvRpcUrl, info.ActiveEnvFaucetUrl = parseSuiEnvsLegacy(string(out), info.ActiveEnv)
		}
	}

	return info
}

func parseSuiEnvsJSON(content, activeEnv string) (rpc, faucet string) {
	// Output is like: [[{"alias":"localnet","rpc":"http://0.0.0.0:9000","ws":null,"basic_auth":null},...],"localnet"]
	// Very simple manual "JSON" parsing to avoid adding heavy dependencies if not needed,
	// but since we already use regex elsewhere, let's just look for the active env entry.

	// Find the block for the active environment
	envPattern := fmt.Sprintf(`"alias":"%s"`, activeEnv)
	idx := strings.Index(content, envPattern)
	if idx == -1 {
		return "", ""
	}

	// Look for RPC
	rpcPattern := regexp.MustCompile(`"rpc":"([^"]+)"`)
	if matches := rpcPattern.FindStringSubmatch(content[idx:]); len(matches) > 1 {
		rpc = matches[1]
	}
	// Look for Faucet
	faucetPattern := regexp.MustCompile(`"faucet":"([^"]+)"`)
	if matches := faucetPattern.FindStringSubmatch(content[idx:]); len(matches) > 1 {
		faucet = matches[1]
	}
	return rpc, faucet
}

func parseSuiEnvsLegacy(content, activeEnv string) (rpc, faucet string) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if !strings.Contains(line, activeEnv) {
			continue
		}
		fields := strings.Fields(line)
		cleanFields := cleanTableFields(fields)

		if len(cleanFields) < 2 {
			continue
		}

		// In table format: [alias, url, active] or [* alias url]
		if isTargetEnv(cleanFields, activeEnv) {
			r, f := extractUrlsFromFields(cleanFields)
			if r != "" && rpc == "" {
				rpc = r
			}
			if f != "" && faucet == "" {
				faucet = f
			}
		}
	}
	return rpc, faucet
}

func cleanTableFields(fields []string) []string {
	var cleanFields []string
	for _, f := range fields {
		if f != "│" && f != "├" && f != "┤" && f != "║" {
			cleanFields = append(cleanFields, f)
		}
	}
	return cleanFields
}

func isTargetEnv(fields []string, activeEnv string) bool {
	if fields[0] == activeEnv {
		return true
	}
	if len(fields) > 1 && fields[0] == "*" && fields[1] == activeEnv {
		return true
	}
	return false
}

func extractUrlsFromFields(fields []string) (rpc, faucet string) {
	for _, f := range fields {
		if strings.HasPrefix(f, "http") {
			if rpc == "" {
				rpc = f
			} else if faucet == "" {
				faucet = f
			}
		}
	}
	return rpc, faucet
}
