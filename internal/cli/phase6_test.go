package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/meredian-labs/lore/internal/blob"
	"github.com/meredian-labs/lore/internal/task"
)

func TestHookFileWrite_InsertsFileWriteTask(t *testing.T) {
	_, _, s := setupTestRepo(t)
	ctx := context.Background()

	if err := hookFileWrite(ctx, "internal/auth/oauth.go", "agent:claude", s); err != nil {
		t.Fatal(err)
	}

	pending, _ := s.PendingTasks(ctx)
	writes := tasksOfKind(pending, task.KindFileWrite)
	if len(writes) != 1 {
		t.Fatalf("expected 1 FileWrite task, got %d", len(writes))
	}
	if writes[0].Path != "internal/auth/oauth.go" {
		t.Errorf("unexpected path: %s", writes[0].Path)
	}
	if writes[0].Source != "agent:claude" {
		t.Errorf("unexpected source: %s", writes[0].Source)
	}
	if writes[0].TrustLevel != 2 {
		t.Errorf("expected trust_level=2, got %d", writes[0].TrustLevel)
	}
}

func TestHookFileWrite_FiltersLorePaths(t *testing.T) {
	_, _, s := setupTestRepo(t)
	ctx := context.Background()

	if err := hookFileWrite(ctx, ".lore/lore.db", "agent:claude", s); err != nil {
		t.Fatal(err)
	}

	pending, _ := s.PendingTasks(ctx)
	if len(pending) != 0 {
		t.Errorf("expected 0 tasks for .lore/ path, got %d", len(pending))
	}
}

func TestHookCommand_InsertsCommandTask(t *testing.T) {
	_, _, s := setupTestRepo(t)
	ctx := context.Background()

	if err := hookCommand(ctx, "go test ./...", "agent:claude", s); err != nil {
		t.Fatal(err)
	}

	pending, _ := s.PendingTasks(ctx)
	cmds := tasksOfKind(pending, task.KindCommand)
	if len(cmds) != 1 {
		t.Fatalf("expected 1 Command task, got %d", len(cmds))
	}
	if cmds[0].Detail != "go test ./..." {
		t.Errorf("unexpected detail: %s", cmds[0].Detail)
	}
	if cmds[0].Source != "agent:claude" {
		t.Errorf("unexpected source: %s", cmds[0].Source)
	}
	if cmds[0].TrustLevel != 2 {
		t.Errorf("expected trust_level=2, got %d", cmds[0].TrustLevel)
	}
}

func TestHookAgentRecap_ValidJSON_InsertsTask(t *testing.T) {
	_, _, s := setupTestRepo(t)
	ctx := context.Background()

	payload := blob.AgentRecapPayload{
		UserIntent: "Add OAuth support",
		Summary:    "Implemented OAuth2 provider flow.",
		Recap:      "Auth subsystem updated.",
		Kind:       "Feature",
		Tags:       []string{"auth", "oauth"},
	}
	raw, _ := json.Marshal(payload)

	var errBuf bytes.Buffer
	if err := hookAgentRecap(ctx, "agent:claude", bytes.NewReader(raw), &errBuf, s); err != nil {
		t.Fatal(err)
	}

	pending, _ := s.PendingTasks(ctx)
	recaps := tasksOfKind(pending, task.KindAgentRecap)
	if len(recaps) != 1 {
		t.Fatalf("expected 1 AgentRecap task, got %d", len(recaps))
	}
	if recaps[0].Source != "agent:claude" {
		t.Errorf("unexpected source: %s", recaps[0].Source)
	}
	if recaps[0].TrustLevel != 2 {
		t.Errorf("expected trust_level=2, got %d", recaps[0].TrustLevel)
	}

	var parsed blob.AgentRecapPayload
	if err := json.Unmarshal([]byte(recaps[0].Detail), &parsed); err != nil {
		t.Fatalf("expected valid JSON in detail: %v", err)
	}
	if parsed.UserIntent != "Add OAuth support" {
		t.Errorf("unexpected user_intent: %s", parsed.UserIntent)
	}
}

func TestHookAgentRecap_MalformedJSON_UsesRawText(t *testing.T) {
	_, _, s := setupTestRepo(t)
	ctx := context.Background()

	raw := "Fixed the JWT expiry bug by adjusting the clock skew."
	var errBuf bytes.Buffer
	if err := hookAgentRecap(ctx, "agent:claude", strings.NewReader(raw), &errBuf, s); err != nil {
		t.Fatal(err)
	}

	pending, _ := s.PendingTasks(ctx)
	recaps := tasksOfKind(pending, task.KindAgentRecap)
	if len(recaps) != 1 {
		t.Fatalf("expected 1 AgentRecap task, got %d", len(recaps))
	}

	var parsed blob.AgentRecapPayload
	if err := json.Unmarshal([]byte(recaps[0].Detail), &parsed); err != nil {
		t.Fatalf("expected wrapped JSON in detail: %v", err)
	}
	if parsed.Summary != raw {
		t.Errorf("expected raw text as summary, got: %s", parsed.Summary)
	}
}

