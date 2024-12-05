-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

ALTER TABLE rule_type
    ADD CONSTRAINT rule_type_provider_id_name_fkey
        FOREIGN KEY (provider_id, provider)
            REFERENCES providers(id, name)
            ON DELETE CASCADE;

ALTER TABLE rule_type
    ALTER COLUMN provider SET NOT NULL;

ALTER TABLE rule_type
    ALTER COLUMN provider_id SET NOT NULL;
