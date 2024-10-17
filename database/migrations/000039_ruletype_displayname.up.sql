-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

ALTER TABLE rule_type ADD COLUMN display_name TEXT NOT NULL DEFAULT '';
