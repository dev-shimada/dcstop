package devcontainer

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindDevcontainerConfigs searches for devcontainer.json files in the given directory.
// It looks for:
// - .devcontainer/devcontainer.json
// - .devcontainer/*/devcontainer.json (subdirectories)
func FindDevcontainerConfigs(dir string) ([]string, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist: %s", dir)
	}

	var configs []string

	devcontainerDir := filepath.Join(dir, ".devcontainer")
	if _, err := os.Stat(devcontainerDir); os.IsNotExist(err) {
		return configs, nil
	}

	// Check .devcontainer/devcontainer.json
	mainConfig := filepath.Join(devcontainerDir, "devcontainer.json")
	if _, err := os.Stat(mainConfig); err == nil {
		configs = append(configs, mainConfig)
	}

	// Check .devcontainer/*/devcontainer.json
	entries, err := os.ReadDir(devcontainerDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read .devcontainer directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		subConfig := filepath.Join(devcontainerDir, entry.Name(), "devcontainer.json")
		if _, err := os.Stat(subConfig); err == nil {
			configs = append(configs, subConfig)
		}
	}

	return configs, nil
}
