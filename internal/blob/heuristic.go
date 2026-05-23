package blob

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// HeuristicExtract builds a Blob from observed window data using deterministic
// rules when no AI model is available. trust_level=4 (LoreInference).
func HeuristicExtract(w Window) Blob {
	return Blob{
		ID:                uuid.NewString(),
		Kind:              inferKind(w),
		Title:             inferTitle(w),
		Summary:           inferSummary(w),
		InferredReasoning: "Inferred from commit messages and file paths; no AI model available.",
		Tags:              inferTags(w),
		TrustLevel:        4,
		AISource:          "lore:heuristic",
		StartedAt:         w.StartedAt,
		EndedAt:           w.EndedAt,
		CommitStart:       w.CommitStart,
		CommitEnd:         w.CommitEnd,
		CreatedAt:         time.Now().UnixNano(),
	}
}

func inferKind(w Window) BlobKind {
	for _, msg := range w.CommitMsgs {
		lower := strings.ToLower(msg)
		switch {
		case strings.HasPrefix(lower, "fix") || strings.HasPrefix(lower, "bug"):
			return KindBugFix
		case strings.HasPrefix(lower, "feat") || strings.HasPrefix(lower, "add"):
			return KindFeature
		case strings.Contains(lower, "migrat"):
			return KindMigration
		case strings.HasPrefix(lower, "refactor") || strings.HasPrefix(lower, "chore"):
			return KindRefactor
		}
	}
	for _, p := range append(w.FilesWritten, w.FilesDeleted...) {
		lower := strings.ToLower(p)
		if strings.Contains(lower, "arch") || strings.Contains(lower, "design") || strings.Contains(lower, "adr") {
			return KindArchitecture
		}
	}
	if !w.HasCommit && len(w.FilesWritten) == 0 {
		return KindInvestigation
	}
	return KindFeature
}

func inferTitle(w Window) string {
	if len(w.CommitMsgs) > 0 {
		msg := w.CommitMsgs[0]
		if len(msg) > 72 {
			return msg[:72]
		}
		return msg
	}
	return "Untitled"
}

func inferSummary(w Window) string {
	var parts []string
	if n := len(w.FilesWritten); n > 0 {
		parts = append(parts, fmt.Sprintf("Modified %d file(s).", n))
	}
	if n := len(w.FilesDeleted); n > 0 {
		parts = append(parts, fmt.Sprintf("Deleted %d file(s).", n))
	}
	if n := len(w.Commands); n > 0 {
		parts = append(parts, fmt.Sprintf("Ran %d command(s).", n))
	}
	if n := len(w.CommitMsgs); n > 0 {
		parts = append(parts, fmt.Sprintf("Produced %d commit(s).", n))
	}
	if len(parts) == 0 {
		return "No observable changes recorded."
	}
	return strings.Join(parts, " ")
}

func inferTags(w Window) []string {
	dirSet := make(map[string]struct{})
	for _, p := range append(w.FilesWritten, w.FilesDeleted...) {
		dir := filepath.Dir(p)
		if dir != "." {
			top := strings.SplitN(dir, "/", 2)[0]
			dirSet[top] = struct{}{}
		}
	}
	tags := make([]string, 0, len(dirSet))
	for d := range dirSet {
		tags = append(tags, d)
	}
	return tags
}
