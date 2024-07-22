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
    contextual_rules,
    migrated
) VALUES (
    $1,
    $2,
    sqlc.arg(contextual_rules)::jsonb,
    TRUE
) RETURNING *;

-- name: UpsertProfileForEntity :one
INSERT INTO entity_profiles (
    entity,
    profile_id,
    contextual_rules,
    migrated
) VALUES ($1, $2, sqlc.arg(contextual_rules)::jsonb, false)
ON CONFLICT (entity, profile_id) DO UPDATE SET
    contextual_rules = sqlc.arg(contextual_rules)::jsonb,
    migrated = TRUE
RETURNING *;

-- name: DeleteProfileForEntity :exec
DELETE FROM entity_profiles WHERE profile_id = $1 AND entity = $2;

-- name: GetProfileForEntity :one
SELECT * FROM entity_profiles WHERE profile_id = $1 AND entity = $2;

-- name: GetProfileByProjectAndID :many
WITH helper AS(
    SELECT pr.id as profid,
           ARRAY_AGG(ROW(ps.id, ps.profile_id, ps.entity, ps.selector, ps.comment)::profile_selector) AS selectors
    FROM profiles pr
             JOIN profile_selectors ps
                  ON pr.id = ps.profile_id
    WHERE pr.project_id = $1
    GROUP BY pr.id
)
SELECT
    sqlc.embed(profiles),
    sqlc.embed(profiles_with_entity_profiles),
    helper.selectors::profile_selector[] AS profiles_with_selectors
FROM profiles
JOIN profiles_with_entity_profiles ON profiles.id = profiles_with_entity_profiles.profid
LEFT JOIN helper ON profiles.id = helper.profid
WHERE profiles.project_id = $1 AND profiles.id = $2;

-- name: GetProfileByID :one
SELECT * FROM profiles WHERE id = $1 AND project_id = $2;

-- name: GetProfileByIDAndLock :one
SELECT * FROM profiles WHERE id = $1 AND project_id = $2 FOR UPDATE;

-- name: GetProfileByNameAndLock :one
SELECT * FROM profiles WHERE lower(name) = lower(sqlc.arg(name)) AND project_id = $1 FOR UPDATE;

-- name: ListProfilesByProjectID :many
WITH helper AS(
     SELECT pr.id as profid,
            ARRAY_AGG(ROW(ps.id, ps.profile_id, ps.entity, ps.selector, ps.comment)::profile_selector) AS selectors
       FROM profiles pr
       JOIN profile_selectors ps
         ON pr.id = ps.profile_id
      WHERE pr.project_id = $1
      GROUP BY pr.id
)
SELECT
    sqlc.embed(profiles),
    sqlc.embed(profiles_with_entity_profiles),
    helper.selectors::profile_selector[] AS profiles_with_selectors
FROM profiles
JOIN profiles_with_entity_profiles ON profiles.id = profiles_with_entity_profiles.profid
LEFT JOIN helper ON profiles.id = helper.profid
WHERE profiles.project_id = $1;

-- name: ListProfilesByProjectIDAndLabel :many
WITH helper AS(
     SELECT pr.id as profid,
     ARRAY_AGG(ROW(ps.id, ps.profile_id, ps.entity, ps.selector, ps.comment)::profile_selector) AS selectors
       FROM profiles pr
       JOIN profile_selectors ps
         ON pr.id = ps.profile_id
      WHERE pr.project_id = $1
      GROUP BY pr.id
)
SELECT sqlc.embed(profiles),
       sqlc.embed(profiles_with_entity_profiles),
       helper.selectors::profile_selector[] AS profiles_with_selectors
FROM profiles
JOIN profiles_with_entity_profiles ON profiles.id = profiles_with_entity_profiles.profid
LEFT JOIN helper ON profiles.id = helper.profid
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
    -- if the exclude_labels arg is not empty, we exclude profiles containing any of the exclude_labels
    NOT profiles.labels::TEXT[] && sqlc.arg(exclude_labels)::TEXT[]
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
SELECT DISTINCT(p.name)
FROM profiles AS p
JOIN rule_instances AS r ON p.id = r.profile_id
WHERE r.rule_type_id = $1;

-- name: CountProfilesByEntityType :many
SELECT COUNT(DISTINCT(p.id)) AS num_profiles, r.entity_type AS profile_entity
FROM profiles AS p
JOIN rule_instances AS r ON p.id = r.profile_id
GROUP BY r.entity_type;

-- name: CountProfilesByName :one
SELECT COUNT(*) AS num_named_profiles FROM profiles WHERE lower(name) = lower(sqlc.arg(name));

-- name: BulkGetProfilesByID :many
SELECT *
FROM profiles
WHERE id = ANY(sqlc.arg(profile_ids)::UUID[]);
