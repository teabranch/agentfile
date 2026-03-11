// Package runtimecfg abstracts MCP server config generation for multiple
// AI coding runtimes (Claude Code, Codex, Gemini CLI).
package runtimecfg

import (
	"fmt"
	"os"
	"path/filepath"
)

// Runtime identifies an AI coding runtime that supports MCP servers.
type Runtime string

const (
	ClaudeCode Runtime = "claude-code"
	Codex      Runtime = "codex"
	Gemini     Runtime = "gemini"
)

// AllRuntimes returns all supported runtimes in deterministic order.
func AllRuntimes() []Runtime {
	return []Runtime{ClaudeCode, Codex, Gemini}
}

// ServerEntry describes an MCP server to register with a runtime.
type ServerEntry struct {
	Command string
	Args    []string
}

// ConfigWriter handles reading/writing MCP server entries for a specific runtime.
type ConfigWriter interface {
	// Runtime returns which runtime this writer targets.
	Runtime() Runtime

	// LocalPath returns the project-local config path (e.g. ".mcp.json").
	LocalPath() string

	// GlobalPath returns the user-global config path (e.g. "~/.claude/mcp.json").
	GlobalPath() (string, error)

	// Merge reads existing config at path, adds/overwrites the given entries,
	// and writes back. Creates the file and parent directories if needed.
	Merge(path string, entries map[string]ServerEntry) error

	// Remove deletes a single server entry from config at path.
	Remove(path string, name string) error
}

// Parse converts a string to a Runtime, returning an error for unknown values.
func Parse(s string) (Runtime, error) {
	switch s {
	case string(ClaudeCode):
		return ClaudeCode, nil
	case string(Codex):
		return Codex, nil
	case string(Gemini):
		return Gemini, nil
	default:
		return "", fmt.Errorf("unknown runtime %q (supported: claude-code, codex, gemini)", s)
	}
}

// For returns the ConfigWriter for a specific runtime.
func For(r Runtime) ConfigWriter {
	switch r {
	case ClaudeCode:
		return &claudeWriter{}
	case Codex:
		return &codexWriter{}
	case Gemini:
		return &geminiWriter{}
	default:
		return &claudeWriter{} // fallback
	}
}

// Detect returns ConfigWriters for all runtimes whose global config directories
// exist on the current system. Falls back to Claude Code if none are detected.
func Detect() []ConfigWriter {
	var writers []ConfigWriter
	for _, r := range AllRuntimes() {
		w := For(r)
		gp, err := w.GlobalPath()
		if err != nil {
			continue
		}
		// Check if the parent directory of the global config exists.
		dir := filepath.Dir(gp)
		if _, err := os.Stat(dir); err == nil {
			writers = append(writers, w)
		}
	}
	if len(writers) == 0 {
		writers = append(writers, For(ClaudeCode))
	}
	return writers
}

// Resolve returns the list of ConfigWriters for a given --runtime flag value.
// Supported values: "auto", "all", or a specific runtime name.
func Resolve(flag string) ([]ConfigWriter, error) {
	switch flag {
	case "auto":
		return Detect(), nil
	case "all":
		var writers []ConfigWriter
		for _, r := range AllRuntimes() {
			writers = append(writers, For(r))
		}
		return writers, nil
	default:
		r, err := Parse(flag)
		if err != nil {
			return nil, err
		}
		return []ConfigWriter{For(r)}, nil
	}
}
