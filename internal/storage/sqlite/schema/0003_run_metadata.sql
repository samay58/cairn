-- 0003: durable import and export summaries for agent-readable status.
ALTER TABLE sync_log ADD COLUMN source_path TEXT NOT NULL DEFAULT '';
ALTER TABLE sync_log ADD COLUMN rows_read INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sync_log ADD COLUMN valid_cards INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sync_log ADD COLUMN inserted_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sync_log ADD COLUMN updated_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sync_log ADD COLUMN unchanged_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sync_log ADD COLUMN tombstoned_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sync_log ADD COLUMN skipped_rows INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sync_log ADD COLUMN warning_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sync_log ADD COLUMN media_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sync_log ADD COLUMN chunk_count INTEGER NOT NULL DEFAULT 0;

CREATE TABLE export_log (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    started_at        TIMESTAMP NOT NULL,
    finished_at       TIMESTAMP,
    target_path       TEXT NOT NULL,
    cards_written     INTEGER NOT NULL DEFAULT 0,
    cards_unchanged   INTEGER NOT NULL DEFAULT 0,
    media_written     INTEGER NOT NULL DEFAULT 0,
    media_skipped     INTEGER NOT NULL DEFAULT 0,
    warning_count     INTEGER NOT NULL DEFAULT 0,
    status            TEXT NOT NULL,
    detail            TEXT NOT NULL DEFAULT ''
);
