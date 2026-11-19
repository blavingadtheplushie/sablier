-- name: InsertEvent :one
INSERT INTO events (ts, kind, session_id, project_id, harness, source, data)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING seq;

-- name: EventsBetween :many
SELECT * FROM events
WHERE ts >= sqlc.arg(from_ts) AND ts < sqlc.arg(to_ts)
ORDER BY ts, seq;

-- name: PendingEvents :many
SELECT * FROM events
WHERE uploaded_at IS NULL
ORDER BY seq
LIMIT ?;

-- name: MarkUploaded :exec
UPDATE events
SET uploaded_at = sqlc.arg(uploaded_at)
WHERE seq <= sqlc.arg(up_to_seq) AND uploaded_at IS NULL;
