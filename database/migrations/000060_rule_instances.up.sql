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

CREATE TABLE IF NOT EXISTS rule_instances(
    id           UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    profile_id   UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    rule_type_id UUID NOT NULL REFERENCES rule_type(id),
    name         TEXT NOT NULL,
    entity_type  entities NOT NULL,
    def          JSONB NOT NULL,
    params       JSONB NOT NULL,
    created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    -- equivalent to constraints enforced by rule validation
    UNIQUE (profile_id, entity_type, name)
);

-- this reflects likely access patterns for this table - retrieving all rule
-- instances for a given profile, and all rules by profile and entity type
CREATE INDEX idx_rule_instances_profile ON rule_instances(profile_id);
CREATE INDEX idx_rule_instances_profile_entity ON rule_instances(profile_id, entity_type);

-- this will be used for migration purposes
ALTER TABLE entity_profiles ADD COLUMN migrated BOOL DEFAULT FALSE NOT NULL;

COMMIT;
