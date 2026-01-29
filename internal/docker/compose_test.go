package docker

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dev-shimada/dcstop/internal/devcontainer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockComposeClient extends MockContainerClient with network and volume operations
type MockComposeClient struct {
	MockContainerClient
}

func (m *MockComposeClient) NetworkList(ctx context.Context, options NetworkListOptions) ([]NetworkInfo, error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]NetworkInfo), args.Error(1)
}

func (m *MockComposeClient) NetworkRemove(ctx context.Context, networkID string) error {
	args := m.Called(ctx, networkID)
	return args.Error(0)
}

func (m *MockComposeClient) VolumeList(ctx context.Context, options VolumeListOptions) ([]VolumeInfo, error) {
	args := m.Called(ctx, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]VolumeInfo), args.Error(1)
}

func (m *MockComposeClient) VolumeRemove(ctx context.Context, volumeName string, force bool) error {
	args := m.Called(ctx, volumeName, force)
	return args.Error(0)
}

func TestFindComposeContainers(t *testing.T) {
	t.Run("finds containers by compose project name", func(t *testing.T) {
		mockClient := new(MockComposeClient)

		containers := []ContainerInfo{
			{
				ID:    "web123",
				Names: []string{"/myproject-web-1"},
				Labels: map[string]string{
					"com.docker.compose.project": "myproject",
					"com.docker.compose.service": "web",
				},
				State: "running",
			},
			{
				ID:    "db456",
				Names: []string{"/myproject-db-1"},
				Labels: map[string]string{
					"com.docker.compose.project": "myproject",
					"com.docker.compose.service": "db",
				},
				State: "running",
			},
		}

		mockClient.On("ContainerList", mock.Anything, mock.MatchedBy(func(opts ContainerListOptions) bool {
			return opts.LabelFilter == "com.docker.compose.project=myproject"
		})).Return(containers, nil)

		ops := NewComposeOps(mockClient)
		result, err := ops.FindComposeContainers(context.Background(), "myproject")

		require.NoError(t, err)
		assert.Len(t, result, 2)
		mockClient.AssertExpectations(t)
	})
}

func TestStopComposeProject(t *testing.T) {
	t.Run("stops all containers in compose project", func(t *testing.T) {
		mockClient := new(MockComposeClient)

		containers := []ContainerInfo{
			{ID: "web123", Names: []string{"/myproject-web-1"}},
			{ID: "db456", Names: []string{"/myproject-db-1"}},
		}

		mockClient.On("ContainerList", mock.Anything, mock.Anything).Return(containers, nil)
		mockClient.On("ContainerStop", mock.Anything, "web123", mock.Anything).Return(nil)
		mockClient.On("ContainerStop", mock.Anything, "db456", mock.Anything).Return(nil)

		ops := NewComposeOps(mockClient)
		err := ops.StopComposeProject(context.Background(), "myproject")

		require.NoError(t, err)
		mockClient.AssertExpectations(t)
	})
}

