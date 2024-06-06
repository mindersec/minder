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

-- This backfills the rule_instances table using information about the rule
-- instances stored in various tables in Minder.
-- The migration query takes the following approach:
--
-- 1) Flatten each list of rules in the entity_profiles table by using
--    JSONB_ARRAY_ELEMENTS() with a lateral cross join.
--    This allows us to join each JSON object with the row in which it is stored.
-- 2) Get the rule type ID by querying rule_type using name and project ID. The
--    rule name comes from each JSON object. To get the project ID, we need to
--    join entity_profiles to profiles on profile ID, and use the project_id
--    column from profiles.
--
-- We also reuse the created_at timestamp from entity_profiles in rule_instances
-- for the sake of consistency.

INSERT INTO rule_instances (
    profile_id,
    entity_type,
    rule_type_id,
    name,
    def,
    params,
    created_at
)
SELECT
    ep.profile_id,
    ep.entity AS entity_type,
    rt.id AS rule_type_id,
    cr->>'name' AS rule_name,
    COALESCE(cr->'def', '{}'::jsonb) AS def,
    COALESCE(cr->'params', '{}'::jsonb) AS params,
    ep.created_at
FROM entity_profiles AS ep
    CROSS JOIN LATERAL JSONB_ARRAY_ELEMENTS(ep.contextual_rules) AS cr
    JOIN profiles AS pf ON pf.id = ep.profile_id
    JOIN rule_type AS rt ON rt.project_id = pf.project_id AND cr->>'type' = rt.name
WHERE ep.migrated = FALSE;

-- As of the previous PR, all new rows are written to both rule_instances and
-- entities. By the time that we get to this part of the transaction, all rows
-- will have been migrated.
UPDATE entity_profiles SET migrated = TRUE;

COMMIT;
