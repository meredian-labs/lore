package glh

import "os"

func passthrough(args []string) {
	os.Exit(gitExitCode(args))
}
