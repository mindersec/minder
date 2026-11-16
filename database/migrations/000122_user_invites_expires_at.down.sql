-- SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

ALTER TABLE user_invites
    DROP COLUMN IF EXISTS expires_at;
