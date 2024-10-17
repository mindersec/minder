-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- Begin transaction
BEGIN;

-- Function to remove the "name" field from a JSONB array. Function is immutable as it will return same results
-- for same input arguments forever.
CREATE OR REPLACE FUNCTION remove_name_from_jsonb_array(input_jsonb jsonb) RETURNS jsonb AS
$$
DECLARE
  updated_array jsonb;
  element       jsonb;
BEGIN
  updated_array := '[]'::jsonb;

  FOR element IN SELECT * FROM jsonb_array_elements(input_jsonb)
    LOOP
      element := element - 'name';
      updated_array := updated_array || jsonb_build_array(element);
    END LOOP;

  RETURN updated_array;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Prevent concurrent updates to entity_profiles
SELECT *
FROM entity_profiles FOR UPDATE;

-- Update the entity_profiles table to remove the "name" key from the "contextual_rules" JSONB array
UPDATE entity_profiles
SET contextual_rules = remove_name_from_jsonb_array(contextual_rules),
    updated_at       = now()
WHERE entity_profiles.profile_id IN (SELECT profile_id FROM migration_profile_backfill_log);

-- Prevent concurrent updates to rule_evaluations
SELECT *
FROM rule_evaluations FOR UPDATE;

-- Update the rule_name column to remove not null constraint
ALTER TABLE rule_evaluations
  ALTER COLUMN rule_name DROP NOT NULL;

-- Delete duplicate rule evaluation results without considering rule_name
-- Using CTID as postgres doesn't have min, max aggregators for uuid (too much code to add one)
DELETE
FROM rule_evaluations
WHERE CTID IN (SELECT MIN(CTID) AS CTID
               FROM rule_evaluations
               GROUP BY entity, profile_id, repository_id, rule_type_id,
                        COALESCE(pull_request_id, '00000000-0000-0000-0000-000000000000'::UUID),
                        COALESCE(artifact_id, '00000000-0000-0000-0000-000000000000'::UUID)
               HAVING COUNT(*) > 1)
  AND profile_id IN (SELECT profile_id FROM migration_profile_backfill_log);

-- Set rule_name column to null
UPDATE rule_evaluations
SET rule_name = null
WHERE profile_id IN (SELECT profile_id FROM migration_profile_backfill_log);

-- Drop the created function
DROP FUNCTION IF EXISTS remove_name_from_jsonb_array(input_jsonb jsonb);

-- Drop the migration log table
DROP TABLE IF EXISTS migration_profile_backfill_log;

-- transaction commit
COMMIT;
