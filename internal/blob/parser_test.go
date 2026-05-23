package blob

import (
	"testing"
)

func TestParseLLMResponse_ValidJSON(t *testing.T) {
	raw := `{"title":"OAuth impl","summary":"Did stuff.","recap":"It matters.","user_intent":"Add OAuth","kind":"Feature","tags":["auth"]}`
	r, err := ParseLLMResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Title != "OAuth impl" {
		t.Errorf("Title = %q", r.Title)
	}
	if r.Kind != "Feature" {
		t.Errorf("Kind = %q", r.Kind)
	}
	if len(r.Tags) != 1 || r.Tags[0] != "auth" {
		t.Errorf("Tags = %v", r.Tags)
	}
}

func TestParseLLMResponse_MarkdownFences(t *testing.T) {
	raw := "```json\n{\"title\":\"T\",\"summary\":\"S\",\"recap\":\"R\",\"user_intent\":\"U\",\"kind\":\"BugFix\",\"tags\":[]}\n```"
	r, err := ParseLLMResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Kind != "BugFix" {
		t.Errorf("Kind = %q, want BugFix", r.Kind)
	}
}

func TestParseLLMResponse_InvalidKindFallback(t *testing.T) {
	raw := `{"title":"T","summary":"S","recap":"R","user_intent":"U","kind":"UNKNOWN","tags":[]}`
	r, err := ParseLLMResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Kind != string(KindFeature) {
		t.Errorf("Kind = %q, want Feature fallback", r.Kind)
	}
}

func TestParseLLMResponse_TitleTruncated(t *testing.T) {
	long := make([]byte, 150)
	for i := range long {
		long[i] = 'a'
	}
	raw := `{"title":"` + string(long) + `","summary":"S","recap":"R","user_intent":"U","kind":"Feature","tags":[]}`
	r, err := ParseLLMResponse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.Title) != 100 {
		t.Errorf("Title len = %d, want 100", len(r.Title))
	}
}

func TestParseLLMResponse_InvalidJSON(t *testing.T) {
	_, err := ParseLLMResponse("not json at all")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
