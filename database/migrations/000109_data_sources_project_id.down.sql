-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

DROP INDEX data_sources_functions_name_lower_idx;

ALTER TABLE data_sources_functions
  DROP COLUMN project_id;

CREATE UNIQUE INDEX data_sources_functions_name_lower_idx
  ON data_sources_functions (data_source_id, lower(name));

ALTER TABLE rule_type_data_sources
  DROP COLUMN project_id;

COMMIT;
