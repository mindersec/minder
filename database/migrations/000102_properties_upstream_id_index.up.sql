-- SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- Create index on properties for upstream_id
CREATE INDEX idx_properties_value_gin ON properties USING GIN (value jsonb_path_ops);

COMMIT;
