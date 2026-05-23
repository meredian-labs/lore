package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nishchay/lore/internal/graph"
	"github.com/nishchay/lore/internal/node"
	"github.com/nishchay/lore/internal/store"
	"github.com/spf13/cobra"
)

var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Manage subsystem nodes",
}

var nodeCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a subsystem node",
	Args:  cobra.ExactArgs(1),
	RunE:  runNodeCreate,
}

var nodeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all subsystem nodes",
	Args:  cobra.NoArgs,
	RunE:  runNodeList,
}

var nodeShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show blobs assigned to a subsystem node",
	Args:  cobra.ExactArgs(1),
	RunE:  runNodeShow,
}

func init() {
	nodeCmd.AddCommand(nodeCreateCmd, nodeListCmd, nodeShowCmd)
}

// --- cobra wrappers ---

func runNodeCreate(cmd *cobra.Command, args []string) error {
	loreRoot, err := findLoreRoot()
	if err != nil {
		return err
	}
	s, err := store.Open(filepath.Join(loreRoot, "lore.db"))
	if err != nil {
		return err
	}
	defer s.Close()
	return nodeCreate(cmd.Context(), cmd.OutOrStdout(), s, graph.New(s), args[0])
}

func runNodeList(cmd *cobra.Command, args []string) error {
	loreRoot, err := findLoreRoot()
	if err != nil {
		return err
	}
	s, err := store.Open(filepath.Join(loreRoot, "lore.db"))
	if err != nil {
		return err
	}
	defer s.Close()
	return nodeList(cmd.Context(), cmd.OutOrStdout(), s)
}

func runNodeShow(cmd *cobra.Command, args []string) error {
	loreRoot, err := findLoreRoot()
	if err != nil {
		return err
	}
	s, err := store.Open(filepath.Join(loreRoot, "lore.db"))
	if err != nil {
		return err
	}
	defer s.Close()
	return nodeShow(cmd.Context(), cmd.OutOrStdout(), s, args[0])
}

// --- testable logic ---

func nodeCreate(ctx context.Context, w io.Writer, s *store.Store, g *graph.Builder, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("error: node name cannot be empty")
	}
	if len(name) > 100 {
		return fmt.Errorf("error: node name too long (max 100 characters)")
	}

	if _, err := s.NodeByTitle(ctx, name); err == nil {
		return fmt.Errorf("error: node %q already exists", name)
	}

	now := time.Now().UnixNano()
	n := node.Node{
		ID:        uuid.NewString(),
		Title:     name,
		Status:    "active",
		CreatedBy: "user",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.InsertNode(ctx, n); err != nil {
		return fmt.Errorf("creating node: %w", err)
	}
	if err := g.UpdateFromNode(ctx, n); err != nil {
		return fmt.Errorf("updating graph: %w", err)
	}
	fmt.Fprintf(w, "Created node: %s\n", n.Title)
	return nil
}

func nodeList(ctx context.Context, w io.Writer, s *store.Store) error {
	nodes, err := s.ListNodes(ctx)
	if err != nil {
		return err
	}
	if len(nodes) == 0 {
		fmt.Fprintln(w, "No subsystem nodes yet.")
		fmt.Fprintln(w, "hint: use 'lore node create <name>' to create one")
		return nil
	}

	fmt.Fprintf(w, "Subsystems (%d):\n\n", len(nodes))
	for _, n := range nodes {
		count, _ := s.NodeBlobCount(ctx, n.ID)
		noun := "blobs"
		if count == 1 {
			noun = "blob"
		}
		fmt.Fprintf(w, "  %-24s  %3d %-5s  %s\n", n.Title, count, noun, n.Status)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "hint: use 'lore assign <blob-id> <subsystem>' to assign unassigned blobs")
	fmt.Fprintln(w, "      use 'lore node show <name>' to see blobs in a subsystem")
	return nil
}

func nodeShow(ctx context.Context, w io.Writer, s *store.Store, name string) error {
	n, err := s.NodeByTitle(ctx, name)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("error: no node %q\nhint: run 'lore node list' to see available nodes", name)
		}
		return err
	}

	blobs, err := s.BlobsForNode(ctx, n.ID)
	if err != nil {
		return err
	}

	desc := n.Description
	if desc == "" {
		desc = "(none)"
	}
	fmt.Fprintf(w, "Node: %s\n", n.Title)
	fmt.Fprintf(w, "Description: %s\n", desc)
	fmt.Fprintf(w, "Status: %s\n", n.Status)
	fmt.Fprintf(w, "Blobs: %d\n", len(blobs))

	if len(blobs) > 0 {
		fmt.Fprintln(w)
		for _, b := range blobs {
			count, _ := s.BlobFileCount(ctx, b.ID)
			printLogLine(w, b, count)
		}
	}
	return nil
}
