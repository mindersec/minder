-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

ALTER TABLE rule_details_remediate ADD COLUMN metadata JSONB NOT NULL DEFAULT '{}';

ALTER TYPE remediation_status_types ADD VALUE 'pending';

COMMIT;
