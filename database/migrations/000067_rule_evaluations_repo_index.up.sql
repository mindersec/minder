-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

CREATE INDEX CONCURRENTLY rule_evaluations_repository_id_idx ON rule_evaluations(repository_id);
