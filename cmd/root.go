package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dev-shimada/dcstop/internal/devcontainer"
	"github.com/dev-shimada/dcstop/internal/docker"
	"github.com/dev-shimada/dcstop/internal/ui"
	"github.com/spf13/cobra"
)

var (
	downFlag    bool
	volumesFlag bool
	contextFlag string
)

var rootCmd = &cobra.Command{
	Use:   "dcstop [directory]",
	Short: "Stop devcontainer created containers",
	Long: `dcstop is a CLI tool to stop containers created by devcontainer.

It searches for devcontainer.json in the specified directory (or current directory)
and stops the associated containers.

For image-based devcontainers, it stops containers by the devcontainer labels.
For compose-based devcontainers, it stops the compose project.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runStop,
}

func init() {
	rootCmd.Flags().BoolVarP(&downFlag, "down", "d", false, "Remove containers after stopping (for compose, also removes networks)")
	rootCmd.Flags().BoolVarP(&volumesFlag, "volumes", "v", false, "Also remove volumes (requires --down)")
	rootCmd.Flags().StringVarP(&contextFlag, "context", "c", "", "Docker context to use (default: current context)")
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runStop(cmd *cobra.Command, args []string) error {
	// Validate flags
	if volumesFlag && !downFlag {
		return fmt.Errorf("--volumes requires --down flag")
	}

	// Determine target directory
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}

	// Convert to absolute path
	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Find devcontainer configs
	configPaths, err := devcontainer.FindDevcontainerConfigs(absDir)
	if err != nil {
		return fmt.Errorf("failed to find devcontainer configs: %w", err)
	}

	if len(configPaths) == 0 {
		fmt.Println("No devcontainer.json found")
		return nil
	}

	// Parse all configs
	configs := make([]*devcontainer.Config, 0, len(configPaths))
	for _, path := range configPaths {
		cfg, err := devcontainer.ParseConfig(path)
		if err != nil {
			fmt.Printf("Warning: failed to parse %s: %v\n", path, err)
			continue
		}
		configs = append(configs, cfg)
	}

	if len(configs) == 0 {
		return fmt.Errorf("no valid devcontainer configs found")
	}

	// Select config if multiple
	selectedConfig, err := ui.SelectConfig(configs)
	if err != nil {
		return err
	}

	// Create Docker client
	dockerClient, err := docker.NewClientWithContext(contextFlag)
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}
	defer func() {
		if closeErr := dockerClient.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close docker client: %v\n", closeErr)
		}
	}()

	ctx := context.Background()

	// Handle based on config type
	if selectedConfig.IsComposeBased() {
		return handleCompose(ctx, dockerClient, selectedConfig)
	}

	return handleImage(ctx, dockerClient, selectedConfig)
}

func handleImage(ctx context.Context, client *docker.RealDockerClient, cfg *devcontainer.Config) error {
	ops := docker.NewContainerOps(client)

	// Find containers by config path
	containers, err := ops.FindDevcontainersByConfigPath(ctx, cfg.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to find containers: %w", err)
	}

	if len(containers) == 0 {
		fmt.Println("No running containers found for this devcontainer")
		return nil
	}

	fmt.Printf("Found %d container(s) to stop\n", len(containers))
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = c.Names[0]
		}
		fmt.Printf("  - %s (%s)\n", name, c.ID[:12])
	}

	// Stop containers
	if err := ops.StopContainers(ctx, containers); err != nil {
		return err
	}

	if downFlag {
		// Remove containers
		if err := ops.RemoveContainers(ctx, containers); err != nil {
			return err
		}
		fmt.Println("Containers stopped and removed successfully")
	} else {
		fmt.Println("Containers stopped successfully")
	}
	return nil
}

func handleCompose(ctx context.Context, client *docker.RealDockerClient, cfg *devcontainer.Config) error {
	ops := docker.NewComposeOps(client)

	// Derive project name from devcontainer config
	projectName := docker.DeriveProjectNameFromConfig(cfg)

	// Find containers
	containers, err := ops.FindComposeContainers(ctx, projectName)
	if err != nil {
		return fmt.Errorf("failed to find compose containers: %w", err)
	}

	if len(containers) == 0 {
		if !downFlag {
			fmt.Printf("No containers found for compose project '%s'\n", projectName)
			return nil
		}
		fmt.Printf("No containers found for compose project '%s', cleaning up resources...\n", projectName)
	} else {
		fmt.Printf("Found %d container(s) in compose project '%s'\n", len(containers), projectName)
		for _, c := range containers {
			name := ""
			if len(c.Names) > 0 {
				name = c.Names[0]
			}
			fmt.Printf("  - %s (%s)\n", name, c.ID[:12])
		}
	}

	if downFlag {
		// Stop and remove containers, networks, and optionally volumes
		if err := ops.DownComposeProject(ctx, projectName, volumesFlag); err != nil {
			return err
		}
		if volumesFlag {
			fmt.Println("Compose project stopped and removed (including volumes) successfully")
		} else {
			fmt.Println("Compose project stopped and removed successfully")
		}
	} else {
		// Just stop containers
		if err := ops.StopComposeProject(ctx, projectName); err != nil {
			return err
		}
		fmt.Println("Compose project stopped successfully")
	}

	return nil
}
