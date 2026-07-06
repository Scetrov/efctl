package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func FuzzConfigValidate(f *testing.F) {
	seeds := []string{
		"{}",
		"world-contracts-url: https://github.com/evefrontier/world-contracts.git\n",
		"world-contracts-url: git://example.com/repo.git\n",
		"world-contracts-ref: main\nbuilder-scaffold-ref: v1.0.0\n",
		"additional-bind-mounts:\n  - hostPath: ./data\n    identifier: data\n",
		"host: localhost\nexpose-postgres: true\n",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		var cfg Config
		if err := yaml.Unmarshal([]byte(input), &cfg); err != nil {
			return
		}
		_ = cfg.Validate()
	})
}
