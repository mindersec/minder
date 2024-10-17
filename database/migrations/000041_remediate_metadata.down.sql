-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

ALTER TABLE rule_details_remediate DROP COLUMN metadata;

-- It is not possible to drop added values from enums, ref. `pending` for remediation_status_types
