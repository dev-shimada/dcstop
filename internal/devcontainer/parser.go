package devcontainer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Config represents a parsed devcontainer.json configuration.
type Config struct {
	Image             string   `json:"image"`
	DockerComposeFile []string `json:"-"`
	Service           string   `json:"service"`
	ConfigPath        string   `json:"-"`
}

// rawConfig is used for initial JSON unmarshaling to handle dockerComposeFile
// which can be either a string or an array.
type rawConfig struct {
	Image             string          `json:"image"`
	DockerComposeFile json.RawMessage `json:"dockerComposeFile"`
	Service           string          `json:"service"`
}

// ParseConfig reads and parses a devcontainer.json file.
// It supports JSONC (JSON with comments) and trailing commas.
func ParseConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Strip comments and trailing commas (JSONC support)
	cleaned := stripJSONC(string(data))

	var raw rawConfig
	if err := json.Unmarshal([]byte(cleaned), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	config := &Config{
		Image:      raw.Image,
		Service:    raw.Service,
		ConfigPath: path,
	}

	// Parse dockerComposeFile (can be string or array)
	if len(raw.DockerComposeFile) > 0 {
		config.DockerComposeFile, err = parseDockerComposeFile(raw.DockerComposeFile)
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

// parseDockerComposeFile handles both string and array formats.
func parseDockerComposeFile(data json.RawMessage) ([]string, error) {
	// Try as string first
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		return []string{single}, nil
	}

	// Try as array
	var multiple []string
	if err := json.Unmarshal(data, &multiple); err == nil {
		return multiple, nil
	}

	return nil, fmt.Errorf("dockerComposeFile must be a string or array of strings")
}

// stripJSONC removes comments and trailing commas from JSONC content.
func stripJSONC(content string) string {
	// Remove single-line comments (// ...)
	singleLineComment := regexp.MustCompile(`//[^\n]*`)
	content = singleLineComment.ReplaceAllString(content, "")

	// Remove multi-line comments (/* ... */)
	multiLineComment := regexp.MustCompile(`/\*[\s\S]*?\*/`)
	content = multiLineComment.ReplaceAllString(content, "")

	// Remove trailing commas before } or ]
	trailingComma := regexp.MustCompile(`,(\s*[}\]])`)
	content = trailingComma.ReplaceAllString(content, "$1")

	return strings.TrimSpace(content)
}

// IsImageBased returns true if the config uses an image directly.
func (c *Config) IsImageBased() bool {
	return c.Image != "" && len(c.DockerComposeFile) == 0
}

// IsComposeBased returns true if the config uses docker-compose.
func (c *Config) IsComposeBased() bool {
	return len(c.DockerComposeFile) > 0
}

// GetComposeFiles returns absolute paths of the compose files.
func (c *Config) GetComposeFiles() []string {
	if len(c.DockerComposeFile) == 0 {
		return nil
	}

	configDir := filepath.Dir(c.ConfigPath)
	files := make([]string, len(c.DockerComposeFile))

	for i, f := range c.DockerComposeFile {
		if filepath.IsAbs(f) {
			files[i] = f
		} else {
			files[i] = filepath.Clean(filepath.Join(configDir, f))
		}
	}

	return files
}
