package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/meredian-labs/lore/internal/blob"
	"github.com/meredian-labs/lore/internal/store"
	"github.com/meredian-labs/lore/internal/task"
)

// setupTestRepo creates a real temp git repo with lore initialized.
func setupTestRepo(t *testing.T) (gitRoot, loreRoot string, s *store.Store) {
	t.Helper()
	gitRoot = t.TempDir()

	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = gitRoot
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init")
	git("config", "user.email", "test@lore.test")
	git("config", "user.name", "Lore Test")

	loreRoot = filepath.Join(gitRoot, ".lore")
	os.MkdirAll(filepath.Join(loreRoot, "cache"), 0755)

	var err error
	s, err = store.Open(filepath.Join(loreRoot, "lore.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	s.SetMeta(context.Background(), "git_root", gitRoot)
	t.Cleanup(func() { s.Close() })
	return gitRoot, loreRoot, s
}

// commitFile writes a file and commits it; returns the commit SHA.
func commitFile(t *testing.T, dir, name, content, message string) string {
	t.Helper()
	os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("add", name)
	run("commit", "-m", message)
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func tasksOfKind(tasks []task.Task, kind task.TaskKind) []task.Task {
	var out []task.Task
	for _, tk := range tasks {
		if tk.Kind == kind {
			out = append(out, tk)
		}
	}
	return out
}

var testCtx = context.Background()

// nonCheckpointBlobs filters out KindCheckpoint blobs from a list.
func nonCheckpointBlobs(blobs []blob.Blob) []blob.Blob {
	var out []blob.Blob
	for _, b := range blobs {
		if b.Kind != blob.KindCheckpoint {
			out = append(out, b)
		}
	}
	return out
}

func TestHookCommit_InsertsCommitTask(t *testing.T) {
	gitRoot, _, s := setupTestRepo(t)
	sha := commitFile(t, gitRoot, "main.go", "package main\n", "feat: initial commit")

	if err := hookCommit(testCtx, gitRoot, s); err != nil {
		t.Fatal(err)
	}

	// Each hookCommit produces a regular blob + a checkpoint blob.
	all, err := s.BlobLog(testCtx, 10)
	if err != nil {
		t.Fatal(err)
	}
	blobs := nonCheckpointBlobs(all)
	if len(blobs) != 1 {
		t.Fatalf("expected 1 non-checkpoint blob after hookCommit, got %d", len(blobs))
	}
	if !strings.HasPrefix(sha, blobs[0].CommitStart) && !strings.HasPrefix(blobs[0].CommitStart, sha[:7]) {
		t.Errorf("blob CommitStart=%q, want prefix of %s", blobs[0].CommitStart, sha)
	}

	// Verify a checkpoint blob was also created.
	checkpoints := all[:len(all)-len(blobs)]
	_ = checkpoints // presence verified by total count
	if len(all) != 2 {
		t.Errorf("expected 2 total blobs (regular + checkpoint), got %d", len(all))
	}
}

func TestHookCommit_SkipsLoreCheckpointCommits(t *testing.T) {
	gitRoot, _, s := setupTestRepo(t)

	// A "lore: checkpoint" commit must be a no-op.
	commitFile(t, gitRoot, "x.go", "package x", "lore: checkpoint")

	if err := hookCommit(testCtx, gitRoot, s); err != nil {
		t.Fatal(err)
	}

	all, _ := s.BlobLog(testCtx, 10)
	if len(all) != 0 {
		t.Errorf("expected 0 blobs for lore: checkpoint commit, got %d", len(all))
	}
}

func TestHookCommit_InsertsFileWriteTasks(t *testing.T) {
	gitRoot, _, s := setupTestRepo(t)
	commitFile(t, gitRoot, "auth.go", "package auth\n", "add auth package")

	if err := hookCommit(testCtx, gitRoot, s); err != nil {
		t.Fatal(err)
	}

	all, err := s.BlobLog(testCtx, 10)
	if err != nil {
		t.Fatal(err)
	}
	blobs := nonCheckpointBlobs(all)
	if len(blobs) != 1 {
		t.Fatalf("expected 1 non-checkpoint blob, got %d", len(blobs))
	}
	files, err := s.BlobFiles(testCtx, blobs[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, f := range files {
		if f.Path == "auth.go" && f.Role == "written" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected blob_file auth.go (written), got: %v", files)
	}
}

func TestHookCommit_FiltersLorePaths(t *testing.T) {
	gitRoot, loreRoot, s := setupTestRepo(t)

	// Commit a source file plus a .lore/ file. The .lore/ path must not appear
	// in the extracted blob's file list.
	commitFile(t, gitRoot, "main.go", "package main", "feat: add main")
	// Stage a .lore/ file that is NOT lore.db (avoids corrupting the open store).
	os.WriteFile(filepath.Join(loreRoot, "scratch.tmp"), []byte("test"), 0644)
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = gitRoot
		cmd.Run()
	}
	run("add", ".lore/scratch.tmp")
	run("commit", "--amend", "--no-edit")

	if err := hookCommit(testCtx, gitRoot, s); err != nil {
		t.Fatal(err)
	}

	all, _ := s.BlobLog(testCtx, 10)
	blobs := nonCheckpointBlobs(all)
	if len(blobs) != 1 {
		t.Fatalf("expected 1 non-checkpoint blob, got %d", len(blobs))
	}
	files, _ := s.BlobFiles(testCtx, blobs[0].ID)
	for _, f := range files {
		if strings.HasPrefix(f.Path, ".lore/") {
			t.Errorf("blob_files must not contain .lore/ path, got: %s", f.Path)
		}
	}
}

func TestHookCommit_InsertsFileDeleteTask(t *testing.T) {
	gitRoot, _, s := setupTestRepo(t)

	// First commit: add the file and extract.
	commitFile(t, gitRoot, "old.go", "package old\n", "add old.go")
	if err := hookCommit(testCtx, gitRoot, s); err != nil {
		t.Fatal(err)
	}

	// Second commit: delete the file.
	os.Remove(filepath.Join(gitRoot, "old.go"))
	cmd := exec.Command("git", "rm", "old.go")
	cmd.Dir = gitRoot
	cmd.Run()
	c := exec.Command("git", "commit", "-m", "remove old.go")
	c.Dir = gitRoot
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("commit: %v\n%s", err, out)
	}
	if err := hookCommit(testCtx, gitRoot, s); err != nil {
		t.Fatal(err)
	}

	// Find the most-recent non-checkpoint blob — it should have old.go deleted.
	all, err := s.BlobLog(testCtx, 10)
	if err != nil {
		t.Fatal(err)
	}
	blobs := nonCheckpointBlobs(all)
	if len(blobs) < 1 {
		t.Fatal("expected at least 1 non-checkpoint blob")
	}
	files, err := s.BlobFiles(testCtx, blobs[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, f := range files {
		if f.Path == "old.go" && f.Role == "deleted" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected blob_file old.go (deleted), got: %v", files)
	}
}

func TestHookCheckout_BranchFlag1_EmitsBranchSwitch(t *testing.T) {
	gitRoot, _, s := setupTestRepo(t)
	commitFile(t, gitRoot, "init.go", "package main\n", "init")

	prevSHA, _ := exec.Command("git", "-C", gitRoot, "rev-parse", "HEAD").Output()
	prev := strings.TrimSpace(string(prevSHA))

	// Create and switch to a new branch.
	exec.Command("git", "-C", gitRoot, "checkout", "-b", "feature").Run()

	newSHA, _ := exec.Command("git", "-C", gitRoot, "rev-parse", "HEAD").Output()
	newRef := strings.TrimSpace(string(newSHA))

	if err := hookCheckout(testCtx, gitRoot, prev, newRef, "1", s); err != nil {
		t.Fatal(err)
	}

	pending, _ := s.PendingTasks(testCtx)
	switches := tasksOfKind(pending, task.KindBranchSwitch)
	if len(switches) != 1 {
		t.Fatalf("expected 1 BranchSwitch task, got %d", len(switches))
	}
	if !strings.Contains(switches[0].Detail, "feature") {
		t.Errorf("expected detail to contain branch name, got: %s", switches[0].Detail)
	}
}

func TestHookCheckout_FileFlag0_NoEmission(t *testing.T) {
	gitRoot, _, s := setupTestRepo(t)
	commitFile(t, gitRoot, "init.go", "package main\n", "init")

	// flag="0" means file checkout — no task should be emitted.
	if err := hookCheckout(testCtx, gitRoot, "abc", "def", "0", s); err != nil {
		t.Fatal(err)
	}

	pending, _ := s.PendingTasks(testCtx)
	if len(pending) != 0 {
		t.Errorf("expected 0 tasks for file checkout, got %d", len(pending))
	}
}

func TestHookMerge_EmitsMergeEvent(t *testing.T) {
	gitRoot, _, s := setupTestRepo(t)
	commitFile(t, gitRoot, "init.go", "package main\n", "init")

	// Simulate MERGE_HEAD (written by git merge before post-merge hook fires).
	mergedSHA := "abc123def456abc123def456abc123def456abc1"
	os.WriteFile(filepath.Join(gitRoot, ".git", "MERGE_HEAD"), []byte(mergedSHA+"\n"), 0644)

	if err := hookMerge(testCtx, gitRoot, s); err != nil {
		t.Fatal(err)
	}

	pending, _ := s.PendingTasks(testCtx)
	merges := tasksOfKind(pending, task.KindMergeEvent)
	if len(merges) != 1 {
		t.Fatalf("expected 1 MergeEvent task, got %d", len(merges))
	}
	if merges[0].Detail != mergedSHA {
		t.Errorf("expected detail=%s, got %s", mergedSHA, merges[0].Detail)
	}
}

func TestRecord_InsertsNoteTask(t *testing.T) {
	_, loreRoot, s := setupTestRepo(t)

	if err := recordNote(testCtx, loreRoot, "investigating JWT expiry bug"); err != nil {
		t.Fatal(err)
	}

	pending, _ := s.PendingTasks(testCtx)
	notes := tasksOfKind(pending, task.KindNote)
	if len(notes) != 1 {
		t.Fatalf("expected 1 Note task, got %d", len(notes))
	}
	if notes[0].Detail != "investigating JWT expiry bug" {
		t.Errorf("unexpected detail: %s", notes[0].Detail)
	}
	if notes[0].Source != "human" {
		t.Errorf("expected source=human, got %s", notes[0].Source)
	}
	if notes[0].TrustLevel != 1 {
		t.Errorf("expected trust_level=1, got %d", notes[0].TrustLevel)
	}
}
