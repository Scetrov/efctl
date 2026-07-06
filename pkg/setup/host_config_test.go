package setup

import (
	"context"
	"testing"
	"time"

	"efctl/pkg/config"
	"efctl/pkg/container"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestStartPostgresKeepsLocalHostByDefault(t *testing.T) {
	oldLoaded := config.Loaded
	config.Loaded = &config.Config{Host: "0.0.0.0"}
	defer func() { config.Loaded = oldLoaded }()

	m := &mockContainerClient{}
	m.On("NetworkName").Return("efctl-test")
	m.On("CreateVolume", mock.Anything, container.VolumePgData).Return(nil)
	m.On("CreateContainer", mock.Anything, mock.MatchedBy(func(cfg container.ContainerConfig) bool {
		return cfg.Name == container.ContainerPostgres && cfg.Host == "127.0.0.1"
	})).Return(nil)
	m.On("StartContainer", mock.Anything, container.ContainerPostgres).Return(nil)
	m.On("WaitHealthy", mock.Anything, container.ContainerPostgres, 60*time.Second).Return(nil)

	require.NoError(t, startPostgres(m, context.Background(), "sui", "pass", "db"))
	m.AssertExpectations(t)
}

func TestStartPostgresUsesServiceHostWhenExposed(t *testing.T) {
	oldLoaded := config.Loaded
	config.Loaded = &config.Config{Host: "0.0.0.0", ExposePostgres: true}
	defer func() { config.Loaded = oldLoaded }()

	m := &mockContainerClient{}
	m.On("NetworkName").Return("efctl-test")
	m.On("CreateVolume", mock.Anything, container.VolumePgData).Return(nil)
	m.On("CreateContainer", mock.Anything, mock.MatchedBy(func(cfg container.ContainerConfig) bool {
		return cfg.Name == container.ContainerPostgres && cfg.Host == "0.0.0.0"
	})).Return(nil)
	m.On("StartContainer", mock.Anything, container.ContainerPostgres).Return(nil)
	m.On("WaitHealthy", mock.Anything, container.ContainerPostgres, 60*time.Second).Return(nil)

	require.NoError(t, startPostgres(m, context.Background(), "sui", "pass", "db"))
	m.AssertExpectations(t)
}
