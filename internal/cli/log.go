package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/meredian-labs/lore/internal/blob"
	"github.com/meredian-labs/lore/internal/store"
	"github.com/meredian-labs/lore/internal/task"
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
	logCmd.Flags().Bool("json", false, "Output as JSON array")
	logCmd.Flags().Bool("all", false, "File-explorer tree: all blobs organized by node → blob → files")
	logCmd.Flags().Bool("all-files", false, "Verbose dump: every blob with files, commands, and agent actions")
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

	ctx := cmd.Context()
	w := cmd.OutOrStdout()

	allTree, _ := cmd.Flags().GetBool("all")
	allFiles, _ := cmd.Flags().GetBool("all-files")

	if allTree {
		return printAllTree(ctx, w, s)
	}
	if allFiles {
		return printAllFiles(ctx, w, s, logLimit)
	}

	blobs, err := s.BlobLog(ctx, logLimit)
	if err != nil {
		return fmt.Errorf("fetching blobs: %w", err)
	}

	asJSON, _ := cmd.Flags().GetBool("json")
	if asJSON {
		var out []BlobJSON
		for _, b := range blobs {
			out = append(out, blobToJSON(b, nil, nil, nil))
		}
		if out == nil {
			out = []BlobJSON{}
		}
		return writeJSON(w, out)
	}

	if len(blobs) == 0 {
		fmt.Fprintln(w, "No blobs recorded yet.")
		return nil
	}

	for _, b := range blobs {
		count, _ := s.BlobFileCount(ctx, b.ID)
		printLogLine(w, b, count)
	}
	return nil
}

// printAllTree renders a file-explorer tree view of every blob, grouped by
// node subsystem. Checkpoint blobs are excluded. Format:
//
//	Node: Authentication  [active]
//	├── abc1234  OAuth impl   Feature  2026-05-20  [AgentTruth]
//	│   ├── [+] internal/auth/oauth.go
//	│   └── [-] internal/auth/token_legacy.go
//	└── def5678  JWT fix      BugFix   2026-05-18  [LoreInferred]
//	    └── [+] internal/auth/jwt.go
func printAllTree(ctx context.Context, w io.Writer, s *store.Store) error {
	nodes, err := s.ListNodes(ctx)
	if err != nil {
		return err
	}

	if len(nodes) == 0 {
		// Fall back to a flat tree of all unassigned blobs.
		fmt.Fprintln(w, "No subsystems defined. Showing all blobs.")
		fmt.Fprintln(w)
	}

	for _, n := range nodes {
		blobs, err := s.BlobsForNode(ctx, n.ID)
		if err != nil {
			continue
		}
		// Filter checkpoints.
		blobs = filterCheckpoints(blobs)
		if len(blobs) == 0 {
			continue
		}

		noun := "blobs"
		if len(blobs) == 1 {
			noun = "blob"
		}
		fmt.Fprintf(w, "Node: %s  [%s]  %d %s\n", bold(n.Title), n.Status, len(blobs), noun)
		printBlobTree(ctx, w, s, blobs)
		fmt.Fprintln(w)
	}

	// Unassigned blobs — fetch all by using a large limit.
	unassigned, err := s.UnassignedBlobs(ctx, 99999)
	if err != nil {
		return err
	}
	unassigned = filterCheckpoints(unassigned)
	if len(unassigned) > 0 {
		fmt.Fprintf(w, "Unassigned  (%d blobs)\n", len(unassigned))
		printBlobTree(ctx, w, s, unassigned)
		fmt.Fprintln(w)
	}

	return nil
}

