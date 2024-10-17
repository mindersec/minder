-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

CREATE VIEW profiles_with_entity_profiles AS(
    SELECT entity_profiles.*, profiles.id as profid FROM profiles LEFT JOIN entity_profiles ON profiles.id = entity_profiles.profile_id
);

COMMIT;
