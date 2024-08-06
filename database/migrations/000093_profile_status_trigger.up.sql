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

-- drop the old triggers based on rule_details_eval and rule_evaluations, and replace with latest_evaluation_statuses
-- (See migrations #7 and #54)
DROP TRIGGER IF EXISTS update_profile_status_after_delete ON rule_evaluations;
DROP TRIGGER IF EXISTS update_profile_status ON rule_details_eval;

-- Trigger function for updates
CREATE OR REPLACE FUNCTION update_profile_status() RETURNS TRIGGER AS $$
DECLARE
    v_status eval_status_types;
    v_new_status eval_status_types;
    v_other_error boolean;
    v_other_failed boolean;
    v_other_success boolean;
    v_other_skipped boolean;
    v_pending boolean;
BEGIN
  -- Fetch the profile_id for the new evaluation status
  SELECT es.status INTO v_new_status
  FROM latest_evaluation_statuses AS les
  JOIN evaluation_statuses AS es ON es.id = les.evaluation_history_id
  WHERE les.profile_id = NEW.profile_id
  AND les.rule_entity_id = NEW.rule_entity_id;

  IF v_new_status IS NULL THEN
      RAISE EXCEPTION 'oh no';
  end if;

  -- The next five statements calculate whether there are, for this
  -- profile, any rules in evaluations in status 'error', 'failure',
  -- 'success', and 'skipped', respectively. This allows to write the
  -- subsequent CASE statement in a more compact and readable fashion.
  --
  -- The consequence is that this version of the stored procedure adds
  -- some load w.r.t. to previous one by unconditionally executing
  -- these statements, but this should not be a problem, as all five
  -- queries hit the same rows, so they'll likely hit the cache.

  -- These queries join on the latest_evaluation_statuses table to ensure that
  -- we exclude historical statuses.

  SELECT EXISTS (
       SELECT 1 FROM latest_evaluation_statuses les
       INNER JOIN evaluation_statuses es ON es.id = les.evaluation_history_id
       WHERE les.profile_id = NEW.profile_id
         AND es.status = 'error'
  ) INTO v_other_error;

  SELECT EXISTS (
      SELECT 1 FROM latest_evaluation_statuses les
      INNER JOIN evaluation_statuses es ON es.id = les.evaluation_history_id
      WHERE les.profile_id = NEW.profile_id
        AND es.status = 'failure'
  ) INTO v_other_failed;

  SELECT EXISTS (
      SELECT 1 FROM latest_evaluation_statuses les
      INNER JOIN evaluation_statuses es ON es.id = les.evaluation_history_id
      WHERE les.profile_id = NEW.profile_id
        AND es.status = 'success'
  ) INTO v_other_success;

  SELECT EXISTS (
      SELECT 1 FROM latest_evaluation_statuses les
      INNER JOIN evaluation_statuses es ON es.id = les.evaluation_history_id
      WHERE les.profile_id = NEW.profile_id
        AND es.status = 'skipped'
  ) INTO v_other_skipped;

  SELECT NOT EXISTS (
      SELECT 1 FROM latest_evaluation_statuses les
      INNER JOIN evaluation_statuses es ON es.id = les.evaluation_history_id
      WHERE les.profile_id = NEW.profile_id
  ) INTO v_pending;

  CASE
      -- A single rule in error state means policy is in error state
      WHEN v_new_status = 'error' THEN
          v_status := 'error';

      -- No rule in error state and at least one rule in failure state
      -- means policy is in error state
      WHEN v_new_status = 'failure' AND v_other_error THEN
          v_status := 'error';
      WHEN v_new_status = 'failure' THEN
          v_status := 'failure';

      -- No rule in error or failure state and at least one rule in
      -- success state means policy is in success state
      WHEN v_new_status = 'success' AND v_other_error THEN
          v_status := 'error';
      WHEN v_new_status = 'success' AND v_other_failed THEN
          v_status := 'failure';
      WHEN v_new_status = 'success' THEN
          v_status := 'success';

      -- No rule in error, failure, or success state and at least one
      -- rule in skipped state means policy is in skipped state
      WHEN v_new_status = 'skipped' AND v_other_error THEN
          v_status := 'error';
      WHEN v_new_status = 'skipped' AND v_other_failed THEN
          v_status := 'failure';
      WHEN v_new_status = 'skipped' AND v_other_success THEN
          v_status := 'success';
      WHEN v_new_status = 'skipped' THEN
          v_status := 'skipped';

    -- This should never happen, if yes, make it visible
    ELSE
      v_status := 'error';
      RAISE WARNING 'default case should not happen';
  END CASE;

  -- This turned out to be very useful during debugging
  --     RAISE LOG '% % % % % % % % => %',
  --       v_other_error,
  --       v_other_failed,
  --       v_other_success,
  --       v_other_skipped,
  --       v_pending,
  --       NEW.evaluation_history_id,
  --       NEW.profile_id,
  --       v_new_status,
  --       v_status;

  UPDATE profile_status
     SET profile_status = v_status, last_updated = NOW()
   WHERE profile_id = NEW.profile_id;

  RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Trigger function for deletions
CREATE OR REPLACE FUNCTION update_profile_status_on_delete() RETURNS TRIGGER AS $$
DECLARE
    v_status eval_status_types;
BEGIN
    SELECT CASE
       WHEN EXISTS (
           SELECT 1 FROM latest_evaluation_statuses AS les
           INNER JOIN evaluation_statuses AS es ON es.id = les.evaluation_history_id
           WHERE les.profile_id = OLD.profile_id AND es.status = 'error'
       ) THEN 'error'
       WHEN EXISTS (
           SELECT 1 FROM latest_evaluation_statuses AS les
           INNER JOIN evaluation_statuses AS es ON es.id = les.evaluation_history_id
           WHERE les.profile_id = OLD.profile_id AND es.status = 'failure'
       ) THEN 'failure'
       WHEN NOT EXISTS (
           SELECT 1 FROM latest_evaluation_statuses
           WHERE profile_id = OLD.profile_id
       ) THEN 'pending'
       WHEN NOT EXISTS (
           SELECT 1 FROM latest_evaluation_statuses AS les
           INNER JOIN evaluation_statuses AS es ON es.id = les.evaluation_history_id
           WHERE les.profile_id = OLD.profile_id AND es.status != 'skipped'
       ) THEN 'skipped'
       WHEN NOT EXISTS (
           SELECT 1 FROM latest_evaluation_statuses AS les
           INNER JOIN evaluation_statuses AS es ON es.id = les.evaluation_history_id
           WHERE les.profile_id = OLD.profile_id AND es.status NOT IN ('success', 'skipped')
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
    AFTER INSERT OR UPDATE ON latest_evaluation_statuses
    FOR EACH ROW
EXECUTE PROCEDURE update_profile_status();

CREATE TRIGGER update_profile_status_after_delete
    AFTER DELETE ON latest_evaluation_statuses
    FOR EACH ROW
EXECUTE FUNCTION update_profile_status_on_delete();

COMMIT;
