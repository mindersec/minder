-- Bundles --

-- name: CreateBundle :one
INSERT INTO bundles (namespace, name) VALUES ($1, $2) RETURNING *;

-- name: DeleteBundle :exec
DELETE FROM bundles WHERE namespace = $1 AND name = $2;

-- name: BundleExists :one
SELECT 1 FROM bundles WHERE namespace = $1 AND name = $2;

-- Streams --

-- name: CreateStream :one
INSERT INTO streams (bundle_id, version) VALUES ($1, $2) RETURNING *;

-- name: DeleteStream :exec
DELETE FROM streams
WHERE bundle_id IN (
    SELECT id FROM bundles WHERE namespace = $1 AND name = $2
);

-- name: StreamExists :one
SELECT 1 FROM streams
JOIN bundles ON bundles.id = streams.bundle_id
WHERE bundles.namespace = $1 AND bundles.name = $2 AND streams.version = $3;

-- Subscriptions --

-- name: CreateSubscription :one
INSERT INTO subscriptions (project_id, bundle_id, stream_version)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListSubscriptionsByBundle :many
SELECT (su.project_id, bu.namespace, bu.name)
FROM subscriptions AS su JOIN bundles AS bu ON bu.id = su.bundle_id
WHERE bu.namespace = $1 AND bu.name = $2;

-- name: GetCurrentVersionByProjectBundle :one
SELECT st.version FROM subscriptions AS su
JOIN bundles AS bu ON bu.id = su.bundle_id
JOIN streams AS st ON st.bundle_id = su.bundle_id AND st.version = su.current_version
WHERE bu.namespace = $1 AND bu.name = $2 AND su.project_id = $3;

-- name: SetCurrentVersion :exec
UPDATE subscriptions
SET stream_version = $1
FROM subscriptions AS su
JOIN bundles as bu ON su.bundle_id = bu.id
WHERE su.project_id = $2 AND bu.namespace = $1 AND bu.name = $2;
