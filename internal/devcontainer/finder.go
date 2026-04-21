package devcontainer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// FindDevcontainerConfigs searches for devcontainer.json files in the given directory.
// It recursively walks .devcontainer/ and returns every devcontainer.json found at any depth:
// - .devcontainer/devcontainer.json
// - .devcontainer/*/devcontainer.json
// - .devcontainer/**/devcontainer.json (any depth)
func FindDevcontainerConfigs(dir string) ([]string, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist: %s", dir)
	}

	var configs []string

	devcontainerDir := filepath.Join(dir, ".devcontainer")
	if _, err := os.Stat(devcontainerDir); os.IsNotExist(err) {
		return configs, nil
	}

	err := filepath.WalkDir(devcontainerDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Base(path) == "devcontainer.json" {
			configs = append(configs, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk .devcontainer directory: %w", err)
	}

	return configs, nil
}
