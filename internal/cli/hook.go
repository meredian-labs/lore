package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nishchay/lore/internal/blob"
	"github.com/nishchay/lore/internal/config"
	gitpkg "github.com/nishchay/lore/internal/git"
	"github.com/nishchay/lore/internal/graph"
	"github.com/nishchay/lore/internal/store"
	"github.com/nishchay/lore/internal/task"
	"github.com/spf13/cobra"
)

var hookCmd = &cobra.Command{
	Use:    "hook",
	Short:  "Internal: called by git hooks",
	Hidden: true,
}

var hookCommitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Record a CommitCreated task",
	Args:  cobra.NoArgs,
	RunE:  runHookCommit,
}

var hookCheckoutCmd = &cobra.Command{
	Use:   "checkout <prev> <new> <flag>",
	Short: "Record a BranchSwitch task",
	Args:  cobra.ExactArgs(3),
	RunE:  runHookCheckout,
}

var hookMergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "Record a MergeEvent task",
	Args:  cobra.NoArgs,
	RunE:  runHookMerge,
}

func init() {
	hookCmd.AddCommand(hookCommitCmd, hookCheckoutCmd, hookMergeCmd)
}

// --- cobra wrappers ---

func runHookCommit(cmd *cobra.Command, args []string) error {
	loreRoot, err := findLoreRoot()
	if err != nil {
		return nil // silent: don't break git
	}
	gitRoot := filepath.Dir(loreRoot)
	s, err := store.Open(filepath.Join(loreRoot, "lore.db"))
	if err != nil {
		return nil
	}
	defer s.Close()
	return hookCommit(cmd.Context(), gitRoot, s)
}

func runHookCheckout(cmd *cobra.Command, args []string) error {
	loreRoot, err := findLoreRoot()
	if err != nil {
		return nil
	}
	gitRoot := filepath.Dir(loreRoot)
	s, err := store.Open(filepath.Join(loreRoot, "lore.db"))
	if err != nil {
		return nil
	}
	defer s.Close()
	return hookCheckout(cmd.Context(), gitRoot, args[0], args[1], args[2], s)
}

func runHookMerge(cmd *cobra.Command, args []string) error {
	loreRoot, err := findLoreRoot()
	if err != nil {
		return nil
	}
	gitRoot := filepath.Dir(loreRoot)
	s, err := store.Open(filepath.Join(loreRoot, "lore.db"))
	if err != nil {
		return nil
	}
	defer s.Close()
	return hookMerge(cmd.Context(), gitRoot, s)
}

// --- testable logic ---

