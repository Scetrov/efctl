package mocks

import (
	"github.com/stretchr/testify/mock"
)

// MockCommandExecutor is a testify mock for sui.CommandExecutor.
type MockCommandExecutor struct {
	mock.Mock
}

func (m *MockCommandExecutor) LookPath(file string) (string, error) {
	args := m.Called(file)
	return args.String(0), args.Error(1)
}

func (m *MockCommandExecutor) Run(name string, cmdArgs ...string) error {
	args := m.Called(name, cmdArgs)
	return args.Error(0)
}

func (m *MockCommandExecutor) RunWithStdin(stdin string, name string, cmdArgs ...string) error {
	args := m.Called(stdin, name, cmdArgs)
	return args.Error(0)
}
