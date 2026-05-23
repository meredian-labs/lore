package blob

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AgentRecapPayload is the structured JSON emitted by an AI agent's Stop hook.
type AgentRecapPayload struct {
	UserIntent string   `json:"user_intent"`
	Summary    string   `json:"summary"`
	Recap      string   `json:"recap"`
	Kind       string   `json:"kind"`
	Tags       []string `json:"tags"`
}

// ParseAgentRecap parses the JSON payload from an AgentRecap task's Detail field.
func ParseAgentRecap(detail string) (AgentRecapPayload, error) {
	var p AgentRecapPayload
	if err := json.Unmarshal([]byte(detail), &p); err != nil {
		return AgentRecapPayload{}, fmt.Errorf("parsing agent recap: %w", err)
	}
	return p, nil
}

// IngestRecap builds a Blob from a Window and an agent-provided recap.
// trust_level=2 (AgentTruth) — the agent that did the work provided the interpretation.
func IngestRecap(w Window, payload AgentRecapPayload, source string) Blob {
	kind := BlobKind(payload.Kind)
	if !validKind(kind) {
		kind = KindFeature
	}

	title := payload.UserIntent
	if len(title) > 100 {
		title = title[:100]
	}
	if title == "" {
		title = firstCommitMsg(w)
	}

	return Blob{
		ID:          uuid.NewString(),
		Kind:        kind,
		Title:       title,
		Summary:     payload.Summary,
		Recap:       payload.Recap,
		UserIntent:  payload.UserIntent,
		Tags:        payload.Tags,
		TrustLevel:  2,
		AISource:    source,
		StartedAt:   w.StartedAt,
		EndedAt:     w.EndedAt,
		CommitStart: w.CommitStart,
		CommitEnd:   w.CommitEnd,
		CreatedAt:   time.Now().UnixNano(),
	}
}

func validKind(k BlobKind) bool {
	switch k {
	case KindFeature, KindBugFix, KindMigration, KindInvestigation,
		KindRefactor, KindArchitecture, KindReview, KindIncident:
		return true
	}
	return false
}

func firstCommitMsg(w Window) string {
	if len(w.CommitMsgs) > 0 {
		msg := w.CommitMsgs[0]
		if len(msg) > 100 {
			return msg[:100]
		}
		return msg
	}
	return "Untitled"
}
