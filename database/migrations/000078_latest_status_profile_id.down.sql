-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- introduce some denormalization to simplify a common access pattern
-- namely: retrieving the latest rule statuses for a specific profile
-- A future PR will backfill this column and make it NOT NULL
ALTER TABLE latest_evaluation_statuses DROP COLUMN profile_id;
