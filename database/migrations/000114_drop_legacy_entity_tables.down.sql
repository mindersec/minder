-- SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- WARNING: This down migration recreates the table structure ONLY.
-- It does NOT restore any data that was deleted when the tables were dropped.
-- This is for structural rollback only - data recovery requires restoring from backup.

-- Recreate repositories table
CREATE TABLE repositories (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    provider TEXT NOT NULL,
    project_id UUID NOT NULL,
    repo_owner TEXT NOT NULL,
    repo_name TEXT NOT NULL,
    repo_id INTEGER NOT NULL,
    is_private BOOLEAN NOT NULL,
    is_fork BOOLEAN NOT NULL,
    webhook_id INTEGER,
    webhook_url TEXT NOT NULL,
    deploy_url TEXT NOT NULL,
    clone_url TEXT NOT NULL,
    default_branch TEXT,
    license VARCHAR(255) DEFAULT 'unknown',
    provider_id UUID,
    reminder_last_sent TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY (project_id, provider) REFERENCES providers(project_id, name) ON DELETE CASCADE
);

ALTER TABLE repositories ADD CONSTRAINT unique_repo_id UNIQUE (repo_id);
ALTER TABLE repositories ADD CONSTRAINT fk_repositories_provider_id FOREIGN KEY (provider_id) REFERENCES providers (id) ON DELETE CASCADE;

-- Recreate artifacts table
CREATE TABLE artifacts (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    artifact_name TEXT NOT NULL,
    artifact_type TEXT NOT NULL,
    artifact_visibility TEXT NOT NULL,
    project_id UUID,
    provider_id UUID,
    provider_name TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

ALTER TABLE artifacts ADD CONSTRAINT fk_artifacts_project_id FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE;
ALTER TABLE artifacts ADD CONSTRAINT fk_artifacts_provider_id_and_name FOREIGN KEY (provider_id, provider_name) REFERENCES providers (id, name) ON DELETE CASCADE;

-- Recreate pull_requests table
CREATE TABLE pull_requests (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    pr_number BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX pr_in_repo_unique ON pull_requests (repository_id, pr_number);

COMMIT;
