-- SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- Postgres can't remove a value for an enum type. So, we can't really
-- do a down migration. Instead, we'll just leave this here as a
-- reminder that we can't remove this value.

-- Drop the existing unique index on rule_evaluations
DROP INDEX IF EXISTS rule_evaluations_results_idx;

-- Recreate the unique index without COALESCE on pull_request_id
CREATE UNIQUE INDEX rule_evaluations_results_idx
    ON rule_evaluations(profile_id, repository_id, COALESCE(artifact_id, '00000000-0000-0000-0000-000000000000'::UUID), entity, rule_type_id)

-- Remove the pr reference from rule_evaluations
ALTER TABLE rule_evaluations DROP COLUMN IF EXISTS pull_request_id;

-- Drop the existing unique index on rule_evaluations
DROP INDEX IF EXISTS pr_in_repo_unique;

-- Drop the pull_requests table
DROP TABLE IF EXISTS pull_requests;
