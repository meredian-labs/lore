CREATE TABLE IF NOT EXISTS graph_nodes (
    id    TEXT PRIMARY KEY,
    kind  TEXT NOT NULL,
    label TEXT NOT NULL DEFAULT '',
    ref   TEXT NOT NULL DEFAULT '',
    UNIQUE(kind, ref)
);

CREATE INDEX IF NOT EXISTS idx_graph_nodes_kind ON graph_nodes(kind);
CREATE INDEX IF NOT EXISTS idx_graph_nodes_ref ON graph_nodes(ref);

CREATE TABLE IF NOT EXISTS graph_edges (
    id       TEXT PRIMARY KEY,
    from_id  TEXT NOT NULL,
    to_id    TEXT NOT NULL,
    relation TEXT NOT NULL,
    weight   INTEGER NOT NULL DEFAULT 1,
    UNIQUE(from_id, to_id, relation)
);

CREATE INDEX IF NOT EXISTS idx_graph_edges_from ON graph_edges(from_id);
CREATE INDEX IF NOT EXISTS idx_graph_edges_to ON graph_edges(to_id);
CREATE INDEX IF NOT EXISTS idx_graph_edges_relation ON graph_edges(relation);

UPDATE meta SET value = '4' WHERE key = 'schema_version';
