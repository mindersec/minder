-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- artifact versions table
CREATE TABLE artifact_versions (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    artifact_id UUID NOT NULL REFERENCES artifacts(id) ON DELETE CASCADE,
    version BIGINT NOT NULL,
    tags TEXT,
    sha TEXT NOT NULL,
    signature_verification JSONB,       -- see /proto/minder/v1/minder.proto#L82
    github_workflow JSONB,              -- see /proto/minder/v1/minder.proto#L75
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX artifact_versions_idx ON artifact_versions (artifact_id, sha);
