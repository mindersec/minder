-- SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- name: UpsertEvaluationOutput :exec
INSERT INTO evaluation_outputs(
    id,
    output,
    debug
) VALUES (
    $1,
    sqlc.arg(output)::jsonb,
    sqlc.narg(debug)
)
ON CONFLICT (id) DO UPDATE
SET output = COALESCE(sqlc.arg(output)::jsonb, evaluation_outputs.output),
    debug  = COALESCE(sqlc.narg(debug), evaluation_outputs.debug);

-- name: GetEvaluationOutput :one
SELECT * FROM evaluation_outputs
WHERE id = $1;

-- name: DeleteEvaluationOutputsByEvaluationIDs :execrows
DELETE FROM evaluation_outputs
WHERE id = ANY(sqlc.slice(evaluationIds)::uuid[]);

-- name: GetEvaluationOutputsByIDs :many
SELECT * FROM evaluation_outputs
WHERE id = ANY(sqlc.slice(evaluationIds)::uuid[]);
