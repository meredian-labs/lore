package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nishchay/lore/internal/blob"
	"github.com/nishchay/lore/internal/config"
	gitpkg "github.com/nishchay/lore/internal/git"
	"github.com/nishchay/lore/internal/store"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check lore installation and configuration",
	Args:  cobra.NoArgs,
	RunE:  runDoctor,
}

const (
	levelOK   = 0
	levelWarn = 1
	levelFail = 2
)

type doctorCheck struct {
	label  string
	detail string
	level  int
}

func runDoctor(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	w := cmd.OutOrStdout()

	var checks []doctorCheck
	var gitRoot, loreRoot string

	// Check 1: git repository
	if root, err := gitpkg.FindGitRoot(); err != nil {
		checks = append(checks, doctorCheck{"Git repository", "not found — run inside a git repo", levelFail})
	} else {
		gitRoot = root
		checks = append(checks, doctorCheck{"Git repository", gitRoot, levelOK})
	}

	// Check 2: lore initialized
	if gitRoot != "" {
		loreRoot = filepath.Join(gitRoot, ".lore")
		if _, err := os.Stat(loreRoot); err != nil {
			checks = append(checks, doctorCheck{"Lore initialized", "not found — run 'lore init'", levelFail})
			loreRoot = ""
		} else {
			// Read initialized_at from meta if store opens.
			detail := ".lore/"
			if s, err := store.Open(filepath.Join(loreRoot, "lore.db")); err == nil {
				if ts, err := s.GetMeta(ctx, "initialized_at"); err == nil && ts != "" {
					var ns int64
					fmt.Sscanf(ts, "%d", &ns)
					if ns > 0 {
						detail = ".lore/  (since " + time.Unix(0, ns).UTC().Format("2006-01-02") + ")"
					}
				}
				s.Close()
			}
			checks = append(checks, doctorCheck{"Lore initialized", detail, levelOK})
		}
	}

	// Check 3: core.hooksPath
	if gitRoot != "" {
		hooksPath, err := gitpkg.Output(ctx, gitRoot, "config", "core.hooksPath")
		if err != nil || hooksPath != ".lore/hooks" {
			checks = append(checks, doctorCheck{"Git hooks wired", "core.hooksPath not set — run 'lore init'", levelFail})
		} else {
			checks = append(checks, doctorCheck{"Git hooks wired", "core.hooksPath = .lore/hooks", levelOK})
		}
	}

	// Check 4: hook scripts on disk
	if loreRoot != "" {
		scripts := []string{"post-commit", "post-checkout", "post-merge"}
		var missing []string
		for _, s := range scripts {
			data, err := os.ReadFile(filepath.Join(loreRoot, "hooks", s))
			if err != nil || !strings.Contains(string(data), "# Managed by lore") {
				missing = append(missing, s)
			}
		}
		if len(missing) > 0 {
			checks = append(checks, doctorCheck{"Hook scripts", "missing: " + strings.Join(missing, ", ") + " — run 'lore init'", levelFail})
		} else {
			checks = append(checks, doctorCheck{"Hook scripts", strings.Join(scripts, ", "), levelOK})
		}
	}

	// Check 5: Claude Code hooks
	if gitRoot != "" {
		settingsPath := filepath.Join(gitRoot, ".claude", "settings.json")
		if data, err := os.ReadFile(settingsPath); err != nil {
			checks = append(checks, doctorCheck{"Claude Code hooks", "not found — run 'lore init'", levelWarn})
		} else if hasClaudeHooks(data) {
			checks = append(checks, doctorCheck{"Claude Code hooks", "PostToolUse (Edit/Write/Read/Bash) + Stop", levelOK})
		} else {
			checks = append(checks, doctorCheck{"Claude Code hooks", "incomplete — run 'lore init'", levelWarn})
		}
	}

	// Check 6: Claude Code MCP
	if gitRoot != "" {
		settingsPath := filepath.Join(gitRoot, ".claude", "settings.json")
		if data, err := os.ReadFile(settingsPath); err != nil {
			checks = append(checks, doctorCheck{"Claude Code MCP", "not found — run 'lore init'", levelWarn})
		} else if hasLoreMCPServer(data) {
			checks = append(checks, doctorCheck{"Claude Code MCP", "lore mcp agent:claude", levelOK})
		} else {
			checks = append(checks, doctorCheck{"Claude Code MCP", "missing mcpServers.lore — run 'lore init'", levelWarn})
		}
	}

	// Check 7: Cursor MCP
	if gitRoot != "" {
		cursorPath := filepath.Join(gitRoot, ".cursor", "mcp.json")
		if data, err := os.ReadFile(cursorPath); err != nil {
			checks = append(checks, doctorCheck{"Cursor MCP", "not found — run 'lore init'", levelWarn})
		} else if hasLoreMCPServer(data) {
			checks = append(checks, doctorCheck{"Cursor MCP", ".cursor/mcp.json", levelOK})
		} else {
			checks = append(checks, doctorCheck{"Cursor MCP", "missing lore entry — run 'lore init'", levelWarn})
		}
	}

	// Check 8: Windsurf MCP
	if gitRoot != "" {
		windsurfPath := filepath.Join(gitRoot, ".windsurf", "mcp.json")
		if data, err := os.ReadFile(windsurfPath); err != nil {
			checks = append(checks, doctorCheck{"Windsurf MCP", "not found — run 'lore init'", levelWarn})
		} else if hasLoreMCPServer(data) {
			checks = append(checks, doctorCheck{"Windsurf MCP", ".windsurf/mcp.json", levelOK})
		} else {
			checks = append(checks, doctorCheck{"Windsurf MCP", "missing lore entry — run 'lore init'", levelWarn})
		}
	}

	// Check 9: LLM availability
	if loreRoot != "" {
		cfg, _ := config.Load(loreRoot)
		if cfg.LLM.Endpoint == "" {
			checks = append(checks, doctorCheck{"LLM", "not configured (config.toml has no endpoint)", levelWarn})
		} else {
			label := cfg.LLM.Provider + "/" + cfg.LLM.Model + " (" + cfg.LLM.Endpoint + ")"
			pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			if blob.NewLLMClient(cfg.LLM.Endpoint, cfg.LLM.Model).Ping(pingCtx) == nil {
				checks = append(checks, doctorCheck{"LLM", label, levelOK})
			} else {
				checks = append(checks, doctorCheck{"LLM", "not reachable — " + label, levelWarn})
			}
		}
	}

	// --- Print results ---
	fmt.Fprintln(w)
	var fails, warns int
	for _, c := range checks {
		icon, colorFn := checkIcon(c.level)
		fmt.Fprintf(w, "%s  %-22s  %s\n", icon, colorFn(c.label), c.detail)
		switch c.level {
		case levelWarn:
			warns++
		case levelFail:
			fails++
		}
	}

	// --- Stats (only when lore is initialized) ---
	if loreRoot != "" {
		if s, err := store.Open(filepath.Join(loreRoot, "lore.db")); err == nil {
			defer s.Close()
			blobCount, _ := s.BlobCount(ctx)
			byKind, _ := s.BlobCountByKind(ctx)
			nodeCount, _ := s.NodeCount(ctx)
			pendingCount, _ := s.PendingTaskCount(ctx)

			fmt.Fprintln(w)
			kindParts := kindSummary(byKind)
			if len(kindParts) > 0 {
				fmt.Fprintf(w, "Blobs: %d  (%s)\n", blobCount, strings.Join(kindParts, ", "))
			} else {
				fmt.Fprintf(w, "Blobs: %d\n", blobCount)
			}
			fmt.Fprintf(w, "Nodes: %d\n", nodeCount)
			fmt.Fprintf(w, "Pending tasks: %d\n", pendingCount)
		}
	}

	// --- Summary line ---
	fmt.Fprintln(w)
	if fails > 0 {
		fmt.Fprintf(w, "%d failure(s), %d warning(s)\n", fails, warns)
		return fmt.Errorf("lore: doctor found %d critical issue(s)", fails)
	}
	if warns > 0 {
		fmt.Fprintf(w, "%d warning(s) — run 'lore init' to fix missing configs\n", warns)
		return nil
	}
	fmt.Fprintln(w, "All checks passed.")
	return nil
}

