-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- the properties ID is a surrogate, we don't want the user to have to provide it
ALTER TABLE properties
ALTER COLUMN id SET DEFAULT gen_random_uuid();

-- all properties must have an ID
ALTER TABLE properties
ALTER COLUMN id SET NOT NULL;

COMMIT;
