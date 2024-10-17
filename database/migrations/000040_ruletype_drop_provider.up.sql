-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- Drop the existing foreign key constraint on provider
ALTER TABLE rule_type DROP CONSTRAINT rule_type_provider_id_name_fkey;

ALTER TABLE rule_type
    ALTER COLUMN provider DROP NOT NULL;

ALTER TABLE rule_type
    ALTER COLUMN provider_id DROP NOT NULL;
