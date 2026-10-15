package setup

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

const customSuiBlock = `# Custom Sui CLI supplied by the user
COPY sui /usr/local/bin/sui
RUN chmod +x /usr/local/bin/sui \
    && /usr/local/bin/sui --version`

// globalSuiBlock pins the last Sui release that still serves fullnode JSON-RPC
// (removed in its successor, testnet-v1.82.1). The digests are a non-enforced
// integrity reference for the official release assets.
const globalSuiBlock = `# Sui CLI via suiup pinned to testnet-1.81.1 (last release with fullnode JSON-RPC)
# Reference SHA-256 (not verified at build time):
#   x86_64  d921910fdedf5ae2bc3f08a7236c57819a457ce53a8cc60c85f3653c7831b5d4
#   aarch64 4c2d948417af9f55cdd1f2f5d8e558533502672004c34f6429aa6cb114a51583
ENV PATH="/root/.local/bin:/root/.suiup/bin:${PATH}"
RUN curl -sSfL https://raw.githubusercontent.com/MystenLabs/suiup/main/install.sh | sh \
    && echo y | suiup install sui@testnet-1.81.1 \
    && sui --version`

// legacyGlobalSuiBlock is the pre-pin floating default that existing
// scaffolds may still contain. It is recognized only so preparation can
// replace it with the pinned globalSuiBlock.
const legacyGlobalSuiBlock = `# Sui CLI via suiup (native binary for build arch – works on ARM and x86)
ENV PATH="/root/.local/bin:/root/.suiup/bin:${PATH}"
RUN curl -sSfL https://raw.githubusercontent.com/MystenLabs/suiup/main/install.sh | sh \
    && echo y | suiup install sui \
    && sui --version`

const relocationBlock = `RUN SUI_PATH=$(command -v sui) && SUIUP_PATH=$(command -v suiup) && \
    mv "$SUI_PATH" /usr/local/bin/sui && \
    mv "$SUIUP_PATH" /usr/local/bin/suiup && \
    chmod +x /usr/local/bin/sui /usr/local/bin/suiup`

var extraDockerBlankLines = regexp.MustCompile(`\n[ \t]*\n(?:[ \t]*\n)+`)

// customSuiAvailable inspects a bounded prefix, never executing the supplied file.
// A missing file selects suiup; invalid supplied artifacts are errors, not fallback.
func customSuiAvailable(dockerDir string) (bool, error) {
	path, err := safePath(dockerDir, "sui")
	if err != nil {
		return false, err
	}
	info, err := os.Lstat(path) // #nosec G703 -- path validated by safePath
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect docker/sui: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return false, fmt.Errorf("docker/sui must be a non-empty regular file (not a symlink)")
	}
	f, err := os.Open(path) // #nosec G304 G703 -- path validated by safePath; no supplied executable is run
	if err != nil {
		return false, fmt.Errorf("read docker/sui: %w", err)
	}
	defer f.Close()
	prefix, err := io.ReadAll(io.LimitReader(f, 256))
	if err != nil {
		return false, fmt.Errorf("inspect docker/sui prefix: %w", err)
	}
	if bytes.HasPrefix(prefix, []byte("version https://git-lfs.github.com/spec/v1")) {
		return false, fmt.Errorf("docker/sui is an unhydrated Git LFS pointer; run git lfs pull in builder-scaffold before retrying")
	}
	return true, nil
}

// dockerInstructions groups continued lines without parsing or executing shell code.
// Only supported, complete Sui instructions are replaced; other content is retained.
func dockerInstructions(content string) []string {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	var instructions []string
	for i := 0; i < len(lines); i++ {
		instruction := lines[i]
		for strings.HasSuffix(strings.TrimSpace(lines[i]), "\\") && i+1 < len(lines) {
			i++
			instruction += "\n" + lines[i]
		}
		instructions = append(instructions, instruction)
	}
	return instructions
}

func dockerInstructionKey(instruction string) string {
	fields := strings.Fields(strings.ReplaceAll(instruction, "\\\n", " "))
	if len(fields) == 0 {
		return ""
	}
	fields[0] = strings.ToUpper(fields[0])
	return strings.Join(fields, " ")
}

func knownSuiInstructions() map[string]bool {
	known := make(map[string]bool)
	for _, block := range []string{globalSuiBlock, legacyGlobalSuiBlock, relocationBlock,
		customSuiBlock, "RUN chmod +x /usr/local/bin/sui \\\n    && sui --version"} {
		for _, instruction := range dockerInstructions(block) {
			known[dockerInstructionKey(instruction)] = true
		}
	}
	// Clean up the comment produced by the previous efctl implementation too.
	known[dockerInstructionKey("# Custom Sui CLI compiled without AVX2")] = true
	return known
}

func unsupportedSuiInstruction(key string) bool {
	if strings.HasPrefix(key, "#") {
		return false
	}
	return strings.Contains(key, "suiup install") ||
		strings.Contains(key, "/usr/local/bin/sui") ||
		strings.Contains(key, "/root/.suiup/bin")
}

func validateSuiLayout(instructions []string, known map[string]bool, custom bool) error {
	workdirs, stages := 0, 0
	for _, instruction := range instructions {
		key := dockerInstructionKey(instruction)
		if key == "WORKDIR /workspace" {
			workdirs++
		}
		if strings.HasPrefix(key, "FROM ") {
			stages++
		}
		if !known[key] && unsupportedSuiInstruction(key) {
			return fmt.Errorf("unsupported Sui installation instruction in Dockerfile; no changes applied")
		}
	}
	if workdirs != 1 || stages != 1 {
		op := "global-sui-insertion"
		if custom {
			op = "custom-sui-insertion"
		}
		warnPatchUnmatched(op, "Dockerfile")
		return fmt.Errorf("Sui preparation requires one WORKDIR /workspace in a single-stage Dockerfile; no changes applied")
	}
	return nil
}

// patchCustomSuiIfPresent validates the complete supported transformation before
// removing setup. Returned content remains unchanged on any preparation error.
func patchCustomSuiIfPresent(content, dockerDir string) (string, error) {
	custom, err := customSuiAvailable(dockerDir)
	if err != nil {
		return content, err
	}
	instructions := dockerInstructions(content)
	known := knownSuiInstructions()
	if err := validateSuiLayout(instructions, known, custom); err != nil {
		return content, err
	}
	selected := globalSuiBlock + "\n\n" + relocationBlock
	if custom {
		selected = customSuiBlock
	}
	var result []string
	for _, instruction := range instructions {
		key := dockerInstructionKey(instruction)
		if known[key] {
			continue
		}
		if key == "WORKDIR /workspace" {
			for len(result) > 0 && strings.TrimSpace(result[len(result)-1]) == "" {
				result = result[:len(result)-1]
			}
			result = append(result, "", selected, "")
		}
		result = append(result, instruction)
	}
	return extraDockerBlankLines.ReplaceAllString(strings.Join(result, "\n"), "\n\n"), nil
}
