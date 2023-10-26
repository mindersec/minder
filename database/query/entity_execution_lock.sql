-- LockIfThresholdNotExceeded is used to lock an entity for execution. It will
-- attempt to insert or update the entity_execution_lock table only if the
-- last_lock_time is older than the threshold. If the lock is successful, it
-- will return the lock record. If the lock is unsuccessful, it will return
-- NULL.

-- name: LockIfThresholdNotExceeded :one
INSERT INTO entity_execution_lock(
    entity,
    locked_by,
    last_lock_time,
    repository_id,
    artifact_id,
    pull_request_id
) VALUES(
    sqlc.arg(entity)::entities,
    gen_random_uuid(),
    NOW(),
    sqlc.arg(repository_id)::UUID,
    sqlc.narg(artifact_id)::UUID,
    sqlc.narg(pull_request_id)::UUID
) ON CONFLICT(entity, repository_id, COALESCE(artifact_id, '00000000-0000-0000-0000-000000000000'::UUID), COALESCE(pull_request_id, '00000000-0000-0000-0000-000000000000'::UUID))
DO UPDATE SET
    locked_by = gen_random_uuid(),
    last_lock_time = NOW()
WHERE entity_execution_lock.last_lock_time < NOW() - (@interval::TEXT || ' seconds')::interval
RETURNING *;

-- ReleaseLock is used to release a lock on an entity. It will delete the
-- entity_execution_lock record if the lock is held by the given locked_by
-- value.

-- name: ReleaseLock :exec
DELETE FROM entity_execution_lock
WHERE entity = sqlc.arg(entity)::entities AND repository_id = sqlc.arg(repository_id)::UUID AND
    COALESCE(artifact_id, '00000000-0000-0000-0000-000000000000'::UUID) = COALESCE(sqlc.narg(artifact_id)::UUID, '00000000-0000-0000-0000-000000000000'::UUID) AND
    COALESCE(pull_request_id, '00000000-0000-0000-0000-000000000000'::UUID) = COALESCE(sqlc.narg(pull_request_id)::UUID, '00000000-0000-0000-0000-000000000000'::UUID) AND
    locked_by = sqlc.arg(locked_by)::UUID;

-- name: UpdateLease :exec
UPDATE entity_execution_lock SET last_lock_time = NOW()
WHERE entity = $1 AND repository_id = $2 AND
COALESCE(artifact_id, '00000000-0000-0000-0000-000000000000'::UUID) = COALESCE(sqlc.narg(artifact_id)::UUID, '00000000-0000-0000-0000-000000000000'::UUID) AND
COALESCE(pull_request_id, '00000000-0000-0000-0000-000000000000'::UUID) = COALESCE(sqlc.narg(pull_request_id)::UUID, '00000000-0000-0000-0000-000000000000'::UUID) AND
locked_by = sqlc.arg(locked_by)::UUID;

-- name: EnqueueFlush :one
INSERT INTO flush_cache(
    entity,
    repository_id,
    artifact_id,
    pull_request_id
) VALUES(
    sqlc.arg(entity)::entities,
    sqlc.arg(repository_id)::UUID,
    sqlc.narg(artifact_id)::UUID,
    sqlc.narg(pull_request_id)::UUID
) ON CONFLICT(entity, repository_id, COALESCE(artifact_id, '00000000-0000-0000-0000-000000000000'::UUID), COALESCE(pull_request_id, '00000000-0000-0000-0000-000000000000'::UUID))
DO NOTHING
RETURNING *;

-- name: FlushCache :one
DELETE FROM flush_cache
WHERE entity = $1 AND repository_id = $2 AND
    COALESCE(artifact_id, '00000000-0000-0000-0000-000000000000'::UUID) = COALESCE(sqlc.narg(artifact_id)::UUID, '00000000-0000-0000-0000-000000000000'::UUID) AND
    COALESCE(pull_request_id, '00000000-0000-0000-0000-000000000000'::UUID) = COALESCE(sqlc.narg(pull_request_id)::UUID, '00000000-0000-0000-0000-000000000000'::UUID)
RETURNING *;

-- name: ListFlushCache :many
SELECT * FROM flush_cache;