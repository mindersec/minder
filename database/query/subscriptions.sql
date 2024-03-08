-- Copyright 2024 Stacklok, Inc
--
-- Licensed under the Apache License, Version 2.0 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
--
--      http://www.apache.org/licenses/LICENSE-2.0
--
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.

-- Bundles --

-- name: CreateBundle :one
INSERT INTO bundles (namespace, name) VALUES ($1, $2) RETURNING *;

-- name: DeleteBundle :exec
DELETE FROM bundles WHERE namespace = $1 AND name = $2;

-- name: BundleExists :one
SELECT bundles.id FROM bundles WHERE namespace = $1 AND name = $2;

-- Subscriptions --

-- name: CreateSubscription :one
INSERT INTO subscriptions (project_id, bundle_id, current_version)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListSubscriptionsByBundle :many
SELECT *
FROM subscriptions AS su JOIN bundles AS bu ON bu.id = su.bundle_id
WHERE bu.namespace = $1 AND bu.name = $2;

-- name: GetSubscriptionByProjectBundle :one
SELECT * FROM subscriptions AS su
JOIN bundles AS bu ON bu.id = su.bundle_id
WHERE bu.namespace = $1 AND bu.name = $2 AND su.project_id = $3;

-- name: GetSubscriptionByProjectBundleVersion :one
SELECT * FROM subscriptions AS su
JOIN bundles AS bu ON bu.id = su.bundle_id
WHERE bu.namespace = $1 AND bu.name = $2 AND su.project_id = $3 AND su.current_version = $4;

-- name: SetCurrentVersion :exec
UPDATE subscriptions
SET current_version = $1
FROM subscriptions AS su
JOIN bundles as bu ON su.bundle_id = bu.id
WHERE su.project_id = $2 AND bu.namespace = $1 AND bu.name = $2;

-- name: ListSubscriptionProfilesInProject :many
SELECT * FROM profiles as p
JOIN bundles AS b ON b.id = p.subscription_id
WHERE p.id = $1 AND b.namespace = $2 AND b.name = $3;

-- name: ListSubscriptionRuleTypesInProject :many
SELECT * FROM rule_type as r
JOIN bundles AS b ON b.id = r.subscription_id
WHERE r.project_id = $1 AND b.namespace = $2 AND b.name = $3;