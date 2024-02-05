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

-- Begin transaction
BEGIN;

-- Function to check whether jsonb array element contain a name key
CREATE OR REPLACE FUNCTION contains_name_key(input_jsonb jsonb) RETURNS boolean AS
$$
DECLARE
  element jsonb;
BEGIN
  FOR element IN SELECT * FROM jsonb_array_elements(input_jsonb)
    LOOP
      IF element ? 'name' THEN
        RETURN true;
      END IF;
    END LOOP;

  RETURN false;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Function to add a descriptive name to each element in a JSONB array
CREATE OR REPLACE FUNCTION add_descriptive_name_to_jsonb_array(input_jsonb jsonb) RETURNS jsonb AS
$$
DECLARE
  updated_array jsonb;
  element       jsonb;
  element_type  text;
BEGIN
  updated_array := '[]'::jsonb;

  FOR element IN SELECT * FROM jsonb_array_elements(input_jsonb)
    LOOP
      element_type := element ->> 'type';

      IF (SELECT COUNT(*) FROM jsonb_array_elements(input_jsonb) WHERE value ->> 'type' = element_type) > 1 THEN
        element := jsonb_set(element, '{name}', ('"' || element_type || '_' || gen_random_uuid() || '"')::jsonb);
      ELSE
        element := jsonb_set(element, '{name}', ('"' || element_type || '"')::jsonb);
      END IF;

      updated_array := updated_array || jsonb_build_array(element);
    END LOOP;

  RETURN updated_array;
END;
$$ LANGUAGE plpgsql;

-- Function to get the rule name for a given entity, profile and rule id.
CREATE OR REPLACE FUNCTION get_rule_name(entity entities, profile_id uuid, rule_type_id uuid) RETURNS text AS
$$
DECLARE
  rule_name      text;
  rule_type_name text;
  rules          jsonb;
BEGIN
  SELECT entity_profiles.contextual_rules
  INTO rules
  FROM entity_profiles
  WHERE entity_profiles.profile_id = get_rule_name.profile_id
    AND entity_profiles.entity = get_rule_name.entity;

  SELECT rule_type.name INTO rule_type_name FROM rule_type WHERE id = get_rule_name.rule_type_id;

  SELECT rule_element ->> 'name'
  INTO rule_name
  FROM jsonb_array_elements(rules) rule_element
  WHERE rule_element ->> 'type' = rule_type_name;

  RETURN rule_name;
END;
$$ LANGUAGE plpgsql STABLE;

CREATE TABLE migration_profile_backfill_log (
  profile_id UUID PRIMARY KEY,
  FOREIGN KEY (profile_id) REFERENCES profiles (id) ON DELETE CASCADE
);

-- Prevent entity_profiles to be updated outside the transaction
SELECT *
FROM entity_profiles FOR UPDATE;

-- Add updated profile_id to migration_profile_backfill_log, don't add if already exists
WITH updated_profile_ids AS (
  -- Update existing rules to have the name field, internally this field is mandatory
  UPDATE entity_profiles
    SET contextual_rules = add_descriptive_name_to_jsonb_array(entity_profiles.contextual_rules),
      updated_at = now()
    WHERE contains_name_key(entity_profiles.contextual_rules) = false
    RETURNING entity_profiles.profile_id)
INSERT
INTO migration_profile_backfill_log (profile_id)
SELECT DISTINCT profile_id
FROM updated_profile_ids
WHERE profile_id NOT IN (SELECT profile_id FROM migration_profile_backfill_log);

-- Prevent rule_evaluations to be updated outside the transaction
SELECT *
FROM rule_evaluations FOR UPDATE;

-- Update rule evaluations
UPDATE rule_evaluations
SET rule_name = get_rule_name(rule_evaluations.entity, rule_evaluations.profile_id, rule_evaluations.rule_type_id)
WHERE rule_name IS NULL;

-- Add non null constraint on rule_name
ALTER TABLE rule_evaluations
  ALTER COLUMN rule_name SET NOT NULL;

-- Drop the created functions
DROP FUNCTION IF EXISTS contains_name_key(input_jsonb jsonb);
DROP FUNCTION IF EXISTS add_descriptive_name_to_jsonb_array(input_jsonb jsonb);
DROP FUNCTION IF EXISTS get_rule_name(entity entities, profile_id uuid, rule_type_id uuid);

-- transaction commit
COMMIT;
