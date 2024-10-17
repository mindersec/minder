-- SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- pull_requests table
CREATE TABLE pull_requests (
                           id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
                           repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
                           pr_number BIGINT NOT NULL, -- BIGINT because GitHub PR numbers are 64-bit integers
                           created_at TIMESTAMP NOT NULL DEFAULT NOW(),
                           updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Add unique constraint on repository_id and pr_number. Needed for upserts.
CREATE UNIQUE INDEX pr_in_repo_unique ON pull_requests (repository_id, pr_number);

-- Add pull_request_id to rule_evaluations
ALTER TABLE rule_evaluations
    ADD COLUMN pull_request_id UUID REFERENCES pull_requests(id) ON DELETE CASCADE;

-- Drop the existing unique index on rule_evaluations
DROP INDEX IF EXISTS rule_evaluations_results_idx;

-- Recreate the unique index with COALESCE on pull_request_id
CREATE UNIQUE INDEX rule_evaluations_results_idx
    ON rule_evaluations(profile_id, repository_id, COALESCE(artifact_id, '00000000-0000-0000-0000-000000000000'::UUID), entity, rule_type_id, COALESCE(pull_request_id, '00000000-0000-0000-0000-000000000000'::UUID));

