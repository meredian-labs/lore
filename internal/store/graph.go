package store

import (
	"context"

	"github.com/google/uuid"
)

func (s *Store) UpsertGraphNode(ctx context.Context, n GraphNode) (string, error) {
	if n.ID == "" {
		n.ID = uuid.NewString()
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO graph_nodes (id, kind, label, ref)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(kind, ref) DO UPDATE SET label = excluded.label`,
		n.ID, n.Kind, n.Label, n.Ref,
	)
	if err != nil {
		return "", err
	}
	// Return the actual ID (may differ from n.ID if row already existed).
	var id string
	err = s.db.QueryRowContext(ctx,
		`SELECT id FROM graph_nodes WHERE kind = ? AND ref = ?`, n.Kind, n.Ref,
	).Scan(&id)
	return id, err
}

func (s *Store) UpsertGraphEdge(ctx context.Context, e GraphEdge) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO graph_edges (id, from_id, to_id, relation, weight)
		 VALUES (?, ?, ?, ?, 1)
		 ON CONFLICT(from_id, to_id, relation) DO UPDATE SET weight = weight + 1`,
		e.ID, e.FromID, e.ToID, e.Relation,
	)
	return err
}

func (s *Store) GraphNodeCount(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM graph_nodes`).Scan(&n)
	return n, err
}

func (s *Store) GraphEdgeCount(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM graph_edges`).Scan(&n)
	return n, err
}

func (s *Store) GraphEdgesFrom(ctx context.Context, fromID string) ([]GraphEdge, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, from_id, to_id, relation, weight FROM graph_edges WHERE from_id = ?`, fromID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var edges []GraphEdge
	for rows.Next() {
		var e GraphEdge
		if err := rows.Scan(&e.ID, &e.FromID, &e.ToID, &e.Relation, &e.Weight); err != nil {
			return nil, err
		}
		edges = append(edges, e)
	}
	return edges, rows.Err()
}
