package setup

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// patchPnpmDependencies creates pnpm-workspace.yaml in each repo directory
// to allow esbuild build scripts. Since pnpm v10.26+, onlyBuiltDependencies
// was replaced by allowBuilds in pnpm-workspace.yaml. The .npmrc file only
// reads auth/registry settings and is ignored for build-related configs.
//
// pnpm-workspace.yaml format (valid for pnpm v10.26+ and v11+):
//
//	allowBuilds:
//	  esbuild: true
func patchPnpmDependencies(workspace string) error {
	repos := []string{"builder-scaffold", "world-contracts"}
	var errs []error
	for _, repo := range repos {
		workspacePath, err := pnpmWorkspacePath(workspace, repo)
		if err != nil {
			errs = append(errs, fmt.Errorf("resolve pnpm-workspace.yaml path in %s: %w", repo, err))
			continue
		}

		// Create pnpm-workspace.yaml with allowBuilds for esbuild.
		// This is the only supported location for build-related settings
		// in pnpm v10.26+ (allowBuilds replaces onlyBuiltDependencies in v11).
		if err := patchPnpmWorkspaceYaml(workspacePath); err != nil {
			errs = append(errs, fmt.Errorf("patch pnpm-workspace.yaml in %s: %w", repo, err))
		}
	}
	return errors.Join(errs...)
}

func pnpmWorkspacePath(workspace, repo string) (string, error) {
	return safePath(workspace, repo, "pnpm-workspace.yaml")
}

// patchPnpmWorkspaceYaml creates or updates pnpm-workspace.yaml with
// allowBuilds configuration for esbuild. Idempotent — safe to re-run.
func patchPnpmWorkspaceYaml(path string) error {
	// Read existing content if present
	existing, err := os.ReadFile(path) // #nosec G304 -- path is validated by patchPnpmDependencies or provided by a test fixture
	if err != nil {
		if os.IsNotExist(err) {
			// Create new file
			content := "allowBuilds:\n  esbuild: true\n"
			return os.WriteFile(path, []byte(content), 0600) // #nosec G306 G703 -- path is validated by patchPnpmDependencies or provided by a test fixture; restricted permissions
		}
		return err
	}

	updated, changed, err := ensureAllowBuildsEsbuild(existing)
	if err != nil {
		return fmt.Errorf("parse pnpm-workspace.yaml: %w", err)
	}
	if !changed {
		return nil
	}

	return os.WriteFile(path, updated, 0600) // #nosec G306 G703 -- path is validated by patchPnpmDependencies or provided by a test fixture; restricted permissions
}

func ensureAllowBuildsEsbuild(content []byte) ([]byte, bool, error) {
	var document yaml.Node
	if err := yaml.Unmarshal(content, &document); err != nil {
		return nil, false, err
	}

	root, changed, err := ensureDocumentMapping(&document)
	if err != nil {
		return nil, false, err
	}

	allowBuildsNode, allowBuildsChanged, err := ensureMappingValue(root, "allowBuilds")
	if err != nil {
		return nil, false, err
	}
	changed = changed || allowBuildsChanged

	esbuildChanged := ensureBoolMappingValue(allowBuildsNode, "esbuild", true)
	changed = changed || esbuildChanged
	if !changed {
		return nil, false, nil
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(&document); err != nil {
		return nil, false, err
	}
	if err := encoder.Close(); err != nil {
		return nil, false, err
	}

	return buf.Bytes(), true, nil
}

func ensureDocumentMapping(document *yaml.Node) (*yaml.Node, bool, error) {
	if len(document.Content) == 0 {
		document.Kind = yaml.DocumentNode
		document.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
		return document.Content[0], true, nil
	}

	root := document.Content[0]
	if root.Kind == yaml.ScalarNode && root.Tag == "!!null" {
		document.Content[0] = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		return document.Content[0], true, nil
	}
	if root.Kind != yaml.MappingNode {
		return nil, false, fmt.Errorf("expected top-level mapping, got YAML node kind %d", root.Kind)
	}

	return root, false, nil
}

func ensureMappingValue(root *yaml.Node, key string) (*yaml.Node, bool, error) {
	valueNode, _ := mappingValue(root, key)
	if valueNode == nil {
		valueNode = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, valueNode)
		return valueNode, true, nil
	}

	if valueNode.Kind == yaml.ScalarNode && valueNode.Tag == "!!null" {
		valueNode.Kind = yaml.MappingNode
		valueNode.Tag = "!!map"
		valueNode.Value = ""
		return valueNode, true, nil
	}
	if valueNode.Kind != yaml.MappingNode {
		return nil, false, fmt.Errorf("expected %s to be a YAML mapping, got node kind %d", key, valueNode.Kind)
	}

	return valueNode, false, nil
}

func ensureBoolMappingValue(root *yaml.Node, key string, want bool) bool {
	valueNode, _ := mappingValue(root, key)
	wantValue := "false"
	if want {
		wantValue = "true"
	}

	if valueNode == nil {
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: wantValue},
		)
		return true
	}

	if valueNode.Kind == yaml.ScalarNode && valueNode.Tag == "!!bool" && valueNode.Value == wantValue {
		return false
	}

	valueNode.Kind = yaml.ScalarNode
	valueNode.Tag = "!!bool"
	valueNode.Value = wantValue
	valueNode.Content = nil
	return true
}

func mappingValue(root *yaml.Node, key string) (*yaml.Node, int) {
	for index := 0; index+1 < len(root.Content); index += 2 {
		if root.Content[index].Value == key {
			return root.Content[index+1], index + 1
		}
	}

	return nil, -1
}

// containsAllowBuildsForEsbuild checks if the content already enables
// allowBuilds.esbuild: true.
func containsAllowBuildsForEsbuild(content string) bool {
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return false
	}

	root, _, err := ensureDocumentMapping(&document)
	if err != nil {
		return false
	}

	allowBuildsNode, _ := mappingValue(root, "allowBuilds")
	if allowBuildsNode == nil || allowBuildsNode.Kind != yaml.MappingNode {
		return false
	}

	esbuildNode, _ := mappingValue(allowBuildsNode, "esbuild")
	return esbuildNode != nil && esbuildNode.Kind == yaml.ScalarNode && esbuildNode.Tag == "!!bool" && esbuildNode.Value == "true"
}
