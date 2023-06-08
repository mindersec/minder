-- name: CreateSessionState :one

INSERT INTO session_store (grp_id, port, session_state) VALUES ($1, $2, $3) RETURNING *;

-- name: GetSessionState :one
SELECT * FROM session_store WHERE id = $1;

-- name: GetSessionStateByGroupID :one
SELECT * FROM session_store WHERE grp_id = $1;

-- name: GetGroupIDPortBySessionState :one
SELECT grp_id, port FROM session_store WHERE session_state = $1;

-- name: DeleteSessionState :exec
DELETE FROM session_store WHERE id = $1;

-- name: DeleteSessionStateByGroupID :exec
DELETE FROM session_store WHERE grp_id = $1;

-- name: DeleteExpiredSessionStates :exec
DELETE FROM session_store WHERE created_at < NOW() - INTERVAL '1 day';
