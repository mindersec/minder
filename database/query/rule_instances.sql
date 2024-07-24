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

-- name: UpsertRuleInstance :one
INSERT INTO rule_instances (
    profile_id,
    rule_type_id,
    name,
    entity_type,
    def,
    params,
    project_id,
    created_at,
    updated_at
) VALUES(
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    NOW(),
    NOW()
)
ON CONFLICT (profile_id, entity_type, name) DO UPDATE SET
    rule_type_id = $2,
    def = $5,
    params = $6,
    updated_at = NOW()
RETURNING id;

-- name: GetRuleInstancesForProfile :many
SELECT * FROM rule_instances WHERE profile_id = $1;

-- name: GetRuleInstancesEntityInProjects :many
SELECT * FROM rule_instances
WHERE entity_type = $1
AND project_id = ANY(sqlc.arg(project_ids)::UUID[]);

-- name: DeleteNonUpdatedRules :exec
DELETE FROM rule_instances
WHERE profile_id = $1
AND entity_type = $2
AND NOT id = ANY(sqlc.arg(updated_ids)::UUID[]);

-- intended as a temporary transition query
-- this will be removed once rule_instances is used consistently in the engine
-- name: GetRuleTypeIDByRuleNameEntityProfile :one
SELECT rule_type_id FROM rule_instances
WHERE name = $1
AND entity_type = $2
AND profile_id = $3;
