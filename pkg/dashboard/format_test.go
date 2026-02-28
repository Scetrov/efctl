package dashboard

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatAge(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"zero", 0, "0s"},
		{"seconds", 30 * time.Second, "30s"},
		{"just under a minute", 59 * time.Second, "59s"},
		{"one minute", 60 * time.Second, "1m"},
		{"minutes", 5 * time.Minute, "5m"},
		{"just under an hour", 59 * time.Minute, "59m"},
		{"one hour", time.Hour, "1h"},
		{"hours", 5 * time.Hour, "5h"},
		{"just under a day", 23 * time.Hour, "23h"},
		{"one day", 24 * time.Hour, "1d"},
		{"multiple days", 72 * time.Hour, "3d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatAge(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatWithCommas(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"zero", "0", "0"},
		{"small number", "42", "42"},
		{"hundreds", "999", "999"},
		{"thousands", "1000", "1,000"},
		{"millions", "1234567", "1,234,567"},
		{"billions", "1234567890", "1,234,567,890"},
		{"negative", "-1234567", "-1,234,567"},
		{"non-numeric", "abc", "abc"},
		{"empty", "", ""},
		{"single digit", "5", "5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatWithCommas(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShortKind(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"ProgrammableTransaction", "ProgrammableTransaction", "PrgTx"},
		{"ConsensusCommitPrologue", "ConsensusCommitPrologue", "Consensus"},
		{"ConsensusCommitPrologueV2", "ConsensusCommitPrologueV2", "Consensus"},
		{"ConsensusCommitPrologueV3", "ConsensusCommitPrologueV3", "Consensus"},
		{"AuthenticatorStateUpdate", "AuthenticatorStateUpdate", "AuthState"},
		{"AuthenticatorStateUpdateV2", "AuthenticatorStateUpdateV2", "AuthState"},
		{"RandomnessStateUpdate", "RandomnessStateUpdate", "Randomness"},
		{"EndOfEpochTransaction", "EndOfEpochTransaction", "EndEpoch"},
		{"ChangeEpoch", "ChangeEpoch", "Epoch"},
		{"Genesis", "Genesis", "Genesis"},
		{"short unknown", "Transfer", "Transfer"},
		{"long unknown", "VeryLongTransactionKindName", "VeryLongTr"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShortKind(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatGas(t *testing.T) {
	tests := []struct {
		name        string
		computation string
		storage     string
		rebate      string
		expected    string
	}{
		{"normal", "1000", "500", "200", "1,300"},
		{"zero total", "100", "100", "200", "-"},
		{"negative total", "100", "0", "200", "-"},
		{"large values", "1000000", "500000", "200000", "1,300,000"},
		{"all zeros", "0", "0", "0", "-"},
		{"non-numeric", "abc", "def", "ghi", "-"},
		{"empty strings", "", "", "", "-"},
		{"rebate equals sum", "500", "500", "1000", "-"},
		{"minimal positive", "2", "0", "1", "1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatGas(tt.computation, tt.storage, tt.rebate)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestColorizeLogLine(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"docker prefix", "[docker] container started"},
		{"db prefix", "[db] postgres ready"},
		{"deploy prefix", "[deploy] deploying contracts"},
		{"frontend prefix", "[frontend] vite dev server"},
		{"no prefix", "some regular log line"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ColorizeLogLine(tt.input)
			if tt.name == "no prefix" || tt.name == "empty" {
				// Passthrough: no prefix to colour
				assert.Equal(t, tt.input, result)
			} else {
				// Prefix has been styled; result should still contain content after prefix
				assert.NotEmpty(t, result)
			}
		})
	}
}

func TestHumanizeCamelCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple camelCase", "governorCap", "Governor Cap"},
		{"PascalCase", "GovernorCap", "Governor Cap"},
		{"single word", "admin", "Admin"},
		{"empty string", "", ""},
		{"all caps abbreviation", "adminAcl", "Admin Acl"},
		{"multiple words", "objectRegistryConfig", "Object Registry Config"},
		{"single char", "a", "A"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HumanizeCamelCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLogViewportRows(t *testing.T) {
	tests := []struct {
		name      string
		height    int
		numEvents int
		minExpect int
	}{
		{"small terminal", 20, 0, 3},
		{"medium terminal", 40, 5, 3},
		{"large terminal", 80, 10, 3},
		{"very small", 10, 0, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LogViewportRows(tt.height, tt.numEvents)
			assert.GreaterOrEqual(t, result, tt.minExpect, "viewport rows should be at least minimum")
		})
	}
}

func TestRenderStatus(t *testing.T) {
	t.Run("stopped", func(t *testing.T) {
		result := RenderStatus("Stopped")
		assert.NotEmpty(t, result)
		// Should contain "Stopped" text (possibly with ANSI codes)
		assert.Contains(t, result, "Stopped")
	})

	t.Run("running", func(t *testing.T) {
		result := RenderStatus("Running")
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "Running")
	})
}

func TestBuildAddresses(t *testing.T) {
	t.Run("with admin and env vars", func(t *testing.T) {
		envVars := map[string]string{
			"SPONSOR_ADDRESS":      "0xsponsor",
			"PLAYER_A_PRIVATE_KEY": "pk_a",
			"PLAYER_B_PRIVATE_KEY": "pk_b",
		}
		mockDerive := func(pk string) string {
			if pk == "pk_a" {
				return "0xplayerA"
			}
			if pk == "pk_b" {
				return "0xplayerB"
			}
			return ""
		}

		result := BuildAddresses("0xadmin", envVars, mockDerive)
		assert.Equal(t, "0xadmin", result["Admin"])
		assert.Equal(t, "0xsponsor", result["Sponsor"])
		assert.Equal(t, "0xplayerA", result["Player A"])
		assert.Equal(t, "0xplayerB", result["Player B"])
	})

	t.Run("admin unknown", func(t *testing.T) {
		result := BuildAddresses("Unknown", nil, func(s string) string { return "" })
		assert.Empty(t, result["Admin"])
	})

	t.Run("admin not found", func(t *testing.T) {
		result := BuildAddresses("Not Found", nil, func(s string) string { return "" })
		assert.Empty(t, result["Admin"])
	})

	t.Run("empty env vars", func(t *testing.T) {
		result := BuildAddresses("0xadmin", map[string]string{}, func(s string) string { return "" })
		assert.Equal(t, "0xadmin", result["Admin"])
		assert.Len(t, result, 1)
	})

	t.Run("derive returns empty", func(t *testing.T) {
		envVars := map[string]string{
			"PLAYER_A_PRIVATE_KEY": "pk_a",
		}
		result := BuildAddresses("0xadmin", envVars, func(s string) string { return "" })
		assert.NotContains(t, result, "Player A")
	})
}
