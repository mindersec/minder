-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- backfill rows which don't have a profile ID
UPDATE latest_evaluation_statuses
SET profile_id = ri.profile_id
FROM rule_instances AS ri
JOIN evaluation_rule_entities AS ere ON ere.rule_id = ri.id
JOIN latest_evaluation_statuses AS les ON les.rule_entity_id = ere.id
WHERE les.profile_id IS NULL;

-- make field mandatory
ALTER TABLE latest_evaluation_statuses ALTER COLUMN profile_id SET NOT NULL;

COMMIT;
