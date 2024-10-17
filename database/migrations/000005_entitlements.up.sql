-- SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0


-- features represents the features that are available to a project.
-- settings are special tunables for the given feature.
CREATE TABLE IF NOT EXISTS features (
    name TEXT NOT NULL PRIMARY KEY,
    settings JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- name is unique
CREATE UNIQUE INDEX IF NOT EXISTS features_name_idx ON features(name);

-- entitlements represents the features that a project has access to.
CREATE TABLE IF NOT EXISTS entitlements (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    feature TEXT NOT NULL REFERENCES features(name) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- feature and project_id are unique
CREATE UNIQUE INDEX IF NOT EXISTS entitlements_feature_project_id_idx ON entitlements(feature, project_id);

-- initial features.

-- private_repositories_enabled is a feature that allows a project to create
-- private repositories.
INSERT INTO features(name, settings)
    VALUES ('private_repositories_enabled', '{}')
    ON CONFLICT DO NOTHING;
