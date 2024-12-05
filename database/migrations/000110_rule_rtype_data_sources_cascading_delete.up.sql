-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

BEGIN;

-- Generally speaking, given two projects R (root) and A within the
-- same hierarchy, a rule type defined in project A can reference a
-- data source defined in project R, and in such cases we want to
-- prevent admins of project R from deleting the data source without
-- fixing said rule type in project A (or having someone else fix it
-- for them), but we still want admins from project A to be able to
-- delete their rule types without hindrance.
--
-- This migration recreates the foreign key constraint to delete rows
-- from `rule_type_data_sources` when a record is deleted from
-- `rule_type`.
--
-- Note that it is safe to just drop and recreate the constraint as
-- the previous version prevented the deletion of rule types if they
-- referenced a data source, so it was not possible to have dangling
-- records.

ALTER TABLE rule_type_data_sources
  DROP CONSTRAINT rule_type_data_sources_rule_type_id_fkey;

ALTER TABLE rule_type_data_sources
  ADD CONSTRAINT rule_type_data_sources_rule_type_id_fkey
  FOREIGN KEY (rule_type_id) REFERENCES rule_type(id) ON DELETE CASCADE;

COMMIT;
