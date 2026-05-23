package git

import (
	"context"
	"os/exec"
	"strings"
)

// Output runs a git command in dir and returns trimmed stdout.
func Output(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// CommitDB stages .lore/lore.db and creates a "lore: checkpoint" commit.
// Errors are intentionally ignored — a failed auto-commit leaves lore.db dirty
// but never blocks the user's git workflow.
func CommitDB(dir string) {
	add := exec.Command("git", "add", ".lore/lore.db")
	add.Dir = dir
	if err := add.Run(); err != nil {
		return
	}
	commit := exec.Command("git", "commit", "-m", "lore: checkpoint")
	commit.Dir = dir
	commit.Run()
}