func TestDownComposeProject(t *testing.T) {
	t.Run("stops and removes containers and networks without volumes", func(t *testing.T) {
		mockClient := new(MockComposeClient)

		containers := []ContainerInfo{
			{ID: "web123", Names: []string{"/myproject-web-1"}},
		}

		networks := []NetworkInfo{
			{ID: "net123", Name: "myproject_default"},
		}

		mockClient.On("ContainerList", mock.Anything, mock.Anything).Return(containers, nil)
		mockClient.On("ContainerStop", mock.Anything, "web123", mock.Anything).Return(nil)
		mockClient.On("ContainerRemove", mock.Anything, "web123", true).Return(nil)
		mockClient.On("NetworkList", mock.Anything, mock.MatchedBy(func(opts NetworkListOptions) bool {
			return opts.LabelFilter == "com.docker.compose.project=myproject"
		})).Return(networks, nil)
		mockClient.On("NetworkRemove", mock.Anything, "net123").Return(nil)

		ops := NewComposeOps(mockClient)
		err := ops.DownComposeProject(context.Background(), "myproject", false)

		require.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("stops and removes containers, networks, and volumes", func(t *testing.T) {
		mockClient := new(MockComposeClient)

		containers := []ContainerInfo{
			{ID: "web123", Names: []string{"/myproject-web-1"}},
		}

		networks := []NetworkInfo{
			{ID: "net123", Name: "myproject_default"},
		}

		volumes := []VolumeInfo{
			{Name: "myproject_data", Labels: map[string]string{"com.docker.compose.project": "myproject"}},
		}

		mockClient.On("ContainerList", mock.Anything, mock.Anything).Return(containers, nil)
		mockClient.On("ContainerStop", mock.Anything, "web123", mock.Anything).Return(nil)
		mockClient.On("ContainerRemove", mock.Anything, "web123", true).Return(nil)
		mockClient.On("NetworkList", mock.Anything, mock.MatchedBy(func(opts NetworkListOptions) bool {
			return opts.LabelFilter == "com.docker.compose.project=myproject"
		})).Return(networks, nil)
		mockClient.On("NetworkRemove", mock.Anything, "net123").Return(nil)
		mockClient.On("VolumeList", mock.Anything, mock.MatchedBy(func(opts VolumeListOptions) bool {
			return opts.LabelFilter == "com.docker.compose.project=myproject"
		})).Return(volumes, nil)
		mockClient.On("VolumeRemove", mock.Anything, "myproject_data", true).Return(nil)

		ops := NewComposeOps(mockClient)
		err := ops.DownComposeProject(context.Background(), "myproject", true)

		require.NoError(t, err)
		mockClient.AssertExpectations(t)
	})
}

func TestDeriveDevcontainerProjectName(t *testing.T) {
	t.Run("standard layout - uses parent directory name", func(t *testing.T) {
		name := DeriveDevcontainerProjectName("/home/user/myproject/.devcontainer/devcontainer.json")
		assert.Equal(t, "myproject_devcontainer", name)
	})

	t.Run("standard layout - converts to lowercase", func(t *testing.T) {
		name := DeriveDevcontainerProjectName("/home/user/MyProject/.devcontainer/devcontainer.json")
		assert.Equal(t, "myproject_devcontainer", name)
	})

	t.Run("standard layout - workspace example", func(t *testing.T) {
		name := DeriveDevcontainerProjectName("/workspace/.devcontainer/devcontainer.json")
		assert.Equal(t, "workspace_devcontainer", name)
	})

	t.Run("multi-config layout - uses subdirectory name only", func(t *testing.T) {
		name := DeriveDevcontainerProjectName("/workspace/.devcontainer/app1/devcontainer.json")
		assert.Equal(t, "app1", name)
	})

	t.Run("multi-config layout - converts to lowercase", func(t *testing.T) {
		name := DeriveDevcontainerProjectName("/home/user/MyProject/.devcontainer/MyApp/devcontainer.json")
		assert.Equal(t, "myapp", name)
	})
}

func TestDeriveProjectNameFromComposeFile(t *testing.T) {
	t.Run("compose file in .devcontainer - uses parent directory name", func(t *testing.T) {
		// docker-compose.yml is in .devcontainer directory
		// /home/user/myproject/.devcontainer/docker-compose.yml
		composeFilePath := "/home/user/myproject/.devcontainer/docker-compose.yml"
		name := DeriveProjectNameFromComposeFile(composeFilePath)
		assert.Equal(t, "myproject_devcontainer", name)
	})

	t.Run("compose file in .devcontainer subdirectory - uses subdirectory name", func(t *testing.T) {
		// docker-compose.yml is in .devcontainer/app1 directory
		// /home/user/myproject/.devcontainer/app1/docker-compose.yml
		composeFilePath := "/home/user/myproject/.devcontainer/app1/docker-compose.yml"
		name := DeriveProjectNameFromComposeFile(composeFilePath)
		assert.Equal(t, "app1", name)
	})

	t.Run("compose file outside .devcontainer - uses compose file directory name", func(t *testing.T) {
		// docker-compose.yml is in project root
		// /home/user/myproject/docker-compose.yml
		composeFilePath := "/home/user/myproject/docker-compose.yml"
		name := DeriveProjectNameFromComposeFile(composeFilePath)
		assert.Equal(t, "myproject", name)
	})

	t.Run("compose file in subdirectory outside .devcontainer - uses that directory name", func(t *testing.T) {
		// docker-compose.yml is in docker/ subdirectory
		// /home/user/myproject/docker/docker-compose.yml
		composeFilePath := "/home/user/myproject/docker/docker-compose.yml"
		name := DeriveProjectNameFromComposeFile(composeFilePath)
		assert.Equal(t, "docker", name)
	})

	t.Run("converts to lowercase", func(t *testing.T) {
		composeFilePath := "/home/user/MyProject/Docker/docker-compose.yml"
		name := DeriveProjectNameFromComposeFile(composeFilePath)
		assert.Equal(t, "docker", name)
	})
}

func TestDeriveProjectNameFromComposeFiles(t *testing.T) {
	t.Run("uses first compose file when multiple files", func(t *testing.T) {
		// Multiple compose files - should use the first one
		composeFiles := []string{
			"/home/user/myproject/.devcontainer/docker-compose.yml",
			"/home/user/myproject/.devcontainer/docker-compose.override.yml",
		}
		name := DeriveProjectNameFromComposeFiles(composeFiles)
		assert.Equal(t, "myproject_devcontainer", name)
	})

	t.Run("uses single compose file", func(t *testing.T) {
		composeFiles := []string{"/home/user/myproject/docker/docker-compose.yml"}
		name := DeriveProjectNameFromComposeFiles(composeFiles)
		assert.Equal(t, "docker", name)
	})

	t.Run("returns empty string for empty list", func(t *testing.T) {
		composeFiles := []string{}
		name := DeriveProjectNameFromComposeFiles(composeFiles)
		assert.Equal(t, "", name)
	})

	t.Run("returns empty string for nil list", func(t *testing.T) {
		name := DeriveProjectNameFromComposeFiles(nil)
		assert.Equal(t, "", name)
	})
}

func TestDeriveProjectNameFromConfig(t *testing.T) {
	t.Run("compose-based config with compose file in .devcontainer", func(t *testing.T) {
		tmpDir := t.TempDir()
		devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
		require.NoError(t, os.MkdirAll(devcontainerDir, 0755))

		configPath := filepath.Join(devcontainerDir, "devcontainer.json")
		content := `{
			"dockerComposeFile": "docker-compose.yml",
			"service": "app"
		}`
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		config, err := devcontainer.ParseConfig(configPath)
		require.NoError(t, err)

		projectName := DeriveProjectNameFromConfig(config)
		baseName := filepath.Base(tmpDir)
		expected := baseName + "_devcontainer"
		assert.Equal(t, expected, projectName)
	})

	t.Run("compose-based config with compose file outside .devcontainer", func(t *testing.T) {
		tmpDir := t.TempDir()
		devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
		require.NoError(t, os.MkdirAll(devcontainerDir, 0755))

		configPath := filepath.Join(devcontainerDir, "devcontainer.json")
		content := `{
			"dockerComposeFile": "../docker-compose.yml",
			"service": "app"
		}`
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		config, err := devcontainer.ParseConfig(configPath)
		require.NoError(t, err)

		projectName := DeriveProjectNameFromConfig(config)
		baseName := filepath.Base(tmpDir)
		assert.Equal(t, baseName, projectName)
	})

	t.Run("compose-based config with compose file in subdirectory", func(t *testing.T) {
		tmpDir := t.TempDir()
		devcontainerDir := filepath.Join(tmpDir, ".devcontainer", "app1")
		require.NoError(t, os.MkdirAll(devcontainerDir, 0755))

		configPath := filepath.Join(devcontainerDir, "devcontainer.json")
		content := `{
			"dockerComposeFile": "docker-compose.yml",
			"service": "app"
		}`
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		config, err := devcontainer.ParseConfig(configPath)
		require.NoError(t, err)

		projectName := DeriveProjectNameFromConfig(config)
		assert.Equal(t, "app1", projectName)
	})

	t.Run("image-based config falls back to devcontainer.json path", func(t *testing.T) {
		tmpDir := t.TempDir()
		devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
		require.NoError(t, os.MkdirAll(devcontainerDir, 0755))

		configPath := filepath.Join(devcontainerDir, "devcontainer.json")
		content := `{
			"image": "golang:1.21"
		}`
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		config, err := devcontainer.ParseConfig(configPath)
		require.NoError(t, err)

		projectName := DeriveProjectNameFromConfig(config)
		baseName := filepath.Base(tmpDir)
		expected := baseName + "_devcontainer"
		assert.Equal(t, expected, projectName)
	})
}
