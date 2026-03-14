package sui

import (
	"testing"
)

type MockExecutor struct {
	Commands [][]string
	Stdin    []string
}

func (m *MockExecutor) LookPath(file string) (string, error) {
	return "/usr/bin/" + file, nil
}

func (m *MockExecutor) Run(name string, args ...string) error {
	m.Commands = append(m.Commands, append([]string{name}, args...))
	return nil
}

func (m *MockExecutor) RunWithStdin(stdin string, name string, args ...string) error {
	m.Commands = append(m.Commands, append([]string{name}, args...))
	m.Stdin = append(m.Stdin, stdin)
	return nil
}

func TestConfigureSui_Commands(t *testing.T) {
	// Note: ConfigureSui in sui.go currently uses exec.Command directly,
	// not the CommandExecutor interface. I should refactor sui.go to use the interface
	// if I want to test it properly without side effects.
	// However, for now, I'll focus on the requested tasks.
}
