-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

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
ON CONFLICT (profile_id, entity_type, lower(name)) DO UPDATE SET
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

-- name: DeleteRuleInstanceOfProfileInProject :exec
DELETE FROM rule_instances WHERE project_id = $1 AND profile_id = $2 AND rule_type_id = $3;
