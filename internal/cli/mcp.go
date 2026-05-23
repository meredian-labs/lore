package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/meredian-labs/lore/internal/blob"
	"github.com/meredian-labs/lore/internal/mcp"
	"github.com/meredian-labs/lore/internal/store"
	"github.com/meredian-labs/lore/internal/task"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp [source]",
	Short: "Start the lore MCP server (stdio)",
	Long: `Start a Model Context Protocol server over stdin/stdout.

Any MCP-compatible agent can connect by launching: lore mcp [source]

The optional source argument identifies the agent (e.g. agent:cursor, agent:claude).
If omitted, source defaults to "agent:mcp".`,
	Args: cobra.MaximumNArgs(1),
	RunE: runMCP,
}

func runMCP(cmd *cobra.Command, args []string) error {
	source := "agent:mcp"
	if len(args) > 0 {
		source = args[0]
	}

	loreRoot, err := findLoreRoot()
	if err != nil {
		return err
	}
	s, err := store.Open(filepath.Join(loreRoot, "lore.db"))
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer s.Close()

	srv := buildMCPServer(s, source)
	return srv.Serve(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout())
}

// buildMCPServer registers all lore tools on the server.
func buildMCPServer(s *store.Store, source string) *mcp.Server {
	srv := mcp.New("lore", Version)

	// --- Record tools ---

	srv.Register(
		"record_file_write",
		"Record that a file was written/modified. Call this after every Edit or Write tool use.",
		strSchema("path", "File path that was written"),
		func(ctx context.Context, args map[string]interface{}) (string, error) {
			path := strArg(args, "path")
			if path == "" {
				return "", fmt.Errorf("path is required")
			}
			if strings.HasPrefix(path, ".lore/") {
				return "skipped (.lore/ path)", nil
			}
			return "recorded", s.InsertTask(ctx, task.Task{
				ID: uuid.NewString(), Kind: task.KindFileWrite,
				Path: path, Source: source, TrustLevel: 2, TS: time.Now().UnixNano(),
			})
		},
	)

	srv.Register(
		"record_file_read",
		"Record that a file was read. Call this after every Read tool use to track context.",
		strSchema("path", "File path that was read"),
		func(ctx context.Context, args map[string]interface{}) (string, error) {
			path := strArg(args, "path")
			if path == "" {
				return "", fmt.Errorf("path is required")
			}
			if strings.HasPrefix(path, ".lore/") {
				return "skipped (.lore/ path)", nil
			}
			return "recorded", s.InsertTask(ctx, task.Task{
				ID: uuid.NewString(), Kind: task.KindFileRead,
				Path: path, Source: source, TrustLevel: 2, TS: time.Now().UnixNano(),
			})
		},
	)

	srv.Register(
		"record_command",
		"Record that a shell command was executed. Call this after every Bash/terminal tool use.",
		strSchema("command", "The shell command that was run"),
		func(ctx context.Context, args map[string]interface{}) (string, error) {
			cmd := strArg(args, "command")
			if cmd == "" {
				return "", fmt.Errorf("command is required")
			}
			return "recorded", s.InsertTask(ctx, task.Task{
				ID: uuid.NewString(), Kind: task.KindCommand,
				Detail: cmd, Source: source, TrustLevel: 2, TS: time.Now().UnixNano(),
			})
		},
	)

	srv.Register(
		"record_note",
		"Record a developer note or observation into lore. Appears in the next blob extraction.",
		strSchema("text", "The note or observation to record"),
		func(ctx context.Context, args map[string]interface{}) (string, error) {
			text := strArg(args, "text")
			if text == "" {
				return "", fmt.Errorf("text is required")
			}
			return "recorded", s.InsertTask(ctx, task.Task{
				ID: uuid.NewString(), Kind: task.KindNote,
				Detail: text, Source: source, TrustLevel: 2, TS: time.Now().UnixNano(),
			})
		},
	)

	srv.Register(
		"submit_recap",
		"Submit a structured recap of the current work session. This becomes the highest-trust blob on next commit.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"user_intent": map[string]interface{}{"type": "string", "description": "What the user wanted to accomplish"},
				"summary":     map[string]interface{}{"type": "string", "description": "What was done (2-5 sentences)"},
				"recap":       map[string]interface{}{"type": "string", "description": "Why it matters in the bigger picture"},
				"kind": map[string]interface{}{
					"type":        "string",
					"description": "Feature | BugFix | Migration | Investigation | Refactor | Architecture | Review | Incident",
				},
				"tags": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Domain concepts"},
			},
			"required": []string{"user_intent", "summary"},
		},
		func(ctx context.Context, args map[string]interface{}) (string, error) {
			payload := blob.AgentRecapPayload{
				UserIntent: strArg(args, "user_intent"),
				Summary:    strArg(args, "summary"),
				Recap:      strArg(args, "recap"),
				Kind:       strArg(args, "kind"),
			}
			if tags, ok := args["tags"].([]interface{}); ok {
				for _, t := range tags {
					if s, ok := t.(string); ok {
						payload.Tags = append(payload.Tags, s)
					}
				}
			}
			detail, _ := json.Marshal(payload)
			return "recap recorded; will be ingested on next commit", s.InsertTask(ctx, task.Task{
				ID: uuid.NewString(), Kind: task.KindAgentRecap,
				Detail: string(detail), Source: source, TrustLevel: 2, TS: time.Now().UnixNano(),
			})
		},
	)

	// --- Query tools ---

	srv.Register(
		"query_status",
		"Get the current lore repository status: blob counts, subsystems, pending tasks.",
		map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		func(ctx context.Context, args map[string]interface{}) (string, error) {
			blobCount, _ := s.BlobCount(ctx)
			byKind, _ := s.BlobCountByKind(ctx)
			nodeCount, _ := s.NodeCount(ctx)
			pending, _ := s.PendingTaskCount(ctx)
			unassigned, _ := s.UnassignedBlobCount(ctx)

			var sb strings.Builder
			fmt.Fprintf(&sb, "Blobs: %d\n", blobCount)
			for k, n := range byKind {
				fmt.Fprintf(&sb, "  %s: %d\n", k, n)
			}
			fmt.Fprintf(&sb, "Subsystems: %d\n", nodeCount)
			fmt.Fprintf(&sb, "Unassigned blobs: %d\n", unassigned)
			fmt.Fprintf(&sb, "Pending tasks (since last commit): %d\n", pending)
			return sb.String(), nil
		},
	)

	srv.Register(
		"query_log",
		"List recent blobs newest-first. Use this to understand what has been worked on.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"limit": map[string]interface{}{"type": "integer", "description": "Max blobs to return (default 10)"},
			},
		},
		func(ctx context.Context, args map[string]interface{}) (string, error) {
			limit := 10
			if v, ok := args["limit"].(float64); ok && v > 0 {
				limit = int(v)
			}
			blobs, err := s.BlobLog(ctx, limit)
			if err != nil {
				return "", err
			}
			var sb strings.Builder
			for _, b := range blobs {
				if b.Kind == blob.KindCheckpoint {
					continue
				}
				fmt.Fprintf(&sb, "%s  %-10s  %s  %s\n",
					b.ID[:7], string(b.Kind), formatDate(b.EndedAt), b.Title)
				if b.Summary != "" {
					fmt.Fprintf(&sb, "  %s\n", b.Summary)
				}
				fmt.Fprintln(&sb)
			}
			if sb.Len() == 0 {
				return "No blobs recorded yet.", nil
			}
			return sb.String(), nil
		},
	)

	srv.Register(
		"query_why",
		"Explain why a file exists by showing all blobs that have modified it.",
		strSchema("file", "File path or suffix to look up (e.g. 'oauth.go' or 'internal/auth/oauth.go')"),
		func(ctx context.Context, args map[string]interface{}) (string, error) {
			file := strArg(args, "file")
			if file == "" {
				return "", fmt.Errorf("file is required")
			}
			blobs, err := s.BlobsByFile(ctx, file)
			if err != nil {
				return "", err
			}
			if len(blobs) == 0 {
				return fmt.Sprintf("No blobs found for %q. The file may not have been committed yet, or lore has not captured its history.", file), nil
			}
			var sb strings.Builder
			fmt.Fprintf(&sb, "History of %s:\n\n", file)
			for _, b := range blobs {
				fmt.Fprintf(&sb, "%s (%s)  %s\n", b.Title, b.Kind, formatDate(b.EndedAt))
				if b.Summary != "" {
					fmt.Fprintf(&sb, "  %s\n", b.Summary)
				}
				if b.CommitStart != "" {
					fmt.Fprintf(&sb, "  Commit: %s\n", shortSHA(b.CommitStart))
				}
				fmt.Fprintln(&sb)
			}
			return sb.String(), nil
		},
	)

	srv.Register(
		"query_blob",
		"Get full detail for a single blob by ID prefix.",
		strSchema("id", "Blob ID or 7-character prefix"),
		func(ctx context.Context, args map[string]interface{}) (string, error) {
			id := strArg(args, "id")
			if id == "" {
				return "", fmt.Errorf("id is required")
			}
			if len(id) < 7 {
				return "", fmt.Errorf("id must be at least 7 characters")
			}
			blobID, err := s.ResolveBlobIDPrefix(ctx, id)
			if err != nil {
				return "", err
			}
			b, err := s.BlobByID(ctx, blobID)
			if err != nil {
				return "", err
			}
			files, _ := s.BlobFiles(ctx, b.ID)
			cmds, _ := s.BlobCommands(ctx, b.ID)

			var sb strings.Builder
			fmt.Fprintf(&sb, "ID:      %s\n", b.ID[:7])
			fmt.Fprintf(&sb, "Title:   %s\n", b.Title)
			fmt.Fprintf(&sb, "Kind:    %s\n", b.Kind)
			fmt.Fprintf(&sb, "Trust:   %s\n", mcpTrustLabel(b.TrustLevel))
			fmt.Fprintf(&sb, "Source:  %s\n", b.AISource)
			fmt.Fprintf(&sb, "Period:  %s → %s\n", formatDate(b.StartedAt), formatDate(b.EndedAt))
			if b.CommitStart != "" {
				fmt.Fprintf(&sb, "Commit:  %s\n", shortSHA(b.CommitStart))
			}
			if b.UserIntent != "" {
				fmt.Fprintf(&sb, "\nIntent: %s\n", b.UserIntent)
			}
			if b.Summary != "" {
				fmt.Fprintf(&sb, "\nSummary:\n%s\n", b.Summary)
			}
			if b.Recap != "" {
				fmt.Fprintf(&sb, "\nRecap:\n%s\n", b.Recap)
			}
			if len(files) > 0 {
				fmt.Fprintln(&sb, "\nFiles:")
				for _, f := range files {
					fmt.Fprintf(&sb, "  [%s] %s\n", f.Role, f.Path)
				}
			}
			if len(cmds) > 0 {
				fmt.Fprintln(&sb, "\nCommands:")
				for _, c := range cmds {
					fmt.Fprintf(&sb, "  $ %s\n", c.Command)
				}
			}
			return sb.String(), nil
		},
	)

	srv.Register(
		"query_nodes",
		"List all subsystem nodes (engineering areas) tracked in this repository.",
		map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		func(ctx context.Context, args map[string]interface{}) (string, error) {
			nodes, err := s.ListNodes(ctx)
			if err != nil {
				return "", err
			}
			if len(nodes) == 0 {
				return "No subsystems defined yet. Use lore node create <name> to create one.", nil
			}
			var sb strings.Builder
			for _, n := range nodes {
				cnt, _ := s.NodeBlobCount(ctx, n.ID)
				noun := "blobs"
				if cnt == 1 {
					noun = "blob"
				}
				fmt.Fprintf(&sb, "%-24s  %d %s  [%s]\n", n.Title, cnt, noun, n.Status)
				if n.Description != "" {
					fmt.Fprintf(&sb, "  %s\n", n.Description)
				}
			}
			return sb.String(), nil
		},
	)

	srv.Register(
		"query_node",
		"Get detail for a subsystem node, including its most recent blobs.",
		strSchema("name", "Node name (or partial match)"),
		func(ctx context.Context, args map[string]interface{}) (string, error) {
			name := strArg(args, "name")
			if name == "" {
				return "", fmt.Errorf("name is required")
			}
			n, err := s.NodeByTitle(ctx, name)
			if err != nil {
				return "", fmt.Errorf("node %q not found", name)
			}
			blobs, err := s.BlobsForNode(ctx, n.ID)
			if err != nil {
				return "", err
			}
			var sb strings.Builder
			fmt.Fprintf(&sb, "Node: %s  [%s]\n", n.Title, n.Status)
			if n.Description != "" {
				fmt.Fprintf(&sb, "Description: %s\n", n.Description)
			}
			fmt.Fprintf(&sb, "Blobs: %d\n\n", len(blobs))
			for _, b := range blobs {
				fmt.Fprintf(&sb, "  %s  %-10s  %s  %s\n",
					b.ID[:7], string(b.Kind), formatDate(b.EndedAt), b.Title)
			}
			return sb.String(), nil
		},
	)

	return srv
}

// --- helpers ---

func strArg(args map[string]interface{}, key string) string {
	if v, ok := args[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func strSchema(param, description string) map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			param: map[string]interface{}{"type": "string", "description": description},
		},
		"required": []string{param},
	}
}

func mcpTrustLabel(level int) string {
	switch level {
	case 1:
		return "GroundTruth"
	case 2:
		return "AgentTruth"
	case 3:
		return "HumanAssertion"
	default:
		return "LoreInference"
	}
}
