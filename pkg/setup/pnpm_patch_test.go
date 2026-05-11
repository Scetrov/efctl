package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPatchPnpmWorkspaceYaml_CreatesNew(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pnpm_workspace_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	yamlPath := filepath.Join(tmpDir, "pnpm-workspace.yaml")

	if err := patchPnpmWorkspaceYaml(yamlPath); err != nil {
		t.Fatalf("patchPnpmWorkspaceYaml failed: %v", err)
	}

	data, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "allowBuilds:") {
		t.Errorf("pnpm-workspace.yaml should contain allowBuilds:. Got:\n%s", content)
	}
	if !strings.Contains(content, "esbuild: true") {
		t.Errorf("pnpm-workspace.yaml should contain esbuild: true. Got:\n%s", content)
	}
}

func TestPatchPnpmWorkspaceYaml_AppendsToExisting(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pnpm_workspace_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	yamlPath := filepath.Join(tmpDir, "pnpm-workspace.yaml")
	existing := "# my workspace config\npackages:\n  - \"packages/*\"\n"
	if err := os.WriteFile(yamlPath, []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	if err := patchPnpmWorkspaceYaml(yamlPath); err != nil {
		t.Fatalf("patchPnpmWorkspaceYaml failed: %v", err)
	}

	data, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "# my workspace config") {
		t.Errorf("original content should be preserved")
	}
	if !strings.Contains(content, "allowBuilds:") {
		t.Errorf("should contain allowBuilds:. Got:\n%s", content)
	}
	if !strings.Contains(content, "esbuild: true") {
		t.Errorf("should contain esbuild: true. Got:\n%s", content)
	}
}

func TestPatchPnpmWorkspaceYaml_Idempotent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pnpm_workspace_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	yamlPath := filepath.Join(tmpDir, "pnpm-workspace.yaml")

	if err := patchPnpmWorkspaceYaml(yamlPath); err != nil {
		t.Fatalf("first patch failed: %v", err)
	}

	data1, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("failed to read after first patch: %v", err)
	}

	if err := patchPnpmWorkspaceYaml(yamlPath); err != nil {
		t.Fatalf("second patch failed: %v", err)
	}

	data2, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("failed to read after second patch: %v", err)
	}

	if string(data1) != string(data2) {
		t.Errorf("patch is not idempotent.\nFirst:  %q\nSecond: %q", data1, data2)
	}
}

func TestPatchPnpmWorkspaceYaml_NotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pnpm_workspace_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	yamlPath := filepath.Join(tmpDir, "nonexistent", "pnpm-workspace.yaml")

	// Should fail for non-existent directory
	err = patchPnpmWorkspaceYaml(yamlPath)
	if err == nil {
		t.Fatal("expected error for non-existent path")
	}
}

func TestPatchPnpmDependencies_CreatesWorkspaceYaml(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pnpm_deps_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create repo directories with package.json (simulating cloned repos)
	for _, repo := range []string{"builder-scaffold", "world-contracts"} {
		repoDir := filepath.Join(tmpDir, repo)
		if err := os.MkdirAll(repoDir, 0755); err != nil {
			t.Fatal(err)
		}
		// Write minimal package.json
		pkgContent := `{"name": "` + repo + `", "packageManager": "pnpm@10.17.0", "scripts": {"test": "echo test"}}`
		if err := os.WriteFile(filepath.Join(repoDir, "package.json"), []byte(pkgContent), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Run the patch function
	if err := patchPnpmDependencies(tmpDir); err != nil {
		t.Fatalf("patchPnpmDependencies failed: %v", err)
	}

	// Verify pnpm-workspace.yaml was created in both repos
	for _, repo := range []string{"builder-scaffold", "world-contracts"} {
		yamlPath := filepath.Join(tmpDir, repo, "pnpm-workspace.yaml")
		data, err := os.ReadFile(yamlPath)
		if err != nil {
			t.Fatalf("%s/pnpm-workspace.yaml: %v", repo, err)
		}

		content := string(data)
		if !strings.Contains(content, "allowBuilds:") {
			t.Errorf("%s/pnpm-workspace.yaml should contain allowBuilds:.", repo)
		}
		if !strings.Contains(content, "esbuild: true") {
			t.Errorf("%s/pnpm-workspace.yaml should contain esbuild: true.", repo)
		}
	}
}

func TestPatchPnpmDependencies_PropagatesErrors(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pnpm_deps_err_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Only create one repo directory, leaving the other missing
	repoDir := filepath.Join(tmpDir, "builder-scaffold")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}

	err = patchPnpmDependencies(tmpDir)
	if err == nil {
		t.Fatal("expected error when world-contracts directory is missing")
	}
	// The error message should mention the missing repo
	if !strings.Contains(err.Error(), "world-contracts") {
		t.Errorf("error should mention world-contracts, got: %v", err)
	}
}

func TestPnpmWorkspacePath_RejectsTraversal(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pnpm_workspace_path_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_, err = pnpmWorkspacePath(tmpDir, "../escape")
	if err == nil {
		t.Fatal("expected traversal to be rejected")
	}
	if !strings.Contains(err.Error(), "escapes base directory") {
		t.Fatalf("expected escape error, got: %v", err)
	}
}

func TestPatchPnpmWorkspaceYaml_DetectsExistingAllowBuilds(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pnpm_workspace_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	yamlPath := filepath.Join(tmpDir, "pnpm-workspace.yaml")

	// Pre-create with allowBuilds for esbuild (maybe from a previous run or manual edit)
	existing := "# existing config\nallowBuilds:\n  esbuild: true\n"
	if err := os.WriteFile(yamlPath, []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	// Patch should be a no-op
	if err := patchPnpmWorkspaceYaml(yamlPath); err != nil {
		t.Fatalf("patchPnpmWorkspaceYaml failed: %v", err)
	}

	data, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	// Should be unchanged
	if content != existing {
		t.Errorf("content should be unchanged.\nExpected: %q\nGot:      %q", existing, content)
	}
}

func TestContainsAllowBuildsForEsbuild(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "has allowBuilds with esbuild true",
			content: "allowBuilds:\n  esbuild: true\n",
			want:    true,
		},
		{
			name:    "has allowBuilds with esbuild false",
			content: "allowBuilds:\n  esbuild: false\n",
			want:    false,
		},
		{
			name:    "has allowBuilds without esbuild",
			content: "allowBuilds:\n  electron: true\n",
			want:    false,
		},
		{
			name:    "no allowBuilds",
			content: "packages:\n  - \"packages/*\"\n",
			want:    false,
		},
		{
			name:    "empty content",
			content: "",
			want:    false,
		},
		{
			name:    "esbuild outside allowBuilds — false positive guard",
			content: "overrides:\n  esbuild: true\nallowBuilds:\n  electron: true\n",
			want:    false,
		},
		{
			name:    "esbuild in allowBuilds with intermediate keys",
			content: "allowBuilds:\n  electron: true\n  esbuild: true\n",
			want:    true,
		},
		{
			name:    "inline allowBuilds map with esbuild true",
			content: "allowBuilds: { esbuild: true }\n",
			want:    true,
		},
		{
			name:    "inline allowBuilds map without esbuild",
			content: "allowBuilds: { electron: true }\n",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsAllowBuildsForEsbuild(tt.content)
			if got != tt.want {
				t.Errorf("containsAllowBuildsForEsbuild(%q) = %v, want %v", tt.content, got, tt.want)
			}
		})
	}
}

