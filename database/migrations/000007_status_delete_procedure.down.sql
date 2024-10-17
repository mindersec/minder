-- SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

DROP TRIGGER IF EXISTS update_profile_status_after_delete ON rule_evaluations;

DROP FUNCTION IF EXISTS update_profile_status_on_delete();

-- This is the same version of the function from the 000001 migration. Just for completeness of down migration
CREATE OR REPLACE FUNCTION update_profile_status() RETURNS TRIGGER AS $$
DECLARE
v_profile_id UUID;
BEGIN
    -- Fetch the profile_id for the current rule_eval_id
SELECT profile_id INTO v_profile_id
FROM rule_evaluations
WHERE id = NEW.rule_eval_id;

-- keep error if profile had errored
IF (NEW.status = 'error') THEN
UPDATE profile_status SET profile_status = 'error', last_updated = NOW()
WHERE profile_id = v_profile_id;
-- only mark profile run as skipped if every evaluation was skipped
ELSEIF (NEW.status = 'skipped') THEN
UPDATE profile_status SET profile_status = 'skipped', last_updated = NOW()
WHERE profile_id = v_profile_id AND NOT EXISTS (SELECT * FROM rule_evaluations res INNER JOIN rule_details_eval rde ON res.id = rde.rule_eval_id WHERE res.profile_id = v_profile_id AND rde.status != 'skipped');
-- mark status as successful if all evaluations are successful or skipped
ELSEIF NOT EXISTS (
        SELECT *
        FROM rule_evaluations res
        INNER JOIN rule_details_eval rde ON res.id = rde.rule_eval_id
        WHERE res.profile_id = v_profile_id AND rde.status != 'success' AND rde.status != 'skipped'
    ) THEN
UPDATE profile_status SET profile_status = 'success', last_updated = NOW()
WHERE profile_id = v_profile_id;
-- mark profile as successful if it was pending and the new status is success
ELSEIF (NEW.status = 'success') THEN
UPDATE profile_status SET profile_status = 'success', last_updated = NOW() WHERE profile_id = v_profile_id AND profile_status = 'pending';
-- mark status as failed if it was successful or pending and the new status is failure
-- and there are no errors
ELSIF (NEW.status = 'failure') AND NOT EXISTS (
        SELECT *
        FROM rule_evaluations res
        INNER JOIN rule_details_eval rde ON res.id = rde.rule_eval_id
        WHERE res.profile_id = v_profile_id AND rde.status = 'error'
    ) THEN
UPDATE profile_status SET profile_status = 'failure', last_updated = NOW()
WHERE profile_id = v_profile_id AND (profile_status = 'success' OR profile_status = 'pending') AND NEW.status = 'failure';
END IF;
RETURN NEW;
END;
$$ LANGUAGE plpgsql;

COMMIT;
