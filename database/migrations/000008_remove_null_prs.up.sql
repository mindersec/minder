-- SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- see bug #1608: we were not linking PRs properly with rule evaluations. Just drop those rows, there's nothing
-- we can do with them.
DELETE FROM rule_evaluations WHERE entity = 'pull_request' AND pull_request_id IS NULL;

-- transaction commit
COMMIT;
