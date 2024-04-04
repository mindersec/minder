-- name: CreateProfile :one
INSERT INTO profiles (  
    project_id,
    remediate,
    alert,
    name,
    subscription_id,
    display_name,
    labels
) VALUES ($1, $2, $3, $4, sqlc.narg(subscription_id), sqlc.arg(display_name), COALESCE(sqlc.arg(labels)::text[], '{}'::text[])) RETURNING *;

-- name: UpdateProfile :one
UPDATE profiles SET
    remediate = $3,
    alert = $4,
    updated_at = NOW(),
    display_name = sqlc.arg(display_name),
    labels = COALESCE(sqlc.arg(labels)::TEXT[], '{}'::TEXT[])
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
SELECT * FROM profiles JOIN profiles_with_entity_profiles ON profiles.id = profiles_with_entity_profiles.profid
WHERE profiles.project_id = $1 AND profiles.id = $2;

-- name: GetProfileByID :one
SELECT * FROM profiles WHERE id = $1 AND project_id = $2;

-- name: GetProfileByIDAndLock :one
SELECT * FROM profiles WHERE id = $1 AND project_id = $2 FOR UPDATE;

-- name: GetProfileByNameAndLock :one
SELECT * FROM profiles WHERE lower(name) = lower(sqlc.arg(name)) AND project_id = $1 FOR UPDATE;

-- name: ListProfilesByProjectID :many
SELECT sqlc.embed(profiles), sqlc.embed(profiles_with_entity_profiles) FROM profiles JOIN profiles_with_entity_profiles ON profiles.id = profiles_with_entity_profiles.profid
WHERE profiles.project_id = $1;

-- name: ListProfilesByProjectIDAndLabel :many
SELECT sqlc.embed(profiles), sqlc.embed(profiles_with_entity_profiles) FROM profiles JOIN profiles_with_entity_profiles ON profiles.id = profiles_with_entity_profiles.profid
WHERE profiles.project_id = $1
AND (
    -- the most common case first, if the include_labels is empty, we list profiles with no labels
    -- we use coalesce to handle the case where the include_labels is null
    (COALESCE(cardinality(sqlc.arg(include_labels)::TEXT[]), 0) = 0 AND profiles.labels = ARRAY[]::TEXT[]) OR
    -- if the include_labels arg is equal to '*', we list all profiles
    sqlc.arg(include_labels)::TEXT[] = ARRAY['*'] OR
    -- if the include_labels arg is not empty and not a wildcard, we list profiles whose labels are a subset of include_labels
    (COALESCE(cardinality(sqlc.arg(include_labels)::TEXT[]), 0) > 0 AND profiles.labels @> sqlc.arg(include_labels)::TEXT[])
) AND (
    -- if the exclude_labels arg is empty, we list all profiles
    COALESCE(cardinality(sqlc.arg(exclude_labels)::TEXT[]), 0) = 0 OR
    -- if the exclude_labels arg is not empty, we list profiles whose labels are not a subset of exclude_labels
    NOT profiles.labels @> sqlc.arg(exclude_labels)::TEXT[]
);

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
