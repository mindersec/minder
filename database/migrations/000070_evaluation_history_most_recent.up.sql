-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- Store the most recent timestamp in a dedicated field to simplify sorting the rows.

ALTER TABLE evaluation_statuses ADD COLUMN most_recent_evaluation TIMESTAMP NOT NULL DEFAULT NOW();
