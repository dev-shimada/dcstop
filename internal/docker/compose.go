package docker

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// NetworkInfo represents network information.
type NetworkInfo struct {
	ID   string
	Name string
}

// NetworkListOptions represents options for listing networks.
type NetworkListOptions struct {
	LabelFilter string
}

// VolumeInfo represents volume information.
type VolumeInfo struct {
	Name   string
	Labels map[string]string
}

// VolumeListOptions represents options for listing volumes.
type VolumeListOptions struct {
	LabelFilter string
}

// ComposeClient extends ContainerClient with network and volume operations.
type ComposeClient interface {
	ContainerClient
	NetworkList(ctx context.Context, options NetworkListOptions) ([]NetworkInfo, error)
	NetworkRemove(ctx context.Context, networkID string) error
	VolumeList(ctx context.Context, options VolumeListOptions) ([]VolumeInfo, error)
	VolumeRemove(ctx context.Context, volumeName string, force bool) error
}

// ComposeOps provides operations on Docker Compose projects.
type ComposeOps struct {
	client ComposeClient
}

// NewComposeOps creates a new ComposeOps with the given client.
func NewComposeOps(client ComposeClient) *ComposeOps {
	return &ComposeOps{client: client}
}

// FindComposeContainers finds containers belonging to a compose project.
func (c *ComposeOps) FindComposeContainers(ctx context.Context, projectName string) ([]ContainerInfo, error) {
	opts := ContainerListOptions{
		All:         true,
		LabelFilter: fmt.Sprintf("com.docker.compose.project=%s", projectName),
	}

	return c.client.ContainerList(ctx, opts)
}

// StopComposeProject stops all containers in a compose project.
func (c *ComposeOps) StopComposeProject(ctx context.Context, projectName string) error {
	containers, err := c.FindComposeContainers(ctx, projectName)
	if err != nil {
		return fmt.Errorf("failed to find compose containers: %w", err)
	}

	for _, container := range containers {
		if err := c.client.ContainerStop(ctx, container.ID, nil); err != nil {
			return fmt.Errorf("failed to stop container %s: %w", container.ID, err)
		}
	}

	return nil
}

// DownComposeProject stops and removes containers and networks for a compose project.
func (c *ComposeOps) DownComposeProject(ctx context.Context, projectName string, removeVolumes bool) error {
	containers, err := c.FindComposeContainers(ctx, projectName)
	if err != nil {
		return fmt.Errorf("failed to find compose containers: %w", err)
	}

	// Stop containers
	for _, container := range containers {
		if err := c.client.ContainerStop(ctx, container.ID, nil); err != nil {
			return fmt.Errorf("failed to stop container %s: %w", container.ID, err)
		}
	}

	// Remove containers
	for _, container := range containers {
		if err := c.client.ContainerRemove(ctx, container.ID, true); err != nil {
			return fmt.Errorf("failed to remove container %s: %w", container.ID, err)
		}
	}

	// Remove networks
	networks, err := c.client.NetworkList(ctx, NetworkListOptions{
		LabelFilter: fmt.Sprintf("com.docker.compose.project=%s", projectName),
	})
	if err != nil {
		return fmt.Errorf("failed to list networks: %w", err)
	}

	for _, network := range networks {
		if err := c.client.NetworkRemove(ctx, network.ID); err != nil {
			return fmt.Errorf("failed to remove network %s: %w", network.Name, err)
		}
	}

	// Remove volumes if requested
	if removeVolumes {
		volumes, err := c.client.VolumeList(ctx, VolumeListOptions{
			LabelFilter: fmt.Sprintf("com.docker.compose.project=%s", projectName),
		})
		if err != nil {
			return fmt.Errorf("failed to list volumes: %w", err)
		}

		for _, volume := range volumes {
			if err := c.client.VolumeRemove(ctx, volume.Name, true); err != nil {
				return fmt.Errorf("failed to remove volume %s: %w", volume.Name, err)
			}
		}
	}

	return nil
}

