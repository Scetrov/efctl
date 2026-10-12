package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestMakeE2ETimeout(t *testing.T) {
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make unavailable")
	}
	for _, tc := range []struct{ name, argument, environment, want string }{
		{"default", "", "", "60m"},
		{"command override", "E2E_TIMEOUT=90m", "", "90m"},
		{"environment override", "", "45m", "45m"},
		{"CI override", "E2E_TIMEOUT=30m", "", "30m"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"-n", "test-e2e"}
			if tc.argument != "" {
				args = append(args, tc.argument)
			}
			cmd := exec.Command("make", args...)
			cmd.Dir = repositoryRoot(t)
			for _, v := range os.Environ() {
				if !strings.HasPrefix(v, "E2E_TIMEOUT=") {
					cmd.Env = append(cmd.Env, v)
				}
			}
			if tc.environment != "" {
				cmd.Env = append(cmd.Env, "E2E_TIMEOUT="+tc.environment)
			}
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("make dry-run: %v: %s", err, output)
			}
			if !strings.Contains(string(output), `-timeout "`+tc.want+`" `) {
				t.Fatalf("expected %s deadline in %s", tc.want, output)
			}
		})
	}
}

func TestCIE2ETimeout(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repositoryRoot(t), ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		Jobs map[string]struct {
			Timeout int `yaml:"timeout-minutes"`
			Steps   []struct {
				Run string `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatal(err)
	}
	job := workflow.Jobs["e2e-tests"]
	if job.Timeout != 35 {
		t.Fatalf("CI job timeout changed: %d", job.Timeout)
	}
	for _, step := range job.Steps {
		if strings.TrimSpace(step.Run) == "make test-e2e E2E_TIMEOUT=30m" {
			return
		}
	}
	t.Fatal("CI must use shared Make target with explicit 30m timeout")
}

func TestCustomSuiGuideTestsBuiltArtifact(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repositoryRoot(t), "docs", "create-your-own-sui-binary.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "./target/release/sui start --force-regenesis") || strings.Contains(string(data), "\nsui start --force-regenesis") {
		t.Fatal("guide must test the newly built binary explicitly")
	}
}
