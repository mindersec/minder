// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: entity_execution_lock.sql

package db

import (
	"context"

	"github.com/google/uuid"
)

const enqueueFlush = `-- name: EnqueueFlush :one
INSERT INTO flush_cache(
    entity,
    project_id,
    entity_instance_id
) VALUES(
    $1::entities,
    $2::UUID,
    $3::UUID
) ON CONFLICT(entity_instance_id)
DO NOTHING
RETURNING id, entity, queued_at, project_id, entity_instance_id
`

type EnqueueFlushParams struct {
	Entity           Entities  `json:"entity"`
	ProjectID        uuid.UUID `json:"project_id"`
	EntityInstanceID uuid.UUID `json:"entity_instance_id"`
}

func (q *Queries) EnqueueFlush(ctx context.Context, arg EnqueueFlushParams) (FlushCache, error) {
	row := q.db.QueryRowContext(ctx, enqueueFlush, arg.Entity, arg.ProjectID, arg.EntityInstanceID)
	var i FlushCache
	err := row.Scan(
		&i.ID,
		&i.Entity,
		&i.QueuedAt,
		&i.ProjectID,
		&i.EntityInstanceID,
	)
	return i, err
}

const flushCache = `-- name: FlushCache :one
DELETE FROM flush_cache
WHERE entity_instance_id= $1
RETURNING id, entity, queued_at, project_id, entity_instance_id
`

func (q *Queries) FlushCache(ctx context.Context, entityInstanceID uuid.UUID) (FlushCache, error) {
	row := q.db.QueryRowContext(ctx, flushCache, entityInstanceID)
	var i FlushCache
	err := row.Scan(
		&i.ID,
		&i.Entity,
		&i.QueuedAt,
		&i.ProjectID,
		&i.EntityInstanceID,
	)
	return i, err
}

const listFlushCache = `-- name: ListFlushCache :many
SELECT id, entity, queued_at, project_id, entity_instance_id FROM flush_cache
`

func (q *Queries) ListFlushCache(ctx context.Context) ([]FlushCache, error) {
	rows, err := q.db.QueryContext(ctx, listFlushCache)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []FlushCache{}
	for rows.Next() {
		var i FlushCache
		if err := rows.Scan(
			&i.ID,
			&i.Entity,
			&i.QueuedAt,
			&i.ProjectID,
			&i.EntityInstanceID,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const lockIfThresholdNotExceeded = `-- name: LockIfThresholdNotExceeded :one

INSERT INTO entity_execution_lock(
    entity,
    locked_by,
    last_lock_time,
    project_id,
    entity_instance_id
) VALUES(
    $1::entities,
    gen_random_uuid(),
    NOW(),
    $2::UUID,
    $3::UUID
) ON CONFLICT(entity_instance_id)
DO UPDATE SET
    locked_by = gen_random_uuid(),
    last_lock_time = NOW()
WHERE entity_execution_lock.last_lock_time < (NOW() - ($4::TEXT || ' seconds')::interval)
RETURNING id, entity, locked_by, last_lock_time, project_id, entity_instance_id
`

type LockIfThresholdNotExceededParams struct {
	Entity           Entities  `json:"entity"`
	ProjectID        uuid.UUID `json:"project_id"`
	EntityInstanceID uuid.UUID `json:"entity_instance_id"`
	Interval         string    `json:"interval"`
}

// LockIfThresholdNotExceeded is used to lock an entity for execution. It will
// attempt to insert or update the entity_execution_lock table only if the
// last_lock_time is older than the threshold. If the lock is successful, it
// will return the lock record. If the lock is unsuccessful, it will return
// NULL.
func (q *Queries) LockIfThresholdNotExceeded(ctx context.Context, arg LockIfThresholdNotExceededParams) (EntityExecutionLock, error) {
	row := q.db.QueryRowContext(ctx, lockIfThresholdNotExceeded,
		arg.Entity,
		arg.ProjectID,
		arg.EntityInstanceID,
		arg.Interval,
	)
	var i EntityExecutionLock
	err := row.Scan(
		&i.ID,
		&i.Entity,
		&i.LockedBy,
		&i.LastLockTime,
		&i.ProjectID,
		&i.EntityInstanceID,
	)
	return i, err
}

const releaseLock = `-- name: ReleaseLock :exec

DELETE FROM entity_execution_lock
WHERE entity_instance_id = $1 AND locked_by = $2::UUID
`

type ReleaseLockParams struct {
	EntityInstanceID uuid.UUID `json:"entity_instance_id"`
	LockedBy         uuid.UUID `json:"locked_by"`
}

// ReleaseLock is used to release a lock on an entity. It will delete the
// entity_execution_lock record if the lock is held by the given locked_by
// value.
func (q *Queries) ReleaseLock(ctx context.Context, arg ReleaseLockParams) error {
	_, err := q.db.ExecContext(ctx, releaseLock, arg.EntityInstanceID, arg.LockedBy)
	return err
}

const updateLease = `-- name: UpdateLease :exec
UPDATE entity_execution_lock SET last_lock_time = NOW()
WHERE entity_instance_id = $1 AND locked_by = $2::UUID
`

type UpdateLeaseParams struct {
	EntityInstanceID uuid.UUID `json:"entity_instance_id"`
	LockedBy         uuid.UUID `json:"locked_by"`
}

func (q *Queries) UpdateLease(ctx context.Context, arg UpdateLeaseParams) error {
	_, err := q.db.ExecContext(ctx, updateLease, arg.EntityInstanceID, arg.LockedBy)
	return err
}
