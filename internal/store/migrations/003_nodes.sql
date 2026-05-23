CREATE TABLE IF NOT EXISTS nodes (
    id          TEXT PRIMARY KEY,
    title       TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'active',
    created_by  TEXT NOT NULL DEFAULT 'user',
    created_at  INTEGER NOT NULL DEFAULT (strftime('%s','now') * 1000000000),
    updated_at  INTEGER NOT NULL DEFAULT (strftime('%s','now') * 1000000000)
);

CREATE INDEX IF NOT EXISTS idx_nodes_title ON nodes(title);
CREATE INDEX IF NOT EXISTS idx_nodes_status ON nodes(status);

CREATE TABLE IF NOT EXISTS node_blobs (
    node_id    TEXT NOT NULL,
    blob_id    TEXT NOT NULL,
    confidence REAL NOT NULL DEFAULT 1.0,
    PRIMARY KEY (node_id, blob_id)
);

CREATE INDEX IF NOT EXISTS idx_node_blobs_node ON node_blobs(node_id);
CREATE INDEX IF NOT EXISTS idx_node_blobs_blob ON node_blobs(blob_id);

UPDATE meta SET value = '3' WHERE key = 'schema_version';
