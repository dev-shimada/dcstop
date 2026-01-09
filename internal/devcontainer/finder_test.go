package devcontainer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindDevcontainerConfigs(t *testing.T) {
	t.Run("finds single devcontainer.json in .devcontainer directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
		require.NoError(t, os.MkdirAll(devcontainerDir, 0755))

		configPath := filepath.Join(devcontainerDir, "devcontainer.json")
		require.NoError(t, os.WriteFile(configPath, []byte(`{"image": "golang:1.21"}`), 0644))

		configs, err := FindDevcontainerConfigs(tmpDir)
		require.NoError(t, err)
		assert.Len(t, configs, 1)
		assert.Equal(t, configPath, configs[0])
	})

	t.Run("finds multiple devcontainer.json in subdirectories", func(t *testing.T) {
		tmpDir := t.TempDir()

		// .devcontainer/devcontainer.json
		devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
		require.NoError(t, os.MkdirAll(devcontainerDir, 0755))
		config1 := filepath.Join(devcontainerDir, "devcontainer.json")
		require.NoError(t, os.WriteFile(config1, []byte(`{"image": "golang:1.21"}`), 0644))

		// .devcontainer/node/devcontainer.json
		nodeDir := filepath.Join(devcontainerDir, "node")
		require.NoError(t, os.MkdirAll(nodeDir, 0755))
		config2 := filepath.Join(nodeDir, "devcontainer.json")
		require.NoError(t, os.WriteFile(config2, []byte(`{"image": "node:18"}`), 0644))

		// .devcontainer/python/devcontainer.json
		pythonDir := filepath.Join(devcontainerDir, "python")
		require.NoError(t, os.MkdirAll(pythonDir, 0755))
		config3 := filepath.Join(pythonDir, "devcontainer.json")
		require.NoError(t, os.WriteFile(config3, []byte(`{"image": "python:3.11"}`), 0644))

		configs, err := FindDevcontainerConfigs(tmpDir)
		require.NoError(t, err)
		assert.Len(t, configs, 3)
		assert.Contains(t, configs, config1)
		assert.Contains(t, configs, config2)
		assert.Contains(t, configs, config3)
	})

	t.Run("returns empty slice when no devcontainer found", func(t *testing.T) {
		tmpDir := t.TempDir()

		configs, err := FindDevcontainerConfigs(tmpDir)
		require.NoError(t, err)
		assert.Empty(t, configs)
	})

	t.Run("returns error for non-existent directory", func(t *testing.T) {
		configs, err := FindDevcontainerConfigs("/non/existent/path")
		assert.Error(t, err)
		assert.Nil(t, configs)
	})
}
