-- name: CreateSessionState :one
INSERT INTO session_store (provider, grp_id, port, session_state, owner_filter) VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: GetSessionState :one
SELECT * FROM session_store WHERE id = $1;

-- name: GetSessionStateByGroupID :one
SELECT * FROM session_store WHERE grp_id = $1;

-- name: GetGroupIDPortBySessionState :one
SELECT provider, grp_id, port, owner_filter FROM session_store WHERE session_state = $1;

-- name: DeleteSessionState :exec
DELETE FROM session_store WHERE id = $1;

-- name: DeleteSessionStateByGroupID :exec
DELETE FROM session_store WHERE provider=$1 AND grp_id = $2;

-- name: DeleteExpiredSessionStates :exec
DELETE FROM session_store WHERE created_at < NOW() - INTERVAL '1 day';