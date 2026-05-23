package graph_test

import (
	"context"
	"testing"

	"github.com/nishchay/lore/internal/blob"
	"github.com/nishchay/lore/internal/graph"
	"github.com/nishchay/lore/internal/store"
)

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("opening store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func insertTestBlob(t *testing.T, s *store.Store, b blob.Blob, files []store.BlobFile) {
	t.Helper()
	if err := s.InsertBlobWithRelations(context.Background(), b, files, nil, nil); err != nil {
		t.Fatalf("inserting blob: %v", err)
	}
}

func TestUpdateFromBlob_CreatesBlobNode(t *testing.T) {
	s := openTestStore(t)
	b := blob.Blob{
		ID: "blob-1", Kind: blob.KindFeature, Title: "OAuth impl",
		TrustLevel: 4, AISource: "lore:heuristic",
		StartedAt: 100, EndedAt: 200, CreatedAt: 200,
	}
	insertTestBlob(t, s, b, nil)

	builder := graph.New(s)
	if err := builder.UpdateFromBlob(context.Background(), b); err != nil {
		t.Fatalf("UpdateFromBlob: %v", err)
	}

	n, err := s.GraphNodeCount(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Error("expected at least one graph node")
	}
}

func TestUpdateFromBlob_FileEdges(t *testing.T) {
	s := openTestStore(t)
	b := blob.Blob{
		ID: "blob-2", Kind: blob.KindFeature, Title: "Add files",
		TrustLevel: 4, AISource: "lore:heuristic",
		StartedAt: 100, EndedAt: 200, CommitStart: "abc123", CreatedAt: 200,
	}
	files := []store.BlobFile{
		{BlobID: "blob-2", Path: "internal/auth/oauth.go", Role: "written"},
		{BlobID: "blob-2", Path: "old.go", Role: "deleted"},
	}
	insertTestBlob(t, s, b, files)

	builder := graph.New(s)
	if err := builder.UpdateFromBlob(context.Background(), b); err != nil {
		t.Fatalf("UpdateFromBlob: %v", err)
	}

	edgeCount, err := s.GraphEdgeCount(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Expect: Produced (commit) + Modified (written file) + Deleted (deleted file) = 3
	if edgeCount < 3 {
		t.Errorf("expected at least 3 edges (Produced+Modified+Deleted), got %d", edgeCount)
	}
}

func TestUpdateFromBlob_ConceptNodes(t *testing.T) {
	s := openTestStore(t)
	b := blob.Blob{
		ID: "blob-3", Kind: blob.KindFeature, Title: "Tagged blob",
		Tags: []string{"auth", "oauth"},
		TrustLevel: 2, AISource: "agent:claude",
		StartedAt: 100, EndedAt: 200, CreatedAt: 200,
	}
	insertTestBlob(t, s, b, nil)

	builder := graph.New(s)
	if err := builder.UpdateFromBlob(context.Background(), b); err != nil {
		t.Fatalf("UpdateFromBlob: %v", err)
	}

	nodeCount, _ := s.GraphNodeCount(context.Background())
	// Blob node + 2 concept nodes = at least 3
	if nodeCount < 3 {
		t.Errorf("expected at least 3 nodes (blob + 2 concepts), got %d", nodeCount)
	}
}

func TestUpdateFromBlob_Idempotent(t *testing.T) {
	s := openTestStore(t)
	b := blob.Blob{
		ID: "blob-4", Kind: blob.KindBugFix, Title: "Fix bug",
		CommitStart: "def456",
		TrustLevel: 4, AISource: "lore:heuristic",
		StartedAt: 100, EndedAt: 200, CreatedAt: 200,
	}
	insertTestBlob(t, s, b, nil)

	builder := graph.New(s)
	// Call twice — nodes must not duplicate, edge weight should increment.
	for i := 0; i < 2; i++ {
		if err := builder.UpdateFromBlob(context.Background(), b); err != nil {
			t.Fatalf("UpdateFromBlob call %d: %v", i+1, err)
		}
	}

	nodeCount, _ := s.GraphNodeCount(context.Background())
	// Blob + Commit = 2 nodes, not 4 after two calls.
	if nodeCount != 2 {
		t.Errorf("expected 2 nodes after idempotent call, got %d", nodeCount)
	}

	edgeCount, _ := s.GraphEdgeCount(context.Background())
	// One Produced edge (not two), weight incremented to 2 internally.
	if edgeCount != 1 {
		t.Errorf("expected 1 edge (upserted), got %d", edgeCount)
	}
}

func TestUpdateFromBlob_EdgeWeightIncrement(t *testing.T) {
	s := openTestStore(t)
	// Two different blobs both produced the same commit — weight should be 2.
	b1 := blob.Blob{
		ID: "blob-w1", Kind: blob.KindFeature, Title: "First",
		CommitStart: "shared-sha",
		TrustLevel: 4, AISource: "lore:heuristic",
		StartedAt: 100, EndedAt: 200, CreatedAt: 200,
	}
	b2 := blob.Blob{
		ID: "blob-w2", Kind: blob.KindBugFix, Title: "Second",
		CommitStart: "shared-sha",
		TrustLevel: 4, AISource: "lore:heuristic",
		StartedAt: 200, EndedAt: 300, CreatedAt: 300,
	}
	insertTestBlob(t, s, b1, nil)
	insertTestBlob(t, s, b2, nil)

	builder := graph.New(s)
	if err := builder.UpdateFromBlob(context.Background(), b1); err != nil {
		t.Fatalf("UpdateFromBlob b1: %v", err)
	}
	if err := builder.UpdateFromBlob(context.Background(), b2); err != nil {
		t.Fatalf("UpdateFromBlob b2: %v", err)
	}

	// Commit node is shared — one Commit graph node exists.
	nodeCount, _ := s.GraphNodeCount(context.Background())
	// blob-w1 node + blob-w2 node + shared-sha commit node = 3
	if nodeCount != 3 {
		t.Errorf("expected 3 graph nodes (2 blobs + 1 shared commit), got %d", nodeCount)
	}
}
