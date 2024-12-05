-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

ALTER TABLE latest_evaluation_statuses ALTER COLUMN profile_id DROP NOT NULL;
