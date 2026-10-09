package scripts

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// OSV-Scanner v2 fails before writing SARIF for --fail or a go.sum lockfile.
func TestOSVWorkflowScannerArguments(t *testing.T) {
	data, err := os.ReadFile("../.github/workflows/osv-scan.yml")
	require.NoError(t, err)
	var workflow struct {
		Jobs map[string]struct {
			Steps []struct {
				Uses string `yaml:"uses"`
				With struct {
					ScanArgs string `yaml:"scan-args"`
				} `yaml:"with"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	require.NoError(t, yaml.Unmarshal(data, &workflow))
	for _, step := range workflow.Jobs["scan"].Steps {
		if strings.HasPrefix(step.Uses, "google/osv-scanner-action/osv-scanner-action@") {
			require.Equal(t, []string{"--lockfile", "go.mod", "--format", "sarif", "--output-file", "osv-results.sarif"}, strings.Fields(step.With.ScanArgs))
			return
		}
	}
	t.Fatal("OSV scanner step not found")
}
