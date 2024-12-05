-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

ALTER TABLE profiles ADD COLUMN labels TEXT[] DEFAULT '{}';

-- This index is a bit besides the point of profile labels, but while profiling the searches
-- we noticed that the profile_id index was missing and it was causing a full table scan
-- which was more costly then the subsequent label filtering on the subset of profiles.
CREATE INDEX idx_entity_profiles_profile_id ON entity_profiles(profile_id);
