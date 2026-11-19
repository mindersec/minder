-- SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

DROP TABLE exceptions;

CREATE TABLE exceptions (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    entity_id UUID NOT NULL REFERENCES entity_instances(id) ON DELETE CASCADE,
    profile_name TEXT NOT NULL,
    rule_name TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT exceptions_reason_length_check
        CHECK (char_length(reason) <= 500)
);

CREATE UNIQUE INDEX exceptions_entity_profile_rule_key
    ON exceptions (entity_id, lower(profile_name), lower(rule_name));

COMMIT;
