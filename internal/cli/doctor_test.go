package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupDoctorRepo creates a git repo with full lore init applied (hooks, claude, cursor, windsurf configs).
func setupDoctorRepo(t *testing.T) (gitRoot string) {
	t.Helper()
	gitRoot, loreRoot, _ := setupTestRepo(t)

	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = gitRoot
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	// Wire core.hooksPath.
	git("config", "core.hooksPath", ".lore/hooks")

	// Write hook scripts with the managed header.
	hooksDir := filepath.Join(loreRoot, "hooks")
	os.MkdirAll(hooksDir, 0755)
	for _, name := range []string{"post-commit", "post-checkout", "post-merge"} {
		content := "#!/bin/sh\n# Managed by lore — do not edit manually\nlore hook " + name[5:] + "\n"
		if name == "post-commit" {
			content = "#!/bin/sh\n# Managed by lore — do not edit manually\nlore hook commit\n"
		}
		os.WriteFile(filepath.Join(hooksDir, name), []byte(content), 0755)
	}

	// Write Claude Code settings.json.
	claudeDir := filepath.Join(gitRoot, ".claude")
	os.MkdirAll(claudeDir, 0755)
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(`{
  "hooks": {
    "Stop": [{"command": "lore hook agent-recap agent:claude"}],
    "PostToolUse": [
      {"matcher": "Edit", "command": "lore hook file-write \"$TOOL_INPUT_file_path\" agent:claude"}
    ]
  },
  "mcpServers": {
    "lore": {"command": "lore", "args": ["mcp", "agent:claude"]}
  }
}`), 0644)

	// Write Cursor + Windsurf MCP configs.
	os.MkdirAll(filepath.Join(gitRoot, ".cursor"), 0755)
	os.WriteFile(filepath.Join(gitRoot, ".cursor", "mcp.json"), []byte(`{"mcpServers":{"lore":{"command":"lore","args":["mcp","agent:cursor"]}}}`), 0644)
	os.MkdirAll(filepath.Join(gitRoot, ".windsurf"), 0755)
	os.WriteFile(filepath.Join(gitRoot, ".windsurf", "mcp.json"), []byte(`{"mcpServers":{"lore":{"command":"lore","args":["mcp","agent:windsurf"]}}}`), 0644)

	return gitRoot
}

func runDoctorIn(t *testing.T, gitRoot string) (string, error) {
	t.Helper()
	orig, _ := os.Getwd()
	os.Chdir(gitRoot)
	defer os.Chdir(orig)

	var buf bytes.Buffer
	cmd := doctorCmd
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	err := runDoctor(cmd, nil)
	return buf.String(), err
}

func TestDoctor_AllPass(t *testing.T) {
	gitRoot := setupDoctorRepo(t)
	out, err := runDoctorIn(t, gitRoot)

	// Critical checks must pass → no error.
	if err != nil {
		t.Errorf("expected no error for fully configured repo, got: %v\noutput:\n%s", err, out)
	}

	for _, want := range []string{
		"Git repository", "Lore initialized", "Git hooks wired",
		"Hook scripts", "Claude Code hooks", "Claude Code MCP",
		"Cursor MCP", "Windsurf MCP",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got:\n%s", want, out)
		}
	}

	// No ✗ failures.
	if strings.Contains(out, "✗") {
		t.Errorf("unexpected failure marker in output:\n%s", out)
	}

	// Stats section present.
	if !strings.Contains(out, "Blobs:") {
		t.Errorf("expected stats section, got:\n%s", out)
	}
}

func TestDoctor_NoLoreInit_FailsCritical(t *testing.T) {
	_, loreRoot, _ := setupTestRepo(t)
	// Remove .lore/ so lore is not initialized.
	os.RemoveAll(loreRoot)

	// Chdir to gitRoot (parent of loreRoot).
	gitRoot := filepath.Dir(loreRoot)
	orig, _ := os.Getwd()
	os.Chdir(gitRoot)
	defer os.Chdir(orig)

	var buf bytes.Buffer
	cmd := doctorCmd
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	err := runDoctor(cmd, nil)

	if err == nil {
		t.Error("expected error when .lore/ is missing")
	}
	if !strings.Contains(buf.String(), "not found") {
		t.Errorf("expected 'not found' message, got:\n%s", buf.String())
	}
}

func TestDoctor_MissingHooksPath_Fails(t *testing.T) {
	gitRoot := setupDoctorRepo(t)
	// Remove core.hooksPath.
	exec.Command("git", "-C", gitRoot, "config", "--unset", "core.hooksPath").Run()

	out, err := runDoctorIn(t, gitRoot)
	if err == nil {
		t.Error("expected error when core.hooksPath is unset")
	}
	if !strings.Contains(out, "Git hooks wired") {
		t.Errorf("expected hooks check in output, got:\n%s", out)
	}
}

func TestDoctor_MissingHookScript_Fails(t *testing.T) {
	gitRoot := setupDoctorRepo(t)
	// Delete one of the hook scripts.
	os.Remove(filepath.Join(gitRoot, ".lore", "hooks", "post-commit"))

	out, err := runDoctorIn(t, gitRoot)
	if err == nil {
		t.Error("expected error when post-commit script is missing")
	}
	if !strings.Contains(out, "post-commit") {
		t.Errorf("expected post-commit in output, got:\n%s", out)
	}
}

func TestDoctor_MissingClaudeSettings_Warns(t *testing.T) {
	gitRoot := setupDoctorRepo(t)
	// Remove Claude settings — should warn, not fail.
	os.RemoveAll(filepath.Join(gitRoot, ".claude"))

	out, err := runDoctorIn(t, gitRoot)
	// Critical checks still pass → no error.
	if err != nil {
		t.Errorf("expected no critical error when claude settings are missing, got: %v\noutput:\n%s", err, out)
	}
	// Should contain warning markers.
	if !strings.Contains(out, "⚠") && !strings.Contains(out, "not found") {
		t.Errorf("expected warning about missing Claude settings, got:\n%s", out)
	}
}

func TestDoctor_OllamaUnavailable_Warns(t *testing.T) {
	gitRoot := setupDoctorRepo(t)
	// Write a config pointing at a non-existent Ollama endpoint.
	loreRoot := filepath.Join(gitRoot, ".lore")
	os.WriteFile(filepath.Join(loreRoot, "config.toml"), []byte(`
[llm]
provider = "ollama"
model    = "llama3"
endpoint = "http://127.0.0.1:19999"
`), 0644)

	out, err := runDoctorIn(t, gitRoot)
	// Should not fail critically — Ollama is optional.
	if err != nil {
		t.Errorf("expected no critical error when Ollama is unreachable, got: %v", err)
	}
	if !strings.Contains(out, "not reachable") && !strings.Contains(out, "LLM") {
		t.Errorf("expected LLM warning in output, got:\n%s", out)
	}
}
