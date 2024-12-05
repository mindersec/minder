-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

ALTER TABLE profiles DROP COLUMN labels;

DROP INDEX IF EXISTS idx_entity_profiles_profile_id;
