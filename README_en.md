English | [日本語](README.md)

[![Go Report Card](https://goreportcard.com/badge/github.com/dev-shimada/dcstop)](https://goreportcard.com/report/github.com/dev-shimada/dcstop)
[![CI](https://github.com/dev-shimada/dcstop/actions/workflows/CI.yaml/badge.svg)](https://github.com/dev-shimada/dcstop/actions/workflows/CI.yaml)
[![License](https://img.shields.io/badge/license-MIT-blue)](https://github.com/dev-shimada/dcstop/blob/main/LICENSE)

<img src="logo.svg" alt="dcstop" />

# dcstop

A CLI tool to stop containers created by devcontainer.

## Overview

Containers created by the VS Code Dev Containers extension sometimes keep running even after "Reopen Locally". `dcstop` identifies and stops those containers.

- **Image-based**: identifies containers by the `devcontainer.config_file` label
- **Compose-based**: identifies projects by the `com.docker.compose.project` label
- **Docker SDK for Go**: communicates with Docker natively without shelling out

## Installation

### Homebrew
```bash
brew install --casks dev-shimada/dcstop/dcstop
```

### Go install

```bash
go install github.com/dev-shimada/dcstop@latest
```

### Download from releases

Download the binary for your platform from [Releases](https://github.com/dev-shimada/dcstop/releases).

### Build from source

```bash
git clone https://github.com/dev-shimada/dcstop.git
cd dcstop
go build -o dcstop .
```

## Shell completion
```bash
dcstop completion bash
dcstop completion zsh
dcstop completion fish
```

## Usage

```bash
# Stop the devcontainer in the current directory
dcstop

# Stop the devcontainer in a specific directory
dcstop /path/to/project

# Stop and remove containers (compose: also removes networks)
dcstop --down
dcstop -d /path/to/project

# Also remove volumes (requires --down)
dcstop --down --volumes
dcstop -dv /path/to/project

# Use a specific Docker context
dcstop --context my-remote-docker
dcstop -c desktop-linux /path/to/project

# List all projects interactively and stop one
dcstop global
```

### global subcommand

Discovers all devcontainer projects by inspecting Docker container labels directly, and lets you interactively select one to stop. Does not require `devcontainer.json` files to be present on disk.

```
  ██████╗   ██████╗ ███████╗ ████████╗  ██████╗  ██████╗
  ██╔══██╗ ██╔════╝ ██╔════╝ ╚══██╔══╝ ██╔═══██╗ ██╔══██╗
  ██║  ██║ ██║      ███████╗    ██║    ██║   ██║ ██████╔╝
  ██║  ██║ ██║      ╚════██║    ██║    ██║   ██║ ██╔═══╝
  ██████╔╝ ╚██████╗ ███████║    ██║    ╚██████╔╝ ██║
  ╚═════╝   ╚═════╝ ╚══════╝    ╚═╝     ╚═════╝  ╚═╝

  Select devcontainer project to stop

  ▸ ● workspace_devcontainer               image     1 running / 1 total
    ● workspace-node_devcontainer          compose   2 running / 2 total
    ○ workspace-python_devcontainer        image     0 running / 1 total

  /  filter   ↑/↓  move   enter  select   q  quit
```

- `●` (green) = has running containers, `○` (grey) = all stopped
- Type `/` to filter projects by name
- A spinner is shown while stopping; `✓` or `✗` indicates the result

| Key | Action |
|-----|--------|
| `↑` / `↓` or `Ctrl+P` / `Ctrl+N` | Move cursor |
| `/` | Filter by name |
| `Enter` | Select and stop |
| `q` / `Ctrl+C` | Cancel |

The `--down` and `--volumes` flags are also available with this subcommand.

### Options

| Flag | Short | Description |
|------|-------|-------------|
| `--context` | `-c` | Docker context to use |
| `--down` | `-d` | Remove containers after stopping (compose: also removes networks) |
| `--volumes` | `-v` | Also remove volumes (requires `--down`) |
| `--help` | `-h` | Show help |

### Docker Context

Use `--context` to specify a Docker context. If omitted, the context is resolved in the following order:

1. `--context` flag
2. `DOCKER_CONTEXT` environment variable
3. `currentContext` in `~/.docker/config.json` (set via `docker context use`)
4. `DOCKER_HOST` environment variable
5. Default Docker socket

## Development

### Requirements

- Go 1.21+
- Docker

### Test

```bash
go test ./... -v
```

### Build

```bash
go build -o dcstop .
```

## Release

Releases are managed with [GoReleaser](https://goreleaser.com/).

### Local snapshot build

```bash
# Install GoReleaser if needed
go install github.com/goreleaser/goreleaser/v2@latest

# Snapshot build (build only, no release)
goreleaser build --snapshot --clean

# Or generate all artifacts
goreleaser release --snapshot --clean
```

### Publishing a release

Pushing a tag triggers an automatic release via GitHub Actions.

```bash
git tag v0.1.0
git push origin v0.1.0
```

To release manually:

```bash
export GITHUB_TOKEN="your_github_token"
goreleaser release --clean
```

## License

MIT
