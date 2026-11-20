-- name: InsertEvent :one
INSERT INTO events (ts, kind, session_id, project_id, harness, source, data)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING seq;

-- name: EventsBetween :many
SELECT * FROM events
WHERE ts >= sqlc.arg(from_ts) AND ts < sqlc.arg(to_ts)
ORDER BY ts, seq;

-- name: CountSessionEvents :one
SELECT count(*) FROM events WHERE session_id = ? AND source = ?;

-- name: PendingEvents :many
SELECT * FROM events
WHERE uploaded_at IS NULL
ORDER BY seq
LIMIT ?;

-- name: MarkUploaded :exec
UPDATE events
SET uploaded_at = sqlc.arg(uploaded_at)
WHERE seq <= sqlc.arg(up_to_seq) AND uploaded_at IS NULL;

-- name: GetMeta :one
SELECT value FROM meta WHERE key = ?;

-- name: SetMetaIfMissing :exec
INSERT INTO meta (key, value) VALUES (?, ?)
ON CONFLICT (key) DO NOTHING;

-- name: GetSessionState :one
SELECT data FROM session_state WHERE session_id = ?;

-- name: PutSessionState :exec
INSERT INTO session_state (session_id, data, updated_at) VALUES (?, ?, ?)
ON CONFLICT (session_id) DO UPDATE SET data = excluded.data, updated_at = excluded.updated_at;
