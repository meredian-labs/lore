package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/meredian-labs/lore/internal/blob"
	"github.com/meredian-labs/lore/internal/store"
	"github.com/spf13/cobra"
)

var whyCmd = &cobra.Command{
	Use:   "why <file>",
	Short: "Show blobs that modified a file (newest first)",
	Args:  cobra.ExactArgs(1),
	RunE:  runWhy,
}

var traceCmd = &cobra.Command{
	Use:   "trace <file>",
	Short: "Chronological blob history for a file",
	Args:  cobra.ExactArgs(1),
	RunE:  runTrace,
}

func init() {
	whyCmd.Flags().Bool("json", false, "Output as JSON")
	traceCmd.Flags().Bool("json", false, "Output as JSON")
}

func runWhy(cmd *cobra.Command, args []string) error {
	loreRoot, err := findLoreRoot()
	if err != nil {
		return err
	}
	s, err := store.Open(filepath.Join(loreRoot, "lore.db"))
	if err != nil {
		return err
	}
	defer s.Close()
	asJSON, _ := cmd.Flags().GetBool("json")
	return whyFile(cmd.Context(), cmd.OutOrStdout(), s, args[0], false, asJSON)
}

func runTrace(cmd *cobra.Command, args []string) error {
	loreRoot, err := findLoreRoot()
	if err != nil {
		return err
	}
	s, err := store.Open(filepath.Join(loreRoot, "lore.db"))
	if err != nil {
		return err
	}
	defer s.Close()
	asJSON, _ := cmd.Flags().GetBool("json")
	return whyFile(cmd.Context(), cmd.OutOrStdout(), s, args[0], true, asJSON)
}

// --- testable logic ---

func whyFile(ctx context.Context, w io.Writer, s *store.Store, file string, chron bool, asJSON bool) error {
	var blobs []blob.Blob
	var err error
	if chron {
		blobs, err = s.BlobsByFileChron(ctx, file)
	} else {
		blobs, err = s.BlobsByFile(ctx, file)
	}
	if err != nil {
		return err
	}

	if errors.Is(err, store.ErrNotFound) || len(blobs) == 0 {
		return fmt.Errorf("error: no blobs found for %q\nhint: run 'lore status' to see what lore has captured", file)
	}

	if asJSON {
		var out []BlobJSON
		for _, b := range blobs {
			files, _ := s.BlobFiles(ctx, b.ID)
			cmds, _ := s.BlobCommands(ctx, b.ID)
			var nodeRef *NodeRefJSON
			if b.PrimaryNodeID != "" {
				if n, err := s.NodeByID(ctx, b.PrimaryNodeID); err == nil {
					nodeRef = &NodeRefJSON{ID: n.ID, Title: n.Title}
				}
			}
			out = append(out, blobToJSON(b, files, cmds, nodeRef))
		}
		if out == nil {
			out = []BlobJSON{}
		}
		return writeJSON(w, out)
	}

	if chron {
		fmt.Fprintf(w, "History of %s:\n\n", file)
	}
	for _, b := range blobs {
		var nodeTitle string
		if b.PrimaryNodeID != "" {
			if n, err := s.NodeByID(ctx, b.PrimaryNodeID); err == nil {
				nodeTitle = n.Title
			}
		}
		printWhyLine(w, b, nodeTitle)
	}
	return nil
}

func printWhyLine(w io.Writer, b blob.Blob, nodeTitle string) {
	fmt.Fprintf(w, "%s (%s)  %s  %s\n",
		bold(b.Title),
		string(b.Kind),
		formatDate(b.EndedAt),
		colorTrustLabel(b.TrustLevel),
	)
	if b.Summary != "" {
		fmt.Fprintf(w, "  %s\n", b.Summary)
	}
	if b.CommitStart != "" {
		if b.CommitEnd != "" && b.CommitEnd != b.CommitStart {
			fmt.Fprintf(w, "  Commits: %s..%s\n", shortSHA(b.CommitStart), shortSHA(b.CommitEnd))
		} else {
			fmt.Fprintf(w, "  Commits: %s\n", shortSHA(b.CommitStart))
		}
	}
	if nodeTitle != "" {
		fmt.Fprintf(w, "  Node: %s\n", nodeTitle)
	}
	fmt.Fprintln(w)
}

func formatDate(ns int64) string {
	if ns == 0 {
		return "unknown"
	}
	return time.Unix(0, ns).UTC().Format("2006-01-02")
}
