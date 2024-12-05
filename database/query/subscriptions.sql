-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- Bundles --

-- name: UpsertBundle :exec
INSERT INTO bundles (namespace, name) VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: GetBundle :one
SELECT * FROM bundles WHERE namespace = $1 AND name = $2;

-- Subscriptions --

-- name: CreateSubscription :one
INSERT INTO subscriptions (project_id, bundle_id, current_version)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetSubscriptionByProjectBundle :one
SELECT su.* FROM subscriptions AS su
JOIN bundles AS bu ON bu.id = su.bundle_id
WHERE bu.namespace = $1 AND bu.name = $2 AND su.project_id = $3;

-- name: SetSubscriptionBundleVersion :exec
UPDATE subscriptions SET current_version = $2 WHERE project_id = $1;
