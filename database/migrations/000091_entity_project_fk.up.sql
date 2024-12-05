-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- Drop the foreign key constraint and then recreate it with the ON DELETE CASCADE option
BEGIN;

ALTER TABLE entity_instances DROP CONSTRAINT entity_instances_project_id_fkey;

ALTER TABLE entity_instances ADD CONSTRAINT entity_instances_project_id_fkey FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE;

-- Do the same for the provider ID, since deleting a provider should delete all entities associated with it
ALTER TABLE entity_instances DROP CONSTRAINT entity_instances_provider_id_fkey;

ALTER TABLE entity_instances ADD CONSTRAINT entity_instances_provider_id_fkey FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE;

COMMIT;
