package docker

import (
	"context"
	"fmt"
)

// ContainerInfo represents container information.
type ContainerInfo struct {
	ID     string
	Names  []string
	Labels map[string]string
	State  string
}

// ContainerListOptions represents options for listing containers.
type ContainerListOptions struct {
	All         bool
	LabelFilter string
}

// ContainerClient is an interface for Docker container operations.
// This interface allows for easy mocking in tests.
type ContainerClient interface {
	ContainerList(ctx context.Context, options ContainerListOptions) ([]ContainerInfo, error)
	ContainerStop(ctx context.Context, containerID string, timeout *int) error
	ContainerRemove(ctx context.Context, containerID string, force bool) error
	Close() error
}

// ContainerOps provides operations on containers.
type ContainerOps struct {
	client ContainerClient
}

// NewContainerOps creates a new ContainerOps with the given client.
func NewContainerOps(client ContainerClient) *ContainerOps {
	return &ContainerOps{client: client}
}

// FindDevcontainersByFolder finds devcontainers by the local folder path.
func (c *ContainerOps) FindDevcontainersByFolder(ctx context.Context, folderPath string) ([]ContainerInfo, error) {
	opts := ContainerListOptions{
		All:         true,
		LabelFilter: fmt.Sprintf("devcontainer.local_folder=%s", folderPath),
	}

	return c.client.ContainerList(ctx, opts)
}

// FindDevcontainersByConfigPath finds devcontainers by the config file path.
func (c *ContainerOps) FindDevcontainersByConfigPath(ctx context.Context, configPath string) ([]ContainerInfo, error) {
	opts := ContainerListOptions{
		All:         true,
		LabelFilter: fmt.Sprintf("devcontainer.config_file=%s", configPath),
	}

	return c.client.ContainerList(ctx, opts)
}

// StopContainers stops the specified containers.
func (c *ContainerOps) StopContainers(ctx context.Context, containers []ContainerInfo) error {
	for _, container := range containers {
		if err := c.client.ContainerStop(ctx, container.ID, nil); err != nil {
			return fmt.Errorf("failed to stop container %s: %w", container.ID, err)
		}
	}
	return nil
}

// RemoveContainers removes the specified containers.
func (c *ContainerOps) RemoveContainers(ctx context.Context, containers []ContainerInfo) error {
	for _, container := range containers {
		if err := c.client.ContainerRemove(ctx, container.ID, true); err != nil {
			return fmt.Errorf("failed to remove container %s: %w", container.ID, err)
		}
	}
	return nil
}
