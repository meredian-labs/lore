package blob

import (
	"encoding/json"
	"fmt"
	"strings"
)

// LLMResponse is the structured JSON Lore expects from an Ollama model.
type LLMResponse struct {
	Title      string   `json:"title"`
	Summary    string   `json:"summary"`
	Recap      string   `json:"recap"`
	UserIntent string   `json:"user_intent"`
	Kind       string   `json:"kind"`
	Tags       []string `json:"tags"`
}

// ParseLLMResponse extracts structured data from an Ollama model response.
// Handles responses that may contain markdown code fences around the JSON.
func ParseLLMResponse(raw string) (LLMResponse, error) {
	raw = strings.TrimSpace(raw)

	// Strip markdown code fences if present.
	if idx := strings.Index(raw, "```"); idx != -1 {
		raw = raw[idx+3:]
		if strings.HasPrefix(raw, "json") {
			raw = raw[4:]
		}
		if end := strings.LastIndex(raw, "```"); end != -1 {
			raw = raw[:end]
		}
		raw = strings.TrimSpace(raw)
	}

	var r LLMResponse
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		return LLMResponse{}, fmt.Errorf("parsing LLM response: %w", err)
	}

	// Enforce length limits from the prompt contract.
	if len(r.Title) > 100 {
		r.Title = r.Title[:100]
	}
	if len(r.Summary) > 500 {
		r.Summary = r.Summary[:500]
	}
	if len(r.Recap) > 300 {
		r.Recap = r.Recap[:300]
	}
	if len(r.UserIntent) > 200 {
		r.UserIntent = r.UserIntent[:200]
	}
	if !validKind(BlobKind(r.Kind)) {
		r.Kind = string(KindFeature)
	}

	return r, nil
}
