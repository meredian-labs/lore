package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nishchay/lore/internal/task"
)

func (s *Store) InsertTask(ctx context.Context, t task.Task) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO tasks (id, kind, path, detail, source, trust_level, ts, extracted, extracted_into)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, string(t.Kind), t.Path, t.Detail, t.Source,
		t.TrustLevel, t.TS, boolToInt(t.Extracted), t.ExtractedInto,
	)
	return err
}

func (s *Store) InsertTasks(ctx context.Context, tasks []task.Task) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO tasks (id, kind, path, detail, source, trust_level, ts, extracted, extracted_into)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, t := range tasks {
		if _, err := stmt.ExecContext(ctx,
			t.ID, string(t.Kind), t.Path, t.Detail, t.Source,
			t.TrustLevel, t.TS, boolToInt(t.Extracted), t.ExtractedInto,
		); err != nil {
			return fmt.Errorf("inserting task %s: %w", t.ID, err)
		}
	}
	return tx.Commit()
}

func (s *Store) PendingTasks(ctx context.Context) ([]task.Task, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, kind, path, detail, source, trust_level, ts, extracted, extracted_into
		 FROM tasks WHERE extracted = 0 ORDER BY ts ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

func (s *Store) MarkTasksExtracted(ctx context.Context, blobID string, taskIDs []string) error {
	if len(taskIDs) == 0 {
		return nil
	}
	placeholders := strings.Repeat("?,", len(taskIDs))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, 0, len(taskIDs)+1)
	args = append(args, blobID)
	for _, id := range taskIDs {
		args = append(args, id)
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET extracted = 1, extracted_into = ? WHERE id IN (`+placeholders+`)`,
		args...,
	)
	return err
}

func (s *Store) PurgeExtractedTasks(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan).UnixNano()
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM tasks WHERE extracted = 1 AND ts < ?`, cutoff,
	)
	return err
}

func (s *Store) PurgeOldTasks(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan).UnixNano()
	_, err := s.db.ExecContext(ctx, `DELETE FROM tasks WHERE ts < ?`, cutoff)
	return err
}

func (s *Store) PendingTaskCount(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks WHERE extracted = 0`).Scan(&n)
	return n, err
}

func scanTasks(rows *sql.Rows) ([]task.Task, error) {
	var tasks []task.Task
	for rows.Next() {
		var t task.Task
		var kind string
		var extracted int
		if err := rows.Scan(
			&t.ID, &kind, &t.Path, &t.Detail, &t.Source,
			&t.TrustLevel, &t.TS, &extracted, &t.ExtractedInto,
		); err != nil {
			return nil, err
		}
		t.Kind = task.TaskKind(kind)
		t.Extracted = extracted == 1
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// TasksExtractedInto returns all tasks that were absorbed into the given blob,
// ordered by timestamp. Returns an empty slice (not an error) when tasks have
// been purged or the blob had no tasks.
func (s *Store) TasksExtractedInto(ctx context.Context, blobID string) ([]task.Task, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, kind, path, detail, source, trust_level, ts, extracted, extracted_into
		 FROM tasks WHERE extracted_into = ? ORDER BY ts ASC`,
		blobID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

func (s *Store) TaskByKindInWindow(ctx context.Context, kind task.TaskKind) (*task.Task, error) {
	var t task.Task
	var k string
	var extracted int
	err := s.db.QueryRowContext(ctx,
		`SELECT id, kind, path, detail, source, trust_level, ts, extracted, extracted_into
		 FROM tasks WHERE kind = ? AND extracted = 0 ORDER BY ts DESC LIMIT 1`,
		string(kind),
	).Scan(&t.ID, &k, &t.Path, &t.Detail, &t.Source, &t.TrustLevel, &t.TS, &extracted, &t.ExtractedInto)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t.Kind = task.TaskKind(k)
	t.Extracted = extracted == 1
	return &t, nil
}
