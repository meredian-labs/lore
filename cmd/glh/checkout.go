package main

import (
	"os/exec"
	"strings"
)

func runCheckout(args []string) int {
	// Capture the current branch before switching.
	prevBranch := currentBranch()

	code := gitExitCode(append([]string{"checkout"}, args...))
	if code != 0 {
		return code
	}

	if !hooksInstalled() {
		newBranch := currentBranch()
		if newBranch != "" && newBranch != prevBranch {
			// flag=1 means this is a branch switch, not a file checkout.
			fireLoreHook("checkout", prevBranch, newBranch, "1")
		}
	}

	return 0
}

func currentBranch() string {
	out, err := exec.Command("git", "branch", "--show-current").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
