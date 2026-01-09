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
	// Get the directory containing devcontainer.json
	configDir := filepath.Dir(configPath)
	// Get the base name of the config directory
	configDirName := filepath.Base(configDir)

	// Check if devcontainer.json is directly in .devcontainer directory
	if configDirName == ".devcontainer" {
		// Standard layout: /foo/bar/.devcontainer/devcontainer.json
		// Use the parent directory name (bar) + "_devcontainer"
		parentDir := filepath.Dir(configDir)
		name := filepath.Base(parentDir)
		return strings.ToLower(name) + "_devcontainer"
	}

	// Multi-config layout: /foo/bar/.devcontainer/app1/devcontainer.json
	// Use the subdirectory name only (app1)
	return strings.ToLower(configDirName)
}
