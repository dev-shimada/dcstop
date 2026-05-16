package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/dev-shimada/dcstop/internal/docker"
	"github.com/dev-shimada/dcstop/internal/ui"
	"github.com/spf13/cobra"
)

var (
	globalDownFlag    bool
	globalVolumesFlag bool
)

var globalCmd = &cobra.Command{
	Use:   "global",
	Short: "List all devcontainer projects and optionally stop one",
	Long: `List all devcontainer projects found in Docker and interactively select one to stop.

Unlike the root command, this does not require devcontainer.json files on disk.
It discovers projects by inspecting Docker container labels directly.`,
	Args: cobra.NoArgs,
	RunE: runGlobal,
}

func init() {
	globalCmd.Flags().StringVarP(&contextFlag, "context", "c", "", "Docker context to use (default: current context)")
	globalCmd.Flags().BoolVarP(&globalDownFlag, "down", "d", false, "Remove containers after stopping (for compose, also removes networks)")
	globalCmd.Flags().BoolVarP(&globalVolumesFlag, "volumes", "v", false, "Also remove volumes (requires --down)")
	rootCmd.AddCommand(globalCmd)
}

func runGlobal(cmd *cobra.Command, args []string) error {
	ui.PrintLogo()

	if globalVolumesFlag && !globalDownFlag {
		return fmt.Errorf("--volumes requires --down flag")
	}

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

	projects, err := docker.DiscoverProjects(ctx, dockerClient)
	if err != nil {
		return fmt.Errorf("failed to discover projects: %w", err)
	}

	if len(projects) == 0 {
		fmt.Println("No devcontainer projects found")
		return nil
	}

	items := make([]ui.ProjectItem, len(projects))
	for i, p := range projects {
		items[i] = ui.ProjectItem{
			Name:       p.Name,
			ConfigType: p.ConfigType,
			Running:    p.Running(),
			Total:      len(p.Containers),
		}
	}

	selectedIdx, err := ui.SelectProject("Select devcontainer project to stop", items)
	if err != nil {
		return err
	}

	selected := projects[selectedIdx]

	if selected.ConfigType == "compose" {
		return stopComposeByName(ctx, dockerClient, selected.Name)
	}
	return stopImageContainers(ctx, dockerClient, selected.Containers)
}

func stopComposeByName(ctx context.Context, client *docker.RealDockerClient, projectName string) error {
	ops := docker.NewComposeOps(client)

	fmt.Printf("Compose project: %s\n", projectName)

	if globalDownFlag {
		label := "Stopping and removing compose project..."
		if globalVolumesFlag {
			label = "Stopping and removing compose project (including volumes)..."
		}
		return ui.RunWithSpinner(label, func() error {
			return ops.DownComposeProject(ctx, projectName, globalVolumesFlag)
		})
	}
	return ui.RunWithSpinner("Stopping compose project...", func() error {
		return ops.StopComposeProject(ctx, projectName)
	})
}

func stopImageContainers(ctx context.Context, client *docker.RealDockerClient, containers []docker.ContainerInfo) error {
	ops := docker.NewContainerOps(client)

	fmt.Printf("Found %d container(s) to stop\n", len(containers))
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = c.Names[0]
		}
		fmt.Printf("  - %s (%s)\n", name, c.ID[:12])
	}

	if err := ui.RunWithSpinner("Stopping containers...", func() error {
		return ops.StopContainers(ctx, containers)
	}); err != nil {
		return err
	}

	if globalDownFlag {
		return ui.RunWithSpinner("Removing containers...", func() error {
			return ops.RemoveContainers(ctx, containers)
		})
	}
	return nil
}
