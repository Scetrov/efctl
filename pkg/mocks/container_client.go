package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockContainerClient is a testify mock for container.ContainerClient.
type MockContainerClient struct {
	mock.Mock
}

func (m *MockContainerClient) ComposeBuild(dir string) error {
	args := m.Called(dir)
	return args.Error(0)
}

func (m *MockContainerClient) ComposeRun(dir string) error {
	args := m.Called(dir)
	return args.Error(0)
}

func (m *MockContainerClient) ComposeUp(dir string, services ...string) error {
	args := m.Called(dir, services)
	return args.Error(0)
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

func (m *MockContainerClient) Cleanup() error {
	args := m.Called()
	return args.Error(0)
}
