-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

ALTER TABLE profiles DROP COLUMN subscription_id;
ALTER TABLE projects
    ADD COLUMN subscription_id UUID DEFAULT NULL
    REFERENCES subscriptions(id);
