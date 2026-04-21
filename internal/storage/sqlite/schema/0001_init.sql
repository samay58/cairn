-- migration 0001: initial schema
CREATE TABLE cards (
    id           TEXT PRIMARY KEY,
    mymind_id    TEXT NOT NULL UNIQUE,
    kind         TEXT NOT NULL,
    title        TEXT NOT NULL,
    url          TEXT,
    body         TEXT,
    excerpt      TEXT,
    source       TEXT,
    captured_at  TIMESTAMP NOT NULL,
    updated_at   TIMESTAMP NOT NULL,
    deleted_at   TIMESTAMP
);
CREATE INDEX cards_captured_at_idx ON cards(captured_at);
CREATE INDEX cards_deleted_at_idx ON cards(deleted_at);

CREATE TABLE card_meta (
    card_id  TEXT NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    key      TEXT NOT NULL,
    value    TEXT,
    PRIMARY KEY (card_id, key)
);

CREATE TABLE tags (
    card_id  TEXT NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    tag      TEXT NOT NULL,
    PRIMARY KEY (card_id, tag)
);
CREATE INDEX tags_tag_idx ON tags(tag);

CREATE TABLE media (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    card_id  TEXT NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    kind     TEXT NOT NULL,
    path     TEXT NOT NULL,
    sha256   TEXT NOT NULL,
    mime     TEXT
);
CREATE INDEX media_card_id_idx ON media(card_id);

CREATE TABLE chunks (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    card_id       TEXT NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    modality      TEXT NOT NULL,
    text          TEXT NOT NULL,
    start_offset  INTEGER NOT NULL,
    end_offset    INTEGER NOT NULL,
    checksum      TEXT NOT NULL
);
CREATE INDEX chunks_card_id_idx ON chunks(card_id);

CREATE TABLE sync_log (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    started_at    TIMESTAMP NOT NULL,
    finished_at   TIMESTAMP,
    delta_count   INTEGER NOT NULL DEFAULT 0,
    status        TEXT NOT NULL
);

CREATE TABLE handles (
    position    INTEGER PRIMARY KEY,
    card_id     TEXT NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    created_at  TIMESTAMP NOT NULL
);

CREATE VIRTUAL TABLE cards_fts USING fts5(title, body, tags_flat, content='');

CREATE TRIGGER cards_ai AFTER INSERT ON cards BEGIN
    INSERT INTO cards_fts(rowid, title, body, tags_flat)
    VALUES (new.rowid, new.title, coalesce(new.body, ''), '');
END;
CREATE TRIGGER cards_ad AFTER DELETE ON cards BEGIN
    INSERT INTO cards_fts(cards_fts, rowid, title, body, tags_flat)
    VALUES ('delete', old.rowid, old.title, coalesce(old.body, ''), '');
END;
CREATE TRIGGER cards_au AFTER UPDATE ON cards BEGIN
    INSERT INTO cards_fts(cards_fts, rowid, title, body, tags_flat)
    VALUES ('delete', old.rowid, old.title, coalesce(old.body, ''), '');
    INSERT INTO cards_fts(rowid, title, body, tags_flat)
    VALUES (new.rowid, new.title, coalesce(new.body, ''), '');
END;
