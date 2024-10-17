-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

-- Drop the existing foreign key constraint on provider
ALTER TABLE session_store DROP CONSTRAINT session_store_project_id_provider_fkey;

-- Add a new foreign key constraint just for project_id
ALTER TABLE session_store
    ADD CONSTRAINT session_store_project_id_fkey
    FOREIGN KEY (project_id)
    REFERENCES projects(id)
    ON DELETE CASCADE;
