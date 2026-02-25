package env

import (
	"os"
	"testing"
)

func TestEngineSelectionWithEnvVar(t *testing.T) {
	tests := []struct {
		name        string
		envVar      string
		hasDocker   bool
		hasPodman   bool
		expected    string
		expectError bool
	}{
		{
			name:        "Prefers podman when env says podman and both installed",
			envVar:      "podman",
			hasDocker:   true,
			hasPodman:   true,
			expected:    "podman",
			expectError: false,
		},
		{
			name:        "Prefers docker when env says docker and both installed",
			envVar:      "docker",
			hasDocker:   true,
			hasPodman:   true,
			expected:    "docker",
			expectError: false,
		},
		{
			name:        "Falls back to default (podman) if env says podman but podman isn't installed",
			envVar:      "podman",
			hasDocker:   true,
			hasPodman:   false,
			expected:    "docker", // Env pref ignored, fallback to docker
			expectError: false,
		},
		{
			name:        "Default precedence prefers podman if both installed and no env var set",
			envVar:      "",
			hasDocker:   true,
			hasPodman:   true,
			expected:    "podman",
			expectError: false,
		},
		{
			name:        "Returns docker if only docker installed",
			envVar:      "",
			hasDocker:   true,
			hasPodman:   false,
			expected:    "docker",
			expectError: false,
		},
		{
			name:        "Returns error if neither installed",
			envVar:      "",
			hasDocker:   false,
			hasPodman:   false,
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("EFCTL_ENGINE", tt.envVar)
			defer os.Unsetenv("EFCTL_ENGINE")

			res := &CheckResult{
				HasDocker: tt.hasDocker,
				HasPodman: tt.hasPodman,
			}

			engine, err := res.Engine()

			if tt.expectError && err == nil {
				t.Errorf("Expected an error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Did not expect an error, got: %v", err)
			}
			if engine != tt.expected {
				t.Errorf("Expected engine %s, got %s", tt.expected, engine)
			}
		})
	}
}
