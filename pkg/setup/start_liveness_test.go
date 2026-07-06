package setup

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"efctl/pkg/container"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestWaitForSuiLivenessSucceedsWhenContainerRunning(t *testing.T) {
	c := new(mockContainerClient)
	c.On("ContainerRunning", container.ContainerSuiPlayground).Return(true).Once()

	start := time.Now()
	err := waitForSuiLiveness(c, container.ContainerSuiPlayground, 0, time.Hour, time.Hour)

	require.NoError(t, err)
	assert.Less(t, time.Since(start), 100*time.Millisecond)
	c.AssertExpectations(t)
}

func TestWaitForSuiLivenessFailsEarlyWithExitDiagnostics(t *testing.T) {
	c := new(mockContainerClient)
	c.On("ContainerRunning", container.ContainerSuiPlayground).Return(false).Once()
	c.On("ContainerExitCode", container.ContainerSuiPlayground).Return(42, nil).Once()
	c.On("ContainerLogs", container.ContainerSuiPlayground, suiLivenessDiagnosticLines).Return("boom").Once()

	err := waitForSuiLiveness(c, container.ContainerSuiPlayground, 0, time.Millisecond, time.Second)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "exit code 42")
	assert.Contains(t, err.Error(), "boom")
	c.AssertExpectations(t)
}

func TestWaitForSuiLivenessTimeoutIncludesDiagnostics(t *testing.T) {
	c := new(mockContainerClient)
	c.On("ContainerRunning", container.ContainerSuiPlayground).Return(false)
	c.On("ContainerExitCode", container.ContainerSuiPlayground).Return(-1, errors.New("not stopped"))
	c.On("ContainerLogs", container.ContainerSuiPlayground, suiLivenessDiagnosticLines).Return("still starting").Once()

	err := waitForSuiLiveness(c, container.ContainerSuiPlayground, 0, time.Millisecond, 2*time.Millisecond)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "did not become live")
	assert.Contains(t, err.Error(), "ExitCode: -1")
	assert.Contains(t, err.Error(), "not stopped")
	assert.Contains(t, err.Error(), "still starting")
	c.AssertExpectations(t)
}

func TestStartSuiDevWaitsForReadyLogAfterLiveness(t *testing.T) {
	oldWaitForSuiLiveness := waitForSuiLivenessFunc
	waitForSuiLivenessFunc = func(c container.ContainerClient, containerName string, gracePeriod, pollInterval, timeout time.Duration) error {
		assert.Equal(t, container.ContainerSuiPlayground, containerName)
		return nil
	}
	t.Cleanup(func() { waitForSuiLivenessFunc = oldWaitForSuiLiveness })

	c := new(mockContainerClient)
	c.On("NetworkName").Return("test-net")
	c.On("GetEngine").Return("docker")
	c.On("CreateContainer", mock.Anything, mock.AnythingOfType("container.ContainerConfig")).Return(nil).Once()
	c.On("StartContainer", mock.Anything, container.ContainerSuiPlayground).Return(nil).Once()
	c.On("Exec", mock.Anything, container.ContainerSuiPlayground, mock.AnythingOfType("[]string")).Return(nil).Maybe()
	c.On("WaitForLogs", mock.Anything, container.ContainerSuiPlayground, container.ContainerLogReadyCtx).Return(nil).Once()
	c.On("ExecCapture", mock.Anything, container.ContainerSuiPlayground, []string{"cat", "/workspace/.sui/.env.sui"}).Return("KEY=value\n", nil).Once()

	err := startSuiDev(c, context.Background(), t.TempDir(), t.TempDir(), false, "sui", "pass", "db")

	require.NoError(t, err)
	c.AssertExpectations(t)
}

func TestStartSuiDevPropagatesLivenessFailureBeforeReadyLog(t *testing.T) {
	oldWaitForSuiLiveness := waitForSuiLivenessFunc
	waitForSuiLivenessFunc = func(c container.ContainerClient, containerName string, gracePeriod, pollInterval, timeout time.Duration) error {
		return errors.New("liveness failed")
	}
	t.Cleanup(func() { waitForSuiLivenessFunc = oldWaitForSuiLiveness })

	c := new(mockContainerClient)
	c.On("NetworkName").Return("test-net")
	c.On("GetEngine").Return("docker")
	c.On("CreateContainer", mock.Anything, mock.AnythingOfType("container.ContainerConfig")).Return(nil).Once()
	c.On("StartContainer", mock.Anything, container.ContainerSuiPlayground).Return(nil).Once()
	c.On("Exec", mock.Anything, container.ContainerSuiPlayground, mock.AnythingOfType("[]string")).Return(nil).Maybe()

	err := startSuiDev(c, context.Background(), t.TempDir(), t.TempDir(), false, "sui", "pass", "db")

	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "liveness failed"))
	c.AssertNotCalled(t, "WaitForLogs", mock.Anything, container.ContainerSuiPlayground, container.ContainerLogReadyCtx)
	c.AssertExpectations(t)
}
