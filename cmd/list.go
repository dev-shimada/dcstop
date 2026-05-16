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
	globalDownFlag    bool
	globalVolumesFlag bool
)

var globalCmd = &cobra.Command{
	Use:   "global [directory]",
	Short: "List all devcontainer projects and optionally stop one",
	Long: `List all devcontainer projects found in the specified directory (or current directory).

Shows project name, type (image or compose), and running container count.
You can then select a project to stop.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runGlobal,
}

func init() {
	globalCmd.Flags().StringVarP(&contextFlag, "context", "c", "", "Docker context to use (default: current context)")
	globalCmd.Flags().BoolVarP(&globalDownFlag, "down", "d", false, "Remove containers after stopping (for compose, also removes networks)")
	globalCmd.Flags().BoolVarP(&globalVolumesFlag, "volumes", "v", false, "Also remove volumes (requires --down)")
	rootCmd.AddCommand(globalCmd)
}

func runGlobal(cmd *cobra.Command, args []string) error {
	if globalVolumesFlag && !globalDownFlag {
		return fmt.Errorf("--volumes requires --down flag")
	}

	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}

	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	configPaths, err := devcontainer.FindDevcontainerConfigs(absDir)
	if err != nil {
		return fmt.Errorf("failed to find devcontainer configs: %w", err)
	}

	if len(configPaths) == 0 {
		fmt.Println("No devcontainer.json found")
		return nil
	}

	configs := make([]*devcontainer.Config, 0, len(configPaths))
	for _, path := range configPaths {
		cfg, err := devcontainer.ParseConfig(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", path, err)
			continue
		}
		configs = append(configs, cfg)
	}

	if len(configs) == 0 {
		return fmt.Errorf("no valid devcontainer configs found")
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

	// Build summary for display in selector
	type projectEntry struct {
		cfg         *devcontainer.Config
		projectName string
		configType  string
		total       int
		running     int
	}

	entries := make([]projectEntry, 0, len(configs))
	for _, cfg := range configs {
		projectName := docker.DeriveProjectNameFromConfig(cfg)
		configType := "image"
		if cfg.IsComposeBased() {
			configType = "compose"
		}
		total, running := countContainers(ctx, dockerClient, cfg, projectName)
		entries = append(entries, projectEntry{cfg, projectName, configType, total, running})
	}

	projects := make([]ui.ProjectItem, len(entries))
	for i, e := range entries {
		projects[i] = ui.ProjectItem{
			Name:       e.projectName,
			ConfigType: e.configType,
			Running:    e.running,
			Total:      e.total,
		}
	}

	selectedIdx, err := ui.SelectProject("Select devcontainer project to stop", projects)
	if err != nil {
		return err
	}

	selected := entries[selectedIdx]

	// Reuse stop logic from root.go, respecting global flags
	savedDown := downFlag
	savedVolumes := volumesFlag
	downFlag = globalDownFlag
	volumesFlag = globalVolumesFlag
	defer func() {
		downFlag = savedDown
		volumesFlag = savedVolumes
	}()

	if selected.cfg.IsComposeBased() {
		return handleCompose(ctx, dockerClient, selected.cfg)
	}
	return handleImage(ctx, dockerClient, selected.cfg)
}

func countContainers(ctx context.Context, client *docker.RealDockerClient, cfg *devcontainer.Config, projectName string) (total, running int) {
	var containers []docker.ContainerInfo
	var err error

	if cfg.IsComposeBased() {
		ops := docker.NewComposeOps(client)
		containers, err = ops.FindComposeContainers(ctx, projectName)
	} else {
		ops := docker.NewContainerOps(client)
		containers, err = ops.FindDevcontainersByConfigPath(ctx, cfg.ConfigPath)
	}

	if err != nil {
		return 0, 0
	}

	for _, c := range containers {
		total++
		if c.State == "running" {
			running++
		}
	}

	return total, running
}
