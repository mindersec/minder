-- name: CreateSelector :one
INSERT INTO profile_selectors (profile_id, entity, selector, comment)
VALUES ($1, $2, $3, $4)
RETURNING id, profile_id, entity, selector, comment;

-- name: GetSelectorsByProfileID :many
SELECT id, profile_id, entity, selector, comment
FROM profile_selectors
WHERE profile_id = $1;

-- name: UpdateSelector :one
UPDATE profile_selectors
SET entity = $2, selector = $3, comment = $4
WHERE id = $1
RETURNING id, profile_id, entity, selector, comment;

-- name: DeleteSelector :exec
DELETE FROM profile_selectors
WHERE id = $1;

-- name: GetSelectorByID :one
SELECT id, profile_id, entity, selector, comment
FROM profile_selectors
WHERE id = $1;

-- name: DeleteSelectorsByProfileID :exec
DELETE FROM profile_selectors
WHERE profile_id = $1;