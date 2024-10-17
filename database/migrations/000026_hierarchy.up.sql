-- SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
-- SPDX-License-Identifier: Apache-2.0

--- Add unique constraint to the providers table between name and id
ALTER TABLE providers
  ADD CONSTRAINT providers_name_id_key
  UNIQUE (name, id);

-- Initialize the provider_id column in the repositories table.
-- Note that the foreign key constraint is not added here, as it will be added
-- after the provider_id column is populated.
ALTER TABLE repositories ADD COLUMN provider_id UUID;

-- set the provider_id for all existing repositories
UPDATE repositories
  SET provider_id = providers.id
  FROM providers
  WHERE repositories.provider = providers.name AND repositories.project_id = providers.project_id;

-- Make provider_id not nullable
ALTER TABLE repositories
  ALTER COLUMN provider_id SET NOT NULL;

-- Now let's add the foreign key constraint in the repositories table
-- for the provider_id and provider columns
ALTER TABLE repositories
  ADD CONSTRAINT repositories_provider_id_name_fkey
  FOREIGN KEY (provider_id, provider)
  REFERENCES providers(id, name)
  ON DELETE CASCADE;

-- change the project_id column from being a foreign key from the providers
-- table to being a foreign key from the projects table
ALTER TABLE repositories
  DROP CONSTRAINT repositories_project_id_provider_fkey;

-- Add constraints so project_id is a foreign key to the projects table
ALTER TABLE repositories
  ADD CONSTRAINT repositories_project_id_fkey
  FOREIGN KEY (project_id)
  REFERENCES projects(id)
  ON DELETE CASCADE;

-- Now let's do the same for the rule_type table

ALTER TABLE rule_type
  ADD COLUMN provider_id UUID;

-- set the provider_id for all existing rule_type
UPDATE rule_type
  SET provider_id = providers.id
  FROM providers
  WHERE rule_type.provider = providers.name AND rule_type.project_id = providers.project_id;

-- Make provider_id not nullable
ALTER TABLE rule_type
  ALTER COLUMN provider_id SET NOT NULL;

-- Now let's add the foreign key constraint in the rule_type table
-- for the provider_id and provider columns
ALTER TABLE rule_type
  ADD CONSTRAINT rule_type_provider_id_name_fkey
  FOREIGN KEY (provider_id, provider)
  REFERENCES providers(id, name)
  ON DELETE CASCADE;

-- change the project_id column from being a foreign key from the providers
-- table to being a foreign key from the projects table
ALTER TABLE rule_type
  DROP CONSTRAINT rule_type_project_id_provider_fkey;

-- Add constraints so project_id is a foreign key to the projects table
ALTER TABLE rule_type
  ADD CONSTRAINT rule_type_project_id_fkey
  FOREIGN KEY (project_id)
  REFERENCES projects(id)
  ON DELETE CASCADE;

-- Now let's cover the `profiles` table

ALTER TABLE profiles
  ADD COLUMN provider_id UUID;

-- set the provider_id for all existing profiles
UPDATE profiles
  SET provider_id = providers.id
  FROM providers
  WHERE profiles.provider = providers.name AND profiles.project_id = providers.project_id;

-- Make provider_id not nullable
ALTER TABLE profiles
  ALTER COLUMN provider_id SET NOT NULL;

-- Now let's add the foreign key constraint in the profiles table
-- for the provider_id and provider columns
ALTER TABLE profiles
  ADD CONSTRAINT profiles_provider_id_name_fkey
  FOREIGN KEY (provider_id, provider)
  REFERENCES providers(id, name)
  ON DELETE CASCADE;

-- change the project_id column from being a foreign key from the providers
-- table to being a foreign key from the projects table
ALTER TABLE profiles
  DROP CONSTRAINT profiles_project_id_provider_fkey;

-- Add constraints so project_id is a foreign key to the projects table
ALTER TABLE profiles
  ADD CONSTRAINT profiles_project_id_fkey
  FOREIGN KEY (project_id)
  REFERENCES projects(id)
  ON DELETE CASCADE;