func hookCommit(ctx context.Context, gitRoot string, s *store.Store) error {
	now := time.Now().UnixNano()

	// Read commit metadata.
	logOut, err := gitpkg.Output(ctx, gitRoot, "log", "-1", "--format=%H%n%s")
	if err != nil {
		return fmt.Errorf("git log: %w", err)
	}
	lines := strings.SplitN(logOut, "\n", 2)
	sha, subject := "", ""
	if len(lines) >= 1 {
		sha = lines[0]
	}
	if len(lines) >= 2 {
		subject = lines[1]
	}

	// lore's own checkpoint commits — skip entirely to prevent an extraction loop.
	if strings.HasPrefix(subject, "lore: ") {
		return nil
	}

	var tasks []task.Task

	// CommitCreated task.
	tasks = append(tasks, task.Task{
		ID:         uuid.NewString(),
		Kind:       task.KindCommitCreated,
		Detail:     sha + "|" + subject,
		Source:     "hook",
		TrustLevel: 1,
		TS:         now,
	})

	// File change tasks from diff-tree.
	// --root ensures the initial commit (no parent) also emits file entries.
	// .lore/ paths are filtered: they are lore-internal and not source knowledge.
	diffOut, err := gitpkg.Output(ctx, gitRoot, "diff-tree", "--root", "--no-commit-id", "-r", "--name-status", "HEAD")
	if err == nil && diffOut != "" {
		for _, line := range strings.Split(diffOut, "\n") {
			if line == "" {
				continue
			}
			parts := strings.Split(line, "\t")
			if len(parts) < 2 {
				continue
			}
			status := parts[0]
			path := parts[1]
			if strings.HasPrefix(path, ".lore/") {
				continue
			}
			switch {
			case status == "A" || status == "M":
				tasks = append(tasks, task.Task{
					ID: uuid.NewString(), Kind: task.KindFileWrite,
					Path: path, Source: "hook", TrustLevel: 1, TS: now,
				})
			case status == "D":
				tasks = append(tasks, task.Task{
					ID: uuid.NewString(), Kind: task.KindFileDelete,
					Path: path, Source: "hook", TrustLevel: 1, TS: now,
				})
			case strings.HasPrefix(status, "R") && len(parts) >= 3:
				if !strings.HasPrefix(parts[2], ".lore/") {
					tasks = append(tasks, task.Task{
						ID: uuid.NewString(), Kind: task.KindFileRename,
						Detail: path + " -> " + parts[2],
						Source: "hook", TrustLevel: 1, TS: now,
					})
				}
			}
		}
	}

	if err := s.InsertTasks(ctx, tasks); err != nil {
		return fmt.Errorf("inserting commit tasks: %w", err)
	}

	cfg, _ := config.Load(filepath.Join(filepath.Dir(gitRoot), ".lore"))
	if err := blob.ExtractIfReady(ctx, s, graph.New(s), cfg); err != nil {
		return err
	}

	// Insert a checkpoint blob for this DB state, then commit lore.db so
	// the working tree stays clean after every commit.
	sha7 := sha
	if len(sha7) > 7 {
		sha7 = sha7[:7]
	}
	checkpoint := blob.Blob{
		ID:         uuid.NewString(),
		Title:      "Lore knowledge base checkpoint",
		Kind:       blob.KindCheckpoint,
		Summary:    "Knowledge base updated after commit " + sha7,
		AISource:   "lore:system",
		TrustLevel: 1,
		StartedAt:  now,
		EndedAt:    now,
		CreatedAt:  now,
	}
	checkpointFile := store.BlobFile{BlobID: checkpoint.ID, Path: ".lore/lore.db", Role: "written"}
	if err := s.InsertBlobWithRelations(ctx, checkpoint, []store.BlobFile{checkpointFile}, nil, nil); err != nil {
		return fmt.Errorf("inserting checkpoint blob: %w", err)
	}

	gitpkg.CommitDB(gitRoot)
	return nil
}

func hookCheckout(ctx context.Context, gitRoot, prevRef, newRef, flag string, s *store.Store) error {
	if flag != "1" {
		return nil // file checkout — not a branch switch
	}
	branch, err := gitpkg.Output(ctx, gitRoot, "branch", "--show-current")
	if err != nil {
		branch = newRef // fallback
	}
	t := task.Task{
		ID:         uuid.NewString(),
		Kind:       task.KindBranchSwitch,
		Detail:     prevRef + "->" + branch,
		Source:     "hook",
		TrustLevel: 1,
		TS:         time.Now().UnixNano(),
	}
	return s.InsertTask(ctx, t)
}

func hookMerge(ctx context.Context, gitRoot string, s *store.Store) error {
	mergedSHA := ""
	if data, err := os.ReadFile(filepath.Join(gitRoot, ".git", "MERGE_HEAD")); err == nil {
		mergedSHA = strings.TrimSpace(string(data))
	}
	t := task.Task{
		ID:         uuid.NewString(),
		Kind:       task.KindMergeEvent,
		Detail:     mergedSHA,
		Source:     "hook",
		TrustLevel: 1,
		TS:         time.Now().UnixNano(),
	}
	return s.InsertTask(ctx, t)
}
