-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

ALTER TABLE providers DROP COLUMN auth_flows;
DROP TYPE authorization_flow;
