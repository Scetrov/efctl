package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestProjectGoBaseline(t *testing.T) {
	root := repositoryRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`(?m)^go 1\.27\.1$`).Match(data) {
		t.Fatal("project minimum Go version must be 1.27.1")
	}
	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), "[Go 1.27.1+]") {
		t.Fatal("source installation documentation must match the Go baseline")
	}
}

type goWorkflow struct {
	Jobs map[string]struct {
		Steps []struct {
			Uses string            `yaml:"uses"`
			With map[string]string `yaml:"with"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

func TestWorkflowsUseProjectGoBaseline(t *testing.T) {
	root := repositoryRoot(t)
	for _, name := range []string{"ci.yml", "release.yml", "codeql.yml"} {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(root, ".github", "workflows", name))
			if err != nil {
				t.Fatal(err)
			}
			var workflow goWorkflow
			if err := yaml.Unmarshal(data, &workflow); err != nil {
				t.Fatal(err)
			}
			setups := 0
			for job, configuration := range workflow.Jobs {
				for _, step := range configuration.Steps {
					if !strings.HasPrefix(step.Uses, "actions/setup-go@") {
						continue
					}
					setups++
					if step.With["go-version-file"] != "go.mod" || step.With["go-version"] != "" {
						t.Errorf("job %s must select its Go version from go.mod", job)
					}
				}
			}
			if setups == 0 {
				t.Fatal("expected a setup-go step")
			}
		})
	}
}
