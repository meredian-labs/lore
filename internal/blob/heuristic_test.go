package blob

import (
	"strings"
	"testing"
)

func TestHeuristicExtract_TrustLevel4(t *testing.T) {
	w := Window{
		StartedAt: 100, EndedAt: 200,
		CommitStart: "abc", CommitEnd: "abc",
		HasCommit: true, CommitMsgs: []string{"feat: new feature"},
	}
	b := HeuristicExtract(w)
	if b.TrustLevel != 4 {
		t.Errorf("TrustLevel = %d, want 4", b.TrustLevel)
	}
	if b.AISource != "lore:heuristic" {
		t.Errorf("AISource = %q, want lore:heuristic", b.AISource)
	}
	if b.ID == "" {
		t.Error("ID must not be empty")
	}
}

func TestHeuristicExtract_KindBugFix(t *testing.T) {
	w := Window{HasCommit: true, CommitMsgs: []string{"fix: correct nil dereference"}}
	b := HeuristicExtract(w)
	if b.Kind != KindBugFix {
		t.Errorf("Kind = %q, want BugFix", b.Kind)
	}
}

func TestHeuristicExtract_KindMigration(t *testing.T) {
	w := Window{HasCommit: true, CommitMsgs: []string{"database migration for users table"}}
	b := HeuristicExtract(w)
	if b.Kind != KindMigration {
		t.Errorf("Kind = %q, want Migration", b.Kind)
	}
}

func TestHeuristicExtract_KindRefactor(t *testing.T) {
	w := Window{HasCommit: true, CommitMsgs: []string{"refactor: extract helper"}}
	b := HeuristicExtract(w)
	if b.Kind != KindRefactor {
		t.Errorf("Kind = %q, want Refactor", b.Kind)
	}
}

func TestHeuristicExtract_TitleTruncated(t *testing.T) {
	long := strings.Repeat("a", 100)
	w := Window{HasCommit: true, CommitMsgs: []string{long}}
	b := HeuristicExtract(w)
	if len(b.Title) != 72 {
		t.Errorf("Title len = %d, want 72", len(b.Title))
	}
}

func TestHeuristicExtract_Tags(t *testing.T) {
	w := Window{
		HasCommit:    true,
		CommitMsgs:   []string{"chore: cleanup"},
		FilesWritten: []string{"internal/auth/oauth.go", "internal/store/blobs.go"},
	}
	b := HeuristicExtract(w)
	tagSet := make(map[string]struct{})
	for _, tag := range b.Tags {
		tagSet[tag] = struct{}{}
	}
	if _, ok := tagSet["internal"]; !ok {
		t.Errorf("expected 'internal' tag in %v", b.Tags)
	}
}

func TestHeuristicExtract_DefaultKind_IsFeature(t *testing.T) {
	// No commit message matches any heuristic keyword.
	w := Window{HasCommit: true, CommitMsgs: []string{"update readme"}}
	b := HeuristicExtract(w)
	if b.Kind != KindFeature {
		t.Errorf("Kind = %q, want Feature as default", b.Kind)
	}
}

func TestHeuristicExtract_SummaryContainsFileCounts(t *testing.T) {
	w := Window{
		HasCommit:    true,
		CommitMsgs:   []string{"feat: stuff"},
		FilesWritten: []string{"a.go", "b.go"},
		FilesDeleted: []string{"c.go"},
	}
	b := HeuristicExtract(w)
	if !strings.Contains(b.Summary, "2 file(s)") {
		t.Errorf("Summary %q should mention 2 file(s)", b.Summary)
	}
	if !strings.Contains(b.Summary, "Deleted 1") {
		t.Errorf("Summary %q should mention Deleted 1", b.Summary)
	}
}
