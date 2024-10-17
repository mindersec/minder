-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

CREATE TABLE IF NOT EXISTS rule_instances(
    id           UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    profile_id   UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    rule_type_id UUID NOT NULL REFERENCES rule_type(id),
    name         TEXT NOT NULL,
    entity_type  entities NOT NULL,
    def          JSONB, -- stores the values needed by the rule type's `def`
    params       JSONB, -- stores the values needed by the rule type's `params`
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