// DeriveDevcontainerProjectName derives a compose project name from a devcontainer.json path.
// This follows the devcontainer naming convention.
//
// For standard layout (/foo/bar/.devcontainer/devcontainer.json):
//
//	project name will be "bar_devcontainer"
//
// For multi-config layout (/foo/bar/.devcontainer/app1/devcontainer.json):
//
//	project name will be "app1"
func DeriveDevcontainerProjectName(configPath string) string {
	configDir := filepath.Dir(configPath)
	configDirName := filepath.Base(configDir)

	if configDirName == ".devcontainer" {
		parentDir := filepath.Dir(configDir)
		name := filepath.Base(parentDir)
		return strings.ToLower(name) + "_devcontainer"
	}

	return strings.ToLower(configDirName)
}

// DeriveProjectNameFromComposeFile derives a compose project name from a docker-compose file path.
// This determines the project name based on where the compose file is located.
//
// For compose file in .devcontainer directory (/foo/bar/.devcontainer/docker-compose.yml):
//
//	project name will be "bar_devcontainer"
//
// For compose file in .devcontainer subdirectory (/foo/bar/.devcontainer/app1/docker-compose.yml):
//
//	project name will be "app1"
//
// For compose file outside .devcontainer (/foo/bar/docker-compose.yml or /foo/bar/docker/docker-compose.yml):
//
//	project name will be the directory name containing the compose file ("bar" or "docker")
func DeriveProjectNameFromComposeFile(composeFilePath string) string {
	composeDir := filepath.Dir(composeFilePath)
	composeDirName := filepath.Base(composeDir)

	pathParts := strings.Split(filepath.Clean(composeFilePath), string(filepath.Separator))

	devcontainerIndex := -1
	for i, part := range pathParts {
		if part == ".devcontainer" {
			devcontainerIndex = i
			break
		}
	}

	if devcontainerIndex == -1 {
		return strings.ToLower(composeDirName)
	}

	if devcontainerIndex == len(pathParts)-2 {
		if devcontainerIndex > 0 {
			parentDirName := pathParts[devcontainerIndex-1]
			return strings.ToLower(parentDirName) + "_devcontainer"
		}
	}

	if devcontainerIndex < len(pathParts)-2 {
		subdirName := pathParts[devcontainerIndex+1]
		return strings.ToLower(subdirName)
	}

	return strings.ToLower(composeDirName)
}

// DeriveProjectNameFromComposeFiles derives a compose project name from a list of compose files.
// It uses the first file in the list to determine the project name.
// Returns an empty string if the list is empty or nil.
func DeriveProjectNameFromComposeFiles(composeFiles []string) string {
	if len(composeFiles) == 0 {
		return ""
	}
	return DeriveProjectNameFromComposeFile(composeFiles[0])
}

// Config interface represents the minimal interface needed from devcontainer.Config.
// This interface avoids a direct import dependency on the devcontainer package.
type Config interface {
	IsComposeBased() bool
	GetComposeFiles() []string
	GetConfigPath() string
}

// DeriveProjectNameFromConfig derives a compose project name from a devcontainer config.
// For compose-based configs, it uses the location of the docker-compose file.
// For image-based configs, it falls back to the devcontainer.json path.
func DeriveProjectNameFromConfig(cfg Config) string {
	if cfg.IsComposeBased() {
		composeFiles := cfg.GetComposeFiles()
		if len(composeFiles) > 0 {
			composeFile := composeFiles[0]

			// Special case: config is in .devcontainer/<sub>/ and compose file is
			// in .devcontainer/ itself (e.g. ../docker-compose.yml from the subdir).
			// Docker Compose names the project from the compose file's directory,
			// stripping leading dots: ".devcontainer" → "devcontainer".
			configDir := filepath.Dir(cfg.GetConfigPath())
			composeDir := filepath.Dir(composeFile)
			if filepath.Base(filepath.Dir(configDir)) == ".devcontainer" &&
				filepath.Base(composeDir) == ".devcontainer" {
				return strings.TrimLeft(filepath.Base(composeDir), ".")
			}

			return DeriveProjectNameFromComposeFile(composeFile)
		}
	}
	// Fallback to devcontainer.json path for image-based or when compose files are not found
	return DeriveDevcontainerProjectName(cfg.GetConfigPath())
}
