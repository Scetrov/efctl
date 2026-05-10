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

func TestPatchNpmrc_CreatesNew(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pnpm_npmrc_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	npmrcPath := filepath.Join(tmpDir, ".npmrc")
	if err := patchNpmrc(npmrcPath); err != nil {
		t.Fatalf("patchNpmrc failed on new file: %v", err)
	}

	data, err := os.ReadFile(npmrcPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "onlyBuiltDependencies=esbuild") {
		t.Errorf("npmrc should contain onlyBuiltDependencies=esbuild. Got:\n%s", content)
	}
}

func TestPatchNpmrc_AppendsToExisting(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pnpm_npmrc_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	npmrcPath := filepath.Join(tmpDir, ".npmrc")
	existing := "strict-peer-dependencies=true\n"
	if err := os.WriteFile(npmrcPath, []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	if err := patchNpmrc(npmrcPath); err != nil {
		t.Fatalf("patchNpmrc failed: %v", err)
	}

	data, err := os.ReadFile(npmrcPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "strict-peer-dependencies=true") {
		t.Errorf("original content should be preserved")
	}
	if !strings.Contains(content, "onlyBuiltDependencies=esbuild") {
		t.Errorf("npmrc should contain onlyBuiltDependencies=esbuild. Got:\n%s", content)
	}
}

func TestPatchNpmrc_Idempotent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pnpm_npmrc_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	npmrcPath := filepath.Join(tmpDir, ".npmrc")
	if err := patchNpmrc(npmrcPath); err != nil {
		t.Fatalf("patchNpmrc failed: %v", err)
	}

	data1, _ := os.ReadFile(npmrcPath)

	if err := patchNpmrc(npmrcPath); err != nil {
		t.Fatalf("second patch failed: %v", err)
	}

	data2, _ := os.ReadFile(npmrcPath)
	if string(data1) != string(data2) {
		t.Errorf("npmrc patch is not idempotent.\nFirst:  %q\nSecond: %q", data1, data2)
	}
}
