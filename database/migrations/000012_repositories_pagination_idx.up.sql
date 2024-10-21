-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- This adds a unique index to the repositories table to support cursor pagination.
CREATE UNIQUE INDEX repositories_cursor_pagination_idx ON repositories(project_id, provider, repo_id);
