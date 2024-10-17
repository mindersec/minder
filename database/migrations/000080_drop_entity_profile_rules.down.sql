-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

CREATE TABLE entity_profile_rules (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    entity_profile_id UUID NOT NULL REFERENCES entity_profiles(id) ON DELETE CASCADE,
    rule_type_id UUID NOT NULL REFERENCES rule_type(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (entity_profile_id, rule_type_id)
);
