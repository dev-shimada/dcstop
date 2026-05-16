package docker

import "context"

// DiscoveredProject represents a devcontainer project found via Docker labels,
// without requiring devcontainer.json files on disk.
type DiscoveredProject struct {
	Name        string
	ConfigType  string // "image" or "compose"
	LocalFolder string // devcontainer.local_folder label value
	Containers  []ContainerInfo
}

// Running returns the number of running containers in the project.
func (p *DiscoveredProject) Running() int {
	n := 0
	for _, c := range p.Containers {
		if c.State == "running" {
			n++
		}
	}
	return n
}

// DiscoverProjects queries Docker for all devcontainer projects using container labels.
// It does not require devcontainer.json files to be present on disk.
func DiscoverProjects(ctx context.Context, client ContainerClient) ([]DiscoveredProject, error) {
	// Image-based: containers with devcontainer.config_file but no compose project label
	imageBased, err := client.ContainerList(ctx, ContainerListOptions{
		All:         true,
		LabelFilter: "devcontainer.config_file",
	})
	if err != nil {
		return nil, err
	}

	imageProjects := make(map[string]*DiscoveredProject)
	for _, c := range imageBased {
		if _, hasCompose := c.Labels["com.docker.compose.project"]; hasCompose {
			continue
		}
		configPath := c.Labels["devcontainer.config_file"]
		name := DeriveDevcontainerProjectName(configPath)
		if p, ok := imageProjects[name]; ok {
			p.Containers = append(p.Containers, c)
		} else {
			imageProjects[name] = &DiscoveredProject{
				Name:        name,
				ConfigType:  "image",
				LocalFolder: c.Labels["devcontainer.local_folder"],
				Containers:  []ContainerInfo{c},
			}
		}
	}

	// Compose-based: containers with both devcontainer.local_folder and com.docker.compose.project
	composeBased, err := client.ContainerList(ctx, ContainerListOptions{
		All:         true,
		LabelFilter: "devcontainer.local_folder",
	})
	if err != nil {
		return nil, err
	}

	composeProjects := make(map[string]*DiscoveredProject)
	for _, c := range composeBased {
		projectName, hasCompose := c.Labels["com.docker.compose.project"]
		if !hasCompose {
			continue
		}
		if p, ok := composeProjects[projectName]; ok {
			p.Containers = append(p.Containers, c)
		} else {
			composeProjects[projectName] = &DiscoveredProject{
				Name:        projectName,
				ConfigType:  "compose",
				LocalFolder: c.Labels["devcontainer.local_folder"],
				Containers:  []ContainerInfo{c},
			}
		}
	}

	result := make([]DiscoveredProject, 0, len(imageProjects)+len(composeProjects))
	for _, p := range imageProjects {
		result = append(result, *p)
	}
	for _, p := range composeProjects {
		result = append(result, *p)
	}
	return result, nil
}
