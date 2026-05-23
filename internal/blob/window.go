package blob

import (
	"strings"

	"github.com/nishchay/lore/internal/task"
)

// Window is a fully-assembled extraction window derived from pending tasks.
type Window struct {
	Tasks        []task.Task
	StartedAt    int64
	EndedAt      int64
	CommitStart  string // SHA of earliest CommitCreated
	CommitEnd    string // SHA of latest CommitCreated
	FilesWritten []string
	FilesDeleted []string
	Commands     []string
	CommitMsgs   []string
	Sources      []string
	HasCommit    bool
	RecapTask    *task.Task // most recent AgentRecap by TS, nil if absent
}

// BuildWindow iterates tasks once and assembles all window fields.
func BuildWindow(tasks []task.Task) Window {
	var w Window
	w.Tasks = tasks

	writtenSet := make(map[string]struct{})
	deletedSet := make(map[string]struct{})
	commandSet := make(map[string]struct{})
	sourceSet := make(map[string]struct{})

	for i := range tasks {
		t := &tasks[i]

		if w.StartedAt == 0 || t.TS < w.StartedAt {
			w.StartedAt = t.TS
		}
		if t.TS > w.EndedAt {
			w.EndedAt = t.TS
		}

		sourceSet[t.Source] = struct{}{}

		switch t.Kind {
		case task.KindCommitCreated:
			w.HasCommit = true
			sha := strings.SplitN(t.Detail, "|", 2)[0]
			if w.CommitStart == "" {
				w.CommitStart = sha
			}
			w.CommitEnd = sha
			if len(t.Detail) > len(sha)+1 {
				w.CommitMsgs = append(w.CommitMsgs, t.Detail[len(sha)+1:])
			}
		case task.KindFileWrite:
			if t.Path != "" {
				writtenSet[t.Path] = struct{}{}
			}
		case task.KindFileDelete:
			if t.Path != "" {
				deletedSet[t.Path] = struct{}{}
			}
		case task.KindCommand:
			if t.Detail != "" {
				commandSet[t.Detail] = struct{}{}
			}
		case task.KindAgentRecap:
			if w.RecapTask == nil || t.TS > w.RecapTask.TS {
				tc := *t
				w.RecapTask = &tc
			}
		}
	}

	for p := range writtenSet {
		w.FilesWritten = append(w.FilesWritten, p)
	}
	for p := range deletedSet {
		w.FilesDeleted = append(w.FilesDeleted, p)
	}
	for c := range commandSet {
		w.Commands = append(w.Commands, c)
	}
	for src := range sourceSet {
		w.Sources = append(w.Sources, src)
	}
	return w
}
