-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

CREATE TABLE bundles(
    -- I have given this a separate PK to simplify some of the queries
    -- the combination of the remaining columns are unique
    id          UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    namespace   TEXT NOT NULL,
    name        TEXT NOT NULL,
    UNIQUE      (namespace, name)
);

CREATE TABLE subscriptions(
    -- same comment as for PK of `bundles`
    id                UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    project_id        UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    -- this FK is not ON DELETE CASCADE
    -- this prevents deletion of bundles which are still in use
    bundle_id         UUID NOT NULL REFERENCES bundles(id),
    current_version    TEXT NOT NULL,
    UNIQUE (bundle_id, project_id)
);

-- none of these FKs are ON DELETE CASCADE
-- prevents deletion of an in-use subscription
ALTER TABLE projects
    ADD COLUMN subscription_id UUID DEFAULT NULL
    REFERENCES subscriptions(id);

ALTER TABLE rule_type
    ADD COLUMN subscription_id UUID DEFAULT NULL
    REFERENCES subscriptions(id);
