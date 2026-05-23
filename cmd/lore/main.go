package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/nishchay/lore/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		switch {
		case errors.Is(err, cli.ErrNotALoreRepo):
			fmt.Fprintf(os.Stderr, "error: not a lore repository (or any parent up to mount point /)\nhint: run 'lore init' to initialize\n")
			os.Exit(128)
		case errors.Is(err, cli.ErrUsage):
			os.Exit(2)
		default:
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}
}