// printBlobTree renders blobs as a ├──/└── tree with their files indented under each.
func printBlobTree(ctx context.Context, w io.Writer, s *store.Store, blobs []blob.Blob) {
	for i, b := range blobs {
		isLastBlob := i == len(blobs)-1
		blobPrefix := "├── "
		contPrefix := "│   "
		if isLastBlob {
			blobPrefix = "└── "
			contPrefix = "    "
		}

		id := b.ID
		if len(id) > 7 {
			id = id[:7]
		}
		title := b.Title
		if len(title) > 36 {
			title = title[:36]
		}
		date := time.Unix(0, b.EndedAt).UTC().Format("2006-01-02")

		fmt.Fprintf(w, "%s%s  %-36s  %-12s  %s  %s\n",
			blobPrefix, id, title, string(b.Kind), date, colorTrustLabel(b.TrustLevel),
		)

		files, _ := s.BlobFiles(ctx, b.ID)
		for j, f := range files {
			isLastFile := j == len(files)-1
			filePrefix := contPrefix + "├── "
			if isLastFile {
				filePrefix = contPrefix + "└── "
			}
			fmt.Fprintf(w, "%s%s %s\n", filePrefix, roleIcon(f.Role), f.Path)
		}
	}
}

// printAllFiles renders a verbose expanded view: each blob gets a header,
// all interpreted fields, files, commands, and agent tasks (if retained).
func printAllFiles(ctx context.Context, w io.Writer, s *store.Store, limit int) error {
	blobs, err := s.BlobLog(ctx, limit)
	if err != nil {
		return err
	}
	blobs = filterCheckpoints(blobs)

	if len(blobs) == 0 {
		fmt.Fprintln(w, "No blobs recorded yet.")
		return nil
	}

	sep := strings.Repeat("─", 68)

	for _, b := range blobs {
		id := b.ID
		if len(id) > 7 {
			id = id[:7]
		}
		date := time.Unix(0, b.EndedAt).UTC().Format("2006-01-02")

		fmt.Fprintln(w, sep)
		fmt.Fprintf(w, "%s  %s  [%s]  %s  %s  (%s)\n",
			bold(id), bold(b.Title), string(b.Kind), date,
			colorTrustLabel(b.TrustLevel), b.AISource,
		)
		fmt.Fprintln(w, sep)

		// Node assignment.
		if b.PrimaryNodeID != "" {
			if n, err := s.NodeByID(ctx, b.PrimaryNodeID); err == nil {
				fmt.Fprintf(w, "Node:    %s\n", n.Title)
			}
		}
		if b.CommitStart != "" {
			end := ""
			if b.CommitEnd != "" && b.CommitEnd != b.CommitStart {
				end = ".." + shortSHA(b.CommitEnd)
			}
			fmt.Fprintf(w, "Commit:  %s%s\n", shortSHA(b.CommitStart), end)
		}

		// Interpreted fields.
		if b.UserIntent != "" {
			fmt.Fprintln(w)
			fmt.Fprintf(w, "Intent:  %s\n", b.UserIntent)
		}
		if b.Summary != "" {
			fmt.Fprintln(w)
			fmt.Fprintln(w, "Summary:")
			fmt.Fprintf(w, "  %s\n", wordWrap(b.Summary, 2, 80))
		}
		if b.Recap != "" {
			fmt.Fprintln(w)
			fmt.Fprintln(w, "Recap:")
			fmt.Fprintf(w, "  %s\n", wordWrap(b.Recap, 2, 80))
		}
		if len(b.Tags) > 0 {
			fmt.Fprintln(w)
			fmt.Fprintf(w, "Tags:    %s\n", strings.Join(b.Tags, ", "))
		}

		// Files.
		files, _ := s.BlobFiles(ctx, b.ID)
		if len(files) > 0 {
			fmt.Fprintln(w)
			fmt.Fprintf(w, "── Files %s\n", strings.Repeat("─", 60))
			for _, f := range files {
				fmt.Fprintf(w, "  %s %s\n", roleIcon(f.Role), f.Path)
			}
		}

		// Commands.
		cmds, _ := s.BlobCommands(ctx, b.ID)
		if len(cmds) > 0 {
			fmt.Fprintln(w)
			fmt.Fprintf(w, "── Commands %s\n", strings.Repeat("─", 57))
			for _, c := range cmds {
				fmt.Fprintf(w, "  $ %s\n", c.Command)
			}
		}

		// Agent actions (extracted tasks — may be empty if purged).
		agentTasks, _ := s.TasksExtractedInto(ctx, b.ID)
		if len(agentTasks) > 0 {
			fmt.Fprintln(w)
			fmt.Fprintf(w, "── Agent Actions %s\n", strings.Repeat("─", 52))
			for _, t := range agentTasks {
				printTaskLine(w, t)
			}
		}

		fmt.Fprintln(w)
	}
	return nil
}

