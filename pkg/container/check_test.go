package container

import (
	"testing"
)

func TestHasIptablesConfig(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name: "Correct config",
			content: `
[network]
firewall_driver = "iptables"
`,
			expected: true,
		},
		{
			name: "Different driver",
			content: `
[network]
firewall_driver = "nftables"
`,
			expected: false,
		},
		{
			name: "Missing section",
			content: `
[containers]
netns = "bridge"
`,
			expected: false,
		},
		{
			name: "Key outside section",
			content: `
firewall_driver = "iptables"
[network]
`,
			expected: false,
		},
		{
			name: "Comments and whitespace",
			content: `
# Some comment
[network]

  firewall_driver  =  "iptables"  # another comment
`,
			expected: true,
		},
		{
			name:     "Empty content",
			content:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasIptablesConfig(tt.content); got != tt.expected {
				t.Errorf("hasIptablesConfig() = %v, want %v", got, tt.expected)
			}
		})
	}
}
