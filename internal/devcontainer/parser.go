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
// It processes the input character-by-character to correctly handle string
// literals, including literal newlines inside strings (common in JSONC files).
func stripJSONC(content string) string {
	var result strings.Builder
	i := 0
	for i < len(content) {
		ch := content[i]

		switch {
		case ch == '"':
			// Copy string literal, escaping any literal newlines/carriage-returns.
			result.WriteByte(ch)
			i++
			for i < len(content) {
				c := content[i]
				if c == '\\' {
					// Escape sequence: copy both characters verbatim.
					result.WriteByte(c)
					i++
					if i < len(content) {
						result.WriteByte(content[i])
						i++
					}
				} else if c == '"' {
					result.WriteByte(c)
					i++
					break
				} else if c == '\n' {
					// Literal newline inside string — replace with \n escape.
					result.WriteString(`\n`)
					i++
				} else if c == '\r' {
					// Literal CR inside string — skip (paired \r\n already handled).
					i++
				} else {
					result.WriteByte(c)
					i++
				}
			}

		case ch == '/' && i+1 < len(content) && content[i+1] == '/':
			// Single-line comment: skip until end of line.
			for i < len(content) && content[i] != '\n' {
				i++
			}

		case ch == '/' && i+1 < len(content) && content[i+1] == '*':
			// Multi-line comment: skip until closing */.
			i += 2
			for i < len(content)-1 {
				if content[i] == '*' && content[i+1] == '/' {
					i += 2
					break
				}
				i++
			}

		default:
			result.WriteByte(ch)
			i++
		}
	}

	// Remove trailing commas before } or ]
	cleaned := result.String()
	trailingComma := regexp.MustCompile(`,(\s*[}\]])`)
	cleaned = trailingComma.ReplaceAllString(cleaned, "$1")

	return strings.TrimSpace(cleaned)
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

// GetConfigPath returns the path to the devcontainer.json file.
func (c *Config) GetConfigPath() string {
	return c.ConfigPath
}