// printTaskLine renders a single task in the agent-actions section.
func printTaskLine(w io.Writer, t task.Task) {
	kind := fmt.Sprintf("[%-14s]", string(t.Kind))
	src := dim(t.Source)

	switch t.Kind {
	case task.KindFileWrite, task.KindFileRead, task.KindFileDelete:
		fmt.Fprintf(w, "  %s  %-48s  %s\n", kind, t.Path, src)
	case task.KindCommand:
		detail := t.Detail
		if len(detail) > 48 {
			detail = detail[:45] + "..."
		}
		fmt.Fprintf(w, "  %s  %-48s  %s\n", kind, detail, src)
	case task.KindAgentRecap:
		// Extract a readable summary from the JSON detail.
		summary := agentRecapSummary(t.Detail)
		fmt.Fprintf(w, "  %s  %s  %s\n", kind, dim(summary), src)
	case task.KindNote:
		detail := t.Detail
		if len(detail) > 48 {
			detail = detail[:45] + "..."
		}
		fmt.Fprintf(w, "  %s  %-48s  %s\n", kind, detail, src)
	default:
		detail := t.Detail
		if t.Path != "" {
			detail = t.Path
		}
		if len(detail) > 48 {
			detail = detail[:45] + "..."
		}
		fmt.Fprintf(w, "  %s  %-48s  %s\n", kind, detail, src)
	}
}

// agentRecapSummary extracts a short human-readable line from a recap JSON detail.
func agentRecapSummary(detail string) string {
	var p struct {
		UserIntent string `json:"user_intent"`
		Summary    string `json:"summary"`
	}
	if json.Unmarshal([]byte(detail), &p) == nil {
		if p.UserIntent != "" {
			if len(p.UserIntent) > 72 {
				return p.UserIntent[:69] + "..."
			}
			return p.UserIntent
		}
		if p.Summary != "" {
			if len(p.Summary) > 72 {
				return p.Summary[:69] + "..."
			}
			return p.Summary
		}
	}
	if len(detail) > 72 {
		return detail[:69] + "..."
	}
	return detail
}

// roleIcon returns a compact colored role marker for tree display.
func roleIcon(role string) string {
	switch role {
	case "written":
		if colorEnabled() {
			return "\033[32m[+]\033[0m"
		}
		return "[+]"
	case "deleted":
		if colorEnabled() {
			return "\033[31m[-]\033[0m"
		}
		return "[-]"
	case "read":
		return dim("[~]")
	default:
		return "[" + role[:1] + "]"
	}
}

// filterCheckpoints removes KindCheckpoint blobs from a slice.
func filterCheckpoints(blobs []blob.Blob) []blob.Blob {
	out := blobs[:0]
	for _, b := range blobs {
		if b.Kind != blob.KindCheckpoint {
			out = append(out, b)
		}
	}
	return out
}

// wordWrap wraps a long string at maxWidth, preserving the given indent
// on continuation lines. Newlines in the source are respected.
func wordWrap(s string, indent, maxWidth int) string {
	lines := strings.Split(s, "\n")
	prefix := strings.Repeat(" ", indent)
	var out []string
	for _, line := range lines {
		if len(line) <= maxWidth-indent {
			out = append(out, line)
			continue
		}
		words := strings.Fields(line)
		cur := ""
		for _, w := range words {
			if cur == "" {
				cur = w
			} else if len(cur)+1+len(w) <= maxWidth-indent {
				cur += " " + w
			} else {
				out = append(out, cur)
				cur = prefix + w
			}
		}
		if cur != "" {
			out = append(out, cur)
		}
	}
	return strings.Join(out, "\n  ")
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
