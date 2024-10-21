-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- add column, do not mark as NOT NULL since we need to populate it
ALTER TABLE rule_instances ADD COLUMN project_id UUID REFERENCES projects(id) ON DELETE CASCADE;
