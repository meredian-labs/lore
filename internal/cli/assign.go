package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/meredian-labs/lore/internal/graph"
	"github.com/meredian-labs/lore/internal/store"
	"github.com/spf13/cobra"
)

var assignCmd = &cobra.Command{
	Use:   "assign <blob-id> <node>",
	Short: "Assign a blob to a subsystem node",
	Args:  cobra.ExactArgs(2),
	RunE:  runAssign,
}

func runAssign(cmd *cobra.Command, args []string) error {
	loreRoot, err := findLoreRoot()
	if err != nil {
		return err
	}
	s, err := store.Open(filepath.Join(loreRoot, "lore.db"))
	if err != nil {
		return err
	}
	defer s.Close()
	return assignBlob(cmd.Context(), cmd.OutOrStdout(), s, graph.New(s), args[0], args[1])
}

// --- testable logic ---

func assignBlob(ctx context.Context, w io.Writer, s *store.Store, g *graph.Builder, blobIDPrefix, nodeName string) error {
	blobID, err := s.ResolveBlobIDPrefix(ctx, blobIDPrefix)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("error: no blob matching prefix %q\nhint: use 'lore log' to see blob IDs", blobIDPrefix)
		}
		if errors.Is(err, store.ErrAmbiguous) {
			return fmt.Errorf("error: prefix %q matches multiple blobs\nhint: use a longer prefix", blobIDPrefix)
		}
		return fmt.Errorf("error: %w\nhint: use 'lore log' to see blob IDs", err)
	}

	n, err := s.NodeByTitle(ctx, nodeName)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("error: no node %q\nhint: run 'lore node list' to see available nodes", nodeName)
		}
		return err
	}

	bl, err := s.BlobByID(ctx, blobID)
	if err != nil {
		return err
	}

	if err := s.SetBlobNode(ctx, blobID, n.ID); err != nil {
		return fmt.Errorf("assigning blob: %w", err)
	}
	if err := g.UpdateAssignment(ctx, blobID, n.ID); err != nil {
		return fmt.Errorf("updating graph: %w", err)
	}

	fmt.Fprintf(w, "Assigned %q to node %q\n", bl.Title, n.Title)
	return nil
}
