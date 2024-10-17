-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

CREATE TYPE provider_class AS enum ('github', 'github-app');

ALTER TABLE providers
  -- Fills existing rows with 'github'
  ADD COLUMN class provider_class DEFAULT 'github' NOT NULL;
