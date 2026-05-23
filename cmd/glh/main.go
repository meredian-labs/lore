package main

import "os"

func main() {
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
