package cli

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/nishchay/lore/internal/blob"
	"github.com/nishchay/lore/internal/config"
	"github.com/nishchay/lore/internal/store"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show repository lore state",
	Args:  cobra.NoArgs,
	RunE:  runStatus,
}

func init() {
	statusCmd.Flags().Bool("json", false, "Output as JSON")
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

	initTS, _ := s.GetMeta(ctx, "initialized_at")
	var initializedAt int64
	if initTS != "" {
		initializedAt, _ = strconv.ParseInt(initTS, 10, 64)
	}

	blobCount, _ := s.BlobCount(ctx)
	byKind, _ := s.BlobCountByKind(ctx)
	byTrust, _ := s.BlobCountByTrust(ctx)
	nodeCount, _ := s.NodeCount(ctx)
	nodes, _ := s.ListNodes(ctx)
	pendingCount, _ := s.PendingTaskCount(ctx)
	unassignedCount, _ := s.UnassignedBlobCount(ctx)

	// LLM ping.
	cfg, _ := config.Load(loreRoot)
	llmAvailable := false
	llmProvider := cfg.LLM.Provider + "/" + cfg.LLM.Model
	if cfg.LLM.Endpoint != "" {
		llmClient := blob.NewLLMClient(cfg.LLM.Endpoint, cfg.LLM.Model)
		llmAvailable = llmClient.Ping(ctx) == nil
	}

	// Build per-node counts.
	type nodeRow struct {
		n     interface{ GetID() string }
		count int
	}
	var nodeRows []NodeJSON
	for _, n := range nodes {
		cnt, _ := s.NodeBlobCount(ctx, n.ID)
		nodeRows = append(nodeRows, NodeJSON{
			ID:          n.ID,
			Title:       n.Title,
			Description: n.Description,
			Status:      n.Status,
			CreatedBy:   n.CreatedBy,
			BlobCount:   cnt,
			CreatedAt:   n.CreatedAt,
		})
	}

	asJSON, _ := cmd.Flags().GetBool("json")
	if asJSON {
		j := StatusJSON{
			Repository:      gitRoot,
			InitializedAt:   initializedAt,
			BlobCount:       blobCount,
			BlobsByKind:     byKind,
			BlobsByTrust:    byTrust,
			NodeCount:       nodeCount,
			Nodes:           nodeRows,
			UnassignedBlobs: unassignedCount,
			PendingTasks:    pendingCount,
			LLMAvailable:    llmAvailable,
			LLMProvider:     llmProvider,
		}
		if j.BlobsByKind == nil {
			j.BlobsByKind = map[string]int{}
		}
		if j.BlobsByTrust == nil {
			j.BlobsByTrust = map[int]int{}
		}
		if j.Nodes == nil {
			j.Nodes = []NodeJSON{}
		}
		return writeJSON(out, j)
	}

	// Human-readable output.
	fmt.Fprintf(out, "Repository: %s\n", gitRoot)
	if initializedAt > 0 {
		fmt.Fprintf(out, "Initialized: %s\n", time.Unix(0, initializedAt).Format("2006-01-02"))
	}
	fmt.Fprintln(out)

	fmt.Fprintf(out, "Blobs: %d\n", blobCount)
	if blobCount > 0 {
		kinds := make([]string, 0, len(byKind))
		for k := range byKind {
			kinds = append(kinds, k)
		}
		sort.Slice(kinds, func(i, j int) bool { return byKind[kinds[i]] > byKind[kinds[j]] })
		for _, k := range kinds {
			fmt.Fprintf(out, "  %-16s %d\n", k+":", byKind[k])
		}
		fmt.Fprintf(out, "  (%d AgentTruth, %d LoreInferred)\n", byTrust[2], byTrust[4])
	}
	fmt.Fprintln(out)

	fmt.Fprintf(out, "Subsystems (Nodes): %d\n", nodeCount)
	for _, nr := range nodeRows {
		noun := "blobs"
		if nr.BlobCount == 1 {
			noun = "blob"
		}
		fmt.Fprintf(out, "  %-24s (%d %s, %s)\n", nr.Title, nr.BlobCount, noun, nr.Status)
	}
	if nodeCount > 0 && unassignedCount > 0 {
		fmt.Fprintln(out)
		fmt.Fprintf(out, "Unassigned Blobs: %d\n", unassignedCount)
		fmt.Fprintln(out, "  hint: use 'lore assign <id> <subsystem>' or 'lore node create <name>'")
	}
	fmt.Fprintln(out)

	fmt.Fprintf(out, "Pending Tasks: %d\n", pendingCount)
	fmt.Fprintln(out)

	llmStatus := "not available"
	if llmAvailable {
		llmStatus = "available"
	}
	fmt.Fprintf(out, "LLM: %s (%s)\n", llmProvider, llmStatus)
	return nil
}
