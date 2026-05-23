CREATE TABLE IF NOT EXISTS blob_tasks (
    blob_id  TEXT NOT NULL,
    task_id  TEXT NOT NULL,
    PRIMARY KEY (blob_id, task_id)
);

CREATE INDEX IF NOT EXISTS idx_blob_tasks_blob ON blob_tasks(blob_id);
CREATE INDEX IF NOT EXISTS idx_blob_tasks_task ON blob_tasks(task_id);

UPDATE meta SET value = '2' WHERE key = 'schema_version';
