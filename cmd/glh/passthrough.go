package main

import "os"

// passthrough runs git with the given args, inheriting all I/O, and exits
// with git's exit code. It never returns.
func passthrough(args []string) {
	os.Exit(gitExitCode(args))
}
