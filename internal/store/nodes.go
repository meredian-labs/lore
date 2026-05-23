package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/meredian-labs/lore/internal/blob"
	"github.com/meredian-labs/lore/internal/node"
)

func (s *Store) InsertNode(ctx context.Context, n node.Node) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO nodes (id, title, description, status, created_by, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		n.ID, n.Title, n.Description, n.Status, n.CreatedBy, n.CreatedAt, n.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return ErrAlreadyExists
		}
		return err
	}
	return nil
}

func (s *Store) NodeByTitle(ctx context.Context, title string) (node.Node, error) {
	var n node.Node
	err := s.db.QueryRowContext(ctx,
		`SELECT id, title, description, status, created_by, created_at, updated_at
		 FROM nodes WHERE LOWER(title) = LOWER(?)`, title,
	).Scan(&n.ID, &n.Title, &n.Description, &n.Status, &n.CreatedBy, &n.CreatedAt, &n.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return node.Node{}, ErrNotFound
	}
	return n, err
}

func (s *Store) NodeByID(ctx context.Context, id string) (node.Node, error) {
	var n node.Node
	err := s.db.QueryRowContext(ctx,
		`SELECT id, title, description, status, created_by, created_at, updated_at
		 FROM nodes WHERE id = ?`, id,
	).Scan(&n.ID, &n.Title, &n.Description, &n.Status, &n.CreatedBy, &n.CreatedAt, &n.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return node.Node{}, ErrNotFound
	}
	return n, err
}

func (s *Store) ListNodes(ctx context.Context) ([]node.Node, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, title, description, status, created_by, created_at, updated_at
		 FROM nodes WHERE status = 'active' ORDER BY title`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var nodes []node.Node
	for rows.Next() {
		var n node.Node
		if err := rows.Scan(&n.ID, &n.Title, &n.Description, &n.Status, &n.CreatedBy, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

func (s *Store) BlobsForNode(ctx context.Context, nodeID string) ([]blob.Blob, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, kind, title, summary, recap, user_intent, inferred_reasoning,
		        tags, trust_level, ai_source, started_at, ended_at,
		        commit_start, commit_end, COALESCE(primary_node_id, ''), created_at
		 FROM blobs WHERE primary_node_id = ? ORDER BY started_at ASC`,
		nodeID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBlobs(rows)
}

func (s *Store) NodeBlobCount(ctx context.Context, nodeID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM blobs WHERE primary_node_id = ?`, nodeID,
	).Scan(&n)
	return n, err
}

func (s *Store) NodeCount(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM nodes WHERE status = 'active'`).Scan(&n)
	return n, err
}
