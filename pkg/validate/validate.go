// Package validate provides reusable input validation functions for CLI arguments
// and configuration values. All validators return an error describing the violation
// or nil if the input is acceptable.
package validate

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// suiAddressRe matches a Sui hex address: 0x followed by 1–64 hex characters.
var suiAddressRe = regexp.MustCompile(`^0x[a-fA-F0-9]{1,64}$`)

// safePathSegmentRe matches path segments that are safe for use in shell commands
// and container paths (alphanumeric, hyphens, underscores, dots).
var safePathSegmentRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// allowedNetworks is the set of supported network names.
var allowedNetworks = map[string]bool{
	"localnet": true,
	"testnet":  true,
}

// SuiAddress validates that s is a well-formed Sui hex address (0x-prefixed, 1–64 hex chars).
func SuiAddress(s string) error {
	if !suiAddressRe.MatchString(s) {
		return fmt.Errorf("invalid Sui address %q: must be 0x followed by 1-64 hex characters", s)
	}
	return nil
}

// Network validates that s is a supported network name.
func Network(s string) error {
	if !allowedNetworks[s] {
		allowed := make([]string, 0, len(allowedNetworks))
		for k := range allowedNetworks {
			allowed = append(allowed, k)
		}
		return fmt.Errorf("invalid network %q: must be one of %s", s, strings.Join(allowed, ", "))
	}
	return nil
}

// ContractPath validates that a relative contract path does not escape the
// expected parent directory via traversal (../).
func ContractPath(s string) error {
	cleaned := filepath.Clean(s)

	if filepath.IsAbs(cleaned) {
		return fmt.Errorf("contract path must be relative, got absolute: %s", s)
	}

	// After cleaning, reject any ".." component (traversal)
	for _, part := range strings.Split(cleaned, string(filepath.Separator)) {
		if part == ".." {
			return fmt.Errorf("contract path must not contain directory traversal (..): %s", s)
		}
	}

	// Each path segment should be safe
	for _, part := range strings.Split(cleaned, string(filepath.Separator)) {
		if part == "." {
			continue
		}
		if !safePathSegmentRe.MatchString(part) {
			return fmt.Errorf("contract path segment %q contains disallowed characters", part)
		}
	}

	return nil
}

// WorkspacePath validates a workspace directory path. It allows absolute and
// relative paths but rejects obviously dangerous patterns.
func WorkspacePath(s string) error {
	if s == "" {
		return fmt.Errorf("workspace path must not be empty")
	}

	cleaned := filepath.Clean(s)

	// Reject null bytes (path injection)
	if strings.ContainsRune(cleaned, 0) {
		return fmt.Errorf("workspace path contains null bytes")
	}

	// Reject paths that resolve to system-sensitive directories
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return fmt.Errorf("cannot resolve workspace path: %w", err)
	}

	systemDirs := []string{"/", "/etc", "/usr", "/bin", "/sbin", "/var", "/boot", "/dev", "/proc", "/sys"}
	for _, d := range systemDirs {
		if abs == d {
			return fmt.Errorf("workspace path must not be a system directory: %s", abs)
		}
	}

	return nil
}
