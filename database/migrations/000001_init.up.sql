-- Copyright 2023 Stacklok, Inc
--
-- Licensed under the Apache License, Version 2.0 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
--
--      http://www.apache.org/licenses/LICENSE-2.0
--
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.

-- projects table
CREATE TABLE projects (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    name TEXT NOT NULL,
    is_organization BOOLEAN NOT NULL DEFAULT FALSE,
    metadata JSONB NOT NULL DEFAULT '{}',
    parent_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX ON projects(name) WHERE parent_id IS NULL; -- if parent_id is null, then name must be unique
CREATE UNIQUE INDEX ON projects(parent_id, name) WHERE parent_id IS NOT NULL; -- if parent_id is not null, then name must be unique for that parent

-- roles table
CREATE TABLE roles (
    id SERIAL PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,    
    is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    is_protected BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- users table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    email TEXT,
    identity_subject TEXT NOT NULL,
    first_name TEXT,
    last_name TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

ALTER TABLE users ADD CONSTRAINT unique_identity_subject UNIQUE (identity_subject);

-- user/projects
CREATE TABLE user_projects (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE
);

-- user/roles
CREATE TABLE user_roles (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE
);

CREATE TYPE provider_type as enum ('github', 'rest', 'git', 'oci');

-- providers table
CREATE TABLE providers (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,  -- NOTE: we could omit this and use project_id + name as primary key, for one less primary key. Downside is that we would always need project_id + name to log or look up, instead of a UUID.
    name TEXT NOT NULL,
    version TEXT NOT NULL DEFAULT 'v1',
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    implements provider_type ARRAY NOT NULL,
    definition JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, name) -- alternative primary key
);

-- provider_access_tokens table
CREATE TABLE provider_access_tokens (
    id SERIAL PRIMARY KEY,
    provider TEXT NOT NULL,
    project_id UUID NOT NULL,
    owner_filter TEXT,
    encrypted_token TEXT NOT NULL,
    expiration_time TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY (project_id, provider) REFERENCES providers(project_id, name) ON DELETE CASCADE,
    UNIQUE (project_id, provider)
);

-- signing_keys table
CREATE TABLE signing_keys (
    id SERIAL PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    private_key TEXT NOT NULL,
    public_key TEXT NOT NULL,
    passphrase TEXT NOT NULL,
    key_identifier TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- repositories table
CREATE TABLE repositories (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    provider TEXT NOT NULL,
    project_id UUID NOT NULL,
    repo_owner TEXT NOT NULL,
    repo_name TEXT NOT NULL,
    repo_id INTEGER NOT NULL,
    is_private BOOLEAN NOT NULL,
    is_fork BOOLEAN NOT NULL,
    webhook_id INTEGER,
    webhook_url TEXT NOT NULL,
    deploy_url TEXT NOT NULL,
    clone_url TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY (project_id, provider) REFERENCES providers(project_id, name) ON DELETE CASCADE

);

-- artifacts table
CREATE TABLE artifacts (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    artifact_name TEXT NOT NULL,    -- this is case insensitive
    artifact_type TEXT NOT NULL,
    artifact_visibility TEXT NOT NULL,      -- comes from github. Can be public, private, internal
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- artifact versions table
CREATE TABLE artifact_versions (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    artifact_id UUID NOT NULL REFERENCES artifacts(id) ON DELETE CASCADE,
    version BIGINT NOT NULL,
    tags TEXT,
    sha TEXT NOT NULL,
    signature_verification JSONB,       -- see /proto/mediator/v1/mediator.proto#L82
    github_workflow JSONB,              -- see /proto/mediator/v1/mediator.proto#L75
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE session_store (
    id SERIAL PRIMARY KEY,
    provider TEXT NOT NULL,
    project_id UUID NOT NULL,
    port INTEGER,
    owner_filter TEXT,
    session_state TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY (project_id, provider) REFERENCES providers(project_id, name) ON DELETE CASCADE
);

-- table for storing rule types
CREATE TABLE rule_type (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    name TEXT NOT NULL,
    provider TEXT NOT NULL,
    project_id UUID NOT NULL,
    description TEXT NOT NULL,
    guidance TEXT NOT NULL,
    definition JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY (project_id, provider) REFERENCES providers(project_id, name) ON DELETE CASCADE
);

CREATE TYPE remediate_type as enum ('on', 'off', 'dry_run');

CREATE TABLE profiles (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    name TEXT NOT NULL,
    provider TEXT NOT NULL,
    project_id UUID NOT NULL,
    remediate remediate_type,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    FOREIGN KEY (project_id, provider) REFERENCES providers(project_id, name) ON DELETE CASCADE
);

CREATE UNIQUE INDEX ON profiles(project_id, name);

CREATE TYPE entities as enum ('repository', 'build_environment', 'artifact', 'pull_request');

CREATE TABLE entity_profiles (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    entity entities NOT NULL,
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    contextual_rules JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

create type eval_status_types as enum ('success', 'failure', 'error', 'skipped', 'pending');

create type remediation_status_types as enum ('success', 'failure', 'error', 'skipped', 'not_available');

-- This table will be used to track the overall status of a profile evaluation
CREATE TABLE profile_status (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    profile_status eval_status_types NOT NULL,
    last_updated TIMESTAMP NOT NULL DEFAULT NOW()
);

-- This table will be used to track the status of each rule evaluation
-- for a given profile
CREATE TABLE rule_evaluation_status (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    entity entities NOT NULL,
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    rule_type_id UUID NOT NULL REFERENCES rule_type(id) ON DELETE CASCADE,
    eval_status eval_status_types NOT NULL,
    remediation_status remediation_status_types NOT NULL,
    -- polimorphic references. A status may be associated with a repository, build environment or artifact
    repository_id UUID REFERENCES repositories(id) ON DELETE CASCADE,
    artifact_id UUID REFERENCES artifacts(id) ON DELETE CASCADE,
    -- These will be added later
    -- build_environment_id UUID REFERENCES build_environments(id) ON DELETE CASCADE,
    eval_details TEXT NOT NULL,
    eval_last_updated TIMESTAMP NOT NULL DEFAULT NOW(),
    remediation_details TEXT NOT NULL,
    remediation_last_updated TIMESTAMP DEFAULT NOW()
);

-- Constraint to ensure we don't have a cycle in the project tree
ALTER TABLE projects ADD CONSTRAINT parent_child_not_equal CHECK (id != parent_id);

-- Unique constraint
ALTER TABLE repositories ADD CONSTRAINT unique_repo_id UNIQUE (repo_id);
ALTER TABLE signing_keys ADD CONSTRAINT unique_key_identifier UNIQUE (key_identifier);

-- Indexes
CREATE INDEX idx_users_organization_id ON users(organization_id);
CREATE INDEX idx_roles_project_id ON roles(project_id);
CREATE UNIQUE INDEX roles_organization_id_name_lower_idx ON roles (organization_id, LOWER(name));
CREATE INDEX idx_provider_access_tokens_project_id ON provider_access_tokens(project_id);
CREATE UNIQUE INDEX repositories_repo_id_idx ON repositories(repo_id);
CREATE UNIQUE INDEX rule_type_idx ON rule_type(provider, project_id, name);
CREATE UNIQUE INDEX rule_evaluation_status_results_idx ON rule_evaluation_status(profile_id, repository_id, COALESCE(artifact_id, '00000000-0000-0000-0000-000000000000'::UUID), entity, rule_type_id);
CREATE UNIQUE INDEX artifact_name_lower_idx ON artifacts (repository_id, LOWER(artifact_name));
CREATE UNIQUE INDEX artifact_versions_idx ON artifact_versions (artifact_id, sha);
CREATE UNIQUE INDEX provider_name_project_id_idx ON providers (name, project_id);

-- triggers

-- Ensure statuses are deleted if a repository is deleted
CREATE OR REPLACE FUNCTION delete_eval_statuses() RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM rule_evaluation_status WHERE repository_id = OLD.id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER delete_eval_statuses
    BEFORE DELETE ON repositories
    FOR EACH ROW
    EXECUTE PROCEDURE delete_eval_statuses();

-- Create a default status for a profile when it's created
CREATE OR REPLACE FUNCTION create_default_profile_status() RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO profile_status (profile_id, profile_status, last_updated) VALUES (NEW.id, 'pending', NOW());
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER create_default_profile_status
    AFTER INSERT ON profiles
    FOR EACH ROW
    EXECUTE PROCEDURE create_default_profile_status();

-- Update overall profile status if a rule evaluation status is updated
-- error takes precedence over failure, failure takes precedence over success
CREATE OR REPLACE FUNCTION update_profile_status() RETURNS TRIGGER AS $$
BEGIN
    -- keep error if profile had errored
    IF (NEW.eval_status = 'error') THEN
        UPDATE profile_status SET profile_status = 'error', last_updated = NOW() WHERE profile_id = NEW.profile_id;
    -- mark status as successful if all evaluations are successful or skipped
    ELSEIF NOT EXISTS (SELECT * FROM rule_evaluation_status WHERE profile_id = NEW.profile_id AND eval_status != 'success' AND eval_status != 'skipped') THEN
        UPDATE profile_status SET profile_status = 'success', last_updated = NOW() WHERE profile_id = NEW.profile_id;
    -- mark profile as successful if it was pending and the new status is success
    ELSEIF (NEW.eval_status = 'success') THEN
        UPDATE profile_status SET profile_status = 'success', last_updated = NOW() WHERE profile_id = NEW.profile_id AND profile_status = 'pending';
    -- mark status as failed if it was successful or pending and the new status is failure
    -- and there are no errors
    ELSEIF (NEW.eval_status = 'failure') AND NOT EXISTS (SELECT * FROM rule_evaluation_status WHERE profile_id = NEW.profile_id and eval_status = 'error') THEN
        UPDATE profile_status SET profile_status = 'failure', last_updated = NOW()
        WHERE profile_id = NEW.profile_id AND (profile_status = 'success' OR profile_status = 'pending') AND NEW.eval_status = 'failure';
    -- only mark profile run as skipped if every evaluation was skipped
    ELSEIF (NEW.eval_status = 'skipped') THEN
        UPDATE profile_status SET profile_status = 'skipped', last_updated = NOW()
        WHERE profile_id = NEW.profile_id AND NOT EXISTS (SELECT * FROM rule_evaluation_status WHERE profile_id = NEW.profile_id AND eval_status != 'skipped');
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_profile_status
    AFTER INSERT OR UPDATE ON rule_evaluation_status
    FOR EACH ROW
    EXECUTE PROCEDURE update_profile_status();

-- Create default root organization and get id so we can create the root project
INSERT INTO projects (name, is_organization) VALUES ('Mediator Root', TRUE);

