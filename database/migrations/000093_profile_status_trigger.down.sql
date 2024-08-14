-- Copyright 2024 Stacklok, Inc
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

BEGIN;

-- Drop the triggers for evaluation_statuses
DROP TRIGGER IF EXISTS update_profile_status_after_delete ON latest_evaluation_statuses;
DROP TRIGGER IF EXISTS update_profile_status_after_delete ON latest_evaluation_statuses;

-- Reinstate the update function as defined in migration script #54

CREATE OR REPLACE FUNCTION update_profile_status() RETURNS TRIGGER AS $$
DECLARE
    v_status eval_status_types;
    v_profile_id UUID;
    v_other_error boolean;
    v_other_failed boolean;
    v_other_success boolean;
    v_other_skipped boolean;
    v_pending boolean;
BEGIN
  -- Fetch the profile_id for the current rule_eval_id
  SELECT profile_id INTO v_profile_id
  FROM rule_evaluations
  WHERE id = NEW.rule_eval_id;

  -- The next five statements calculate whether there are, for this
  -- profile, any rules in evaluations in status 'error', 'failure',
  -- 'success', and 'skipped', respectively. This allows to write the
  -- subsequent CASE statement in a more compact and readable fashion.
  --
  -- The consequence is that this version of the stored procedure adds
  -- some load w.r.t. to previous one by unconditionally executing
  -- these statements, but this should not be a problem, as all five
  -- queries hit the same rows, so they'll likely hit the cache.

  SELECT EXISTS (
       SELECT 1 FROM rule_details_eval rde
        INNER JOIN rule_evaluations res ON res.id = rde.rule_eval_id
        WHERE res.profile_id = v_profile_id
          AND rde.status = 'error'
  ) INTO v_other_error;

  SELECT EXISTS (
       SELECT 1 FROM rule_details_eval rde
        INNER JOIN rule_evaluations res ON res.id = rde.rule_eval_id
        WHERE res.profile_id = v_profile_id
          AND rde.status = 'failure'
  ) INTO v_other_failed;

  SELECT EXISTS (
       SELECT 1 FROM rule_details_eval rde
        INNER JOIN rule_evaluations res ON res.id = rde.rule_eval_id
        WHERE res.profile_id = v_profile_id
          AND rde.status = 'success'
  ) INTO v_other_success;

  SELECT EXISTS (
       SELECT 1 FROM rule_details_eval rde
        INNER JOIN rule_evaluations res ON res.id = rde.rule_eval_id
        WHERE res.profile_id = v_profile_id
          AND rde.status = 'skipped'
  ) INTO v_other_skipped;

  SELECT NOT EXISTS (
       SELECT 1 FROM rule_details_eval rde
        INNER JOIN rule_evaluations res ON res.id = rde.rule_eval_id
        WHERE res.profile_id = v_profile_id
  ) INTO v_pending;

  CASE
    -- A single rule in error state means policy is in error state
    WHEN NEW.status = 'error' THEN
      v_status := 'error';

    -- No rule in error state and at least one rule in failure state
    -- means policy is in error state
    WHEN NEW.STATUS = 'failure' AND v_other_error THEN
      v_status := 'error';
    WHEN NEW.STATUS = 'failure' THEN
      v_status := 'failure';

    -- No rule in error or failure state and at least one rule in
    -- success state means policy is in success state
    WHEN NEW.STATUS = 'success' AND v_other_error THEN
      v_status := 'error';
    WHEN NEW.STATUS = 'success' AND v_other_failed THEN
      v_status := 'failure';
    WHEN NEW.STATUS = 'success' THEN
      v_status := 'success';

    -- No rule in error, failure, or success state and at least one
    -- rule in skipped state means policy is in skipped state
    WHEN NEW.STATUS = 'skipped' AND v_other_error THEN
      v_status := 'error';
    WHEN NEW.STATUS = 'skipped' AND v_other_failed THEN
      v_status := 'failure';
    WHEN NEW.STATUS = 'skipped' AND v_other_success THEN
      v_status := 'success';
    WHEN NEW.STATUS = 'skipped' THEN
      v_status := 'skipped';

    -- No rule evaluations means the policy is pending evaluation
    WHEN v_pending THEN
      v_status := 'pending';

    -- This should never happen, if yes, make it visible
    ELSE
      v_status := 'error';
      RAISE WARNING 'default case should not happen';
  END CASE;

  -- This turned out to be very useful during debugging
  --   RAISE LOG '% % % % % % % => %',
  --     v_other_error,
  --     v_other_failed,
  --     v_other_success,
  --     v_other_skipped,
  --     v_pending,
  --     OLD.status,
  --     NEW.status,
  --     v_status;

  UPDATE profile_status
     SET profile_status = v_status, last_updated = NOW()
   WHERE profile_id = v_profile_id;

  RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Reinstate the delete function as defined in migration script #7
CREATE OR REPLACE FUNCTION update_profile_status_on_delete() RETURNS TRIGGER AS $$
DECLARE
    v_status eval_status_types;
BEGIN
    SELECT CASE
       WHEN EXISTS (
           SELECT 1 FROM rule_details_eval rde
                             INNER JOIN rule_evaluations res ON res.id = rde.rule_eval_id
           WHERE profile_id = OLD.profile_id AND status = 'error'
       ) THEN 'error'
       WHEN EXISTS (
           SELECT 1 FROM rule_details_eval rde
                             INNER JOIN rule_evaluations res ON res.id = rde.rule_eval_id
           WHERE profile_id = OLD.profile_id AND status = 'failure'
       ) THEN 'failure'
       WHEN NOT EXISTS (
           SELECT 1 FROM rule_details_eval rde
                             INNER JOIN rule_evaluations res ON res.id = rde.rule_eval_id
           WHERE profile_id = OLD.profile_id
       ) THEN 'pending'
       WHEN NOT EXISTS (
           SELECT 1 FROM rule_details_eval rde
                             INNER JOIN rule_evaluations res ON res.id = rde.rule_eval_id
           WHERE profile_id = OLD.profile_id AND status != 'skipped'
       ) THEN 'skipped'
       WHEN NOT EXISTS (
           SELECT 1 FROM rule_details_eval rde
                             INNER JOIN rule_evaluations res ON res.id = rde.rule_eval_id
           WHERE profile_id = OLD.profile_id AND status NOT IN ('success', 'skipped')
       ) THEN 'success'
       ELSE (
           'error' -- This should never happen, if yes, make it visible
           )
       END INTO v_status;

    UPDATE profile_status SET profile_status = v_status, last_updated = NOW()
    WHERE profile_id = OLD.profile_id;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- recreate triggers for evaluation_statuses
CREATE TRIGGER update_profile_status
    AFTER INSERT OR UPDATE ON rule_details_eval
    FOR EACH ROW
EXECUTE PROCEDURE update_profile_status();

CREATE TRIGGER update_profile_status_after_delete
    AFTER DELETE ON rule_evaluations
    FOR EACH ROW
EXECUTE FUNCTION update_profile_status_on_delete();

COMMIT;
