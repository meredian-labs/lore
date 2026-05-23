package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "lore",
	Short:         "Local-first engineering memory system",
	Version:       fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, BuildDate),
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		name := cmd.Name()
		if name == "init" || name == "doctor" || name == "version" {
			return nil
		}
		// Hook sub-commands run from inside git hooks; they handle their own
		// lore root lookup and fail silently so git operations are never blocked.
		if cmd.Parent() != nil && cmd.Parent().Name() == "hook" {
			return nil
		}
		_, err := findLoreRoot()
		return err
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().Bool("no-color", false, "Disable color output")
	rootCmd.AddCommand(initCmd, statusCmd, recordCmd, hookCmd, logCmd, showCmd, nodeCmd, assignCmd, whyCmd, traceCmd, graphCmd, mcpCmd)
	hookCmd.Hidden = true
}

// findLoreRoot walks parent directories looking for a .lore/ directory.
func findLoreRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".lore")); err == nil {
			return filepath.Join(dir, ".lore"), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotALoreRepo
		}
		dir = parent
	}
}
