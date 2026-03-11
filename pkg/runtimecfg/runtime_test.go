package runtimecfg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input string
		want  Runtime
		err   bool
	}{
		{"claude-code", ClaudeCode, false},
		{"codex", Codex, false},
		{"gemini", Gemini, false},
		{"unknown", "", true},
	}
	for _, tt := range tests {
		got, err := Parse(tt.input)
		if tt.err && err == nil {
			t.Errorf("Parse(%q) expected error", tt.input)
		}
		if !tt.err && err != nil {
			t.Errorf("Parse(%q) unexpected error: %v", tt.input, err)
		}
		if got != tt.want {
			t.Errorf("Parse(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResolve(t *testing.T) {
	// "all" should return 3 writers.
	writers, err := Resolve("all")
	if err != nil {
		t.Fatal(err)
	}
	if len(writers) != 3 {
		t.Errorf("Resolve(all) returned %d writers, want 3", len(writers))
	}

	// Specific runtime.
	writers, err = Resolve("codex")
	if err != nil {
		t.Fatal(err)
	}
	if len(writers) != 1 || writers[0].Runtime() != Codex {
		t.Errorf("Resolve(codex) unexpected result")
	}

	// Invalid.
	_, err = Resolve("nope")
	if err == nil {
		t.Error("Resolve(nope) expected error")
	}
}

func TestFor(t *testing.T) {
	for _, r := range AllRuntimes() {
		w := For(r)
		if w.Runtime() != r {
			t.Errorf("For(%s).Runtime() = %s", r, w.Runtime())
		}
	}
}

func TestClaudeWriterMergeAndRemove(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mcp.json")
	w := For(ClaudeCode)

	// Merge.
	entries := map[string]ServerEntry{
		"my-agent": {Command: "/usr/local/bin/my-agent", Args: []string{"serve-mcp"}},
	}
	if err := w.Merge(path, entries); err != nil {
		t.Fatal(err)
	}

	// Verify JSON structure.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	servers := cfg["mcpServers"].(map[string]any)
	agent := servers["my-agent"].(map[string]any)
	if agent["command"] != "/usr/local/bin/my-agent" {
		t.Errorf("command = %v", agent["command"])
	}

	// Merge a second entry — first should still exist.
	entries2 := map[string]ServerEntry{
		"other": {Command: "/bin/other", Args: []string{"serve-mcp"}},
	}
	if err := w.Merge(path, entries2); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(path)
	json.Unmarshal(data, &cfg)
	servers = cfg["mcpServers"].(map[string]any)
	if _, ok := servers["my-agent"]; !ok {
		t.Error("my-agent was lost after second merge")
	}
	if _, ok := servers["other"]; !ok {
		t.Error("other was not added")
	}

	// Remove.
	if err := w.Remove(path, "my-agent"); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(path)
	json.Unmarshal(data, &cfg)
	servers = cfg["mcpServers"].(map[string]any)
	if _, ok := servers["my-agent"]; ok {
		t.Error("my-agent still present after remove")
	}
	if _, ok := servers["other"]; !ok {
		t.Error("other was removed unexpectedly")
	}
}

func TestGeminiWriterMergeAndRemove(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gemini", "settings.json")
	w := For(Gemini)

	entries := map[string]ServerEntry{
		"my-agent": {Command: "/bin/my-agent", Args: []string{"serve-mcp"}},
	}
	if err := w.Merge(path, entries); err != nil {
		t.Fatal(err)
	}

	// Verify.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	json.Unmarshal(data, &cfg)
	servers := cfg["mcpServers"].(map[string]any)
	if _, ok := servers["my-agent"]; !ok {
		t.Error("my-agent not found")
	}

	// Remove.
	if err := w.Remove(path, "my-agent"); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(path)
	json.Unmarshal(data, &cfg)
	servers = cfg["mcpServers"].(map[string]any)
	if _, ok := servers["my-agent"]; ok {
		t.Error("my-agent still present after remove")
	}
}

func TestCodexWriterMergeAndRemove(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".codex", "config.toml")
	w := For(Codex)

	entries := map[string]ServerEntry{
		"my-agent": {Command: "/bin/my-agent", Args: []string{"serve-mcp"}},
	}
	if err := w.Merge(path, entries); err != nil {
		t.Fatal(err)
	}

	// Verify file exists and has content.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if len(content) == 0 {
		t.Error("empty config file")
	}

	// Merge a second entry.
	entries2 := map[string]ServerEntry{
		"other": {Command: "/bin/other", Args: []string{"serve-mcp"}},
	}
	if err := w.Merge(path, entries2); err != nil {
		t.Fatal(err)
	}

	// Remove first entry.
	if err := w.Remove(path, "my-agent"); err != nil {
		t.Fatal(err)
	}

	// Remove from non-existent file should not error.
	if err := w.Remove(filepath.Join(dir, "nope.toml"), "x"); err != nil {
		t.Errorf("remove from missing file: %v", err)
	}
}

func TestCodexWriterPreservesExistingConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	w := For(Codex)

	// Write an existing config with non-MCP fields.
	existing := `model = "o3"
approval_mode = "full-auto"
`
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	// Merge an MCP server.
	entries := map[string]ServerEntry{
		"test": {Command: "/bin/test", Args: []string{"serve-mcp"}},
	}
	if err := w.Merge(path, entries); err != nil {
		t.Fatal(err)
	}

	// Re-read and verify non-MCP fields are preserved.
	data, _ := os.ReadFile(path)
	content := string(data)

	// The TOML encoder should preserve the model and approval_mode fields.
	// We can't check exact format but can re-parse and verify.
	cfg := make(map[string]any)
	if _, err := tomlDecode(content, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg["model"] != "o3" {
		t.Errorf("model field lost: %v", cfg["model"])
	}
	if cfg["approval_mode"] != "full-auto" {
		t.Errorf("approval_mode field lost: %v", cfg["approval_mode"])
	}
	servers, ok := cfg["mcp_servers"].(map[string]any)
	if !ok {
		t.Fatal("mcp_servers section missing")
	}
	if _, ok := servers["test"]; !ok {
		t.Error("test server missing")
	}
}

func TestDetectFallback(t *testing.T) {
	// With HOME set to empty dir, Detect should still return at least Claude Code.
	t.Setenv("HOME", t.TempDir())
	writers := Detect()
	if len(writers) == 0 {
		t.Error("Detect returned no writers")
	}
	if writers[0].Runtime() != ClaudeCode {
		t.Errorf("fallback should be ClaudeCode, got %s", writers[0].Runtime())
	}
}

func TestDetectFindsExisting(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create Codex global config dir.
	os.MkdirAll(filepath.Join(home, ".codex"), 0o755)

	writers := Detect()
	found := false
	for _, w := range writers {
		if w.Runtime() == Codex {
			found = true
		}
	}
	if !found {
		t.Error("Detect did not find Codex with .codex dir present")
	}
}

func TestRemoveFromNonExistentFile(t *testing.T) {
	dir := t.TempDir()
	w := For(ClaudeCode)
	if err := w.Remove(filepath.Join(dir, "nope.json"), "x"); err != nil {
		t.Errorf("remove from missing file: %v", err)
	}
}

func TestClaudeWriterPaths(t *testing.T) {
	w := For(ClaudeCode)
	if w.LocalPath() != ".mcp.json" {
		t.Errorf("LocalPath = %q", w.LocalPath())
	}
	gp, err := w.GlobalPath()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(gp) != "mcp.json" {
		t.Errorf("GlobalPath base = %q", filepath.Base(gp))
	}
}

func TestGeminiWriterPaths(t *testing.T) {
	w := For(Gemini)
	if w.LocalPath() != filepath.Join(".gemini", "settings.json") {
		t.Errorf("LocalPath = %q", w.LocalPath())
	}
	gp, err := w.GlobalPath()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(gp) != "settings.json" {
		t.Errorf("GlobalPath base = %q", filepath.Base(gp))
	}
}

func TestCodexWriterPaths(t *testing.T) {
	w := For(Codex)
	if w.LocalPath() != filepath.Join(".codex", "config.toml") {
		t.Errorf("LocalPath = %q", w.LocalPath())
	}
	gp, err := w.GlobalPath()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(gp) != "config.toml" {
		t.Errorf("GlobalPath base = %q", filepath.Base(gp))
	}
}

// tomlDecode is a test helper wrapping toml.Decode.
func tomlDecode(data string, v any) (any, error) {
	_, err := toml.Decode(data, v)
	return v, err
}
