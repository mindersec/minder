-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

CREATE TABLE IF NOT EXISTS profile_selectors (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    entity entities, -- this is nullable since it can be applicable to all
    selector TEXT NOT NULL, -- CEL expression
    comment TEXT NOT NULL -- optional comment (can be empty string)
);

-- Ensure we have performant search based on profile_id
CREATE INDEX idx_profile_selectors_on_profile ON profile_selectors(profile_id);

COMMIT;
