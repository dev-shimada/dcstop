package docker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

// RealDockerClient wraps the Docker SDK client to implement ContainerClient interface.
type RealDockerClient struct {
	cli *client.Client
}

// NewClient creates a new Docker client using default context.
func NewClient() (*RealDockerClient, error) {
	return NewClientWithContext("")
}

// NewClientWithContext creates a new Docker client with the specified context.
// If contextName is empty, it uses the current context from Docker config.
func NewClientWithContext(contextName string) (*RealDockerClient, error) {
	opts := []client.Opt{client.WithAPIVersionNegotiation()}

	// Determine which context to use
	resolvedContext := contextName
	if resolvedContext == "" {
		// Check DOCKER_CONTEXT environment variable first
		if envContext := os.Getenv("DOCKER_CONTEXT"); envContext != "" {
			resolvedContext = envContext
		} else if os.Getenv("DOCKER_HOST") == "" {
			// If DOCKER_HOST is not set, check current context from config
			if currentCtx := getCurrentContext(); currentCtx != "" && currentCtx != "default" {
				resolvedContext = currentCtx
			}
		}
	}

	if resolvedContext != "" && resolvedContext != "default" {
		// Use specified context
		host, err := resolveContextEndpoint(resolvedContext)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve context %q: %w", resolvedContext, err)
		}
		opts = append(opts, client.WithHost(host))
	} else {
		// Use default behavior (DOCKER_HOST or default socket)
		opts = append(opts, client.FromEnv)
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &RealDockerClient{cli: cli}, nil
}

// getCurrentContext reads the current context from Docker config.json.
func getCurrentContext() string {
	configPath := filepath.Join(getDockerConfigDir(), "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}

	var config dockerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return ""
	}

	return config.CurrentContext
}

// dockerConfig represents Docker's config.json.
type dockerConfig struct {
	CurrentContext string `json:"currentContext"`
}

// resolveContextEndpoint resolves a Docker context name to its endpoint.
func resolveContextEndpoint(contextName string) (string, error) {
	// Special case: "default" context uses the default Docker socket
	if contextName == "default" {
		return client.DefaultDockerHost, nil
	}

	dockerConfigDir := getDockerConfigDir()

	// Context ID is SHA256 of the context name
	hash := sha256.Sum256([]byte(contextName))
	contextID := hex.EncodeToString(hash[:])

	metaPath := filepath.Join(dockerConfigDir, "contexts", "meta", contextID, "meta.json")

	data, err := os.ReadFile(metaPath)
	if err != nil {
		return "", fmt.Errorf("context %q not found: %w", contextName, err)
	}

	var meta contextMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return "", fmt.Errorf("failed to parse context metadata: %w", err)
	}

	endpoint, ok := meta.Endpoints["docker"]
	if !ok {
		return "", fmt.Errorf("no docker endpoint found in context %q", contextName)
	}

	if endpoint.Host == "" {
		return "", fmt.Errorf("empty host in context %q", contextName)
	}

	return endpoint.Host, nil
}

// contextMeta represents Docker context metadata.
type contextMeta struct {
	Name      string                    `json:"Name"`
	Endpoints map[string]contextEndpoint `json:"Endpoints"`
}

// contextEndpoint represents a Docker context endpoint.
type contextEndpoint struct {
	Host string `json:"Host"`
}

// getDockerConfigDir returns the Docker config directory.
func getDockerConfigDir() string {
	if dir := os.Getenv("DOCKER_CONFIG"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".docker")
	}
	return filepath.Join(home, ".docker")
}

// ContainerList lists containers matching the given options.
func (c *RealDockerClient) ContainerList(ctx context.Context, options ContainerListOptions) ([]ContainerInfo, error) {
	filterArgs := filters.NewArgs()
	if options.LabelFilter != "" {
		filterArgs.Add("label", options.LabelFilter)
	}

	containers, err := c.cli.ContainerList(ctx, container.ListOptions{
		All:     options.All,
		Filters: filterArgs,
	})
	if err != nil {
		return nil, err
	}

	result := make([]ContainerInfo, len(containers))
	for i, cont := range containers {
		result[i] = ContainerInfo{
			ID:     cont.ID,
			Names:  cont.Names,
			Labels: cont.Labels,
			State:  cont.State,
		}
	}

	return result, nil
}

// ContainerStop stops a container.
func (c *RealDockerClient) ContainerStop(ctx context.Context, containerID string, timeout *int) error {
	opts := container.StopOptions{}
	if timeout != nil {
		opts.Timeout = timeout
	}
	return c.cli.ContainerStop(ctx, containerID, opts)
}

// ContainerRemove removes a container.
func (c *RealDockerClient) ContainerRemove(ctx context.Context, containerID string, force bool) error {
	return c.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force: force,
	})
}

// Close closes the Docker client.
func (c *RealDockerClient) Close() error {
	return c.cli.Close()
}

// NetworkList lists networks matching the given options.
func (c *RealDockerClient) NetworkList(ctx context.Context, options NetworkListOptions) ([]NetworkInfo, error) {
	filterArgs := filters.NewArgs()
	if options.LabelFilter != "" {
		filterArgs.Add("label", options.LabelFilter)
	}

	networks, err := c.cli.NetworkList(ctx, network.ListOptions{
		Filters: filterArgs,
	})
	if err != nil {
		return nil, err
	}

	result := make([]NetworkInfo, len(networks))
	for i, net := range networks {
		result[i] = NetworkInfo{
			ID:   net.ID,
			Name: net.Name,
		}
	}

	return result, nil
}

// NetworkRemove removes a network.
func (c *RealDockerClient) NetworkRemove(ctx context.Context, networkID string) error {
	return c.cli.NetworkRemove(ctx, networkID)
}

// VolumeList lists volumes matching the given options.
func (c *RealDockerClient) VolumeList(ctx context.Context, options VolumeListOptions) ([]VolumeInfo, error) {
	filterArgs := filters.NewArgs()
	if options.LabelFilter != "" {
		filterArgs.Add("label", options.LabelFilter)
	}

	resp, err := c.cli.VolumeList(ctx, volume.ListOptions{
		Filters: filterArgs,
	})
	if err != nil {
		return nil, err
	}

	result := make([]VolumeInfo, len(resp.Volumes))
	for i, vol := range resp.Volumes {
		result[i] = VolumeInfo{
			Name:   vol.Name,
			Labels: vol.Labels,
		}
	}

	return result, nil
}

// VolumeRemove removes a volume.
func (c *RealDockerClient) VolumeRemove(ctx context.Context, volumeName string, force bool) error {
	return c.cli.VolumeRemove(ctx, volumeName, force)
}
