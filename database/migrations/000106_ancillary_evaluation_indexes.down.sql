-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- The following two indexes on alert_events and remediation_events
-- are necessary to optimize deletions.
DROP INDEX alert_events_evaluation_id_fk_idx;
DROP INDEX remediation_events_evaluation_id_fk_idx;

COMMIT;
