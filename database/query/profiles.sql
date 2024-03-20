-- name: CreateProfile :one
INSERT INTO profiles (  
    provider,
    project_id,
    remediate,
    alert,
    name,
    provider_id,
    subscription_id,
    display_name
) VALUES ($1, $2, $3, $4, $5, sqlc.arg(provider_id), sqlc.narg(subscription_id), sqlc.arg(display_name)) RETURNING *;

-- name: UpdateProfile :one
UPDATE profiles SET
    remediate = $3,
    alert = $4,
    updated_at = NOW(),
    display_name = sqlc.arg(display_name)
WHERE id = $1 AND project_id = $2 RETURNING *;

-- name: CreateProfileForEntity :one
INSERT INTO entity_profiles (
    entity,
    profile_id,
    contextual_rules) VALUES ($1, $2, sqlc.arg(contextual_rules)::jsonb) RETURNING *;

-- name: UpsertProfileForEntity :one
INSERT INTO entity_profiles (
    entity,
    profile_id,
    contextual_rules) VALUES ($1, $2, sqlc.arg(contextual_rules)::jsonb)
ON CONFLICT (entity, profile_id) DO UPDATE SET
    contextual_rules = sqlc.arg(contextual_rules)::jsonb
RETURNING *;

-- name: DeleteProfileForEntity :exec
DELETE FROM entity_profiles WHERE profile_id = $1 AND entity = $2;

-- name: GetProfileForEntity :one
SELECT * FROM entity_profiles WHERE profile_id = $1 AND entity = $2;

-- name: GetProfileByProjectAndID :many
SELECT * FROM profiles JOIN entity_profiles ON profiles.id = entity_profiles.profile_id
WHERE profiles.project_id = $1 AND profiles.id = $2;

-- name: GetProfileByID :one
SELECT * FROM profiles WHERE id = $1 AND project_id = $2;

-- name: GetProfileByIDAndLock :one
SELECT * FROM profiles WHERE id = $1 AND project_id = $2 FOR UPDATE;

-- name: GetProfileByNameAndLock :one
SELECT * FROM profiles WHERE lower(name) = lower(sqlc.arg(name)) AND project_id = $1 FOR UPDATE;

-- name: GetEntityProfileByProjectAndName :many
SELECT * FROM profiles JOIN entity_profiles ON profiles.id = entity_profiles.profile_id
WHERE profiles.project_id = $1 AND lower(profiles.name) = lower(sqlc.arg(name));

-- name: ListProfilesByProjectID :many
SELECT * FROM profiles JOIN entity_profiles ON profiles.id = entity_profiles.profile_id
WHERE profiles.project_id = $1;

-- name: DeleteProfile :exec
DELETE FROM profiles
WHERE id = $1 AND project_id = $2;

-- name: UpsertRuleInstantiation :one
INSERT INTO entity_profile_rules (entity_profile_id, rule_type_id)
VALUES ($1, $2)
ON CONFLICT (entity_profile_id, rule_type_id) DO NOTHING RETURNING *;

-- name: DeleteRuleInstantiation :exec
DELETE FROM entity_profile_rules WHERE entity_profile_id = $1 AND rule_type_id = $2;

-- name: ListProfilesInstantiatingRuleType :many
-- get profile information that instantiate a rule. This is done by joining the profiles with entity_profiles, then correlating those
-- with entity_profile_rules. The rule_type_id is used to filter the results. Note that we only really care about the overal profile,
-- so we only return the profile information. We also should group the profiles so that we don't get duplicates.
SELECT profiles.id, profiles.name, profiles.created_at FROM profiles
JOIN entity_profiles ON profiles.id = entity_profiles.profile_id 
JOIN entity_profile_rules ON entity_profiles.id = entity_profile_rules.entity_profile_id
WHERE entity_profile_rules.rule_type_id = $1
GROUP BY profiles.id;


-- name: CountProfilesByEntityType :many
SELECT COUNT(p.id) AS num_profiles, ep.entity AS profile_entity
FROM profiles AS p
         JOIN entity_profiles AS ep ON p.id = ep.profile_id
GROUP BY ep.entity;

-- name: CountProfilesByName :one
SELECT COUNT(*) AS num_named_profiles FROM profiles WHERE lower(name) = lower(sqlc.arg(name));
