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
