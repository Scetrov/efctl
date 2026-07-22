package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

var mandatoryPassingControls = []string{
	"description_good", "interact", "contribution", "floss_license", "license_location",
	"documentation_basics", "documentation_interface", "sites_https", "discussion", "maintained",
	"repo_public", "repo_track", "repo_interim", "version_unique", "release_notes",
	"release_notes_vulns", "report_process", "report_responses", "report_archive",
	"vulnerability_report_process", "vulnerability_report_private", "vulnerability_report_response",
	"build", "test", "test_policy", "tests_are_added", "warnings", "warnings_fixed",
	"know_secure_design", "know_common_errors", "crypto_published", "crypto_floss", "crypto_keylength",
	"crypto_working", "crypto_password_storage", "crypto_random", "delivery_mitm", "delivery_unsigned",
	"vulnerabilities_fixed_60_days", "no_leaked_credentials", "static_analysis", "static_analysis_fixed",
	"dynamic_analysis_fixed",
}

var urlRequiredControls = map[string]bool{
	"contribution":                 true,
	"license_location":             true,
	"release_notes":                true,
	"report_process":               true,
	"report_archive":               true,
	"vulnerability_report_process": true,
	"vulnerability_report_private": true,
}

var auditedReleaseTags = []string{
	"v0.0.1", "v0.0.2", "v0.0.3", "v0.0.4", "v0.0.5", "v0.0.6", "v0.0.7", "v0.0.8", "v0.0.9",
	"v0.1.0", "v0.1.1", "v0.1.2", "v0.2.0", "v0.2.1", "v0.2.2", "v0.2.3", "v0.2.4", "v0.3.0",
	"v0.3.1", "v0.3.2", "v0.3.3", "v0.3.4", "v0.3.5", "v0.3.6",
}

var httpsURL = regexp.MustCompile(`https://[^\s)]+`)
var applicabilityExplanation = regexp.MustCompile(`(?i)\b(no|not applicable|does not|not used|without)\b`)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller: cannot determine repository root")
	}
	return filepath.Dir(file)
}

func readBestPracticesManifest(t *testing.T) map[string]string {
	t.Helper()
	path := filepath.Join(repositoryRoot(t), ".bestpractices.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read .bestpractices.json: %v", err)
	}
	var manifest map[string]string
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse .bestpractices.json: %v", err)
	}
	return manifest
}

func TestBestPracticesMandatoryControlAttestation(t *testing.T) {
	manifest := readBestPracticesManifest(t)
	want := make(map[string]bool, len(mandatoryPassingControls))
	for _, control := range mandatoryPassingControls {
		want[control] = true
		status, ok := manifest[control+"_status"]
		if !ok {
			t.Errorf("missing mandatory status for %q", control)
			continue
		}
		if status != "Met" && status != "N/A" {
			t.Errorf("%q has status %q; want Met or N/A", control, status)
		}

		justification := strings.TrimSpace(manifest[control+"_justification"])
		if justification == "" {
			t.Errorf("%q is missing a paired justification", control)
			continue
		}
		if status == "N/A" && !applicabilityExplanation.MatchString(justification) {
			t.Errorf("%q is N/A without an applicability explanation", control)
		}
		if urlRequiredControls[control] && status == "Met" && !httpsURL.MatchString(justification) {
			t.Errorf("%q is Met but lacks required HTTPS evidence URL", control)
		}
	}

	var unexpected []string
	for key := range manifest {
		if !strings.HasSuffix(key, "_status") {
			continue
		}
		control := strings.TrimSuffix(key, "_status")
		if !want[control] {
			unexpected = append(unexpected, control)
		}
	}
	sort.Strings(unexpected)
	if len(unexpected) > 0 {
		t.Errorf("unexpected non-mandatory status keys: %s", strings.Join(unexpected, ", "))
	}
}

func TestBestPracticesReleaseHistory(t *testing.T) {
	path := filepath.Join(repositoryRoot(t), "CHANGELOG.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read CHANGELOG.md: %v", err)
	}
	content := string(data)
	if !regexp.MustCompile(`(?m)^## Unreleased\s*$`).MatchString(content) {
		t.Error("CHANGELOG.md must contain a level-two Unreleased heading")
	}
	for _, tag := range auditedReleaseTags {
		heading := regexp.MustCompile(fmt.Sprintf(`(?m)^## \[?%s\]?\s*$`, regexp.QuoteMeta(tag)))
		if !heading.MatchString(content) {
			t.Errorf("CHANGELOG.md is missing a level-two heading for audited release %s", tag)
		}
	}
}
