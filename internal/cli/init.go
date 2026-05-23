package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nishchay/lore/internal/config"
	"github.com/nishchay/lore/internal/git"
	"github.com/nishchay/lore/internal/store"
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

	// Step 10: print success.
	fmt.Fprintf(cmd.OutOrStdout(),
		"Initialized lore repository in %s/\n  Database:    .lore/lore.db\n  Config:      .lore/config.toml\n  Git hooks:   .lore/hooks/ (core.hooksPath)\n\nCommit .lore/ to git. On clones, run 'lore init' to wire hooks.\n",
		loreRoot,
	)
	return nil
}
