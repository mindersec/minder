-- SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- name: InsertEvaluationOutput :exec
INSERT INTO evaluation_outputs(
    evaluation_id,
    output,
    debug
) VALUES (
    $1,
    sqlc.narg(output)::jsonb,
    sqlc.narg(debug)
)
ON CONFLICT (evaluation_id) DO UPDATE
SET output = COALESCE(sqlc.narg(output)::jsonb, evaluation_outputs.output),
    debug  = COALESCE(sqlc.narg(debug), evaluation_outputs.debug);

-- name: GetEvaluationOutput :one
SELECT * FROM evaluation_outputs
WHERE evaluation_id = $1;

-- name: DeleteEvaluationOutputsByEvaluationIDs :execrows
DELETE FROM evaluation_outputs
WHERE evaluation_id = ANY(sqlc.slice(evaluationIds)::uuid[]);
