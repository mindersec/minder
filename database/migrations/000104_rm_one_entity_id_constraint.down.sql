-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- recreate `one_entity_id` constraint
ALTER TABLE evaluation_rule_entities ADD CONSTRAINT one_entity_id CHECK (num_nonnulls(repository_id, artifact_id, pull_request_id) = 1);
