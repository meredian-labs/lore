package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/nishchay/lore/internal/graph"
)

func TestNodeCreate_Succeeds(t *testing.T) {
	_, _, s := setupTestRepo(t)
	var buf bytes.Buffer
	if err := nodeCreate(context.Background(), &buf, s, graph.New(s), "Authentication"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "Authentication") {
		t.Errorf("expected output to mention node name, got: %s", buf.String())
	}
}

func TestNodeCreate_DuplicateTitle_Errors(t *testing.T) {
	_, _, s := setupTestRepo(t)
	ctx := context.Background()
	g := graph.New(s)
	if err := nodeCreate(ctx, &bytes.Buffer{}, s, g, "Billing"); err != nil {
		t.Fatalf("first create: %v", err)
	}
	err := nodeCreate(ctx, &bytes.Buffer{}, s, g, "Billing")
	if err == nil {
		t.Fatal("expected error for duplicate node title")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNodeCreate_DuplicateTitle_CaseInsensitive(t *testing.T) {
	_, _, s := setupTestRepo(t)
	ctx := context.Background()
	g := graph.New(s)
	if err := nodeCreate(ctx, &bytes.Buffer{}, s, g, "Auth"); err != nil {
		t.Fatalf("first create: %v", err)
	}
	err := nodeCreate(ctx, &bytes.Buffer{}, s, g, "auth")
	if err == nil {
		t.Fatal("expected error for duplicate node title (case-insensitive)")
	}
}

func TestNodeCreate_EmptyName_Errors(t *testing.T) {
	_, _, s := setupTestRepo(t)
	err := nodeCreate(context.Background(), &bytes.Buffer{}, s, graph.New(s), "   ")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestNodeCreate_NameTooLong_Errors(t *testing.T) {
	_, _, s := setupTestRepo(t)
	long := strings.Repeat("x", 101)
	err := nodeCreate(context.Background(), &bytes.Buffer{}, s, graph.New(s), long)
	if err == nil {
		t.Fatal("expected error for name > 100 chars")
	}
}

func TestNodeCreate_CreatesTopicGraphNode(t *testing.T) {
	_, _, s := setupTestRepo(t)
	if err := nodeCreate(context.Background(), &bytes.Buffer{}, s, graph.New(s), "SessionMgmt"); err != nil {
		t.Fatalf("create: %v", err)
	}
	count, err := s.GraphNodeCount(context.Background())
	if err != nil {
		t.Fatalf("graph count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 graph node, got %d", count)
	}
}

func TestNodeList_Empty(t *testing.T) {
	_, _, s := setupTestRepo(t)
	var buf bytes.Buffer
	if err := nodeList(context.Background(), &buf, s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "No subsystem nodes") {
		t.Errorf("expected empty hint, got: %s", buf.String())
	}
}

func TestNodeList_ShowsNodes(t *testing.T) {
	_, _, s := setupTestRepo(t)
	ctx := context.Background()
	g := graph.New(s)
	nodeCreate(ctx, &bytes.Buffer{}, s, g, "Auth")
	nodeCreate(ctx, &bytes.Buffer{}, s, g, "Billing")

	var buf bytes.Buffer
	if err := nodeList(ctx, &buf, s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Auth") || !strings.Contains(out, "Billing") {
		t.Errorf("expected both node names, got: %s", out)
	}
}

func TestNodeShow_UnknownNode_Errors(t *testing.T) {
	_, _, s := setupTestRepo(t)
	err := nodeShow(context.Background(), &bytes.Buffer{}, s, "NonExistent")
	if err == nil {
		t.Fatal("expected error for unknown node")
	}
	if !strings.Contains(err.Error(), "no node") {
		t.Errorf("unexpected error: %v", err)
	}
}
