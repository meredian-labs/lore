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
	"github.com/nishchay/lore/internal/graph"
	"github.com/nishchay/lore/internal/store"
)

func insertTestBlobDirect(t *testing.T, s *store.Store, kind blob.BlobKind, trust int, ts int64) blob.Blob {
	t.Helper()
	b := blob.Blob{
		ID:         uuid.NewString(),
		Kind:       kind,
		Title:      "test " + string(kind),
		TrustLevel: trust,
		AISource:   "lore:heuristic",
		StartedAt:  ts,
		EndedAt:    ts,
		CreatedAt:  ts,
	}
	if err := s.InsertBlobWithRelations(context.Background(), b, nil, nil, nil); err != nil {
		t.Fatalf("insertTestBlobDirect: %v", err)
	}
	return b
}

func TestStatusAggregation_ZeroState(t *testing.T) {
	_, _, s := setupTestRepo(t)
	var buf bytes.Buffer
	// runStatus requires a cobra command; test by calling through the status helper
	// which relies on store methods. We verify no panics and zero counts.
	ctx := context.Background()
	blobCount, err := s.BlobCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if blobCount != 0 {
		t.Errorf("expected 0 blobs, got %d", blobCount)
	}
	_ = buf
}

func TestStatusAggregation_AllCounts(t *testing.T) {
	_, _, s := setupTestRepo(t)
	ctx := context.Background()
	g := graph.New(s)
	now := time.Now().UnixNano()

	insertTestBlobDirect(t, s, blob.KindFeature, 2, now)
	insertTestBlobDirect(t, s, blob.KindFeature, 4, now+1)
	insertTestBlobDirect(t, s, blob.KindBugFix, 4, now+2)
	nodeCreate(ctx, &bytes.Buffer{}, s, g, "Auth")

	blobCount, _ := s.BlobCount(ctx)
	if blobCount != 3 {
		t.Errorf("expected 3 blobs, got %d", blobCount)
	}
	byKind, _ := s.BlobCountByKind(ctx)
	if byKind["Feature"] != 2 {
		t.Errorf("expected 2 Feature blobs, got %d", byKind["Feature"])
	}
	if byKind["BugFix"] != 1 {
		t.Errorf("expected 1 BugFix blob, got %d", byKind["BugFix"])
	}
	byTrust, _ := s.BlobCountByTrust(ctx)
	if byTrust[2] != 1 {
		t.Errorf("expected 1 AgentTruth blob, got %d", byTrust[2])
	}
	if byTrust[4] != 2 {
		t.Errorf("expected 2 LoreInferred blobs, got %d", byTrust[4])
	}
	nodeCount, _ := s.NodeCount(ctx)
	if nodeCount != 1 {
		t.Errorf("expected 1 node, got %d", nodeCount)
	}
}

func TestJSONOutput_BlobLog_IsArray(t *testing.T) {
	_, _, s := setupTestRepo(t)
	now := time.Now().UnixNano()
	insertTestBlobDirect(t, s, blob.KindFeature, 4, now)

	blobs, _ := s.BlobLog(context.Background(), 20)
	var out []BlobJSON
	for _, b := range blobs {
		out = append(out, blobToJSON(b, nil, nil, nil))
	}
	if out == nil {
		out = []BlobJSON{}
	}

	data, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	var arr []map[string]interface{}
	if err := json.Unmarshal(data, &arr); err != nil {
		t.Fatalf("expected JSON array: %v", err)
	}
	if len(arr) != 1 {
		t.Errorf("expected 1 element, got %d", len(arr))
	}
}

func TestJSONOutput_BlobShow_IsObject(t *testing.T) {
	_, _, s := setupTestRepo(t)
	now := time.Now().UnixNano()
	b := insertTestBlobDirect(t, s, blob.KindBugFix, 4, now)

	j := blobToJSON(b, nil, nil, nil)
	data, err := json.Marshal(j)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatalf("expected JSON object: %v", err)
	}
	if obj["kind"] != "BugFix" {
		t.Errorf("expected kind=BugFix, got: %v", obj["kind"])
	}
}

func TestColorDetection_NoColorEnv_DisablesColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	// colorEnabled checks NO_COLOR env var.
	if colorEnabled() {
		t.Error("expected color disabled when NO_COLOR is set")
	}
}

func TestStatusJSON_Output(t *testing.T) {
	_, _, s := setupTestRepo(t)
	ctx := context.Background()
	g := graph.New(s)
	now := time.Now().UnixNano()

	insertTestBlobDirect(t, s, blob.KindFeature, 4, now)
	nodeCreate(ctx, &bytes.Buffer{}, s, g, "Storage")

	blobCount, _ := s.BlobCount(ctx)
	byKind, _ := s.BlobCountByKind(ctx)
	byTrust, _ := s.BlobCountByTrust(ctx)
	nodeCount, _ := s.NodeCount(ctx)
	pendingCount, _ := s.PendingTaskCount(ctx)
	unassignedCount, _ := s.UnassignedBlobCount(ctx)
	nodes, _ := s.ListNodes(ctx)

	var nodeRows []NodeJSON
	for _, n := range nodes {
		cnt, _ := s.NodeBlobCount(ctx, n.ID)
		nodeRows = append(nodeRows, NodeJSON{
			ID: n.ID, Title: n.Title, Status: n.Status, BlobCount: cnt,
		})
	}
	if nodeRows == nil {
		nodeRows = []NodeJSON{}
	}

	j := StatusJSON{
		Repository:      "/test",
		BlobCount:       blobCount,
		BlobsByKind:     byKind,
		BlobsByTrust:    byTrust,
		NodeCount:       nodeCount,
		Nodes:           nodeRows,
		UnassignedBlobs: unassignedCount,
		PendingTasks:    pendingCount,
	}
	if j.BlobsByKind == nil {
		j.BlobsByKind = map[string]int{}
	}
	if j.BlobsByTrust == nil {
		j.BlobsByTrust = map[int]int{}
	}

	data, err := json.Marshal(j)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatalf("expected JSON object: %v", err)
	}
	if obj["blob_count"].(float64) != 1 {
		t.Errorf("expected blob_count=1, got %v", obj["blob_count"])
	}

	out := string(data)
	if !strings.Contains(out, "Storage") {
		t.Errorf("expected node Storage in JSON, got: %s", out)
	}
}
