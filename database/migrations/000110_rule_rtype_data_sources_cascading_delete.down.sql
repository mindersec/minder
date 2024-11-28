-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

ALTER TABLE rule_type_data_sources
  DROP CONSTRAINT rule_type_data_sources_rule_type_id_fkey;

ALTER TABLE rule_type_data_sources
  ADD CONSTRAINT rule_type_data_sources_rule_type_id_fkey
  FOREIGN KEY (rule_type_id) REFERENCES rule_type(id);

COMMIT;
