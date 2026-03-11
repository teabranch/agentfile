package runtimecfg

import "os"
import "path/filepath"

// geminiWriter implements ConfigWriter for Gemini CLI.
// Same JSON schema as Claude Code, different file paths.
type geminiWriter struct{}

func (g *geminiWriter) Runtime() Runtime { return Gemini }

func (g *geminiWriter) LocalPath() string {
	return filepath.Join(".gemini", "settings.json")
}

func (g *geminiWriter) GlobalPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gemini", "settings.json"), nil
}

func (g *geminiWriter) Merge(path string, entries map[string]ServerEntry) error {
	return mergeJSON(path, entries)
}

func (g *geminiWriter) Remove(path string, name string) error {
	return removeJSON(path, name)
}
