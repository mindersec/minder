-- name: CreateProfile :one
INSERT INTO profiles (  
    provider,
    project_id,
    remediate,
    name) VALUES ($1, $2, $3, $4) RETURNING *;

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
