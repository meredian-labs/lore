package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/meredian-labs/lore/internal/config"
	"github.com/meredian-labs/lore/internal/git"
	"github.com/meredian-labs/lore/internal/store"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize lore repository and install git hooks",
	Args:  cobra.NoArgs,
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	// Step 1: find git root.
	gitRoot, err := git.FindGitRoot()
	if err != nil {
		return fmt.Errorf("not a git repository")
	}

	// Step 2: compute lore root.
	loreRoot := filepath.Join(gitRoot, ".lore")

	// Step 3: clone case — .lore/ already exists (committed in the repo).
	// Ensure hook scripts exist and wire core.hooksPath; skip db and config.
	if _, err := os.Stat(loreRoot); err == nil {
		if err := git.WriteHookScripts(loreRoot); err != nil {
			return fmt.Errorf("writing hook scripts: %w", err)
		}
		if err := git.WireHooksPath(gitRoot); err != nil {
			return fmt.Errorf("wiring hooks: %w", err)
		}
		if err := writeClaudeSettings(gitRoot); err != nil {
			return fmt.Errorf("writing claude settings: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "lore: hooks wired (core.hooksPath = .lore/hooks)")
		return nil
	}

	// Step 4: create .lore/ and .lore/cache/.
	if err := os.MkdirAll(filepath.Join(loreRoot, "cache"), 0755); err != nil {
		return fmt.Errorf("creating .lore directory: %w", err)
	}

	// Step 5: open (and migrate) database.
	s, err := store.Open(filepath.Join(loreRoot, "lore.db"))
	if err != nil {
		return fmt.Errorf("initializing database: %w", err)
	}
	defer s.Close()
	if err := s.SetMeta(cmd.Context(), "git_root", gitRoot); err != nil {
		return fmt.Errorf("setting git_root meta: %w", err)
	}

	// Step 6: write config.toml.
	if err := config.WriteDefault(loreRoot); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	// Step 7: write .lore/.gitignore to exclude cache only.
	// lore.db, config.toml, and hooks/ are intentionally committed so clones
	// get the full knowledge history and hook scripts.
	if err := os.WriteFile(filepath.Join(loreRoot, ".gitignore"), []byte("cache/\n"), 0644); err != nil {
		return fmt.Errorf("writing .lore/.gitignore: %w", err)
	}

	// Step 8: write hook scripts into .lore/hooks/ (committed with the repo).
	if err := git.WriteHookScripts(loreRoot); err != nil {
		return fmt.Errorf("writing hook scripts: %w", err)
	}

	// Step 9: wire core.hooksPath so git uses .lore/hooks/.
	if err := git.WireHooksPath(gitRoot); err != nil {
		return fmt.Errorf("wiring hooks: %w", err)
	}

	// Step 10: write .claude/settings.json for Claude Code agent integration.
	if err := writeClaudeSettings(gitRoot); err != nil {
		return fmt.Errorf("writing claude settings: %w", err)
	}

	// Step 11: print success.
	fmt.Fprintf(cmd.OutOrStdout(),
		"Initialized lore repository in %s/\n  Database:    .lore/lore.db\n  Config:      .lore/config.toml\n  Git hooks:   .lore/hooks/ (core.hooksPath)\n  Claude Code: .claude/settings.json\n\nCommit .lore/ to git. On clones, run 'lore init' to wire hooks.\n",
		loreRoot,
	)
	return nil
}

// claudeHook is one entry in a hooks event list in .claude/settings.json.
type claudeHook struct {
	Matcher string `json:"matcher,omitempty"`
	Command string `json:"command"`
}

// loreHooks are the hook entries lore installs into .claude/settings.json.
// Includes file writes, reads, commands, and the Stop recap.
var loreHooks = map[string][]claudeHook{
	"PostToolUse": {
		{Matcher: "Edit", Command: `lore hook file-write "$TOOL_INPUT_file_path" agent:claude`},
		{Matcher: "Write", Command: `lore hook file-write "$TOOL_INPUT_file_path" agent:claude`},
		{Matcher: "Read", Command: `lore hook file-read "$TOOL_INPUT_file_path" agent:claude`},
		{Matcher: "Bash", Command: `lore hook command "$TOOL_INPUT_command" agent:claude`},
	},
	"Stop": {
		{Command: "lore hook agent-recap agent:claude"},
	},
}

// loreMCPServer is the mcpServers entry added to .claude/settings.json.
var loreMCPServer = map[string]interface{}{
	"command": "lore",
	"args":    []string{"mcp", "agent:claude"},
}

// writeClaudeSettings writes (or merges) lore's hooks and MCP server config into
// .claude/settings.json, then writes MCP configs for Cursor and Windsurf.
// Unknown top-level keys in existing files are preserved.
func writeClaudeSettings(gitRoot string) error {
	if err := writeClaudeCodeSettings(gitRoot); err != nil {
		return err
	}
	if err := writeCursorMCP(gitRoot); err != nil {
		return err
	}
	if err := writeWindsurfMCP(gitRoot); err != nil {
		return err
	}
	return nil
}

// writeClaudeCodeSettings merges lore hooks and MCP server into .claude/settings.json.
func writeClaudeCodeSettings(gitRoot string) error {
	claudeDir := filepath.Join(gitRoot, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return err
	}
	settingsPath := filepath.Join(claudeDir, "settings.json")

	// Load existing file as a raw map to preserve unknown keys.
	rawMap := map[string]json.RawMessage{}
	if data, err := os.ReadFile(settingsPath); err == nil {
		json.Unmarshal(data, &rawMap)
	}

	// Merge hooks.
	existingHooks := map[string][]claudeHook{}
	if hookRaw, ok := rawMap["hooks"]; ok {
		json.Unmarshal(hookRaw, &existingHooks)
	}
	for event, hooks := range loreHooks {
		cur := existingHooks[event]
		for _, h := range hooks {
			if !hookEntryExists(cur, h) {
				cur = append(cur, h)
			}
		}
		existingHooks[event] = cur
	}
	hookBytes, err := json.Marshal(existingHooks)
	if err != nil {
		return err
	}
	rawMap["hooks"] = json.RawMessage(hookBytes)

	// Merge mcpServers — add "lore" entry if not present.
	mcpServers := map[string]json.RawMessage{}
	if mcpRaw, ok := rawMap["mcpServers"]; ok {
		json.Unmarshal(mcpRaw, &mcpServers)
	}
	if _, exists := mcpServers["lore"]; !exists {
		serverBytes, _ := json.Marshal(loreMCPServer)
		mcpServers["lore"] = json.RawMessage(serverBytes)
	}
	mcpBytes, err := json.Marshal(mcpServers)
	if err != nil {
		return err
	}
	rawMap["mcpServers"] = json.RawMessage(mcpBytes)

	out, err := json.MarshalIndent(rawMap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, append(out, '\n'), 0644)
}

// mcpConfig is the common MCP server config written for non-Claude agents.
type mcpConfig struct {
	MCPServers map[string]mcpServerEntry `json:"mcpServers"`
}

type mcpServerEntry struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// writeCursorMCP writes .cursor/mcp.json for Cursor IDE.
func writeCursorMCP(gitRoot string) error {
	return writeMCPConfig(filepath.Join(gitRoot, ".cursor", "mcp.json"), "agent:cursor")
}

// writeWindsurfMCP writes .windsurf/mcp.json for Windsurf IDE.
func writeWindsurfMCP(gitRoot string) error {
	return writeMCPConfig(filepath.Join(gitRoot, ".windsurf", "mcp.json"), "agent:windsurf")
}

// writeMCPConfig writes (or merges) the lore MCP server entry into a standard
// mcpServers JSON file at path. The format is compatible with Cursor and Windsurf.
func writeMCPConfig(path, source string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	// Load existing file as a raw map to preserve other servers.
	rawMap := map[string]json.RawMessage{}
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &rawMap)
	}

	// Parse existing mcpServers (best-effort).
	servers := map[string]json.RawMessage{}
	if raw, ok := rawMap["mcpServers"]; ok {
		json.Unmarshal(raw, &servers)
	}

	// Add lore entry if absent.
	if _, exists := servers["lore"]; !exists {
		entry := mcpServerEntry{Command: "lore", Args: []string{"mcp", source}}
		b, _ := json.Marshal(entry)
		servers["lore"] = json.RawMessage(b)
	}

	serversBytes, _ := json.Marshal(servers)
	rawMap["mcpServers"] = json.RawMessage(serversBytes)

	out, err := json.MarshalIndent(rawMap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0644)
}

// hookEntryExists reports whether an identical (matcher, command) entry already exists.
func hookEntryExists(hooks []claudeHook, h claudeHook) bool {
	for _, existing := range hooks {
		if existing.Matcher == h.Matcher && existing.Command == h.Command {
			return true
		}
	}
	return false
}

// hookCommandExists reports whether any entry with the given command exists.
// Used by tests that check for presence of a specific command regardless of matcher.
func hookCommandExists(hooks []claudeHook, cmd string) bool {
	for _, h := range hooks {
		if h.Command == cmd {
			return true
		}
	}
	return false
}
