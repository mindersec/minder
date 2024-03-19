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

-- name: UpsertBundle :one
INSERT INTO bundles (namespace, name) VALUES ($1, $2)
ON CONFLICT DO NOTHING
RETURNING *;

-- Subscriptions --

-- name: CreateSubscription :one
INSERT INTO subscriptions (project_id, bundle_id, current_version)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetSubscriptionByProjectBundle :one
SELECT su.* FROM subscriptions AS su
JOIN bundles AS bu ON bu.id = su.bundle_id
WHERE bu.namespace = $1 AND bu.name = $2 AND su.project_id = $3;

-- name: SetCurrentVersion :exec
UPDATE subscriptions
SET current_version = $1
FROM subscriptions AS su
JOIN bundles as bu ON su.bundle_id = bu.id
WHERE su.project_id = $2 AND bu.namespace = $1 AND bu.name = $2;
