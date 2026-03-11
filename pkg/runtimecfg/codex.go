package runtimecfg

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// codexWriter implements ConfigWriter for Codex CLI.
// Config format: TOML with [mcp_servers.<name>] sections.
type codexWriter struct{}

// codexConfig represents the subset of Codex config we need to read/write.
// We preserve unknown fields by using a generic map for decode/encode.
type codexServerEntry struct {
	Command string   `toml:"command"`
	Args    []string `toml:"args"`
}

func (c *codexWriter) Runtime() Runtime { return Codex }

func (c *codexWriter) LocalPath() string {
	return filepath.Join(".codex", "config.toml")
}

func (c *codexWriter) GlobalPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", "config.toml"), nil
}

func (c *codexWriter) Merge(path string, entries map[string]ServerEntry) error {
	cfg := make(map[string]any)

	data, err := os.ReadFile(path)
	if err == nil {
		_ = toml.Unmarshal(data, &cfg)
	}

	// Get or create the mcp_servers map.
	servers, _ := cfg["mcp_servers"].(map[string]any)
	if servers == nil {
		servers = make(map[string]any)
	}

	for k, v := range entries {
		servers[k] = codexServerEntry{
			Command: v.Command,
			Args:    v.Args,
		}
	}
	cfg["mcp_servers"] = servers

	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := toml.NewEncoder(f)
	return enc.Encode(cfg)
}

func (c *codexWriter) Remove(path string, name string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	cfg := make(map[string]any)
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return err
	}

	servers, ok := cfg["mcp_servers"].(map[string]any)
	if !ok {
		return nil
	}

	delete(servers, name)
	cfg["mcp_servers"] = servers

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := toml.NewEncoder(f)
	return enc.Encode(cfg)
}
