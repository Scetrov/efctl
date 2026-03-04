package mocks

import (
	"context"
	"time"

	"efctl/pkg/container"

	"github.com/stretchr/testify/mock"
)

// MockContainerClient is a testify mock for container.ContainerClient.
type MockContainerClient struct {
	mock.Mock
}

func (m *MockContainerClient) BuildImage(ctx context.Context, contextDir string, dockerfilePath string, tag string) error {
	args := m.Called(ctx, contextDir, dockerfilePath, tag)
	return args.Error(0)
}

func (m *MockContainerClient) CreateNetwork(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockContainerClient) RemoveNetwork(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockContainerClient) CreateVolume(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockContainerClient) CreateContainer(ctx context.Context, cfg container.ContainerConfig) error {
	args := m.Called(ctx, cfg)
	return args.Error(0)
}

func (m *MockContainerClient) StartContainer(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockContainerClient) StopContainer(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockContainerClient) RemoveContainer(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockContainerClient) WaitHealthy(ctx context.Context, name string, timeout time.Duration) error {
	args := m.Called(ctx, name, timeout)
	return args.Error(0)
}

func (m *MockContainerClient) GetEngine() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockContainerClient) NetworkName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockContainerClient) ContainerRunning(name string) bool {
	args := m.Called(name)
	return args.Bool(0)
}

func (m *MockContainerClient) ContainerLogs(name string, tail int) string {
	args := m.Called(name, tail)
	return args.String(0)
}

func (m *MockContainerClient) ContainerExitCode(name string) (int, error) {
	args := m.Called(name)
	return args.Int(0), args.Error(1)
}

func (m *MockContainerClient) WaitForLogs(ctx context.Context, containerName string, searchString string) error {
	args := m.Called(ctx, containerName, searchString)
	return args.Error(0)
}

func (m *MockContainerClient) InteractiveShell(containerName string) error {
	args := m.Called(containerName)
	return args.Error(0)
}

func (m *MockContainerClient) Exec(containerName string, command []string) error {
	args := m.Called(containerName, command)
	return args.Error(0)
}

func (m *MockContainerClient) ExecCapture(containerName string, command []string) (string, error) {
	args := m.Called(containerName, command)
	return args.String(0), args.Error(1)
}

func (m *MockContainerClient) RemoveImages(names []string) {
	m.Called(names)
}

func (m *MockContainerClient) Cleanup() error {
	args := m.Called()
	return args.Error(0)
}
