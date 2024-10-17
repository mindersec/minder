-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- Drop the existing foreign key constraint on provider
ALTER TABLE profiles DROP CONSTRAINT profiles_provider_id_name_fkey;

ALTER TABLE profiles
    ALTER COLUMN provider DROP NOT NULL;

ALTER TABLE profiles
    ALTER COLUMN provider_id DROP NOT NULL;

COMMIT;
