package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/nishchay/lore/internal/blob"
	"github.com/nishchay/lore/internal/store"
	"github.com/spf13/cobra"
)

var logLimit int

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "List blobs newest-first",
	Args:  cobra.NoArgs,
	RunE:  runLog,
}

func init() {
	logCmd.Flags().IntVarP(&logLimit, "limit", "n", 20, "Maximum number of blobs to show")
}

func runLog(cmd *cobra.Command, args []string) error {
	loreRoot, err := findLoreRoot()
	if err != nil {
		return err
	}
	s, err := store.Open(filepath.Join(loreRoot, "lore.db"))
	if err != nil {
		return err
	}
	defer s.Close()

	blobs, err := s.BlobLog(cmd.Context(), logLimit)
	if err != nil {
		return fmt.Errorf("fetching blobs: %w", err)
	}

	if len(blobs) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No blobs recorded yet.")
		return nil
	}

	for _, b := range blobs {
		count, _ := s.BlobFileCount(cmd.Context(), b.ID)
		printLogLine(cmd.OutOrStdout(), b, count)
	}
	return nil
}

func printLogLine(w io.Writer, b blob.Blob, fileCount int) {
	id := b.ID
	if len(id) > 8 {
		id = id[:8]
	}
	title := b.Title
	if len(title) > 30 {
		title = title[:30]
	}
	date := time.Unix(0, b.EndedAt).UTC().Format("2006-01-02")

	fmt.Fprintf(w, "%-8s  %-30s  %-16s  %s  %-16s  %d files\n",
		id, title, string(b.Kind), date, trustLabel(b.TrustLevel), fileCount,
	)
}
