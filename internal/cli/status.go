package cli

import (
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/nishchay/lore/internal/store"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show repository lore state",
	Args:  cobra.NoArgs,
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	loreRoot, err := findLoreRoot()
	if err != nil {
		return err
	}

	s, err := store.Open(filepath.Join(loreRoot, "lore.db"))
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer s.Close()

	ctx := cmd.Context()
	out := cmd.OutOrStdout()

	gitRoot, _ := s.GetMeta(ctx, "git_root")
	if gitRoot == "" {
		gitRoot = filepath.Dir(loreRoot)
	}
	fmt.Fprintf(out, "Repository: %s\n", gitRoot)

	initTS, _ := s.GetMeta(ctx, "initialized_at")
	if initTS != "" {
		var ns int64
		fmt.Sscanf(initTS, "%d", &ns)
		if ns > 0 {
			t := time.Unix(0, ns)
			fmt.Fprintf(out, "Initialized: %s\n", t.Format("2006-01-02"))
		}
	}
	fmt.Fprintln(out)

	blobCount, _ := s.BlobCount(ctx)
	byKind, _ := s.BlobCountByKind(ctx)
	byTrust, _ := s.BlobCountByTrust(ctx)
	fmt.Fprintf(out, "Blobs: %d\n", blobCount)
	if blobCount > 0 {
		kinds := make([]string, 0, len(byKind))
		for k := range byKind {
			kinds = append(kinds, k)
		}
		sort.Slice(kinds, func(i, j int) bool { return byKind[kinds[i]] > byKind[kinds[j]] })
		for _, k := range kinds {
			agentTruth := 0
			loreInferred := 0
			// We can't split per-kind by trust without a more complex query;
			// show total trust breakdown at the bottom instead.
			_ = agentTruth
			_ = loreInferred
			fmt.Fprintf(out, "  %-14s %d\n", k+":", byKind[k])
		}
		if len(byTrust) > 0 {
			fmt.Fprintf(out, "  (%d AgentTruth, %d LoreInferred)\n", byTrust[2], byTrust[4])
		}
	}
	fmt.Fprintln(out)

	nodeCount, _ := s.NodeCount(ctx)
	nodes, _ := s.ListNodes(ctx)
	fmt.Fprintf(out, "Subsystems (Nodes): %d\n", nodeCount)
	for _, n := range nodes {
		cnt, _ := s.NodeBlobCount(ctx, n.ID)
		noun := "blobs"
		if cnt == 1 {
			noun = "blob"
		}
		fmt.Fprintf(out, "  %-24s (%d %s, %s)\n", n.Title, cnt, noun, n.Status)
	}
	if nodeCount > 0 {
		unassigned, _ := s.UnassignedBlobCount(ctx)
		if unassigned > 0 {
			fmt.Fprintln(out)
			fmt.Fprintf(out, "Unassigned Blobs: %d\n", unassigned)
			fmt.Fprintln(out, "  hint: use 'lore assign <id> <subsystem>' or 'lore node create <name>'")
		}
	}
	fmt.Fprintln(out)

	pendingCount, _ := s.PendingTaskCount(ctx)
	fmt.Fprintf(out, "Pending Tasks: %d\n", pendingCount)
	fmt.Fprintln(out)

	fmt.Fprintln(out, "LLM: ollama/llama3 (not checked — run 'lore doctor')")
	return nil
}
