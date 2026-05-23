package graph

import (
	"context"
	"path/filepath"

	"github.com/nishchay/lore/internal/blob"
	"github.com/nishchay/lore/internal/node"
	"github.com/nishchay/lore/internal/store"
)

// Builder derives graph nodes and edges from Blobs and writes them to the store.
type Builder struct {
	s *store.Store
}

// New creates a Builder backed by the given store.
func New(s *store.Store) *Builder {
	return &Builder{s: s}
}

// UpdateFromBlob inserts graph nodes and edges for the given Blob.
func (b *Builder) UpdateFromBlob(ctx context.Context, bl blob.Blob) error {
	blobNodeID, err := b.s.UpsertGraphNode(ctx, store.GraphNode{
		Kind:  "Blob",
		Label: bl.Title,
		Ref:   bl.ID,
	})
	if err != nil {
		return err
	}

	if bl.CommitStart != "" {
		ref := bl.CommitStart
		label := ref
		if len(label) > 7 {
			label = label[:7]
		}
		commitID, err := b.s.UpsertGraphNode(ctx, store.GraphNode{
			Kind:  "Commit",
			Label: label,
			Ref:   ref,
		})
		if err != nil {
			return err
		}
		if err := b.s.UpsertGraphEdge(ctx, store.GraphEdge{
			FromID:   blobNodeID,
			ToID:     commitID,
			Relation: "Produced",
		}); err != nil {
			return err
		}
	}

	files, err := b.s.BlobFiles(ctx, bl.ID)
	if err != nil {
		return err
	}
	for _, f := range files {
		fileID, err := b.s.UpsertGraphNode(ctx, store.GraphNode{
			Kind:  "File",
			Label: filepath.Base(f.Path),
			Ref:   f.Path,
		})
		if err != nil {
			return err
		}
		relation := "Modified"
		if f.Role == "deleted" {
			relation = "Deleted"
		}
		if err := b.s.UpsertGraphEdge(ctx, store.GraphEdge{
			FromID:   blobNodeID,
			ToID:     fileID,
			Relation: relation,
		}); err != nil {
			return err
		}
	}

	for _, tag := range bl.Tags {
		conceptID, err := b.s.UpsertGraphNode(ctx, store.GraphNode{
			Kind:  "Concept",
			Label: tag,
			Ref:   tag,
		})
		if err != nil {
			return err
		}
		if err := b.s.UpsertGraphEdge(ctx, store.GraphEdge{
			FromID:   blobNodeID,
			ToID:     conceptID,
			Relation: "Involves",
		}); err != nil {
			return err
		}
	}

	return nil
}

// UpdateFromNode upserts a Topic graph node for the given subsystem Node.
func (b *Builder) UpdateFromNode(ctx context.Context, n node.Node) error {
	_, err := b.s.UpsertGraphNode(ctx, store.GraphNode{
		Kind:  "Topic",
		Label: n.Title,
		Ref:   n.ID,
	})
	return err
}

// UpdateAssignment creates a Contains edge from the Topic (node) to the Blob
// in the knowledge graph, reflecting a human assignment.
func (b *Builder) UpdateAssignment(ctx context.Context, blobID, nodeID string) error {
	n, err := b.s.NodeByID(ctx, nodeID)
	if err != nil {
		return err
	}
	bl, err := b.s.BlobByID(ctx, blobID)
	if err != nil {
		return err
	}

	topicID, err := b.s.UpsertGraphNode(ctx, store.GraphNode{
		Kind:  "Topic",
		Label: n.Title,
		Ref:   nodeID,
	})
	if err != nil {
		return err
	}
	blobGNodeID, err := b.s.UpsertGraphNode(ctx, store.GraphNode{
		Kind:  "Blob",
		Label: bl.Title,
		Ref:   blobID,
	})
	if err != nil {
		return err
	}
	return b.s.UpsertGraphEdge(ctx, store.GraphEdge{
		FromID:   topicID,
		ToID:     blobGNodeID,
		Relation: "Contains",
	})
}
