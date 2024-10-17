-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- add columns for new encrypted data format

-- This column is not used at this point in time, it is always NULL.
BEGIN;
-- can't cast between TEXT and JSONB, drop and recreate
ALTER TABLE session_store DROP COLUMN encrypted_redirect;
ALTER TABLE session_store ADD COLUMN encrypted_redirect TEXT;
COMMIT;
