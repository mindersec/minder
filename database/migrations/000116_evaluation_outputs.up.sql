-- SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- evaluation_outputs stores structured output and debug text from rule
-- evaluations. It is keyed as a child of evaluation_statuses so that
-- output data can be retained/purged independently of the compact
-- status rows.
CREATE TABLE evaluation_outputs (
    id     UUID NOT NULL REFERENCES evaluation_statuses(id) ON DELETE CASCADE PRIMARY KEY,
    output JSONB,
    debug  TEXT
);

COMMIT;