func TestPatchPnpmWorkspaceYaml_MergesIntoExistingAllowBuilds(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pnpm_workspace_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	yamlPath := filepath.Join(tmpDir, "pnpm-workspace.yaml")
	// Pre-create with allowBuilds for electron but NOT esbuild.
	existing := "packages:\n  - \"packages/*\"\nallowBuilds:\n  electron: true\n"
	if err := os.WriteFile(yamlPath, []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	if err := patchPnpmWorkspaceYaml(yamlPath); err != nil {
		t.Fatalf("patchPnpmWorkspaceYaml failed: %v", err)
	}

	data, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	// Should have exactly ONE allowBuilds: key.
	if countOccurrences(content, "allowBuilds:") != 1 {
		t.Errorf("should have exactly one allowBuilds: key. Got:\n%s", content)
	}
	if !strings.Contains(content, "electron: true") {
		t.Errorf("original electron: true should be preserved. Got:\n%s", content)
	}
	if !strings.Contains(content, "esbuild: true") {
		t.Errorf("should contain esbuild: true. Got:\n%s", content)
	}

	// Idempotency check: patching again should not duplicate esbuild entry.
	if err := patchPnpmWorkspaceYaml(yamlPath); err != nil {
		t.Fatalf("second patch failed: %v", err)
	}

	data2, _ := os.ReadFile(yamlPath)
	if string(data) != string(data2) {
		t.Errorf("double-merge: patch is not idempotent.\nFirst:  %q\nSecond: %q", data, data2)
	}
}

func TestPatchPnpmWorkspaceYaml_MergesInlineAllowBuildsMap(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pnpm_workspace_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	yamlPath := filepath.Join(tmpDir, "pnpm-workspace.yaml")
	existing := "packages:\n  - \"packages/*\"\nallowBuilds: { electron: true }\n"
	if err := os.WriteFile(yamlPath, []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	if err := patchPnpmWorkspaceYaml(yamlPath); err != nil {
		t.Fatalf("patchPnpmWorkspaceYaml failed: %v", err)
	}

	data, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if countOccurrences(content, "allowBuilds:") != 1 {
		t.Errorf("should have exactly one allowBuilds: key. Got:\n%s", content)
	}
	if !strings.Contains(content, "electron: true") {
		t.Errorf("original electron entry should be preserved. Got:\n%s", content)
	}
	if !strings.Contains(content, "esbuild: true") {
		t.Errorf("should contain esbuild: true. Got:\n%s", content)
	}
	if strings.Count(content, "esbuild: true") != 1 {
		t.Errorf("should contain exactly one esbuild: true entry. Got:\n%s", content)
	}
	if err := patchPnpmWorkspaceYaml(yamlPath); err != nil {
		t.Fatalf("second patch failed: %v", err)
	}
	data2, _ := os.ReadFile(yamlPath)
	if string(data) != string(data2) {
		t.Errorf("inline merge is not idempotent.\nFirst:  %q\nSecond: %q", data, data2)
	}
}

func TestPatchPnpmWorkspaceYaml_UpdatesExistingFalseValue(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pnpm_workspace_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	yamlPath := filepath.Join(tmpDir, "pnpm-workspace.yaml")
	existing := "allowBuilds:\n  esbuild: false\n"
	if err := os.WriteFile(yamlPath, []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	if err := patchPnpmWorkspaceYaml(yamlPath); err != nil {
		t.Fatalf("patchPnpmWorkspaceYaml failed: %v", err)
	}

	data, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "esbuild: true") {
		t.Errorf("should update esbuild to true. Got:\n%s", content)
	}
	if strings.Contains(content, "esbuild: false") {
		t.Errorf("should not leave esbuild: false behind. Got:\n%s", content)
	}
}

func countOccurrences(content, substr string) int {
	count := 0
	idx := 0
	for {
		pos := strings.Index(content[idx:], substr)
		if pos == -1 {
			break
		}
		idx += pos + len(substr)
		count++
	}
	return count
}
