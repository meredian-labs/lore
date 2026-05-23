package blob

import (
	"context"
	"errors"
	"testing"

	"github.com/nishchay/lore/internal/config"
	"github.com/nishchay/lore/internal/task"
)

// fakeStore satisfies blob.Storer for testing.
type fakeStore struct {
	tasks   []task.Task
	blobs   []Blob
	files   []BlobFile
	cmds    []BlobCommand
	taskIDs []string
	err     error
}

func (f *fakeStore) PendingTasks(ctx context.Context) ([]task.Task, error) {
	return f.tasks, f.err
}

func (f *fakeStore) InsertBlobWithRelations(ctx context.Context, b Blob, files []BlobFile, cmds []BlobCommand, taskIDs []string) error {
	f.blobs = append(f.blobs, b)
	f.files = append(f.files, files...)
	f.cmds = append(f.cmds, cmds...)
	f.taskIDs = append(f.taskIDs, taskIDs...)
	return nil
}

// fakeGraphUpdater satisfies blob.GraphUpdater for testing.
type fakeGraphUpdater struct {
	blobs []Blob
	err   error
}

func (g *fakeGraphUpdater) UpdateFromBlob(ctx context.Context, b Blob) error {
	g.blobs = append(g.blobs, b)
	return g.err
}

func TestExtractIfReady_NoTasks_NoBlob(t *testing.T) {
	s := &fakeStore{}
	g := &fakeGraphUpdater{}
	if err := ExtractIfReady(context.Background(), s, g, config.Config{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.blobs) != 0 {
		t.Errorf("expected no blob for empty tasks, got %d", len(s.blobs))
	}
}

func TestExtractIfReady_NoCommit_NoBlob(t *testing.T) {
	s := &fakeStore{
		tasks: []task.Task{
			{ID: "a", Kind: task.KindFileWrite, Path: "x.go", TS: 100},
		},
	}
	// No commit task — should not extract.
	if err := ExtractIfReady(context.Background(), s, nil, config.Config{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.blobs) != 0 {
		t.Errorf("expected no blob without commit, got %d", len(s.blobs))
	}
}

func TestExtractIfReady_HeuristicPath(t *testing.T) {
	s := &fakeStore{
		tasks: []task.Task{
			{ID: "a", Kind: task.KindCommitCreated, Detail: "sha1|feat: add login", Source: "hook", TS: 100},
			{ID: "b", Kind: task.KindFileWrite, Path: "auth.go", Source: "hook", TS: 90},
		},
	}
	g := &fakeGraphUpdater{}
	cfg := config.Config{} // no LLM endpoint

	if err := ExtractIfReady(context.Background(), s, g, cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.blobs) != 1 {
		t.Fatalf("expected 1 blob, got %d", len(s.blobs))
	}
	b := s.blobs[0]
	if b.TrustLevel != 4 {
		t.Errorf("TrustLevel = %d, want 4 (heuristic)", b.TrustLevel)
	}
	if b.AISource != "lore:heuristic" {
		t.Errorf("AISource = %q, want lore:heuristic", b.AISource)
	}
	if len(s.taskIDs) != 2 {
		t.Errorf("taskIDs = %d, want 2", len(s.taskIDs))
	}
	// Graph updater must be called.
	if len(g.blobs) != 1 {
		t.Errorf("graph updater called %d times, want 1", len(g.blobs))
	}
}

func TestExtractIfReady_AgentRecapPath(t *testing.T) {
	recapJSON := `{"user_intent":"Add login","summary":"Login added.","recap":"Users can log in.","kind":"Feature","tags":["auth"]}`
	s := &fakeStore{
		tasks: []task.Task{
			{ID: "a", Kind: task.KindCommitCreated, Detail: "sha1|feat: login", Source: "hook", TS: 100},
			{ID: "b", Kind: task.KindAgentRecap, Detail: recapJSON, Source: "agent:claude", TS: 110},
		},
	}
	g := &fakeGraphUpdater{}
	if err := ExtractIfReady(context.Background(), s, g, config.Config{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.blobs) != 1 {
		t.Fatalf("expected 1 blob, got %d", len(s.blobs))
	}
	b := s.blobs[0]
	if b.TrustLevel != 2 {
		t.Errorf("TrustLevel = %d, want 2 (agent recap)", b.TrustLevel)
	}
	if b.AISource != "agent:claude" {
		t.Errorf("AISource = %q, want agent:claude", b.AISource)
	}
}

func TestExtractIfReady_StoreError_Propagates(t *testing.T) {
	s := &fakeStore{err: errors.New("db error")}
	err := ExtractIfReady(context.Background(), s, nil, config.Config{})
	if err == nil {
		t.Error("expected error from store, got nil")
	}
}

func TestExtractIfReady_NilGraphUpdater_OK(t *testing.T) {
	s := &fakeStore{
		tasks: []task.Task{
			{ID: "a", Kind: task.KindCommitCreated, Detail: "sha1|chore: cleanup", Source: "hook", TS: 100},
		},
	}
	// g=nil must not panic
	if err := ExtractIfReady(context.Background(), s, nil, config.Config{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
