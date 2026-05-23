// Package glh implements the git lore handler — a git wrapper that fires
// lore task capture on commit/checkout/merge and provides enriched log
// and status views.
package glh

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Run is the entry point when the binary is invoked as "glh".
func Run() {
	if len(os.Args) < 2 {
		passthrough([]string{"--help"})
		return
	}
	switch os.Args[1] {
	case "commit":
		os.Exit(runCommit(os.Args[2:]))
	case "checkout", "switch":
		os.Exit(runCheckout(os.Args[2:]))
	case "merge":
		os.Exit(runMerge(os.Args[2:]))
	case "log":
		os.Exit(runLog(os.Args[2:]))
	case "status", "st":
		os.Exit(runStatus(os.Args[2:]))
	default:
		passthrough(os.Args[1:])
	}
}

// hooksInstalled reports whether core.hooksPath is set to .lore/hooks.
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

// fireLoreHook runs "lore hook <args>" silently — never blocks the caller.
func fireLoreHook(args ...string) {
	full := append([]string{"hook"}, args...)
	exec.CommandContext(context.Background(), "lore", full...).Run()
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
