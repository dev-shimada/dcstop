package ui

import (
	"fmt"

	"github.com/dev-shimada/dcstop/internal/devcontainer"
	"github.com/dev-shimada/dcstop/internal/docker"
	"github.com/manifoldco/promptui"
)

// deduplicateConfigs removes configs with duplicate project names,
// keeping only the first occurrence of each unique project name.
func deduplicateConfigs(configs []*devcontainer.Config) []*devcontainer.Config {
	if len(configs) <= 1 {
		return configs
	}

	seen := make(map[string]bool)
	result := make([]*devcontainer.Config, 0, len(configs))

	for _, cfg := range configs {
		projectName := docker.DeriveProjectNameFromConfig(cfg)
		if !seen[projectName] {
			seen[projectName] = true
			result = append(result, cfg)
		}
	}

	return result
}

// SelectConfig prompts the user to select a devcontainer config when multiple are found.
func SelectConfig(configs []*devcontainer.Config) (*devcontainer.Config, error) {
	if len(configs) == 0 {
		return nil, fmt.Errorf("no configs to select from")
	}

	// Deduplicate configs by project name
	uniqueConfigs := deduplicateConfigs(configs)

	if len(uniqueConfigs) == 1 {
		return uniqueConfigs[0], nil
	}

	// Build display items
	items := make([]string, len(uniqueConfigs))
	for i, cfg := range uniqueConfigs {
		configType := "image"
		if cfg.IsComposeBased() {
			configType = "compose"
		}

		projectName := docker.DeriveProjectNameFromConfig(cfg)

		items[i] = fmt.Sprintf("%s (%s)", projectName, configType)
	}

	prompt := promptui.Select{
		Label: "Select devcontainer",
		Items: items,
		Size:  10,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		return nil, fmt.Errorf("selection cancelled: %w", err)
	}

	return uniqueConfigs[idx], nil
}

// Confirm prompts the user for confirmation.
func Confirm(message string) (bool, error) {
	prompt := promptui.Prompt{
		Label:     message,
		IsConfirm: true,
	}

	_, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrAbort {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
