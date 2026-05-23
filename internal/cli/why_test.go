package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nishchay/lore/internal/blob"
	"github.com/nishchay/lore/internal/store"
)

func insertBlobWithFile(t *testing.T, s *store.Store, title, filePath string, ts int64) blob.Blob {
	t.Helper()
	b := blob.Blob{
		ID:         uuid.NewString(),
		Kind:       blob.KindFeature,
		Title:      title,
		TrustLevel: 4,
		AISource:   "lore:heuristic",
		StartedAt:  ts,
		EndedAt:    ts,
		CreatedAt:  ts,
	}
	f := store.BlobFile{BlobID: b.ID, Path: filePath, Role: "written"}
	if err := s.InsertBlobWithRelations(context.Background(), b, []store.BlobFile{f}, nil, nil); err != nil {
		t.Fatalf("insertBlobWithFile: %v", err)
	}
	return b
}

func TestWhyQuery_ReturnsNewestFirst(t *testing.T) {
	_, _, s := setupTestRepo(t)
	ctx := context.Background()
	now := time.Now().UnixNano()

	insertBlobWithFile(t, s, "Older blob", "internal/auth/oauth.go", now-2000)
	insertBlobWithFile(t, s, "Newer blob", "internal/auth/oauth.go", now-1000)

	var buf bytes.Buffer
	if err := whyFile(ctx, &buf, s, "oauth.go", false, false); err != nil {
		t.Fatalf("whyFile: %v", err)
	}

	out := buf.String()
	newerIdx := strings.Index(out, "Newer blob")
	olderIdx := strings.Index(out, "Older blob")
	if newerIdx == -1 || olderIdx == -1 {
		t.Fatalf("expected both blobs in output, got:\n%s", out)
	}
	if newerIdx > olderIdx {
		t.Errorf("expected Newer blob before Older blob (newest-first), got:\n%s", out)
	}
}

func TestWhyQuery_SuffixMatch(t *testing.T) {
	_, _, s := setupTestRepo(t)
	now := time.Now().UnixNano()
	insertBlobWithFile(t, s, "Auth impl", "internal/auth/oauth.go", now)

	var buf bytes.Buffer
	// Query by filename only — should match the full path.
	if err := whyFile(context.Background(), &buf, s, "oauth.go", false, false); err != nil {
		t.Fatalf("whyFile suffix: %v", err)
	}
	if !strings.Contains(buf.String(), "Auth impl") {
		t.Errorf("expected suffix match, got: %s", buf.String())
	}
}

func TestWhyQuery_NoResults_ErrorMessage(t *testing.T) {
	_, _, s := setupTestRepo(t)
	err := whyFile(context.Background(), &bytes.Buffer{}, s, "nonexistent.go", false, false)
	if err == nil {
		t.Fatal("expected error for unknown file")
	}
	if !strings.Contains(err.Error(), "no blobs found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTraceQuery_ReturnsOldestFirst(t *testing.T) {
	_, _, s := setupTestRepo(t)
	ctx := context.Background()
	now := time.Now().UnixNano()

	insertBlobWithFile(t, s, "Older blob", "auth.go", now-2000)
	insertBlobWithFile(t, s, "Newer blob", "auth.go", now-1000)

	var buf bytes.Buffer
	if err := whyFile(ctx, &buf, s, "auth.go", true, false); err != nil {
		t.Fatalf("whyFile chron: %v", err)
	}

	out := buf.String()
	olderIdx := strings.Index(out, "Older blob")
	newerIdx := strings.Index(out, "Newer blob")
	if olderIdx == -1 || newerIdx == -1 {
		t.Fatalf("expected both blobs, got:\n%s", out)
	}
	if olderIdx > newerIdx {
		t.Errorf("expected Older blob first (chronological), got:\n%s", out)
	}
}

func TestJSONOutput_BlobWhy_IsArray(t *testing.T) {
	_, _, s := setupTestRepo(t)
	now := time.Now().UnixNano()
	insertBlobWithFile(t, s, "Auth impl", "auth.go", now)

	var buf bytes.Buffer
	if err := whyFile(context.Background(), &buf, s, "auth.go", false, true); err != nil {
		t.Fatalf("whyFile json: %v", err)
	}
	var arr []BlobJSON
	if err := json.Unmarshal(buf.Bytes(), &arr); err != nil {
		t.Fatalf("expected JSON array, got: %s\nerr: %v", buf.String(), err)
	}
	if len(arr) != 1 {
		t.Errorf("expected 1 blob in array, got %d", len(arr))
	}
}
