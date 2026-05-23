package cli

import (
	"encoding/json"
	"io"

	"github.com/meredian-labs/lore/internal/blob"
	"github.com/meredian-labs/lore/internal/store"
)

// JSON output structs — used by --json flag on all query commands.

type BlobFileJSON struct {
	Path string `json:"path"`
	Role string `json:"role"`
}

type BlobCommandJSON struct {
	Command string `json:"command"`
	TS      int64  `json:"ts"`
}

type NodeRefJSON struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type BlobJSON struct {
	ID                string            `json:"id"`
	Kind              string            `json:"kind"`
	Title             string            `json:"title"`
	Summary           string            `json:"summary"`
	Recap             string            `json:"recap"`
	UserIntent        string            `json:"user_intent"`
	InferredReasoning string            `json:"inferred_reasoning,omitempty"`
	Tags              []string          `json:"tags"`
	TrustLevel        int               `json:"trust_level"`
	AISource          string            `json:"ai_source"`
	StartedAt         int64             `json:"started_at"`
	EndedAt           int64             `json:"ended_at"`
	CommitStart       string            `json:"commit_start"`
	CommitEnd         string            `json:"commit_end"`
	PrimaryNodeID     string            `json:"primary_node_id,omitempty"`
	CreatedAt         int64             `json:"created_at"`
	Files             []BlobFileJSON    `json:"files,omitempty"`
	Commands          []BlobCommandJSON `json:"commands,omitempty"`
	Node              *NodeRefJSON      `json:"node,omitempty"`
}

type NodeJSON struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedBy   string `json:"created_by"`
	BlobCount   int    `json:"blob_count"`
	CreatedAt   int64  `json:"created_at"`
}

type StatusJSON struct {
	Repository      string         `json:"repository"`
	InitializedAt   int64          `json:"initialized_at"`
	BlobCount       int            `json:"blob_count"`
	BlobsByKind     map[string]int `json:"blobs_by_kind"`
	BlobsByTrust    map[int]int    `json:"blobs_by_trust"`
	NodeCount       int            `json:"node_count"`
	Nodes           []NodeJSON     `json:"nodes"`
	UnassignedBlobs int            `json:"unassigned_blobs"`
	PendingTasks    int            `json:"pending_tasks"`
	LLMAvailable    bool           `json:"llm_available"`
	LLMProvider     string         `json:"llm_provider"`
}

// blobToJSON converts a blob.Blob to BlobJSON, optionally attaching files,
// commands, and node reference from the store.
func blobToJSON(b blob.Blob, files []store.BlobFile, cmds []store.BlobCommand, node *NodeRefJSON) BlobJSON {
	j := BlobJSON{
		ID:                b.ID,
		Kind:              string(b.Kind),
		Title:             b.Title,
		Summary:           b.Summary,
		Recap:             b.Recap,
		UserIntent:        b.UserIntent,
		InferredReasoning: b.InferredReasoning,
		Tags:              b.Tags,
		TrustLevel:        b.TrustLevel,
		AISource:          b.AISource,
		StartedAt:         b.StartedAt,
		EndedAt:           b.EndedAt,
		CommitStart:       b.CommitStart,
		CommitEnd:         b.CommitEnd,
		PrimaryNodeID:     b.PrimaryNodeID,
		CreatedAt:         b.CreatedAt,
		Node:              node,
	}
	if j.Tags == nil {
		j.Tags = []string{}
	}
	for _, f := range files {
		j.Files = append(j.Files, BlobFileJSON{Path: f.Path, Role: f.Role})
	}
	for _, c := range cmds {
		j.Commands = append(j.Commands, BlobCommandJSON{Command: c.Command, TS: c.TS})
	}
	return j
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
