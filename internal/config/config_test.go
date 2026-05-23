package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()
	if cfg.LLM.Provider != "ollama" {
		t.Errorf("expected ollama, got %s", cfg.LLM.Provider)
	}
	if cfg.LLM.Model != "llama3" {
		t.Errorf("expected llama3, got %s", cfg.LLM.Model)
	}
	if cfg.Extraction.MinTasks != 1 {
		t.Errorf("expected 1, got %d", cfg.Extraction.MinTasks)
	}
	if !cfg.Output.Color {
		t.Error("expected color=true")
	}
}

func TestWriteAndLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := WriteDefault(dir); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.LLM.Provider != "ollama" {
		t.Errorf("expected ollama, got %s", cfg.LLM.Provider)
	}
	if cfg.LLM.Endpoint != "http://localhost:11434" {
		t.Errorf("unexpected endpoint: %s", cfg.LLM.Endpoint)
	}
	if cfg.NodeResolution.MinConfidence != 0.4 {
		t.Errorf("expected 0.4, got %f", cfg.NodeResolution.MinConfidence)
	}
}

func TestLoad_LocalOverridesGlobal(t *testing.T) {
	dir := t.TempDir()

	local := `[llm]
provider = "custom"
model    = "gpt4"
endpoint = "http://custom:11434"
`
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(local), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LLM.Provider != "custom" {
		t.Errorf("expected custom, got %s", cfg.LLM.Provider)
	}
}

func TestLoad_NoFile_ReturnsDefaults(t *testing.T) {
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LLM.Model != "llama3" {
		t.Errorf("expected llama3, got %s", cfg.LLM.Model)
	}
}
