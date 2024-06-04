-- Copyright 2023 Stacklok, Inc
--
-- Licensed under the Apache License, Version 2.0 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
--
--      http://www.apache.org/licenses/LICENSE-2.0
--
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.

-- Start to make sure the function and trigger are either both added or none
BEGIN;

-- Replace the update_profile_status() function to the one from migration 00007
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
        -- CHANGE: this is the only branch that changed from the original version in this migration
        -- mark status as failed if it was successful or pending or skipped and the new status is failure
        -- and there are no errors
    ELSIF (NEW.status = 'failure') AND NOT EXISTS (
        SELECT *
        FROM rule_evaluations res
                 INNER JOIN rule_details_eval rde ON res.id = rde.rule_eval_id
        WHERE res.profile_id = v_profile_id AND rde.status = 'error'
    ) THEN
        UPDATE profile_status SET profile_status = 'failure', last_updated = NOW()
        WHERE profile_id = v_profile_id AND (profile_status = 'success' OR profile_status = 'pending' OR profile_status = 'skipped') AND NEW.status = 'failure';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- transaction commit
COMMIT;
