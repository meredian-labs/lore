package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/nishchay/lore/internal/blob"
)

func (s *Store) InsertBlobWithRelations(
	ctx context.Context,
	b blob.Blob,
	files []BlobFile,
	commands []BlobCommand,
	taskIDs []string,
) error {
	tags, err := json.Marshal(b.Tags)
	if err != nil {
		return fmt.Errorf("marshaling tags: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO blobs
		 (id, kind, title, summary, recap, user_intent, inferred_reasoning,
		  tags, trust_level, ai_source, started_at, ended_at,
		  commit_start, commit_end, primary_node_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		b.ID, string(b.Kind), b.Title, b.Summary, b.Recap, b.UserIntent,
		b.InferredReasoning, string(tags), b.TrustLevel, b.AISource,
		b.StartedAt, b.EndedAt, b.CommitStart, b.CommitEnd,
		nullableString(b.PrimaryNodeID), b.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting blob: %w", err)
	}

	for _, f := range files {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO blob_files (blob_id, path, role) VALUES (?, ?, ?)`,
			f.BlobID, f.Path, f.Role,
		); err != nil {
			return fmt.Errorf("inserting blob_file %s: %w", f.Path, err)
		}
	}

	for _, c := range commands {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO blob_commands (blob_id, command, ts) VALUES (?, ?, ?)`,
			c.BlobID, c.Command, c.TS,
		); err != nil {
			return fmt.Errorf("inserting blob_command: %w", err)
		}
	}

	if len(taskIDs) > 0 {
		for _, tid := range taskIDs {
			if _, err := tx.ExecContext(ctx,
				`INSERT OR IGNORE INTO blob_tasks (blob_id, task_id) VALUES (?, ?)`,
				b.ID, tid,
			); err != nil {
				return fmt.Errorf("inserting blob_task %s: %w", tid, err)
			}
		}
		// Mark tasks as extracted.
		placeholders := strings.Repeat("?,", len(taskIDs))
		placeholders = placeholders[:len(placeholders)-1]
		args := make([]any, 0, len(taskIDs)+1)
		args = append(args, b.ID)
		for _, id := range taskIDs {
			args = append(args, id)
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE tasks SET extracted = 1, extracted_into = ? WHERE id IN (`+placeholders+`)`,
			args...,
		); err != nil {
			return fmt.Errorf("marking tasks extracted: %w", err)
		}
	}

	return tx.Commit()
}

func (s *Store) BlobByID(ctx context.Context, id string) (blob.Blob, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, kind, title, summary, recap, user_intent, inferred_reasoning,
		        tags, trust_level, ai_source, started_at, ended_at,
		        commit_start, commit_end, COALESCE(primary_node_id, ''), created_at
		 FROM blobs WHERE id = ?`, id,
	)
	b, err := scanBlob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return blob.Blob{}, ErrNotFound
	}
	return b, err
}

func (s *Store) ResolveBlobIDPrefix(ctx context.Context, prefix string) (string, error) {
	if len(prefix) < 7 {
		return "", fmt.Errorf("blob ID prefix too short (minimum 7 characters)")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM blobs WHERE id LIKE ? LIMIT 2`, prefix+"%")
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return "", err
		}
		ids = append(ids, id)
	}
	switch len(ids) {
	case 0:
		return "", ErrNotFound
	case 1:
		return ids[0], nil
	default:
		return "", ErrAmbiguous
	}
}

func (s *Store) BlobsByFile(ctx context.Context, path string) ([]blob.Blob, error) {
	return s.blobsByFileSorted(ctx, path, "DESC")
}

func (s *Store) BlobsByFileChron(ctx context.Context, path string) ([]blob.Blob, error) {
	return s.blobsByFileSorted(ctx, path, "ASC")
}

func (s *Store) blobsByFileSorted(ctx context.Context, path, order string) ([]blob.Blob, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT b.id, b.kind, b.title, b.summary, b.recap, b.user_intent, b.inferred_reasoning,
		        b.tags, b.trust_level, b.ai_source, b.started_at, b.ended_at,
		        b.commit_start, b.commit_end, COALESCE(b.primary_node_id, ''), b.created_at
		 FROM blobs b
		 JOIN blob_files bf ON b.id = bf.blob_id
		 WHERE bf.path = ? OR bf.path LIKE '%/' || ?
		 ORDER BY b.started_at `+order,
		path, path,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBlobs(rows)
}

func (s *Store) BlobLog(ctx context.Context, limit int) ([]blob.Blob, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, kind, title, summary, recap, user_intent, inferred_reasoning,
		        tags, trust_level, ai_source, started_at, ended_at,
		        commit_start, commit_end, COALESCE(primary_node_id, ''), created_at
		 FROM blobs ORDER BY ended_at DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBlobs(rows)
}

func (s *Store) BlobCount(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM blobs`).Scan(&n)
	return n, err
}

func (s *Store) BlobCountByKind(ctx context.Context) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT kind, COUNT(*) FROM blobs GROUP BY kind ORDER BY COUNT(*) DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]int)
	for rows.Next() {
		var kind string
		var cnt int
		if err := rows.Scan(&kind, &cnt); err != nil {
			return nil, err
		}
		m[kind] = cnt
	}
	return m, rows.Err()
}

func (s *Store) BlobCountByTrust(ctx context.Context) (map[int]int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT trust_level, COUNT(*) FROM blobs GROUP BY trust_level`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[int]int)
	for rows.Next() {
		var trust, cnt int
		if err := rows.Scan(&trust, &cnt); err != nil {
			return nil, err
		}
		m[trust] = cnt
	}
	return m, rows.Err()
}

func (s *Store) SetBlobNode(ctx context.Context, blobID, nodeID string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE blobs SET primary_node_id = ? WHERE id = ?`, nodeID, blobID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) BlobFiles(ctx context.Context, blobID string) ([]BlobFile, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT blob_id, path, role FROM blob_files WHERE blob_id = ? ORDER BY role, path`,
		blobID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var files []BlobFile
	for rows.Next() {
		var f BlobFile
		if err := rows.Scan(&f.BlobID, &f.Path, &f.Role); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func (s *Store) BlobCommands(ctx context.Context, blobID string) ([]BlobCommand, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT blob_id, command, ts FROM blob_commands WHERE blob_id = ? ORDER BY ts`,
		blobID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cmds []BlobCommand
	for rows.Next() {
		var c BlobCommand
		if err := rows.Scan(&c.BlobID, &c.Command, &c.TS); err != nil {
			return nil, err
		}
		cmds = append(cmds, c)
	}
	return cmds, rows.Err()
}

func (s *Store) UnassignedBlobs(ctx context.Context, limit int) ([]blob.Blob, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, kind, title, summary, recap, user_intent, inferred_reasoning,
		        tags, trust_level, ai_source, started_at, ended_at,
		        commit_start, commit_end, COALESCE(primary_node_id, ''), created_at
		 FROM blobs WHERE primary_node_id IS NULL ORDER BY ended_at DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBlobs(rows)
}

func (s *Store) UnassignedBlobCount(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM blobs WHERE primary_node_id IS NULL`,
	).Scan(&n)
	return n, err
}

func (s *Store) BlobFileCount(ctx context.Context, blobID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM blob_files WHERE blob_id = ?`, blobID,
	).Scan(&n)
	return n, err
}

func (s *Store) LastExtractionTime(ctx context.Context) (int64, error) {
	var ts sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT MAX(created_at) FROM blobs`).Scan(&ts)
	if err != nil {
		return 0, err
	}
	return ts.Int64, nil
}

// scanBlob scans a single blob row from a QueryRow result.
func scanBlob(row *sql.Row) (blob.Blob, error) {
	var b blob.Blob
	var kind, tagsJSON string
	err := row.Scan(
		&b.ID, &kind, &b.Title, &b.Summary, &b.Recap, &b.UserIntent,
		&b.InferredReasoning, &tagsJSON, &b.TrustLevel, &b.AISource,
		&b.StartedAt, &b.EndedAt, &b.CommitStart, &b.CommitEnd,
		&b.PrimaryNodeID, &b.CreatedAt,
	)
	if err != nil {
		return blob.Blob{}, err
	}
	b.Kind = blob.BlobKind(kind)
	_ = json.Unmarshal([]byte(tagsJSON), &b.Tags)
	return b, nil
}

// scanBlobs scans multiple blob rows.
func scanBlobs(rows *sql.Rows) ([]blob.Blob, error) {
	var blobs []blob.Blob
	for rows.Next() {
		var b blob.Blob
		var kind, tagsJSON string
		if err := rows.Scan(
			&b.ID, &kind, &b.Title, &b.Summary, &b.Recap, &b.UserIntent,
			&b.InferredReasoning, &tagsJSON, &b.TrustLevel, &b.AISource,
			&b.StartedAt, &b.EndedAt, &b.CommitStart, &b.CommitEnd,
			&b.PrimaryNodeID, &b.CreatedAt,
		); err != nil {
			return nil, err
		}
		b.Kind = blob.BlobKind(kind)
		_ = json.Unmarshal([]byte(tagsJSON), &b.Tags)
		blobs = append(blobs, b)
	}
	return blobs, rows.Err()
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
