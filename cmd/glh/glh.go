package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// hooksInstalled reports whether core.hooksPath is set to .lore/hooks,
// meaning git will already fire lore hooks on commit/checkout/merge.
func hooksInstalled() bool {
	out, err := exec.Command("git", "config", "core.hooksPath").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == ".lore/hooks"
}

// findLoreRoot walks parent directories until it finds a .lore/ directory.
func findLoreRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, ".lore")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errNoLore
		}
		dir = parent
	}
}

// fireLoreHook runs "lore hook <args>" silently — never blocks the caller on failure.
func fireLoreHook(args ...string) {
	full := append([]string{"hook"}, args...)
	cmd := exec.CommandContext(context.Background(), "lore", full...)
	cmd.Run()
}

// gitExitCode runs git with the given args, inheriting stdin/stdout/stderr,
// and returns git's exit code.
func gitExitCode(args []string) int {
	cmd := exec.Command("git", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode()
		}
		return 1
	}
	return 0
}
