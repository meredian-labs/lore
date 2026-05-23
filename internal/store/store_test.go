package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nishchay/lore/internal/blob"
	"github.com/nishchay/lore/internal/node"
	"github.com/nishchay/lore/internal/task"
)

func openMemory(t *testing.T) *Store {
	t.Helper()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open :memory: store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func makeTask(kind task.TaskKind) task.Task {
	return task.Task{
		ID:         uuid.NewString(),
		Kind:       kind,
		Detail:     "test detail",
		Source:     "hook",
		TrustLevel: 1,
		TS:         time.Now().UnixNano(),
	}
}

func makeBlob(id string, kind blob.BlobKind) blob.Blob {
	return blob.Blob{
		ID:         id,
		Kind:       kind,
		Title:      "Test blob " + id,
		Summary:    "summary",
		TrustLevel: 4,
		AISource:   "lore:heuristic",
		StartedAt:  time.Now().Add(-time.Hour).UnixNano(),
		EndedAt:    time.Now().UnixNano(),
		Tags:       []string{"test"},
		CreatedAt:  time.Now().UnixNano(),
	}
}

var ctx = context.Background()

func TestInsertAndFetchTask(t *testing.T) {
	s := openMemory(t)
	tk := makeTask(task.KindCommitCreated)
	if err := s.InsertTask(ctx, tk); err != nil {
		t.Fatal(err)
	}
	pending, err := s.PendingTasks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending task, got %d", len(pending))
	}
	if pending[0].ID != tk.ID {
		t.Errorf("expected task ID %s, got %s", tk.ID, pending[0].ID)
	}
	if pending[0].Kind != task.KindCommitCreated {
		t.Errorf("expected kind CommitCreated, got %s", pending[0].Kind)
	}
}

func TestMarkTasksExtracted(t *testing.T) {
	s := openMemory(t)
	tk := makeTask(task.KindFileWrite)
	if err := s.InsertTask(ctx, tk); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkTasksExtracted(ctx, "blob-1", []string{tk.ID}); err != nil {
		t.Fatal(err)
	}
	pending, err := s.PendingTasks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Errorf("expected 0 pending tasks after mark, got %d", len(pending))
	}
}

func TestPurgeExtractedTasks(t *testing.T) {
	s := openMemory(t)
	tk := makeTask(task.KindNote)
	tk.TS = time.Now().Add(-2 * time.Hour).UnixNano()
	if err := s.InsertTask(ctx, tk); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkTasksExtracted(ctx, "blob-purge", []string{tk.ID}); err != nil {
		t.Fatal(err)
	}
	// Purge tasks extracted more than 0 seconds ago (all extracted tasks).
	if err := s.PurgeExtractedTasks(ctx, 0); err != nil {
		t.Fatal(err)
	}
	pending, err := s.PendingTasks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Errorf("expected 0 tasks after purge, got %d", len(pending))
	}
}

