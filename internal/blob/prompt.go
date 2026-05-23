package blob

import (
	"fmt"
	"strings"
	"time"
)

// BuildPrompt assembles a structured text prompt for the Ollama model from
// observed window data. Raw file contents are never included — only metadata.
func BuildPrompt(w Window) string {
	var sb strings.Builder

	sb.WriteString("You are analyzing a software development session. Based on the observed activity below, generate a structured JSON summary.\n\n")

	start := time.Unix(0, w.StartedAt).UTC()
	end := time.Unix(0, w.EndedAt).UTC()
	fmt.Fprintf(&sb, "Time range: %s to %s\n\n", start.Format("2006-01-02 15:04"), end.Format("2006-01-02 15:04"))

	if len(w.Sources) > 0 {
		fmt.Fprintf(&sb, "Sources: %s\n\n", strings.Join(w.Sources, ", "))
	}

	if len(w.CommitMsgs) > 0 {
		sb.WriteString("Commit messages:\n")
		for _, msg := range w.CommitMsgs {
			fmt.Fprintf(&sb, "  - %s\n", msg)
		}
		sb.WriteString("\n")
	}

	if len(w.FilesWritten) > 0 {
		sb.WriteString("Files modified:\n")
		for _, f := range w.FilesWritten {
			fmt.Fprintf(&sb, "  - %s\n", f)
		}
		sb.WriteString("\n")
	}

	if len(w.FilesDeleted) > 0 {
		sb.WriteString("Files deleted:\n")
		for _, f := range w.FilesDeleted {
			fmt.Fprintf(&sb, "  - %s\n", f)
		}
		sb.WriteString("\n")
	}

	if len(w.Commands) > 0 {
		sb.WriteString("Commands executed:\n")
		for _, c := range w.Commands {
			fmt.Fprintf(&sb, "  - %s\n", c)
		}
		sb.WriteString("\n")
	}

	sb.WriteString(`Respond with only a JSON object matching this schema (no markdown, no explanation):
{
  "title": "string (max 100 chars)",
  "summary": "string (max 500 chars, 2-5 sentences: what was done)",
  "recap": "string (max 300 chars, 1-3 sentences: why it matters)",
  "user_intent": "string (max 200 chars, what the developer was trying to accomplish)",
  "kind": "Feature | BugFix | Migration | Investigation | Refactor | Architecture | Review | Incident",
  "tags": ["array of domain concept strings"]
}`)

	return sb.String()
}
