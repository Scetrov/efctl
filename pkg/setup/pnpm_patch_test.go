package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPatchPackageJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pnpm_patch_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	pkgPath := filepath.Join(tmpDir, "package.json")
	content := `{
  "name": "test-repo",
  "packageManager": "pnpm@10.17.0",
  "scripts": {
    "test": "echo test"
  }
}`
	if err := os.WriteFile(pkgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := patchPackageJSON(pkgPath); err != nil {
		t.Fatalf("patchPackageJSON failed: %v", err)
	}

	newData, err := os.ReadFile(pkgPath)
	if err != nil {
		t.Fatal(err)
	}

	newContent := string(newData)
	if !strings.Contains(newContent, "\"onlyBuiltDependencies\"") {
		t.Errorf("patch did not inject onlyBuiltDependencies. Content:\n%s", newContent)
	}
	if !strings.Contains(newContent, "\"esbuild\"") {
		t.Errorf("patch did not inject esbuild")
	}

	// Verify idempotency
	if err := patchPackageJSON(pkgPath); err != nil {
		t.Fatalf("second patch failed: %v", err)
	}
	newData2, _ := os.ReadFile(pkgPath)
	if string(newData2) != newContent {
		t.Errorf("patch is not idempotent")
	}
}

func TestPatchPackageJSON_NoPackageManager(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pnpm_patch_test_no_pkg")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	pkgPath := filepath.Join(tmpDir, "package.json")
	content := `{
  "name": "test-repo",
  "scripts": {
    "test": "echo test"
  }
}`
	if err := os.WriteFile(pkgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := patchPackageJSON(pkgPath); err != nil {
		t.Fatalf("patchPackageJSON failed: %v", err)
	}

	newData, _ := os.ReadFile(pkgPath)
	newContent := string(newData)
	if !strings.Contains(newContent, "\"onlyBuiltDependencies\"") {
		t.Errorf("patch did not inject onlyBuiltDependencies. Content:\n%s", newContent)
	}
}
