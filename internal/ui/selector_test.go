package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dev-shimada/dcstop/internal/devcontainer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeduplicateConfigs(t *testing.T) {
	t.Run("removes duplicate project names", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create two configs that reference the same compose file location
		// Both reference ../docker-compose.yml which results in the same project name
		devcontainerDir1 := filepath.Join(tmpDir, ".devcontainer", "app1")
		require.NoError(t, os.MkdirAll(devcontainerDir1, 0755))
		configPath1 := filepath.Join(devcontainerDir1, "devcontainer.json")
		content1 := `{
			"dockerComposeFile": "../docker-compose.yml",
			"service": "web"
		}`
		require.NoError(t, os.WriteFile(configPath1, []byte(content1), 0644))

		devcontainerDir2 := filepath.Join(tmpDir, ".devcontainer", "app2")
		require.NoError(t, os.MkdirAll(devcontainerDir2, 0755))
		configPath2 := filepath.Join(devcontainerDir2, "devcontainer.json")
		content2 := `{
			"dockerComposeFile": "../docker-compose.yml",
			"service": "db"
		}`
		require.NoError(t, os.WriteFile(configPath2, []byte(content2), 0644))

		config1, err := devcontainer.ParseConfig(configPath1)
		require.NoError(t, err)
		config2, err := devcontainer.ParseConfig(configPath2)
		require.NoError(t, err)

		configs := []*devcontainer.Config{config1, config2}
		deduplicated := deduplicateConfigs(configs)

		// Should only have one config since both reference the same compose file location
		// and thus have the same project name
		assert.Len(t, deduplicated, 1)
	})

	t.Run("keeps configs with different project names", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create two configs with different project names
		devcontainerDir1 := filepath.Join(tmpDir, ".devcontainer")
		require.NoError(t, os.MkdirAll(devcontainerDir1, 0755))
		configPath1 := filepath.Join(devcontainerDir1, "devcontainer.json")
		content1 := `{
			"dockerComposeFile": "docker-compose.yml",
			"service": "web"
		}`
		require.NoError(t, os.WriteFile(configPath1, []byte(content1), 0644))

		devcontainerDir2 := filepath.Join(tmpDir, ".devcontainer", "app1")
		require.NoError(t, os.MkdirAll(devcontainerDir2, 0755))
		configPath2 := filepath.Join(devcontainerDir2, "devcontainer.json")
		content2 := `{
			"dockerComposeFile": "docker-compose.yml",
			"service": "web"
		}`
		require.NoError(t, os.WriteFile(configPath2, []byte(content2), 0644))

		config1, err := devcontainer.ParseConfig(configPath1)
		require.NoError(t, err)
		config2, err := devcontainer.ParseConfig(configPath2)
		require.NoError(t, err)

		configs := []*devcontainer.Config{config1, config2}
		deduplicated := deduplicateConfigs(configs)

		// Should have both configs since they have different project names
		assert.Len(t, deduplicated, 2)
	})

	t.Run("handles empty list", func(t *testing.T) {
		configs := []*devcontainer.Config{}
		deduplicated := deduplicateConfigs(configs)
		assert.Len(t, deduplicated, 0)
	})

	t.Run("handles single config", func(t *testing.T) {
		tmpDir := t.TempDir()
		devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
		require.NoError(t, os.MkdirAll(devcontainerDir, 0755))
		configPath := filepath.Join(devcontainerDir, "devcontainer.json")
		content := `{"image": "golang:1.21"}`
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		config, err := devcontainer.ParseConfig(configPath)
		require.NoError(t, err)

		configs := []*devcontainer.Config{config}
		deduplicated := deduplicateConfigs(configs)
		assert.Len(t, deduplicated, 1)
	})
}
