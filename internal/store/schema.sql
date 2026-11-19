CREATE TABLE IF NOT EXISTS events (
    seq         INTEGER PRIMARY KEY AUTOINCREMENT, -- never reused, gaps mean tampering
    ts          INTEGER NOT NULL,                  -- unix ms, utc
    kind        TEXT    NOT NULL,
    session_id  TEXT    NOT NULL,
    project_id  TEXT    NOT NULL,
    harness     TEXT    NOT NULL,
    source      TEXT    NOT NULL,
    data        TEXT    NOT NULL DEFAULT '{}',     -- json payload
    uploaded_at INTEGER                            -- unix ms, null until sent
);

CREATE INDEX IF NOT EXISTS events_ts ON events (ts);
CREATE INDEX IF NOT EXISTS events_pending ON events (seq) WHERE uploaded_at IS NULL;