// checkIcon returns the display icon and a color function for a given level.
func checkIcon(level int) (string, func(string) string) {
	identity := func(s string) string { return s }
	switch level {
	case levelOK:
		if colorEnabled() {
			return "\033[32m✓\033[0m", identity
		}
		return "✓", identity
	case levelWarn:
		if colorEnabled() {
			return "\033[33m⚠\033[0m", func(s string) string { return "\033[33m" + s + "\033[0m" }
		}
		return "⚠", identity
	default:
		if colorEnabled() {
			return "\033[31m✗\033[0m", func(s string) string { return "\033[31m" + s + "\033[0m" }
		}
		return "✗", identity
	}
}

// hasClaudeHooks returns true if settings JSON has the lore Stop hook.
func hasClaudeHooks(data []byte) bool {
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil {
		return false
	}
	var hooks map[string]json.RawMessage
	if json.Unmarshal(raw["hooks"], &hooks) != nil {
		return false
	}
	var stopHooks []map[string]string
	if json.Unmarshal(hooks["Stop"], &stopHooks) != nil {
		return false
	}
	for _, h := range stopHooks {
		if strings.Contains(h["command"], "lore hook agent-recap") {
			return true
		}
	}
	return false
}

// hasLoreMCPServer returns true if the JSON has mcpServers.lore.
func hasLoreMCPServer(data []byte) bool {
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil {
		return false
	}
	var servers map[string]json.RawMessage
	if json.Unmarshal(raw["mcpServers"], &servers) != nil {
		return false
	}
	_, ok := servers["lore"]
	return ok
}

// kindSummary returns sorted "N Kind" strings, excluding Checkpoint.
func kindSummary(byKind map[string]int) []string {
	type kv struct{ k string; v int }
	var pairs []kv
	for k, v := range byKind {
		if k == "Checkpoint" {
			continue
		}
		pairs = append(pairs, kv{k, v})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].v > pairs[j].v })
	out := make([]string, len(pairs))
	for i, p := range pairs {
		out[i] = fmt.Sprintf("%d %s", p.v, p.k)
	}
	return out
}
