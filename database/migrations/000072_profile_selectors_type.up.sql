-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

CREATE TYPE profile_selector AS (
    id UUID,
    profile_id UUID,
    entity entities,
    selector TEXT,
    comment TEXT
);

COMMIT;
