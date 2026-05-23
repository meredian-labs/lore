package cli

import (
	"github.com/spf13/cobra"
)

var hookCmd = &cobra.Command{
	Use:   "hook <kind> [args]",
	Short: "Internal: called by git hooks",
	Args:  cobra.MinimumNArgs(1),
}

func init() {
	hookCmd.Hidden = true
}
