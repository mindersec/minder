-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- The following two indexes on alert_events and remediation_events
-- are necessary to optimize deletions.
CREATE INDEX alert_events_evaluation_id_fk_idx ON alert_events (evaluation_id);
CREATE INDEX remediation_events_evaluation_id_fk_idx ON remediation_events (evaluation_id);

COMMIT;
