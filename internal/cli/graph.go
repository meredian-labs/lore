package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/meredian-labs/lore/internal/blob"
	"github.com/meredian-labs/lore/internal/store"
	"github.com/spf13/cobra"
)

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "ASCII knowledge graph (subsystems → blobs → files)",
	Args:  cobra.NoArgs,
	RunE:  runGraph,
}

func runGraph(cmd *cobra.Command, args []string) error {
	loreRoot, err := findLoreRoot()
	if err != nil {
		return err
	}
	s, err := store.Open(filepath.Join(loreRoot, "lore.db"))
	if err != nil {
		return err
	}
	defer s.Close()
	return renderGraph(cmd.Context(), cmd.OutOrStdout(), s)
}

// --- testable logic ---

func renderGraph(ctx context.Context, w io.Writer, s *store.Store) error {
	nodes, err := s.ListNodes(ctx)
	if err != nil {
		return err
	}

	for _, n := range nodes {
		fmt.Fprintf(w, "Subsystem: %s\n", bold(n.Title))

		allBlobs, err := s.BlobsForNode(ctx, n.ID)
		if err != nil {
			return err
		}
		// Exclude checkpoint blobs from the graph view.
		var blobs []blob.Blob
		for _, b := range allBlobs {
			if b.Kind != blob.KindCheckpoint {
				blobs = append(blobs, b)
			}
		}
		// Cap at 3 most recent.
		if len(blobs) > 3 {
			blobs = blobs[:3]
		}

		if len(blobs) == 0 {
			fmt.Fprintln(w, "  (no blobs assigned yet)")
		} else {
			for i, b := range blobs {
				isLast := i == len(blobs)-1
				blobPrefix := "├── "
				childPrefix := "│   "
				if isLast {
					blobPrefix = "└── "
					childPrefix = "    "
				}

				fmt.Fprintf(w, "%s%s  (%s, %s)  %s\n",
					blobPrefix,
					truncate(b.Title, 40),
					string(b.Kind),
					formatDate(b.EndedAt),
					colorTrustLabel(b.TrustLevel),
				)

				files, _ := s.BlobFiles(ctx, b.ID)
				for j, f := range files {
					fileIsLast := j == len(files)-1
					filePrefix := childPrefix + "├── "
					if fileIsLast {
						filePrefix = childPrefix + "└── "
					}
					rel := "Modified"
					if f.Role == "deleted" {
						rel = "Deleted"
					}
					fmt.Fprintf(w, "%s%s  %s\n", filePrefix, rel, f.Path)
				}
			}
		}
		fmt.Fprintln(w)
	}

	// Unassigned blobs (max 5, excluding checkpoints).
	unassigned, err := s.UnassignedBlobs(ctx, 20)
	if err != nil {
		return err
	}
	var filtered []blob.Blob
	for _, b := range unassigned {
		if b.Kind != blob.KindCheckpoint {
			filtered = append(filtered, b)
			if len(filtered) == 5 {
				break
			}
		}
	}

	if len(filtered) > 0 {
		fmt.Fprintln(w, "Unassigned Blobs:")
		for i, b := range filtered {
			isLast := i == len(filtered)-1
			blobPrefix := "├── "
			childPrefix := "│   "
			if isLast {
				blobPrefix = "└── "
				childPrefix = "    "
			}

			fmt.Fprintf(w, "%s%s  (%s, %s)  %s\n",
				blobPrefix,
				truncate(b.Title, 40),
				string(b.Kind),
				formatDate(b.EndedAt),
				colorTrustLabel(b.TrustLevel),
			)

			files, _ := s.BlobFiles(ctx, b.ID)
			for j, f := range files {
				fileIsLast := j == len(files)-1
				filePrefix := childPrefix + "├── "
				if fileIsLast {
					filePrefix = childPrefix + "└── "
				}
				rel := "Modified"
				if f.Role == "deleted" {
					rel = "Deleted"
				}
				fmt.Fprintf(w, "%s%s  %s\n", filePrefix, rel, f.Path)
			}

			if isLast {
				id := b.ID
				if len(id) > 8 {
					id = id[:8]
				}
				fmt.Fprintf(w, "    hint: use 'lore assign %s <subsystem>'\n", id)
			}
		}
	} else if len(nodes) == 0 {
		fmt.Fprintln(w, "No subsystems or blobs yet.")
		fmt.Fprintln(w, "hint: use 'lore node create <name>' to create a subsystem")
	}

	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
