package docker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockContainerClient is a mock implementation of ContainerClient
type MockContainerClient struct {
	mock.Mock
}

func (m *MockContainerClient) ContainerList(ctx context.Context, options ContainerListOptions) ([]ContainerInfo, error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]ContainerInfo), args.Error(1)
}

func (m *MockContainerClient) ContainerStop(ctx context.Context, containerID string, timeout *int) error {
	args := m.Called(ctx, containerID, timeout)
	return args.Error(0)
}

func (m *MockContainerClient) ContainerRemove(ctx context.Context, containerID string, force bool) error {
	args := m.Called(ctx, containerID, force)
	return args.Error(0)
}

func (m *MockContainerClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestFindDevcontainersByFolder(t *testing.T) {
	t.Run("finds containers by local folder label", func(t *testing.T) {
		mockClient := new(MockContainerClient)

		containers := []ContainerInfo{
			{
				ID:    "abc123",
				Names: []string{"/devcontainer-myproject"},
				Labels: map[string]string{
					"devcontainer.local_folder": "/home/user/myproject",
					"devcontainer.config_file":  "/home/user/myproject/.devcontainer/devcontainer.json",
				},
				State: "running",
			},
		}

		mockClient.On("ContainerList", mock.Anything, mock.MatchedBy(func(opts ContainerListOptions) bool {
			return opts.LabelFilter == "devcontainer.local_folder=/home/user/myproject"
		})).Return(containers, nil)

		ops := NewContainerOps(mockClient)
		result, err := ops.FindDevcontainersByFolder(context.Background(), "/home/user/myproject")

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "abc123", result[0].ID)
		mockClient.AssertExpectations(t)
	})

	t.Run("returns empty when no containers found", func(t *testing.T) {
		mockClient := new(MockContainerClient)

		mockClient.On("ContainerList", mock.Anything, mock.Anything).Return([]ContainerInfo{}, nil)

		ops := NewContainerOps(mockClient)
		result, err := ops.FindDevcontainersByFolder(context.Background(), "/home/user/empty")

		require.NoError(t, err)
		assert.Empty(t, result)
		mockClient.AssertExpectations(t)
	})
}

func TestFindDevcontainersByConfigPath(t *testing.T) {
	t.Run("finds containers by config file label", func(t *testing.T) {
		mockClient := new(MockContainerClient)

		containers := []ContainerInfo{
			{
				ID:    "def456",
				Names: []string{"/devcontainer-subproject"},
				Labels: map[string]string{
					"devcontainer.local_folder": "/home/user/myproject",
					"devcontainer.config_file":  "/home/user/myproject/.devcontainer/node/devcontainer.json",
				},
				State: "running",
			},
		}

		mockClient.On("ContainerList", mock.Anything, mock.MatchedBy(func(opts ContainerListOptions) bool {
			return opts.LabelFilter == "devcontainer.config_file=/home/user/myproject/.devcontainer/node/devcontainer.json"
		})).Return(containers, nil)

		ops := NewContainerOps(mockClient)
		result, err := ops.FindDevcontainersByConfigPath(context.Background(), "/home/user/myproject/.devcontainer/node/devcontainer.json")

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "def456", result[0].ID)
		mockClient.AssertExpectations(t)
	})
}

func TestStopContainers(t *testing.T) {
	t.Run("stops all specified containers", func(t *testing.T) {
		mockClient := new(MockContainerClient)

		containers := []ContainerInfo{
			{ID: "container1", Names: []string{"/dev1"}},
			{ID: "container2", Names: []string{"/dev2"}},
		}

		mockClient.On("ContainerStop", mock.Anything, "container1", mock.Anything).Return(nil)
		mockClient.On("ContainerStop", mock.Anything, "container2", mock.Anything).Return(nil)

		ops := NewContainerOps(mockClient)
		err := ops.StopContainers(context.Background(), containers)

		require.NoError(t, err)
		mockClient.AssertExpectations(t)
	})
}

func TestRemoveContainers(t *testing.T) {
	t.Run("removes all specified containers", func(t *testing.T) {
		mockClient := new(MockContainerClient)

		containers := []ContainerInfo{
			{ID: "container1", Names: []string{"/dev1"}},
		}

		mockClient.On("ContainerRemove", mock.Anything, "container1", true).Return(nil)

		ops := NewContainerOps(mockClient)
		err := ops.RemoveContainers(context.Background(), containers)

		require.NoError(t, err)
		mockClient.AssertExpectations(t)
	})
}
