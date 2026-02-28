package setup

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// mockContainerClient is a local testify mock of container.ContainerClient
// used by orchestration tests in this package.
type mockContainerClient struct {
	mock.Mock
}

func (m *mockContainerClient) ComposeBuild(dir string) error {
	return m.Called(dir).Error(0)
}

func (m *mockContainerClient) ComposeRun(dir string) error {
	return m.Called(dir).Error(0)
}

func (m *mockContainerClient) ComposeUp(dir string, services ...string) error {
	return m.Called(dir, services).Error(0)
}

func (m *mockContainerClient) ContainerRunning(name string) bool {
	return m.Called(name).Bool(0)
}

func (m *mockContainerClient) ContainerLogs(name string, tail int) string {
	return m.Called(name, tail).String(0)
}

func (m *mockContainerClient) WaitForLogs(ctx context.Context, containerName string, searchString string) error {
	return m.Called(ctx, containerName, searchString).Error(0)
}

func (m *mockContainerClient) InteractiveShell(containerName string) error {
	return m.Called(containerName).Error(0)
}

func (m *mockContainerClient) Exec(containerName string, command []string) error {
	return m.Called(containerName, command).Error(0)
}

func (m *mockContainerClient) ExecCapture(containerName string, command []string) (string, error) {
	args := m.Called(containerName, command)
	return args.String(0), args.Error(1)
}

func (m *mockContainerClient) Cleanup() error {
	return m.Called().Error(0)
}

// mockGitClient is a local testify mock of git.GitClient
// used by orchestration tests in this package.
type mockGitClient struct {
	mock.Mock
}

func (m *mockGitClient) CloneRepository(url, dest string) error {
	return m.Called(url, dest).Error(0)
}

func (m *mockGitClient) CheckoutBranch(repoDir, branch string) error {
	return m.Called(repoDir, branch).Error(0)
}

func (m *mockGitClient) SetupWorkDir(workspace string) error {
	return m.Called(workspace).Error(0)
}
