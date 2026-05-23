package blob

import (
	"testing"
)

func TestIngestRecap_TrustLevel2(t *testing.T) {
	w := Window{
		StartedAt: 100, EndedAt: 200,
		CommitStart: "abc123", CommitEnd: "abc123",
		HasCommit: true,
	}
	payload := AgentRecapPayload{
		UserIntent: "Add OAuth support",
		Summary:    "Implemented OAuth2 flow.",
		Recap:      "Replaced legacy token auth.",
		Kind:       "Feature",
		Tags:       []string{"auth", "oauth"},
	}
	b := IngestRecap(w, payload, "agent:claude")
	if b.TrustLevel != 2 {
		t.Errorf("TrustLevel = %d, want 2", b.TrustLevel)
	}
	if b.AISource != "agent:claude" {
		t.Errorf("AISource = %q, want agent:claude", b.AISource)
	}
	if b.Kind != KindFeature {
		t.Errorf("Kind = %q, want Feature", b.Kind)
	}
	if b.ID == "" {
		t.Error("ID must not be empty")
	}
}

func TestIngestRecap_InvalidKindFallsBackToFeature(t *testing.T) {
	w := Window{StartedAt: 1, EndedAt: 2}
	payload := AgentRecapPayload{Kind: "NotAKind", UserIntent: "something"}
	b := IngestRecap(w, payload, "agent:claude")
	if b.Kind != KindFeature {
		t.Errorf("Kind = %q, want Feature for invalid kind", b.Kind)
	}
}

func TestIngestRecap_EmptyUserIntentFallsToCommitMsg(t *testing.T) {
	w := Window{
		StartedAt:  1,
		EndedAt:    2,
		CommitMsgs: []string{"feat: add login"},
	}
	payload := AgentRecapPayload{Kind: "Feature"}
	b := IngestRecap(w, payload, "agent:claude")
	if b.Title != "feat: add login" {
		t.Errorf("Title = %q, want commit message", b.Title)
	}
}

func TestParseAgentRecap_Valid(t *testing.T) {
	raw := `{"user_intent":"Fix bug","summary":"Fixed it.","recap":"Stable now.","kind":"BugFix","tags":["auth"]}`
	p, err := ParseAgentRecap(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Kind != "BugFix" {
		t.Errorf("Kind = %q, want BugFix", p.Kind)
	}
	if len(p.Tags) != 1 || p.Tags[0] != "auth" {
		t.Errorf("Tags = %v, want [auth]", p.Tags)
	}
}

func TestParseAgentRecap_InvalidJSON(t *testing.T) {
	_, err := ParseAgentRecap("not-json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
