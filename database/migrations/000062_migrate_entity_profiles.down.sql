-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- in case we need to rollback, wipe out the rule_instances table and mark all rows
-- in entity_profiles as unmigrated. A re-run of the migration will recreate all
-- rows in the rule_instances of table since we are dual writing at this point.
DELETE FROM rule_instances;
UPDATE entity_profiles SET migrated = FALSE;

COMMIT;
