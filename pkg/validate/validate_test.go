package validate

import (
	"testing"
)

func TestSuiAddress_Valid(t *testing.T) {
	valid := []string{
		"0x1",
		"0xabcdef1234567890",
		"0xABCDEF1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		"0x0000000000000000000000000000000000000000000000000000000000000002",
	}
	for _, addr := range valid {
		if err := SuiAddress(addr); err != nil {
			t.Errorf("expected %q to be valid, got: %v", addr, err)
		}
	}
}

func TestSuiAddress_Invalid(t *testing.T) {
	invalid := []string{
		"",
		"abc",
		"0x",                            // no hex digits after 0x
		"0xGHIJ",                        // non-hex characters
		"1234",                          // missing 0x prefix
		"0x; rm -rf /",                  // injection attempt
		"0x123\n0x456",                  // newline
		"0x" + string(make([]byte, 65)), // too long
	}
	for _, addr := range invalid {
		if err := SuiAddress(addr); err == nil {
			t.Errorf("expected %q to be invalid, got nil", addr)
		}
	}
}

func TestNetwork_Valid(t *testing.T) {
	for _, n := range []string{"localnet", "testnet"} {
		if err := Network(n); err != nil {
			t.Errorf("expected %q to be valid, got: %v", n, err)
		}
	}
}

func TestNetwork_Invalid(t *testing.T) {
	for _, n := range []string{"", "mainnet", "devnet", "localnet; rm -rf /", "LOCALNET"} {
		if err := Network(n); err == nil {
			t.Errorf("expected %q to be invalid, got nil", n)
		}
	}
}

func TestContractPath_Valid(t *testing.T) {
	valid := []string{
		"smart_gate",
		"smart_gate/sources",
		"my-contract",
		"my_contract.v2",
		".",
	}
	for _, p := range valid {
		if err := ContractPath(p); err != nil {
			t.Errorf("expected %q to be valid, got: %v", p, err)
		}
	}
}

func TestContractPath_Invalid(t *testing.T) {
	invalid := []string{
		"/absolute/path",
		"../escape",
		"foo/../../etc/passwd",
		"foo bar",      // space
		"foo;rm -rf /", // semicolon
		"foo`whoami`",  // backtick
	}
	for _, p := range invalid {
		if err := ContractPath(p); err == nil {
			t.Errorf("expected %q to be invalid, got nil", p)
		}
	}
}

func TestWorkspacePath_Valid(t *testing.T) {
	valid := []string{
		".",
		"./my-workspace",
		"/home/user/dev",
		"/tmp/efctl-test",
	}
	for _, p := range valid {
		if err := WorkspacePath(p); err != nil {
			t.Errorf("expected %q to be valid, got: %v", p, err)
		}
	}
}

func TestWorkspacePath_Invalid(t *testing.T) {
	invalid := []string{
		"",
		"/",
		"/etc",
		"/usr",
		"/bin",
		"/proc",
	}
	for _, p := range invalid {
		if err := WorkspacePath(p); err == nil {
			t.Errorf("expected %q to be invalid, got nil", p)
		}
	}
}
