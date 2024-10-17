-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- populate by joining on profiles table
UPDATE rule_instances AS ri
SET project_id = pf.project_id
FROM profiles AS pf
WHERE ri.profile_id = pf.id;

-- now we can add the not null constraint
ALTER TABLE rule_instances ALTER COLUMN project_id SET NOT NULL;

COMMIT;
