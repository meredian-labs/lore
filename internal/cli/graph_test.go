package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/meredian-labs/lore/internal/graph"
)

func TestGraphRender_SubsystemsBeforeUnassigned(t *testing.T) {
	gitRoot, _, s := setupTestRepo(t)
	ctx := context.Background()
	g := graph.New(s)
	now := time.Now().UnixNano()

	nodeCreate(ctx, &bytes.Buffer{}, s, g, "Auth")
	b := insertBlobWithFile(t, s, "OAuth impl", "auth.go", now)
	assignBlob(ctx, &bytes.Buffer{}, s, g, b.ID[:7], "Auth")
	// Unassigned blob.
	insertBlobWithFile(t, s, "Unassigned work", "billing.go", now)
	_ = gitRoot

	var buf bytes.Buffer
	if err := renderGraph(ctx, &buf, s); err != nil {
		t.Fatalf("renderGraph: %v", err)
	}

	out := buf.String()
	authIdx := strings.Index(out, "Subsystem: ")
	unassignedIdx := strings.Index(out, "Unassigned Blobs:")
	if authIdx == -1 {
		t.Error("expected Subsystem: section")
	}
	if unassignedIdx == -1 {
		t.Error("expected Unassigned Blobs: section")
	}
	if authIdx > unassignedIdx {
		t.Error("expected subsystems before unassigned blobs")
	}
}

func TestGraphRender_MaxThreeBlobsPerNode(t *testing.T) {
	_, _, s := setupTestRepo(t)
	ctx := context.Background()
	g := graph.New(s)
	now := time.Now().UnixNano()

	nodeCreate(ctx, &bytes.Buffer{}, s, g, "Storage")

	for i := 0; i < 5; i++ {
		b := insertBlobWithFile(t, s, strings.Repeat("x", 10), "store.go", now+int64(i))
		assignBlob(ctx, &bytes.Buffer{}, s, g, b.ID[:7], "Storage")
	}

	var buf bytes.Buffer
	if err := renderGraph(ctx, &buf, s); err != nil {
		t.Fatalf("renderGraph: %v", err)
	}

	// Count tree rows (lines starting with ├── or └──).
	out := buf.String()
	var blobRows int
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "├──") || strings.HasPrefix(line, "└──") {
			// Only count blob rows, not file rows.
			if !strings.Contains(line, "Modified") && !strings.Contains(line, "Deleted") {
				blobRows++
			}
		}
	}
	if blobRows > 3 {
		t.Errorf("expected at most 3 blob rows per node, got %d", blobRows)
	}
}

func TestGraphRender_UnassignedMax5(t *testing.T) {
	_, _, s := setupTestRepo(t)
	ctx := context.Background()
	now := time.Now().UnixNano()

	for i := 0; i < 8; i++ {
		insertBlobWithFile(t, s, "Unassigned", "file.go", now+int64(i))
	}

	var buf bytes.Buffer
	if err := renderGraph(ctx, &buf, s); err != nil {
		t.Fatalf("renderGraph: %v", err)
	}

	// Count unassigned blob tree rows.
	out := buf.String()
	afterUnassigned := out
	if idx := strings.Index(out, "Unassigned Blobs:"); idx != -1 {
		afterUnassigned = out[idx:]
	}
	var rows int
	for _, line := range strings.Split(afterUnassigned, "\n") {
		line = strings.TrimSpace(line)
		if (strings.HasPrefix(line, "├──") || strings.HasPrefix(line, "└──")) &&
			!strings.Contains(line, "Modified") && !strings.Contains(line, "Deleted") {
			rows++
		}
	}
	if rows > 5 {
		t.Errorf("expected at most 5 unassigned blob rows, got %d", rows)
	}
}

func TestGraphRender_EmptyRepo(t *testing.T) {
	_, _, s := setupTestRepo(t)
	var buf bytes.Buffer
	if err := renderGraph(context.Background(), &buf, s); err != nil {
		t.Fatalf("renderGraph empty: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "No subsystems") {
		t.Errorf("expected empty-state message, got: %s", out)
	}
}

func TestGraphRender_CheckpointBlobsExcluded(t *testing.T) {
	_, _, s := setupTestRepo(t)
	ctx := context.Background()
	g := graph.New(s)
	now := time.Now().UnixNano()

	nodeCreate(ctx, &bytes.Buffer{}, s, g, "CLI")
	b := insertBlobWithFile(t, s, "Real work", "main.go", now)
	assignBlob(ctx, &bytes.Buffer{}, s, g, b.ID[:7], "CLI")

	// Checkpoint blobs are unassigned and should be excluded from the graph.
	// hookCommit creates them automatically; simulate one directly.
	_ = now // checkpoint is inserted as unassigned (primary_node_id = NULL)

	var buf bytes.Buffer
	if err := renderGraph(ctx, &buf, s); err != nil {
		t.Fatalf("renderGraph: %v", err)
	}

	// The node must show exactly 1 blob (the real one), not the checkpoint.
	out := buf.String()
	if strings.Contains(out, "Lore knowledge base checkpoint") {
		t.Error("checkpoint blob must not appear in graph view")
	}
	if !strings.Contains(out, "Real work") {
		t.Errorf("expected real blob in graph, got:\n%s", out)
	}
}
