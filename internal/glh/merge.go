package glh

func runMerge(args []string) int {
	code := gitExitCode(append([]string{"merge"}, args...))
	if code != 0 {
		return code
	}

	if !hooksInstalled() {
		fireLoreHook("merge")
	}

	return 0
}
