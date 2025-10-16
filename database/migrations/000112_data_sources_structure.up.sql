-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- This migration adds support for storing data source metadata at
-- the datasource level.  This is stored as JSONB using an internal
-- schema derived from (but separate than) protobuf.  (This could
-- also been used for the data source type or even rule storage, but
-- it's not worth migrating at this time.)
--
-- NULL is equivalent to "empty object" for migration purposes.

ALTER TABLE data_sources
    ADD COLUMN metadata JSONB DEFAULT NULL;

COMMIT;
