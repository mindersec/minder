-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- backfill rows which don't have an entity type
UPDATE evaluation_rule_entities
SET entity_type = (CASE
    WHEN artifact_id IS NOT NULL
        THEN 'artifact'::entities
    WHEN pull_request_id IS NOT NULL
        THEN 'pull_request'::entities
    WHEN repository_id IS NOT NULL
        THEN 'repository'::entities
END)
WHERE entity_type IS NULL;

-- make field mandatory
ALTER TABLE evaluation_rule_entities ALTER COLUMN entity_type SET NOT NULL;

COMMIT;