func TestInsertBlobAndFetch(t *testing.T) {
	s := openMemory(t)
	id := uuid.NewString()
	b := makeBlob(id, blob.KindFeature)
	b.Tags = []string{"auth", "oauth"}
	files := []BlobFile{{BlobID: id, Path: "internal/auth/oauth.go", Role: "written"}}
	cmds := []BlobCommand{{BlobID: id, Command: "go test ./...", TS: 1}}

	if err := s.InsertBlobWithRelations(ctx, b, files, cmds, nil); err != nil {
		t.Fatal(err)
	}
	got, err := s.BlobByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != id {
		t.Errorf("expected ID %s, got %s", id, got.ID)
	}
	if got.Kind != blob.KindFeature {
		t.Errorf("expected Feature, got %s", got.Kind)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "auth" {
		t.Errorf("tags round-trip failed: %v", got.Tags)
	}
}

func TestBlobsByFile(t *testing.T) {
	s := openMemory(t)
	id := uuid.NewString()
	b := makeBlob(id, blob.KindFeature)
	files := []BlobFile{{BlobID: id, Path: "internal/auth/oauth.go", Role: "written"}}
	if err := s.InsertBlobWithRelations(ctx, b, files, nil, nil); err != nil {
		t.Fatal(err)
	}
	blobs, err := s.BlobsByFile(ctx, "internal/auth/oauth.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(blobs) != 1 || blobs[0].ID != id {
		t.Errorf("expected 1 blob, got %d", len(blobs))
	}
}

func TestBlobsByFile_SuffixMatch(t *testing.T) {
	s := openMemory(t)
	id := uuid.NewString()
	b := makeBlob(id, blob.KindFeature)
	files := []BlobFile{{BlobID: id, Path: "internal/auth/oauth.go", Role: "written"}}
	if err := s.InsertBlobWithRelations(ctx, b, files, nil, nil); err != nil {
		t.Fatal(err)
	}
	// Query by filename only — should match via suffix.
	blobs, err := s.BlobsByFile(ctx, "oauth.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(blobs) != 1 || blobs[0].ID != id {
		t.Errorf("suffix match failed: got %d blobs", len(blobs))
	}
}

func TestBlobLog_OrderNewestFirst(t *testing.T) {
	s := openMemory(t)
	now := time.Now().UnixNano()

	older := makeBlob(uuid.NewString(), blob.KindFeature)
	older.EndedAt = now - int64(2*time.Hour)
	newer := makeBlob(uuid.NewString(), blob.KindBugFix)
	newer.EndedAt = now - int64(time.Hour)

	for _, b := range []blob.Blob{older, newer} {
		if err := s.InsertBlobWithRelations(ctx, b, nil, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	blobs, err := s.BlobLog(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(blobs) != 2 {
		t.Fatalf("expected 2 blobs, got %d", len(blobs))
	}
	if blobs[0].ID != newer.ID {
		t.Errorf("expected newest first: got %s, want %s", blobs[0].ID, newer.ID)
	}
}

func TestInsertNode_UniqueTitle(t *testing.T) {
	s := openMemory(t)
	now := time.Now().UnixNano()
	n := node.Node{
		ID: uuid.NewString(), Title: "Authentication",
		Status: "active", CreatedBy: "user", CreatedAt: now, UpdatedAt: now,
	}
	if err := s.InsertNode(ctx, n); err != nil {
		t.Fatal(err)
	}
	n2 := n
	n2.ID = uuid.NewString()
	if err := s.InsertNode(ctx, n2); err != ErrAlreadyExists {
		t.Errorf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestSetBlobNode(t *testing.T) {
	s := openMemory(t)
	now := time.Now().UnixNano()
	n := node.Node{
		ID: uuid.NewString(), Title: "Billing",
		Status: "active", CreatedBy: "user", CreatedAt: now, UpdatedAt: now,
	}
	if err := s.InsertNode(ctx, n); err != nil {
		t.Fatal(err)
	}
	blobID := uuid.NewString()
	b := makeBlob(blobID, blob.KindFeature)
	if err := s.InsertBlobWithRelations(ctx, b, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.SetBlobNode(ctx, blobID, n.ID); err != nil {
		t.Fatal(err)
	}
	got, err := s.BlobByID(ctx, blobID)
	if err != nil {
		t.Fatal(err)
	}
	if got.PrimaryNodeID != n.ID {
		t.Errorf("expected primary_node_id=%s, got %s", n.ID, got.PrimaryNodeID)
	}
}

func TestUpsertGraphEdge_Weight(t *testing.T) {
	s := openMemory(t)
	fromID, _ := s.UpsertGraphNode(ctx, GraphNode{Kind: "Blob", Label: "A", Ref: "ref-a"})
	toID, _ := s.UpsertGraphNode(ctx, GraphNode{Kind: "File", Label: "f.go", Ref: "f.go"})

	e := GraphEdge{FromID: fromID, ToID: toID, Relation: "Modified"}
	if err := s.UpsertGraphEdge(ctx, e); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertGraphEdge(ctx, e); err != nil {
		t.Fatal(err)
	}
	edges, err := s.GraphEdgesFrom(ctx, fromID)
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	if edges[0].Weight != 2 {
		t.Errorf("expected weight 2, got %d", edges[0].Weight)
	}
}

func TestMigration_RunsInOrder(t *testing.T) {
	s := openMemory(t)
	val, err := s.GetMeta(ctx, "schema_version")
	if err != nil {
		t.Fatal(err)
	}
	if val != "4" {
		t.Errorf("expected schema_version=4, got %s", val)
	}
}

func TestMigration_Idempotent(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "lore-*.db")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	f.Close()

	s1, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	s1.Close()

	// Second open — migrations must not re-run or fail.
	s2, err := Open(path)
	if err != nil {
		t.Fatalf("second open failed: %v", err)
	}
	defer s2.Close()

	val, err := s2.GetMeta(ctx, "schema_version")
	if err != nil {
		t.Fatal(err)
	}
	if val != "4" {
		t.Errorf("expected schema_version=4 after second open, got %s", val)
	}
}

func TestBlobCountByKind(t *testing.T) {
	s := openMemory(t)
	for range 2 {
		b := makeBlob(uuid.NewString(), blob.KindFeature)
		s.InsertBlobWithRelations(ctx, b, nil, nil, nil)
	}
	b := makeBlob(uuid.NewString(), blob.KindBugFix)
	s.InsertBlobWithRelations(ctx, b, nil, nil, nil)

	counts, err := s.BlobCountByKind(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if counts["Feature"] != 2 {
		t.Errorf("expected Feature=2, got %d", counts["Feature"])
	}
	if counts["BugFix"] != 1 {
		t.Errorf("expected BugFix=1, got %d", counts["BugFix"])
	}
}

func TestBlobCountByTrust(t *testing.T) {
	s := openMemory(t)
	b1 := makeBlob(uuid.NewString(), blob.KindFeature)
	b1.TrustLevel = 2
	b1.AISource = "agent:claude"
	s.InsertBlobWithRelations(ctx, b1, nil, nil, nil)

	b2 := makeBlob(uuid.NewString(), blob.KindFeature)
	b2.TrustLevel = 4
	s.InsertBlobWithRelations(ctx, b2, nil, nil, nil)

	counts, err := s.BlobCountByTrust(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if counts[2] != 1 {
		t.Errorf("expected trust=2 count 1, got %d", counts[2])
	}
	if counts[4] != 1 {
		t.Errorf("expected trust=4 count 1, got %d", counts[4])
	}
}

func TestResolveBlobIDPrefix_Ambiguous(t *testing.T) {
	s := openMemory(t)
	// Two blobs with the same 7-char prefix.
	prefix := "aabbcc0"
	b1 := makeBlob(prefix+"11111111-1111-1111-1111-111111111111", blob.KindFeature)
	b2 := makeBlob(prefix+"22222222-2222-2222-2222-222222222222", blob.KindFeature)
	s.InsertBlobWithRelations(ctx, b1, nil, nil, nil)
	s.InsertBlobWithRelations(ctx, b2, nil, nil, nil)

	_, err := s.ResolveBlobIDPrefix(ctx, prefix)
	if err != ErrAmbiguous {
		t.Errorf("expected ErrAmbiguous, got %v", err)
	}
}

func TestResolveBlobIDPrefix_NotFound(t *testing.T) {
	s := openMemory(t)
	_, err := s.ResolveBlobIDPrefix(ctx, "zzzzzzz")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestInsertBlobWithRelations_AtomicRollback(t *testing.T) {
	s := openMemory(t)
	id := uuid.NewString()
	b := makeBlob(id, blob.KindFeature)

	// Pre-insert a blob_commands row directly — this will conflict when
	// InsertBlobWithRelations tries to insert the same command for the same blob.
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO blob_commands (blob_id, command, ts) VALUES (?, ?, ?)`,
		id, "go test ./...", 0,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Now call InsertBlobWithRelations: blob INSERT succeeds, file INSERT succeeds,
	// then blob_commands INSERT fails (duplicate primary key) → transaction rolls back.
	files := []BlobFile{{BlobID: id, Path: "file.go", Role: "written"}}
	cmds := []BlobCommand{{BlobID: id, Command: "go test ./...", TS: 0}}
	err = s.InsertBlobWithRelations(ctx, b, files, cmds, nil)
	if err == nil {
		t.Fatal("expected error from duplicate blob_command")
	}

	// Blob should NOT have been committed.
	_, err = s.BlobByID(ctx, id)
	if err != ErrNotFound {
		t.Errorf("blob should not exist after rollback, got err=%v", err)
	}

	// blob_files should NOT have any entry for this blob.
	blobFiles, _ := s.BlobFiles(ctx, id)
	if len(blobFiles) != 0 {
		t.Errorf("blob_files should be empty after rollback, got %d entries", len(blobFiles))
	}
}

func TestBlobByCommitStart(t *testing.T) {
	s := openMemory(t)

	sha := "abc1234567890"
	b := makeBlob(uuid.NewString(), blob.KindFeature)
	b.CommitStart = sha
	if err := s.InsertBlobWithRelations(ctx, b, nil, nil, nil); err != nil {
		t.Fatal(err)
	}

	// Insert a checkpoint with the same SHA — should not be returned.
	cp := makeBlob(uuid.NewString(), blob.KindCheckpoint)
	cp.CommitStart = sha
	if err := s.InsertBlobWithRelations(ctx, cp, nil, nil, nil); err != nil {
		t.Fatal(err)
	}

	got, err := s.BlobByCommitStart(ctx, sha)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected blob, got nil")
	}
	if got.ID != b.ID {
		t.Errorf("got blob %s, want %s", got.ID, b.ID)
	}

	// Unknown SHA returns nil.
	got, err = s.BlobByCommitStart(ctx, "unknown")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("expected nil for unknown SHA, got %+v", got)
	}
}
