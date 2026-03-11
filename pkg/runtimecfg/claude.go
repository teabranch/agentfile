package runtimecfg

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// claudeWriter implements ConfigWriter for Claude Code.
// Config format: JSON with {"mcpServers": {"name": {"command": "...", "args": [...]}}}
type claudeWriter struct{}

// mcpServerJSON is the JSON representation of an MCP server entry.
type mcpServerJSON struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

func (c *claudeWriter) Runtime() Runtime { return ClaudeCode }

func (c *claudeWriter) LocalPath() string { return ".mcp.json" }

func (c *claudeWriter) GlobalPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "mcp.json"), nil
}

func (c *claudeWriter) Merge(path string, entries map[string]ServerEntry) error {
	return mergeJSON(path, entries)
}

func (c *claudeWriter) Remove(path string, name string) error {
	return removeJSON(path, name)
}

// mergeJSON reads an existing MCP JSON config (if present), merges new entries,
// and writes it back. Preserves all existing keys and server entries that are
// not being overwritten. Creates parent directories as needed.
func mergeJSON(path string, entries map[string]ServerEntry) error {
	cfg := make(map[string]any)

	data, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(data, &cfg)
	}

	servers, _ := cfg["mcpServers"].(map[string]any)
	if servers == nil {
		servers = make(map[string]any)
	}

	for k, v := range entries {
		servers[k] = mcpServerJSON{
			Command: v.Command,
			Args:    v.Args,
		}
	}
	cfg["mcpServers"] = servers

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	return os.WriteFile(path, append(out, '\n'), 0o644)
}

// removeJSON removes a single server entry from an MCP JSON config file.
func removeJSON(path, name string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	servers, ok := cfg["mcpServers"].(map[string]any)
	if !ok {
		return nil
	}

	delete(servers, name)
	cfg["mcpServers"] = servers

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o644)
}
