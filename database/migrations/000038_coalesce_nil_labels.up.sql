-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- Update profiles with NULL labels to empty array
UPDATE profiles
SET labels = '{}'
WHERE labels IS NULL;

COMMIT;
