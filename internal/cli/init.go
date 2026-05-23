package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nishchay/lore/internal/config"
	"github.com/nishchay/lore/internal/git"
	"github.com/nishchay/lore/internal/store"
	"github.com/spf13/cobra"
)

func loreExecutable() string {
	exe, err := os.Executable()
	if err != nil {
		return "lore"
	}
	return exe
}

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

	// Step 3: idempotent check.
	if _, err := os.Stat(loreRoot); err == nil {
		fmt.Fprintln(cmd.OutOrStdout(), "lore: already initialized")
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

	// Step 7: install git hooks using the absolute path to the running binary.
	if err := git.InstallHooks(gitRoot, loreExecutable()); err != nil {
		return fmt.Errorf("installing hooks: %w", err)
	}

	// Step 8: append .lore/ to .gitignore if not already present.
	if err := ensureGitignore(gitRoot); err != nil {
		return fmt.Errorf("updating .gitignore: %w", err)
	}

	// Step 9: print success.
	fmt.Fprintf(cmd.OutOrStdout(),
		"Initialized lore repository in %s/\n  Database:    .lore/lore.db\n  Config:      .lore/config.toml\n  Git hooks:   post-commit, post-checkout, post-merge\n\nRun 'lore doctor' to verify the installation.\n",
		loreRoot,
	)
	return nil
}

func ensureGitignore(gitRoot string) error {
	giPath := filepath.Join(gitRoot, ".gitignore")
	const entry = ".lore/"

	content, err := os.ReadFile(giPath)
	if err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(content)))
		for scanner.Scan() {
			if strings.TrimSpace(scanner.Text()) == entry {
				return nil // already present
			}
		}
		f, err := os.OpenFile(giPath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = fmt.Fprintf(f, "\n%s\n", entry)
		return err
	}

	return os.WriteFile(giPath, []byte(entry+"\n"), 0644)
}
