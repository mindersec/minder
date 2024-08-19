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
    repository_id,
    artifact_id,
    pull_request_id,
    project_id,
    entity_instance_id
) VALUES(
    $1::entities,
    $2::UUID,
    $3::UUID,
    $4::UUID,
    $5::UUID,
    $6::UUID
) ON CONFLICT(entity, COALESCE(repository_id, '00000000-0000-0000-0000-000000000000'::UUID), COALESCE(artifact_id, '00000000-0000-0000-0000-000000000000'::UUID), COALESCE(pull_request_id, '00000000-0000-0000-0000-000000000000'::UUID))
DO NOTHING
RETURNING id, entity, repository_id, artifact_id, pull_request_id, queued_at, project_id, entity_instance_id
`

type EnqueueFlushParams struct {
	Entity           Entities      `json:"entity"`
	RepositoryID     uuid.NullUUID `json:"repository_id"`
	ArtifactID       uuid.NullUUID `json:"artifact_id"`
	PullRequestID    uuid.NullUUID `json:"pull_request_id"`
	ProjectID        uuid.UUID     `json:"project_id"`
	EntityInstanceID uuid.NullUUID `json:"entity_instance_id"`
}

func (q *Queries) EnqueueFlush(ctx context.Context, arg EnqueueFlushParams) (FlushCache, error) {
	row := q.db.QueryRowContext(ctx, enqueueFlush,
		arg.Entity,
		arg.RepositoryID,
		arg.ArtifactID,
		arg.PullRequestID,
		arg.ProjectID,
		arg.EntityInstanceID,
	)
	var i FlushCache
	err := row.Scan(
		&i.ID,
		&i.Entity,
		&i.RepositoryID,
		&i.ArtifactID,
		&i.PullRequestID,
		&i.QueuedAt,
		&i.ProjectID,
		&i.EntityInstanceID,
	)
	return i, err
}

const flushCache = `-- name: FlushCache :one
DELETE FROM flush_cache
WHERE entity = $1 AND
    COALESCE(repository_id, '00000000-0000-0000-0000-000000000000'::UUID) = COALESCE($2::UUID, '00000000-0000-0000-0000-000000000000'::UUID) AND
    COALESCE(artifact_id, '00000000-0000-0000-0000-000000000000'::UUID) = COALESCE($3::UUID, '00000000-0000-0000-0000-000000000000'::UUID) AND
    COALESCE(pull_request_id, '00000000-0000-0000-0000-000000000000'::UUID) = COALESCE($4::UUID, '00000000-0000-0000-0000-000000000000'::UUID)
RETURNING id, entity, repository_id, artifact_id, pull_request_id, queued_at, project_id, entity_instance_id
`

type FlushCacheParams struct {
	Entity        Entities      `json:"entity"`
	RepositoryID  uuid.NullUUID `json:"repository_id"`
	ArtifactID    uuid.NullUUID `json:"artifact_id"`
	PullRequestID uuid.NullUUID `json:"pull_request_id"`
}

func (q *Queries) FlushCache(ctx context.Context, arg FlushCacheParams) (FlushCache, error) {
	row := q.db.QueryRowContext(ctx, flushCache,
		arg.Entity,
		arg.RepositoryID,
		arg.ArtifactID,
		arg.PullRequestID,
	)
	var i FlushCache
	err := row.Scan(
		&i.ID,
		&i.Entity,
		&i.RepositoryID,
		&i.ArtifactID,
		&i.PullRequestID,
		&i.QueuedAt,
		&i.ProjectID,
		&i.EntityInstanceID,
	)
	return i, err
}

const listFlushCache = `-- name: ListFlushCache :many
SELECT id, entity, repository_id, artifact_id, pull_request_id, queued_at, project_id, entity_instance_id FROM flush_cache
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
			&i.RepositoryID,
			&i.ArtifactID,
			&i.PullRequestID,
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
    repository_id,
    artifact_id,
    pull_request_id,
    project_id,
    entity_instance_id
) VALUES(
    $1::entities,
    gen_random_uuid(),
    NOW(),
    $2::UUID,
    $3::UUID,
    $4::UUID,
    $5::UUID,
    $6::UUID
) ON CONFLICT(entity, COALESCE(repository_id, '00000000-0000-0000-0000-000000000000'::UUID), COALESCE(artifact_id, '00000000-0000-0000-0000-000000000000'::UUID), COALESCE(pull_request_id, '00000000-0000-0000-0000-000000000000'::UUID))
DO UPDATE SET
    locked_by = gen_random_uuid(),
    last_lock_time = NOW(),
    entity_instance_id = $6::UUID
