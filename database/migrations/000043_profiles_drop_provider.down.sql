-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

ALTER TABLE profiles
    ADD CONSTRAINT profiles_provider_id_name_fkey
        FOREIGN KEY (provider_id, provider)
            REFERENCES providers(id, name)
            ON DELETE CASCADE;

ALTER TABLE profiles
    ALTER COLUMN provider SET NOT NULL;

ALTER TABLE profiles
    ALTER COLUMN provider_id SET NOT NULL;

COMMIT;