func TestHookAgentRecap_EmptyInput_InsertsMinimalTask(t *testing.T) {
	_, _, s := setupTestRepo(t)
	ctx := context.Background()

	var errBuf bytes.Buffer
	if err := hookAgentRecap(ctx, "agent:claude", strings.NewReader(""), &errBuf, s); err != nil {
		t.Fatal(err)
	}

	pending, _ := s.PendingTasks(ctx)
	recaps := tasksOfKind(pending, task.KindAgentRecap)
	if len(recaps) != 1 {
		t.Fatalf("expected 1 AgentRecap task, got %d", len(recaps))
	}

	// Should have written a warning to stderr.
	if !strings.Contains(errBuf.String(), "no agent recap") {
		t.Errorf("expected warning in stderr, got: %s", errBuf.String())
	}

	var parsed blob.AgentRecapPayload
	if err := json.Unmarshal([]byte(recaps[0].Detail), &parsed); err != nil {
		t.Fatalf("expected JSON in detail: %v", err)
	}
	if parsed.Summary == "" {
		t.Error("expected non-empty summary in minimal task")
	}
}

func TestHookAgentRecap_EnvVar_TakesPrecedence(t *testing.T) {
	_, _, s := setupTestRepo(t)
	ctx := context.Background()

	payload := blob.AgentRecapPayload{Summary: "From env var"}
	raw, _ := json.Marshal(payload)
	t.Setenv("CLAUDE_STOP_HOOK_PAYLOAD", string(raw))

	var errBuf bytes.Buffer
	// stdin is empty — env var should win.
	if err := hookAgentRecap(ctx, "agent:claude", strings.NewReader(""), &errBuf, s); err != nil {
		t.Fatal(err)
	}

	pending, _ := s.PendingTasks(ctx)
	recaps := tasksOfKind(pending, task.KindAgentRecap)
	if len(recaps) != 1 {
		t.Fatalf("expected 1 AgentRecap task, got %d", len(recaps))
	}
	var parsed blob.AgentRecapPayload
	json.Unmarshal([]byte(recaps[0].Detail), &parsed)
	if parsed.Summary != "From env var" {
		t.Errorf("expected summary from env var, got: %s", parsed.Summary)
	}
}

func TestSettingsJSON_WrittenByInit(t *testing.T) {
	gitRoot := t.TempDir()

	if err := writeClaudeSettings(gitRoot); err != nil {
		t.Fatal(err)
	}

	settingsPath := filepath.Join(gitRoot, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("settings.json not created: %v", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("expected hooks object")
	}

	postToolUse, ok := hooks["PostToolUse"].([]interface{})
	if !ok || len(postToolUse) == 0 {
		t.Error("expected PostToolUse hooks")
	}

	stop, ok := hooks["Stop"].([]interface{})
	if !ok || len(stop) == 0 {
		t.Error("expected Stop hooks")
	}

	out := string(data)
	if !strings.Contains(out, "lore hook file-write") {
		t.Error("expected file-write hook in settings")
	}
	if !strings.Contains(out, "lore hook agent-recap") {
		t.Error("expected agent-recap hook in settings")
	}
}

func TestSettingsJSON_MergesExistingHooks(t *testing.T) {
	gitRoot := t.TempDir()
	claudeDir := filepath.Join(gitRoot, ".claude")
	os.MkdirAll(claudeDir, 0755)

	// Pre-existing settings with a custom hook and one lore hook already present.
	existing := map[string]interface{}{
		"hooks": map[string]interface{}{
			"Stop": []map[string]string{
				{"command": "my-custom-stop-hook"},
				{"command": "lore hook agent-recap agent:claude"},
			},
		},
		"someOtherKey": "preserved",
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644)

	if err := writeClaudeSettings(gitRoot); err != nil {
		t.Fatal(err)
	}

	result, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}

	var merged map[string]interface{}
	if err := json.Unmarshal(result, &merged); err != nil {
		t.Fatalf("invalid JSON after merge: %v", err)
	}

	// Custom key must be preserved.
	if merged["someOtherKey"] != "preserved" {
		t.Errorf("someOtherKey should be preserved, got: %v", merged["someOtherKey"])
	}

	hooks := merged["hooks"].(map[string]interface{})
	stop := hooks["Stop"].([]interface{})

	// Should have: my-custom-stop-hook + lore hook agent-recap (not duplicated).
	if len(stop) != 2 {
		t.Errorf("expected 2 Stop hooks (custom + lore, no duplicate), got %d", len(stop))
	}

	out := string(result)
	if !strings.Contains(out, "my-custom-stop-hook") {
		t.Error("custom stop hook should be preserved")
	}

	// Count occurrences of lore hook agent-recap — must appear exactly once.
	count := strings.Count(out, "lore hook agent-recap")
	if count != 1 {
		t.Errorf("lore hook agent-recap should appear exactly once, got %d", count)
	}

	// PostToolUse hooks should have been added.
	if !strings.Contains(out, "lore hook file-write") {
		t.Error("expected file-write hook added during merge")
	}
}
