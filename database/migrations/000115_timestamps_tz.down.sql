-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

SET timezone = 'UTC';

ALTER TABLE projects
    ALTER COLUMN created_at TYPE TIMESTAMP,
    ALTER COLUMN created_at SET DEFAULT NOW(),
    ALTER COLUMN updated_at TYPE TIMESTAMP,
    ALTER COLUMN updated_at SET DEFAULT NOW();

ALTER TABLE users
    ALTER COLUMN created_at TYPE TIMESTAMP,
    ALTER COLUMN created_at SET DEFAULT NOW(),
    ALTER COLUMN updated_at TYPE TIMESTAMP,
    ALTER COLUMN updated_at SET DEFAULT NOW();

ALTER TABLE providers
    ALTER COLUMN created_at TYPE TIMESTAMP,
    ALTER COLUMN created_at SET DEFAULT NOW(),
    ALTER COLUMN updated_at TYPE TIMESTAMP,
    ALTER COLUMN updated_at SET DEFAULT NOW();

ALTER TABLE provider_access_tokens
    ALTER COLUMN expiration_time TYPE TIMESTAMP,
    ALTER COLUMN created_at TYPE TIMESTAMP,
    ALTER COLUMN created_at SET DEFAULT NOW(),
    ALTER COLUMN updated_at TYPE TIMESTAMP,
    ALTER COLUMN updated_at SET DEFAULT NOW();

ALTER TABLE session_store
    ALTER COLUMN created_at TYPE TIMESTAMP,
    ALTER COLUMN created_at SET DEFAULT NOW();

ALTER TABLE rule_type
    ALTER COLUMN created_at TYPE TIMESTAMP,
    ALTER COLUMN created_at SET DEFAULT NOW(),
    ALTER COLUMN updated_at TYPE TIMESTAMP,
    ALTER COLUMN updated_at SET DEFAULT NOW();

ALTER TABLE profiles
    ALTER COLUMN created_at TYPE TIMESTAMP,
    ALTER COLUMN created_at SET DEFAULT NOW(),
    ALTER COLUMN updated_at TYPE TIMESTAMP,
    ALTER COLUMN updated_at SET DEFAULT NOW();

-- profiles_with_entity_profiles depends on entity_profiles columns; drop and recreate after altering
DROP VIEW IF EXISTS profiles_with_entity_profiles;

ALTER TABLE entity_profiles
    ALTER COLUMN created_at TYPE TIMESTAMP,
    ALTER COLUMN created_at SET DEFAULT NOW(),
    ALTER COLUMN updated_at TYPE TIMESTAMP,
    ALTER COLUMN updated_at SET DEFAULT NOW();

CREATE VIEW profiles_with_entity_profiles WITH (security_invoker = true) AS (
    SELECT entity_profiles.*, profiles.id as profid FROM profiles LEFT JOIN entity_profiles ON profiles.id = entity_profiles.profile_id
);

ALTER TABLE profile_status
    ALTER COLUMN last_updated TYPE TIMESTAMP,
    ALTER COLUMN last_updated SET DEFAULT NOW();

ALTER TABLE features
    ALTER COLUMN created_at TYPE TIMESTAMP,
    ALTER COLUMN created_at SET DEFAULT NOW(),
    ALTER COLUMN updated_at TYPE TIMESTAMP,
    ALTER COLUMN updated_at SET DEFAULT NOW();

ALTER TABLE entitlements
    ALTER COLUMN created_at TYPE TIMESTAMP,
    ALTER COLUMN created_at SET DEFAULT NOW();

ALTER TABLE entity_execution_lock
    ALTER COLUMN last_lock_time TYPE TIMESTAMP;

ALTER TABLE flush_cache
    ALTER COLUMN queued_at TYPE TIMESTAMP,
    ALTER COLUMN queued_at SET DEFAULT NOW();

ALTER TABLE provider_github_app_installations
    ALTER COLUMN created_at TYPE TIMESTAMP,
    ALTER COLUMN created_at SET DEFAULT NOW(),
    ALTER COLUMN updated_at TYPE TIMESTAMP,
    ALTER COLUMN updated_at SET DEFAULT NOW();

ALTER TABLE rule_instances
    ALTER COLUMN created_at TYPE TIMESTAMP,
    ALTER COLUMN created_at SET DEFAULT NOW(),
    ALTER COLUMN updated_at TYPE TIMESTAMP,
    ALTER COLUMN updated_at SET DEFAULT NOW();

ALTER TABLE user_invites
    ALTER COLUMN created_at TYPE TIMESTAMP,
    ALTER COLUMN created_at SET DEFAULT NOW(),
    ALTER COLUMN updated_at TYPE TIMESTAMP,
    ALTER COLUMN updated_at SET DEFAULT NOW();

ALTER TABLE remediation_events
    ALTER COLUMN created_at TYPE TIMESTAMP,
    ALTER COLUMN created_at SET DEFAULT NOW();

ALTER TABLE alert_events
    ALTER COLUMN created_at TYPE TIMESTAMP,
    ALTER COLUMN created_at SET DEFAULT NOW();

COMMIT;
