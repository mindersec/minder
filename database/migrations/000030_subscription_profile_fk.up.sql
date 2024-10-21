-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- FK was created on wrong table (I mixed up profiles and projects again...)
ALTER TABLE projects DROP COLUMN subscription_id;
ALTER TABLE profiles
    ADD COLUMN subscription_id UUID DEFAULT NULL
    REFERENCES subscriptions(id);
