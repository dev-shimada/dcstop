package devcontainer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig(t *testing.T) {
	t.Run("parses image-based config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "devcontainer.json")
		content := `{
			"name": "Go Dev",
			"image": "golang:1.21"
		}`
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		config, err := ParseConfig(configPath)
		require.NoError(t, err)
		assert.Equal(t, "golang:1.21", config.Image)
		assert.Empty(t, config.DockerComposeFile)
		assert.True(t, config.IsImageBased())
		assert.False(t, config.IsComposeBased())
	})

	t.Run("parses compose-based config with single file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "devcontainer.json")
		content := `{
			"name": "Full Stack",
			"dockerComposeFile": "docker-compose.yml",
			"service": "app"
		}`
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		config, err := ParseConfig(configPath)
		require.NoError(t, err)
		assert.Empty(t, config.Image)
		assert.Equal(t, []string{"docker-compose.yml"}, config.DockerComposeFile)
		assert.Equal(t, "app", config.Service)
		assert.False(t, config.IsImageBased())
		assert.True(t, config.IsComposeBased())
	})

	t.Run("parses compose-based config with multiple files", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "devcontainer.json")
		content := `{
			"name": "Multi Compose",
			"dockerComposeFile": ["docker-compose.yml", "docker-compose.override.yml"],
			"service": "web"
		}`
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		config, err := ParseConfig(configPath)
		require.NoError(t, err)
		assert.Equal(t, []string{"docker-compose.yml", "docker-compose.override.yml"}, config.DockerComposeFile)
		assert.True(t, config.IsComposeBased())
	})

	t.Run("handles JSON with comments (JSONC)", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "devcontainer.json")
		content := `{
			// This is a comment
			"name": "JSONC Test",
			"image": "ubuntu:22.04"
			/* Another comment */
		}`
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		config, err := ParseConfig(configPath)
		require.NoError(t, err)
		assert.Equal(t, "ubuntu:22.04", config.Image)
	})

	t.Run("handles trailing commas", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "devcontainer.json")
		content := `{
			"name": "Trailing Comma",
			"image": "node:18",
		}`
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		config, err := ParseConfig(configPath)
		require.NoError(t, err)
		assert.Equal(t, "node:18", config.Image)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		config, err := ParseConfig("/non/existent/devcontainer.json")
		assert.Error(t, err)
		assert.Nil(t, config)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "devcontainer.json")
		require.NoError(t, os.WriteFile(configPath, []byte(`{invalid`), 0644))

		config, err := ParseConfig(configPath)
		assert.Error(t, err)
		assert.Nil(t, config)
	})

	t.Run("stores config file path", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "devcontainer.json")
		content := `{"image": "alpine"}`
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		config, err := ParseConfig(configPath)
		require.NoError(t, err)
		assert.Equal(t, configPath, config.ConfigPath)
	})
}

func TestConfig_GetComposeFiles(t *testing.T) {
	t.Run("returns absolute paths for compose files", func(t *testing.T) {
		tmpDir := t.TempDir()
		devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
		require.NoError(t, os.MkdirAll(devcontainerDir, 0755))

		configPath := filepath.Join(devcontainerDir, "devcontainer.json")
		content := `{
			"dockerComposeFile": ["../docker-compose.yml", "docker-compose.dev.yml"],
			"service": "app"
		}`
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		config, err := ParseConfig(configPath)
		require.NoError(t, err)

		files := config.GetComposeFiles()
		assert.Len(t, files, 2)
		assert.Equal(t, filepath.Join(tmpDir, "docker-compose.yml"), files[0])
		assert.Equal(t, filepath.Join(devcontainerDir, "docker-compose.dev.yml"), files[1])
	})
}
