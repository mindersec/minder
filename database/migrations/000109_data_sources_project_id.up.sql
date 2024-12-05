-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- In the previous migration we forgot to add `project_id` foreign key
-- to both `data_sources_functions` and `rule_type_data_sources`
-- tables.
--
-- While having that foreign key is not terribly important from the
-- data model perspective, since a function is indirectly connected to
-- a project id anyway, from the security perspective we want to
-- ensure that all database objects are tied to a single project and
-- all statements operating on them explicitly filter by project id,
-- since project is the entity by which we enforce permissions.

-- fix data_sources_functions

ALTER TABLE data_sources_functions
  ADD COLUMN project_id UUID;

DO $$
DECLARE
  ds_id UUID;
  pj_id UUID;
BEGIN
  FOR ds_id, pj_id IN SELECT id, project_id FROM data_sources
    LOOP
    UPDATE data_sources_functions
       SET project_id = pj_id
     WHERE data_source_id = ds_id;
  END LOOP;
END $$;

ALTER TABLE data_sources_functions
  ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE data_sources_functions
  ADD CONSTRAINT data_sources_functions_project_id_fkey
  FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE;

DROP INDEX data_sources_functions_name_lower_idx;
CREATE UNIQUE INDEX data_sources_functions_name_lower_idx
  ON data_sources_functions (data_source_id, project_id, lower(name));

-- fix rule_type_data_sources

ALTER TABLE rule_type_data_sources
  ADD COLUMN project_id UUID;

DO $$
DECLARE
  ds_id UUID;
  pj_id UUID;
BEGIN
  FOR ds_id, pj_id IN SELECT id, project_id FROM data_sources
    LOOP
    UPDATE rule_type_data_sources
       SET project_id = pj_id
     WHERE data_sources_id = ds_id;
  END LOOP;
END $$;

ALTER TABLE rule_type_data_sources
  ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE rule_type_data_sources
  ADD CONSTRAINT rule_type_data_sources_project_id_fkey
  FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE;

COMMIT;
