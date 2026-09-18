-- SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

CREATE TABLE exceptions (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    entity_name TEXT NOT NULL,
    entity_id UUID NOT NULL REFERENCES entity_instances(id) ON DELETE CASCADE,
    rule_type_id UUID NOT NULL REFERENCES rule_type(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (entity_id, rule_type_id)
);
