# Development

Guide for contributing to the Agentfile framework itself.

## Prerequisites

- Go 1.24+
- Make

## Build Commands

```bash
make all          # fmtcheck → vet → test → build
make build        # build the agentfile CLI → build/agentfile
make agents       # build agent binaries from Agentfile
make integration  # end-to-end tests against built binary
make install      # install agentfile CLI to /usr/local/bin
make clean        # remove build artifacts
```

## Testing

```bash
# Unit tests (with race detector)
make test

# Integration tests (builds CLI + test agent, exercises all subcommands)
make integration

# Manual end-to-end
make build && ./build/agentfile build
./build/go-pro validate
./build/go-pro --describe
```

## Project Structure

```
Agentfile           Manifest declaring agents to build (also accepts agentfile.yaml)
.claude/agents/     Agent .md files (prompt + frontmatter)
build/              Compiled binaries (agentfile CLI + agents)

pkg/agent/          Core runtime: New(), Execute(), functional options
pkg/builtins/       Shared tool implementations (read, write, edit, bash, glob, grep)
pkg/definition/     Agentfile YAML + agent .md parser
pkg/builder/        Code generation + go build compilation
pkg/tools/          Tool registry, executor, validation
pkg/memory/         File-based KV store, limits, concurrency-safe manager
pkg/prompt/         Embed.FS loader with override support
pkg/mcp/            MCP-over-stdio bridge
pkg/plugin/         Claude Code plugin directory generation (--plugin)
pkg/registry/       Installed agents tracking (~/.agentfile/registry.json)
pkg/github/         GitHub Releases client for remote install/update
internal/cli/       Cobra commands: root, run-tool, memory, serve-mcp, validate
cmd/agentfile/      CLI: build, install, publish, list, update, uninstall
```

## CLI Reference

```bash
# Build
agentfile build                   # build all agents (auto-finds Agentfile or agentfile.yaml)
agentfile build --agent my-agent  # build a single agent
agentfile build -o ./dist         # custom output directory
agentfile build --plugin          # also generate Claude Code plugin directories

# Install
agentfile install my-agent                            # install locally from ./build/
agentfile install -g my-agent                         # install globally (/usr/local/bin/)
agentfile install github.com/owner/repo/agent         # install from GitHub Releases
agentfile install github.com/owner/repo/agent@1.0.0   # specific version

# Publish
agentfile publish                 # cross-compile + create GitHub Release
agentfile publish --agent my-agent
agentfile publish --dry-run       # cross-compile only, no release

# Manage
agentfile list                    # show installed agents
agentfile update                  # update all remote agents
agentfile update my-agent         # update a specific agent
agentfile uninstall my-agent      # remove binary + MCP entry + registry
```

## Built-in Tools

Agents declare tools by Claude Code name in their `.md` frontmatter:

| Declare | MCP tool | Description |
|---------|----------|-------------|
| `Read` | `read_file` | Read file contents |
| `Write` | `write_file` | Write file with dir creation |
| `Edit` | `edit_file` | Find-and-replace in file |
| `Bash` | `run_command` | Shell command execution |
| `Glob` | `glob_files` | File pattern matching |
| `Grep` | `grep_search` | Regex content search |

Memory tools (`memory_read`, `memory_write`, `memory_list`, `memory_delete`) are added automatically when `memory` is set.