WHERE entity_execution_lock.last_lock_time < (NOW() - ($7::TEXT || ' seconds')::interval)
RETURNING id, entity, locked_by, last_lock_time, repository_id, artifact_id, pull_request_id, project_id, entity_instance_id
`

type LockIfThresholdNotExceededParams struct {
	Entity           Entities      `json:"entity"`
	RepositoryID     uuid.NullUUID `json:"repository_id"`
	ArtifactID       uuid.NullUUID `json:"artifact_id"`
	PullRequestID    uuid.NullUUID `json:"pull_request_id"`
	ProjectID        uuid.UUID     `json:"project_id"`
	EntityInstanceID uuid.NullUUID `json:"entity_instance_id"`
	Interval         string        `json:"interval"`
}

// LockIfThresholdNotExceeded is used to lock an entity for execution. It will
// attempt to insert or update the entity_execution_lock table only if the
// last_lock_time is older than the threshold. If the lock is successful, it
// will return the lock record. If the lock is unsuccessful, it will return
// NULL.
func (q *Queries) LockIfThresholdNotExceeded(ctx context.Context, arg LockIfThresholdNotExceededParams) (EntityExecutionLock, error) {
	row := q.db.QueryRowContext(ctx, lockIfThresholdNotExceeded,
		arg.Entity,
		arg.RepositoryID,
		arg.ArtifactID,
		arg.PullRequestID,
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
		&i.RepositoryID,
		&i.ArtifactID,
		&i.PullRequestID,
		&i.ProjectID,
		&i.EntityInstanceID,
	)
	return i, err
}

const releaseLock = `-- name: ReleaseLock :exec

DELETE FROM entity_execution_lock
WHERE entity = $1::entities AND
    COALESCE(repository_id, '00000000-0000-0000-0000-000000000000'::UUID) = COALESCE($2::UUID, '00000000-0000-0000-0000-000000000000'::UUID) AND
    COALESCE(artifact_id, '00000000-0000-0000-0000-000000000000'::UUID) = COALESCE($3::UUID, '00000000-0000-0000-0000-000000000000'::UUID) AND
    COALESCE(pull_request_id, '00000000-0000-0000-0000-000000000000'::UUID) = COALESCE($4::UUID, '00000000-0000-0000-0000-000000000000'::UUID) AND
    locked_by = $5::UUID
`

type ReleaseLockParams struct {
	Entity        Entities      `json:"entity"`
	RepositoryID  uuid.NullUUID `json:"repository_id"`
	ArtifactID    uuid.NullUUID `json:"artifact_id"`
	PullRequestID uuid.NullUUID `json:"pull_request_id"`
	LockedBy      uuid.UUID     `json:"locked_by"`
}

// ReleaseLock is used to release a lock on an entity. It will delete the
// entity_execution_lock record if the lock is held by the given locked_by
// value.
func (q *Queries) ReleaseLock(ctx context.Context, arg ReleaseLockParams) error {
	_, err := q.db.ExecContext(ctx, releaseLock,
		arg.Entity,
		arg.RepositoryID,
		arg.ArtifactID,
		arg.PullRequestID,
		arg.LockedBy,
	)
	return err
}

const updateLease = `-- name: UpdateLease :exec
UPDATE entity_execution_lock SET last_lock_time = NOW()
WHERE entity = $1 AND
COALESCE(repository_id, '00000000-0000-0000-0000-000000000000'::UUID) = COALESCE($2, '00000000-0000-0000-0000-000000000000'::UUID) AND
COALESCE(artifact_id, '00000000-0000-0000-0000-000000000000'::UUID) = COALESCE($3::UUID, '00000000-0000-0000-0000-000000000000'::UUID) AND
COALESCE(pull_request_id, '00000000-0000-0000-0000-000000000000'::UUID) = COALESCE($4::UUID, '00000000-0000-0000-0000-000000000000'::UUID) AND
locked_by = $5::UUID
`

type UpdateLeaseParams struct {
	Entity        Entities      `json:"entity"`
	RepositoryID  uuid.NullUUID `json:"repository_id"`
	ArtifactID    uuid.NullUUID `json:"artifact_id"`
	PullRequestID uuid.NullUUID `json:"pull_request_id"`
	LockedBy      uuid.UUID     `json:"locked_by"`
}

func (q *Queries) UpdateLease(ctx context.Context, arg UpdateLeaseParams) error {
	_, err := q.db.ExecContext(ctx, updateLease,
		arg.Entity,
		arg.RepositoryID,
		arg.ArtifactID,
		arg.PullRequestID,
		arg.LockedBy,
	)
	return err
}
