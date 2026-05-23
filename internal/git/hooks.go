package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const loreManagedHeader = "# Managed by lore — do not edit manually"

var hookScripts = map[string]string{
	"post-commit": "#!/bin/sh\n" + loreManagedHeader + "\nlore hook commit\n",
	"post-checkout": "#!/bin/sh\n" + loreManagedHeader + "\nlore hook checkout \"$1\" \"$2\" \"$3\"\n",
	"post-merge": "#!/bin/sh\n" + loreManagedHeader + "\nlore hook merge\n",
}

// InstallHooks writes git hook scripts into gitRoot/.git/hooks/.
// If a hook already exists and was not written by lore, the lore call is appended.
func InstallHooks(gitRoot, _ string) error {
	hooksDir := filepath.Join(gitRoot, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("creating hooks directory: %w", err)
	}
	for name, content := range hookScripts {
		hookPath := filepath.Join(hooksDir, name)
		existing, err := os.ReadFile(hookPath)
		if err == nil {
			// File exists — check if it's already managed by lore.
			if strings.Contains(string(existing), loreManagedHeader) {
				continue
			}
			// Append lore call to existing hook.
			loreCall := strings.SplitN(content, "\n", 3)[2] // extract the lore hook line
			appended := strings.TrimRight(string(existing), "\n") + "\n" + loreCall
			if err := os.WriteFile(hookPath, []byte(appended), 0755); err != nil {
				return fmt.Errorf("appending to hook %s: %w", name, err)
			}
			continue
		}
		if err := os.WriteFile(hookPath, []byte(content), 0755); err != nil {
			return fmt.Errorf("writing hook %s: %w", name, err)
		}
	}
	return nil
}
