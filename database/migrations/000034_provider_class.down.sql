-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

ALTER TABLE providers DROP COLUMN class;

DROP TYPE provider_class;
