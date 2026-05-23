package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const loreManagedHeader = "# Managed by lore — do not edit manually"

var hookScripts = map[string]string{
	"post-commit":   "#!/bin/sh\n" + loreManagedHeader + "\nlore hook commit\n",
	"post-checkout": "#!/bin/sh\n" + loreManagedHeader + "\nlore hook checkout \"$1\" \"$2\" \"$3\"\n",
	"post-merge":    "#!/bin/sh\n" + loreManagedHeader + "\nlore hook merge\n",
}

// WriteHookScripts writes lore-managed hook scripts into loreRoot/hooks/.
// The scripts are committed with the repo so clones get them automatically.
// Idempotent: skips any script that already contains the managed header.
func WriteHookScripts(loreRoot string) error {
	hooksDir := filepath.Join(loreRoot, "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("creating hooks directory: %w", err)
	}
	for name, content := range hookScripts {
		hookPath := filepath.Join(hooksDir, name)
		existing, err := os.ReadFile(hookPath)
		if err == nil && strings.Contains(string(existing), loreManagedHeader) {
			continue
		}
		if err := os.WriteFile(hookPath, []byte(content), 0755); err != nil {
			return fmt.Errorf("writing hook %s: %w", name, err)
		}
	}
	return nil
}

// WireHooksPath sets core.hooksPath = .lore/hooks in the repo's local git
// config so git uses the committed hook scripts in .lore/hooks/.
// Idempotent: running it multiple times has no effect.
func WireHooksPath(gitRoot string) error {
	cmd := exec.Command("git", "config", "core.hooksPath", ".lore/hooks")
	cmd.Dir = gitRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("setting core.hooksPath: %w\n%s", err, out)
	}
	return nil
}
