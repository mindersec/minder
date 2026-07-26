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
    project_id,
    entity_instance_id
) VALUES(
    sqlc.arg(entity)::entities,
    gen_random_uuid(),
    NOW(),
    sqlc.arg(project_id)::UUID,
    sqlc.arg(entity_instance_id)::UUID
) ON CONFLICT(entity_instance_id)
DO UPDATE SET
    locked_by = gen_random_uuid(),
    last_lock_time = NOW()
WHERE entity_execution_lock.last_lock_time < (NOW() - (@interval::TEXT || ' seconds')::interval)
RETURNING *;

-- ReleaseLock is used to release a lock on an entity. It will delete the
-- entity_execution_lock record if the lock is held by the given locked_by
-- value.

-- name: ReleaseLock :exec
DELETE FROM entity_execution_lock
WHERE entity_instance_id = sqlc.arg(entity_instance_id) AND locked_by = sqlc.arg(locked_by)::UUID;

-- name: UpdateLease :exec
UPDATE entity_execution_lock SET last_lock_time = NOW()
WHERE entity_instance_id = $1 AND locked_by = sqlc.arg(locked_by)::UUID;

-- name: EnqueueFlush :one
INSERT INTO flush_cache(
    entity,
    project_id,
    entity_instance_id
) VALUES(
    sqlc.arg(entity)::entities,
    sqlc.arg(project_id)::UUID,
    sqlc.arg(entity_instance_id)::UUID
) ON CONFLICT(entity_instance_id)
DO NOTHING
RETURNING *;

-- name: FlushCache :one
DELETE FROM flush_cache
WHERE flush_cache.entity_instance_id = sqlc.arg(entity_instance_id)
    AND flush_cache.project_id = sqlc.arg(project_id)
    AND EXISTS (
        SELECT 1 FROM entity_instances ei
        WHERE ei.id = flush_cache.entity_instance_id 
        AND ei.provider_id = sqlc.arg(provider_id)
    )
RETURNING *;

-- name: ListFlushCache :many
SELECT 
    fc.entity, 
    fc.project_id, 
    fc.entity_instance_id, 
    fc.queued_at, 
    ei.provider_id
FROM flush_cache fc
JOIN entity_instances ei ON fc.entity_instance_id = ei.id;