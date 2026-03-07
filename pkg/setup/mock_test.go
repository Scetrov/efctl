package setup

import (
	"context"
	"time"

	"efctl/pkg/container"

	"github.com/stretchr/testify/mock"
)

// mockContainerClient is a local testify mock of container.ContainerClient
// used by orchestration tests in this package.
type mockContainerClient struct {
	mock.Mock
}

func (m *mockContainerClient) BuildImage(ctx context.Context, contextDir string, dockerfilePath string, tag string) error {
	return m.Called(ctx, contextDir, dockerfilePath, tag).Error(0)
}

func (m *mockContainerClient) CreateNetwork(ctx context.Context, name string) error {
	return m.Called(ctx, name).Error(0)
}

func (m *mockContainerClient) RemoveNetwork(ctx context.Context, name string) error {
	return m.Called(ctx, name).Error(0)
}

func (m *mockContainerClient) CreateVolume(ctx context.Context, name string) error {
	return m.Called(ctx, name).Error(0)
}

func (m *mockContainerClient) CreateContainer(ctx context.Context, cfg container.ContainerConfig) error {
	return m.Called(ctx, cfg).Error(0)
}

func (m *mockContainerClient) StartContainer(ctx context.Context, name string) error {
	return m.Called(ctx, name).Error(0)
}

func (m *mockContainerClient) StopContainer(ctx context.Context, name string) error {
	return m.Called(ctx, name).Error(0)
}

func (m *mockContainerClient) RemoveContainer(ctx context.Context, name string) error {
	return m.Called(ctx, name).Error(0)
}

func (m *mockContainerClient) WaitHealthy(ctx context.Context, name string, timeout time.Duration) error {
	return m.Called(ctx, name, timeout).Error(0)
}

func (m *mockContainerClient) GetEngine() string {
	return m.Called().String(0)
}

func (m *mockContainerClient) NetworkName() string {
	return m.Called().String(0)
}

func (m *mockContainerClient) ContainerRunning(name string) bool {
	return m.Called(name).Bool(0)
}

func (m *mockContainerClient) ContainerLogs(name string, tail int) string {
	return m.Called(name, tail).String(0)
}

func (m *mockContainerClient) ContainerExitCode(name string) (int, error) {
	args := m.Called(name)
	return args.Int(0), args.Error(1)
}

func (m *mockContainerClient) WaitForLogs(ctx context.Context, containerName string, searchString string) error {
	return m.Called(ctx, containerName, searchString).Error(0)
}

func (m *mockContainerClient) InteractiveShell(containerName string) error {
	return m.Called(containerName).Error(0)
}

func (m *mockContainerClient) Exec(ctx context.Context, containerName string, command []string) error {
	return m.Called(ctx, containerName, command).Error(0)
}

func (m *mockContainerClient) ExecCapture(ctx context.Context, containerName string, command []string) (string, error) {
	args := m.Called(ctx, containerName, command)
	return args.String(0), args.Error(1)
}

func (m *mockContainerClient) RemoveImages(names []string) {
	m.Called(names)
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

func (m *mockGitClient) CheckoutRef(repoDir, ref string) error {
	return m.Called(repoDir, ref).Error(0)
}

func (m *mockGitClient) SetupWorkDir(workspace string) error {
	return m.Called(workspace).Error(0)
}
