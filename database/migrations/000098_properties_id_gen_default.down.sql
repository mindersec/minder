-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

ALTER TABLE properties
    ALTER COLUMN id DROP NOT NULL;

ALTER TABLE properties
    ALTER COLUMN id DROP DEFAULT;

COMMIT;
