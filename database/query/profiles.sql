-- name: CreateProfile :one
INSERT INTO profiles (  
    provider,
    project_id,
    remediate,
    alert,
    name) VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: CreateProfileForEntity :one
INSERT INTO entity_profiles (
    entity,
    profile_id,
    contextual_rules) VALUES ($1, $2, sqlc.arg(contextual_rules)::jsonb) RETURNING *;

-- name: GetProfileByProjectAndID :many
SELECT * FROM profiles JOIN entity_profiles ON profiles.id = entity_profiles.profile_id
WHERE profiles.project_id = $1 AND profiles.id = $2;

-- name: GetProfileByID :one
SELECT * FROM profiles WHERE id = $1;

-- name: GetProfileByProjectAndName :many
SELECT * FROM profiles JOIN entity_profiles ON profiles.id = entity_profiles.profile_id
WHERE profiles.project_id = $1 AND profiles.name = $2;

-- name: ListProfilesByProjectID :many
SELECT * FROM profiles JOIN entity_profiles ON profiles.id = entity_profiles.profile_id
WHERE profiles.project_id = $1;

-- name: DeleteProfile :exec
DELETE FROM profiles
WHERE id = $1;

-- name: UpsertRuleInstantiation :one
INSERT INTO entity_profile_rules (entity_profile_id, rule_type_id)
VALUES ($1, $2)
ON CONFLICT (entity_profile_id, rule_type_id) DO NOTHING RETURNING *;

-- name: ListProfilesInstantiatingRuleType :many
-- get profile information that instantiate a rule. This is done by joining the profiles with entity_profiles, then correlating those
-- with entity_profile_rules. The rule_type_id is used to filter the results. Note that we only really care about the overal profile,
-- so we only return the profile information. We also should group the profiles so that we don't get duplicates.
SELECT profiles.id, profiles.name, profiles.created_at FROM profiles
JOIN entity_profiles ON profiles.id = entity_profiles.profile_id 
JOIN entity_profile_rules ON entity_profiles.id = entity_profile_rules.entity_profile_id
WHERE entity_profile_rules.rule_type_id = $1
GROUP BY profiles.id;


