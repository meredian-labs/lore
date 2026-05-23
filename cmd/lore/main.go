package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/meredian-labs/lore/internal/cli"
	"github.com/meredian-labs/lore/internal/glh"
)

func main() {
	// When invoked as "glh" (via symlink or rename), run the git lore handler.
	if filepath.Base(os.Args[0]) == "glh" {
		glh.Run()
		return
	}

	if err := cli.Execute(); err != nil {
		switch {
		case errors.Is(err, cli.ErrNotALoreRepo):
			fmt.Fprintf(os.Stderr, "error: not a lore repository (or any parent up to mount point /)\nhint: run 'lore init' to initialize\n")
			os.Exit(128)
		case errors.Is(err, cli.ErrUsage):
			os.Exit(2)
		default:
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	}
}
