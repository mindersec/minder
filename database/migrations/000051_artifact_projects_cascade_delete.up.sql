-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- Artifact changes

BEGIN;

-- delete artifacts that are associated with a project that gets deleted
-- otherwise removing a project would cause:
-- "update or delete on table "projects" violates foreign key constraint "fk_artifacts_project_id" on table "artifacts""
ALTER TABLE artifacts DROP CONSTRAINT fk_artifacts_project_id;
ALTER TABLE artifacts ADD CONSTRAINT fk_artifacts_project_id FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE;

-- delete artifacts that are associated with a provider that gets deleted
-- same reason as above. This is mostly a noop now because we also have a on delete cascade on the repository_id column
-- which while technically nullable is always set to a value, but would bite us in the future
ALTER TABLE artifacts DROP CONSTRAINT fk_artifacts_provider_id_and_name;
ALTER TABLE artifacts ADD CONSTRAINT fk_artifacts_provider_id_and_name FOREIGN KEY (provider_id, provider_name) REFERENCES providers (id, name) ON DELETE CASCADE;

COMMIT;
