-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

DROP INDEX IF EXISTS idx_roles_project_id;

DROP INDEX IF EXISTS roles_organization_id_name_lower_idx;

DROP TABLE IF EXISTS user_roles;

DROP TABLE IF EXISTS roles;
