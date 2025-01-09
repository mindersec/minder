-- SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

ALTER TABLE data_sources
    ADD COLUMN subscription_id UUID DEFAULT NULL
        REFERENCES subscriptions(id);

COMMIT;