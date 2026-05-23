package blob

import (
	"testing"

	"github.com/meredian-labs/lore/internal/task"
)

func TestBuildWindow_Empty(t *testing.T) {
	w := BuildWindow(nil)
	if w.StartedAt != 0 || w.EndedAt != 0 || w.HasCommit {
		t.Fatalf("empty window should have zero values, got %+v", w)
	}
}

func TestBuildWindow_TimeBounds(t *testing.T) {
	tasks := []task.Task{
		{ID: "a", Kind: task.KindFileWrite, Path: "main.go", Source: "hook", TS: 200},
		{ID: "b", Kind: task.KindFileWrite, Path: "lib.go", Source: "hook", TS: 100},
		{ID: "c", Kind: task.KindFileWrite, Path: "x.go", Source: "hook", TS: 300},
	}
	w := BuildWindow(tasks)
	if w.StartedAt != 100 {
		t.Errorf("StartedAt = %d, want 100", w.StartedAt)
	}
	if w.EndedAt != 300 {
		t.Errorf("EndedAt = %d, want 300", w.EndedAt)
	}
}

func TestBuildWindow_CommitFields(t *testing.T) {
	tasks := []task.Task{
		{ID: "a", Kind: task.KindCommitCreated, Detail: "sha1|first commit", Source: "hook", TS: 100},
		{ID: "b", Kind: task.KindCommitCreated, Detail: "sha2|second commit", Source: "hook", TS: 200},
	}
	w := BuildWindow(tasks)
	if !w.HasCommit {
		t.Error("HasCommit should be true")
	}
	if w.CommitStart != "sha1" {
		t.Errorf("CommitStart = %q, want sha1", w.CommitStart)
	}
	if w.CommitEnd != "sha2" {
		t.Errorf("CommitEnd = %q, want sha2", w.CommitEnd)
	}
	if len(w.CommitMsgs) != 2 {
		t.Errorf("CommitMsgs len = %d, want 2", len(w.CommitMsgs))
	}
}

func TestBuildWindow_FileDeduplication(t *testing.T) {
	tasks := []task.Task{
		{ID: "a", Kind: task.KindFileWrite, Path: "main.go", Source: "hook", TS: 100},
		{ID: "b", Kind: task.KindFileWrite, Path: "main.go", Source: "hook", TS: 110},
		{ID: "c", Kind: task.KindFileDelete, Path: "old.go", Source: "hook", TS: 120},
	}
	w := BuildWindow(tasks)
	if len(w.FilesWritten) != 1 {
		t.Errorf("FilesWritten len = %d, want 1 (deduped)", len(w.FilesWritten))
	}
	if len(w.FilesDeleted) != 1 {
		t.Errorf("FilesDeleted len = %d, want 1", len(w.FilesDeleted))
	}
}

func TestBuildWindow_RecapTask_Latest(t *testing.T) {
	tasks := []task.Task{
		{ID: "a", Kind: task.KindAgentRecap, Detail: `{"kind":"Feature"}`, Source: "agent:claude", TS: 100},
		{ID: "b", Kind: task.KindAgentRecap, Detail: `{"kind":"BugFix"}`, Source: "agent:claude", TS: 200},
	}
	w := BuildWindow(tasks)
	if w.RecapTask == nil {
		t.Fatal("RecapTask should not be nil")
	}
	if w.RecapTask.TS != 200 {
		t.Errorf("RecapTask.TS = %d, want 200 (latest)", w.RecapTask.TS)
	}
}

func TestBuildWindow_HasCommit_FalseWhenNoCommit(t *testing.T) {
	tasks := []task.Task{
		{ID: "a", Kind: task.KindFileWrite, Path: "x.go", Source: "hook", TS: 100},
		{ID: "b", Kind: task.KindCommand, Detail: "go test", Source: "hook", TS: 110},
	}
	w := BuildWindow(tasks)
	if w.HasCommit {
		t.Error("HasCommit should be false when no CommitCreated task is present")
	}
}

func TestBuildWindow_Sources(t *testing.T) {
	tasks := []task.Task{
		{ID: "a", Kind: task.KindFileWrite, Path: "a.go", Source: "hook", TS: 100},
		{ID: "b", Kind: task.KindCommand, Detail: "go test", Source: "agent:claude", TS: 110},
		{ID: "c", Kind: task.KindFileWrite, Path: "b.go", Source: "hook", TS: 120},
	}
	w := BuildWindow(tasks)
	srcSet := make(map[string]struct{})
	for _, s := range w.Sources {
		srcSet[s] = struct{}{}
	}
	if _, ok := srcSet["hook"]; !ok {
		t.Error("Sources should contain 'hook'")
	}
	if _, ok := srcSet["agent:claude"]; !ok {
		t.Error("Sources should contain 'agent:claude'")
	}
}
