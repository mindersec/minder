-- SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

ALTER TABLE user_invites
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;

UPDATE user_invites
    SET expires_at = updated_at + INTERVAL '7 days'
    WHERE expires_at IS NULL;

ALTER TABLE user_invites
    ALTER COLUMN expires_at SET NOT NULL,
    ALTER COLUMN expires_at SET DEFAULT NOW() + INTERVAL '7 days';
