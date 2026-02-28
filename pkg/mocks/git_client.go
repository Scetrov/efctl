package mocks

import (
	"github.com/stretchr/testify/mock"
)

// MockGitClient is a testify mock for git.GitClient.
type MockGitClient struct {
	mock.Mock
}

func (m *MockGitClient) CloneRepository(url string, dest string) error {
	args := m.Called(url, dest)
	return args.Error(0)
}

func (m *MockGitClient) CheckoutBranch(repoPath string, branch string) error {
	args := m.Called(repoPath, branch)
	return args.Error(0)
}

func (m *MockGitClient) SetupWorkDir(path string) error {
	args := m.Called(path)
	return args.Error(0)
}
