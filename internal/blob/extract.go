package blob

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nishchay/lore/internal/config"
	"github.com/nishchay/lore/internal/task"
)

// Storer is the storage interface ExtractIfReady needs.
// Defined here to avoid circular imports with internal/store.
type Storer interface {
	PendingTasks(ctx context.Context) ([]task.Task, error)
	InsertBlobWithRelations(ctx context.Context, b Blob, files []BlobFile, commands []BlobCommand, taskIDs []string) error
}

// GraphUpdater derives graph nodes and edges from a Blob.
// Defined here to avoid circular imports with internal/graph.
type GraphUpdater interface {
	UpdateFromBlob(ctx context.Context, b Blob) error
}

// ExtractIfReady reads pending tasks and, if a commit is present, builds and
// stores a Blob. Safe to call on every commit hook — returns nil when there is
// nothing to extract. g may be nil (graph update skipped, blob still stored).
func ExtractIfReady(ctx context.Context, s Storer, g GraphUpdater, cfg config.Config) error {
	tasks, err := s.PendingTasks(ctx)
	if err != nil {
		return fmt.Errorf("fetching pending tasks: %w", err)
	}

	w := BuildWindow(tasks)

	// Commits define the unit-of-work boundary. Extraction only fires after a commit.
	if !w.HasCommit {
		return nil
	}

	b := extractBlob(ctx, w, cfg)

	files := buildBlobFiles(b.ID, w)
	commands := buildBlobCommands(b.ID, w)
	taskIDs := make([]string, len(tasks))
	for i, t := range tasks {
		taskIDs[i] = t.ID
	}

	if err := s.InsertBlobWithRelations(ctx, b, files, commands, taskIDs); err != nil {
		return fmt.Errorf("storing blob: %w", err)
	}

	if g != nil {
		// Graph update failure is non-fatal — Blobs are the primary artifact.
		_ = g.UpdateFromBlob(ctx, b)
	}

	return nil
}

// extractBlob picks the highest-trust extraction path available.
func extractBlob(ctx context.Context, w Window, cfg config.Config) Blob {
	// Path 1: agent recap (trust=2).
	if w.RecapTask != nil {
		if payload, err := ParseAgentRecap(w.RecapTask.Detail); err == nil {
			return IngestRecap(w, payload, w.RecapTask.Source)
		}
	}

	// Path 2: Ollama inference (trust=4) if configured and reachable.
	if cfg.LLM.Endpoint != "" && cfg.LLM.Model != "" {
		llm := NewLLMClient(cfg.LLM.Endpoint, cfg.LLM.Model)
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if llm.Ping(pingCtx) == nil {
			if raw, err := llm.Complete(ctx, BuildPrompt(w)); err == nil {
				if r, err := ParseLLMResponse(raw); err == nil {
					return blobFromLLM(r, w)
				}
			}
		}
	}

	// Path 3: heuristic fallback (trust=4).
	return HeuristicExtract(w)
}

func blobFromLLM(r LLMResponse, w Window) Blob {
	return Blob{
		ID:          uuid.NewString(),
		Kind:        BlobKind(r.Kind),
		Title:       r.Title,
		Summary:     r.Summary,
		Recap:       r.Recap,
		UserIntent:  r.UserIntent,
		Tags:        r.Tags,
		TrustLevel:  4,
		AISource:    "lore:ollama",
		StartedAt:   w.StartedAt,
		EndedAt:     w.EndedAt,
		CommitStart: w.CommitStart,
		CommitEnd:   w.CommitEnd,
		CreatedAt:   time.Now().UnixNano(),
	}
}

func buildBlobFiles(blobID string, w Window) []BlobFile {
	files := make([]BlobFile, 0, len(w.FilesWritten)+len(w.FilesDeleted)+len(w.FilesRead))
	for _, p := range w.FilesWritten {
		files = append(files, BlobFile{BlobID: blobID, Path: p, Role: "written"})
	}
	for _, p := range w.FilesDeleted {
		files = append(files, BlobFile{BlobID: blobID, Path: p, Role: "deleted"})
	}
	for _, p := range w.FilesRead {
		files = append(files, BlobFile{BlobID: blobID, Path: p, Role: "read"})
	}
	return files
}

func buildBlobCommands(blobID string, w Window) []BlobCommand {
	cmds := make([]BlobCommand, 0, len(w.Commands))
	for _, c := range w.Commands {
		cmds = append(cmds, BlobCommand{BlobID: blobID, Command: c, TS: w.EndedAt})
	}
	return cmds
}
