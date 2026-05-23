package cli

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/meredian-labs/lore/internal/store"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show full detail for a blob",
	Args:  cobra.ExactArgs(1),
	RunE:  runShow,
}

func init() {
	showCmd.Flags().Bool("json", false, "Output as JSON")
}

func runShow(cmd *cobra.Command, args []string) error {
	loreRoot, err := findLoreRoot()
	if err != nil {
		return err
	}
	s, err := store.Open(filepath.Join(loreRoot, "lore.db"))
	if err != nil {
		return err
	}
	defer s.Close()

	fullID, err := s.ResolveBlobIDPrefix(cmd.Context(), args[0])
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("error: no blob found for %q\nhint: run 'lore log' to list blobs", args[0])
		}
		if errors.Is(err, store.ErrAmbiguous) {
			return fmt.Errorf("error: ambiguous blob ID prefix %q\nhint: provide more characters", args[0])
		}
		return err
	}

	b, err := s.BlobByID(cmd.Context(), fullID)
	if err != nil {
		return err
	}
	files, err := s.BlobFiles(cmd.Context(), fullID)
	if err != nil {
		return err
	}
	cmds, err := s.BlobCommands(cmd.Context(), fullID)
	if err != nil {
		return err
	}

	asJSON, _ := cmd.Flags().GetBool("json")
	if asJSON {
		var nodeRef *NodeRefJSON
		if b.PrimaryNodeID != "" {
			if n, nErr := s.NodeByID(cmd.Context(), b.PrimaryNodeID); nErr == nil {
				nodeRef = &NodeRefJSON{ID: n.ID, Title: n.Title}
			}
		}
		return writeJSON(cmd.OutOrStdout(), blobToJSON(b, files, cmds, nodeRef))
	}

	out := cmd.OutOrStdout()

	// Header
	fmt.Fprintf(out, "ID:           %s\n", b.ID[:8])
	fmt.Fprintf(out, "Title:        %s\n", bold(b.Title))
	fmt.Fprintf(out, "Kind:         %s\n", string(b.Kind))
	fmt.Fprintf(out, "Trust:        %s (source: %s)\n", colorTrustLabel(b.TrustLevel), b.AISource)
	fmt.Fprintln(out)

	// Observed section
	fmt.Fprintln(out, dim("── Observed ────────────────────────────────────────"))
	fmt.Fprintf(out, "Started:      %s\n", time.Unix(0, b.StartedAt).UTC().Format("2006-01-02 15:04"))
	fmt.Fprintf(out, "Ended:        %s\n", time.Unix(0, b.EndedAt).UTC().Format("2006-01-02 15:04"))

	if b.CommitStart != "" {
		if b.CommitEnd != "" && b.CommitEnd != b.CommitStart {
			fmt.Fprintf(out, "Commits:      %s..%s\n", shortSHA(b.CommitStart), shortSHA(b.CommitEnd))
		} else {
			fmt.Fprintf(out, "Commits:      %s\n", shortSHA(b.CommitStart))
		}
	}

	var written, deleted []store.BlobFile
	for _, f := range files {
		if f.Role == "deleted" {
			deleted = append(deleted, f)
		} else {
			written = append(written, f)
		}
	}

	if len(written) > 0 {
		fmt.Fprintln(out, "\nFiles Modified:")
		for _, f := range written {
			fmt.Fprintf(out, "  %s\n", f.Path)
		}
	}
	if len(deleted) > 0 {
		fmt.Fprintln(out, "\nFiles Deleted:")
		for _, f := range deleted {
			fmt.Fprintf(out, "  %s\n", f.Path)
		}
	}
	if len(cmds) > 0 {
		fmt.Fprintln(out, "\nCommands:")
		for _, c := range cmds {
			fmt.Fprintf(out, "  %s\n", c.Command)
		}
	}

	// Interpreted section
	fmt.Fprintln(out)
	fmt.Fprintln(out, dim("── Interpreted ─────────────────────────────────────"))
	if b.UserIntent != "" {
		fmt.Fprintf(out, "User Intent:  %s\n", b.UserIntent)
	}
	if b.Summary != "" {
		fmt.Fprintf(out, "Summary:      %s\n", wrapIndent(b.Summary, 14))
	}
	if b.Recap != "" {
		fmt.Fprintf(out, "Recap:        %s\n", wrapIndent(b.Recap, 14))
	}
	if len(b.Tags) > 0 {
		fmt.Fprintf(out, "Tags:         %s\n", strings.Join(b.Tags, ", "))
	}

	// Node section
	if b.PrimaryNodeID != "" {
		n, err := s.NodeByID(cmd.Context(), b.PrimaryNodeID)
		if err == nil {
			fmt.Fprintln(out)
			fmt.Fprintln(out, dim("── Part of ─────────────────────────────────────────"))
			fmt.Fprintf(out, "Node: %s\n", n.Title)
		}
	}

	return nil
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// wrapIndent returns s with any newlines re-indented by indent spaces.
func wrapIndent(s string, indent int) string {
	pad := strings.Repeat(" ", indent)
	return strings.ReplaceAll(s, "\n", "\n"+pad)
}
