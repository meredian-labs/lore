CREATE TABLE IF NOT EXISTS meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tasks (
    id            TEXT PRIMARY KEY,
    kind          TEXT NOT NULL,
    path          TEXT NOT NULL DEFAULT '',
    detail        TEXT NOT NULL DEFAULT '',
    source        TEXT NOT NULL DEFAULT '',
    trust_level   INTEGER NOT NULL DEFAULT 1,
    ts            INTEGER NOT NULL,
    extracted     INTEGER NOT NULL DEFAULT 0,
    extracted_into TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_tasks_extracted ON tasks(extracted);
CREATE INDEX IF NOT EXISTS idx_tasks_kind ON tasks(kind);
CREATE INDEX IF NOT EXISTS idx_tasks_ts ON tasks(ts);

CREATE TABLE IF NOT EXISTS blobs (
    id                  TEXT PRIMARY KEY,
    kind                TEXT NOT NULL DEFAULT '',
    title               TEXT NOT NULL DEFAULT '',
    summary             TEXT NOT NULL DEFAULT '',
    recap               TEXT NOT NULL DEFAULT '',
    user_intent         TEXT NOT NULL DEFAULT '',
    inferred_reasoning  TEXT NOT NULL DEFAULT '',
    tags                TEXT NOT NULL DEFAULT '[]',
    trust_level         INTEGER NOT NULL DEFAULT 4,
    ai_source           TEXT NOT NULL DEFAULT '',
    started_at          INTEGER NOT NULL DEFAULT 0,
    ended_at            INTEGER NOT NULL DEFAULT 0,
    commit_start        TEXT NOT NULL DEFAULT '',
    commit_end          TEXT NOT NULL DEFAULT '',
    primary_node_id     TEXT,
    created_at          INTEGER NOT NULL DEFAULT (strftime('%s','now') * 1000000000)
);

CREATE INDEX IF NOT EXISTS idx_blobs_ended_at ON blobs(ended_at DESC);
CREATE INDEX IF NOT EXISTS idx_blobs_primary_node_id ON blobs(primary_node_id);

CREATE TABLE IF NOT EXISTS blob_files (
    blob_id TEXT NOT NULL,
    path    TEXT NOT NULL,
    role    TEXT NOT NULL DEFAULT 'written',
    PRIMARY KEY (blob_id, path, role)
);

CREATE INDEX IF NOT EXISTS idx_blob_files_blob ON blob_files(blob_id);
CREATE INDEX IF NOT EXISTS idx_blob_files_path ON blob_files(path);

CREATE TABLE IF NOT EXISTS blob_commands (
    blob_id TEXT NOT NULL,
    command TEXT NOT NULL,
    ts      INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (blob_id, command)
);

CREATE INDEX IF NOT EXISTS idx_blob_commands_blob ON blob_commands(blob_id);

INSERT OR IGNORE INTO meta (key, value) VALUES ('schema_version', '1');
INSERT OR IGNORE INTO meta (key, value) VALUES ('initialized_at', CAST(strftime('%s','now') AS INTEGER) * 1000000000);
INSERT OR IGNORE INTO meta (key, value) VALUES ('git_root', '');
