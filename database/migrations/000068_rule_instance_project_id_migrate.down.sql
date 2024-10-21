-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

ALTER TABLE rule_instances ALTER COLUMN project_id DROP NOT NULL;
