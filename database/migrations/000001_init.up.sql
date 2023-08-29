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
-- TODO(jaosorior): We should look back at our primary key strategy. I
-- chose to use UUIDs but we want to look at alternatives.
CREATE TABLE projects (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    name TEXT NOT NULL,
    metadata JSONB NOT NULL,
    parent_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- organizations table
CREATE TABLE organizations (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    company TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- groups table
CREATE TABLE groups (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    is_protected BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);


-- roles table
CREATE TABLE roles (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    group_id INTEGER REFERENCES groups(id) ON DELETE CASCADE,
    name TEXT NOT NULL,    
    is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    is_protected BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- users table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email TEXT,
    username TEXT NOT NULL,
    password TEXT NOT NULL,
    needs_password_change BOOLEAN NOT NULL DEFAULT TRUE,
    first_name TEXT,
    last_name TEXT,
    is_protected BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    min_token_issued_time TIMESTAMP
);

-- user/groups
CREATE TABLE user_groups (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id INTEGER NOT NULL REFERENCES groups(id) ON DELETE CASCADE
);

-- user/roles
CREATE TABLE user_roles (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE
);

-- provider_access_tokens table
CREATE TABLE provider_access_tokens (
    id SERIAL PRIMARY KEY,
    provider TEXT NOT NULL,
    group_id INTEGER NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    owner_filter TEXT,
    encrypted_token TEXT NOT NULL,
    expiration_time TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- signing_keys table
CREATE TABLE signing_keys (
    id SERIAL PRIMARY KEY,
    group_id INTEGER NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    private_key TEXT NOT NULL,
    public_key TEXT NOT NULL,
    passphrase TEXT NOT NULL,
    key_identifier TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- repositories table
CREATE TABLE repositories (
    id SERIAL PRIMARY KEY,
    provider TEXT NOT NULL,
    group_id INTEGER NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    repo_owner TEXT NOT NULL,
    repo_name TEXT NOT NULL,
    repo_id INTEGER NOT NULL,
    is_private BOOLEAN NOT NULL,
    is_fork BOOLEAN NOT NULL,
    webhook_id INTEGER,
    webhook_url TEXT NOT NULL,
    deploy_url TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- artifacts table
CREATE TABLE artifacts (
    id SERIAL PRIMARY KEY,
    repository_id INTEGER NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    artifact_name TEXT NOT NULL,    -- this is case insensitive
    artifact_type TEXT NOT NULL,
    artifact_visibility TEXT NOT NULL,      -- comes from github. Can be public, private, internal
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- artifact versions table
CREATE TABLE artifact_versions (
    id SERIAL PRIMARY KEY,
    artifact_id INTEGER NOT NULL REFERENCES artifacts(id) ON DELETE CASCADE,
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
    grp_id INTEGER,
    port INTEGER,
    owner_filter TEXT,
    session_state TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE rule_type (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    provider TEXT NOT NULL,
    group_id INTEGER NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    definition JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE policies (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    provider TEXT NOT NULL,
    group_id INTEGER NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TYPE entities as enum ('repository', 'build_environment', 'artifact');

CREATE TABLE entity_policies (
    id SERIAL PRIMARY KEY,
    entity entities NOT NULL,
    policy_id INTEGER NOT NULL REFERENCES policies(id) ON DELETE CASCADE,
    contextual_rules JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

create type eval_status_types as enum ('success', 'failure', 'error', 'skipped', 'pending');

-- This table will be used to track the overall status of a policy evaluation
CREATE TABLE policy_status (
    id SERIAL PRIMARY KEY,
    policy_id INTEGER NOT NULL REFERENCES policies(id) ON DELETE CASCADE,
    policy_status eval_status_types NOT NULL,
    last_updated TIMESTAMP NOT NULL DEFAULT NOW()
);

-- This table will be used to track the status of each rule evaluation
-- for a given policy
CREATE TABLE rule_evaluation_status (
    id SERIAL PRIMARY KEY,
    entity entities NOT NULL,
    policy_id INTEGER NOT NULL REFERENCES policies(id) ON DELETE CASCADE,
    rule_type_id INTEGER NOT NULL REFERENCES rule_type(id) ON DELETE CASCADE,
    eval_status eval_status_types NOT NULL,
    -- polimorphic references. A status may be associated with a repository, build environment or artifact
    repository_id INTEGER REFERENCES repositories(id) ON DELETE CASCADE,
    artifact_id INTEGER REFERENCES artifacts(id) ON DELETE CASCADE,
    -- These will be added later
    -- build_environment_id INTEGER REFERENCES build_environments(id) ON DELETE CASCADE,
    details TEXT NOT NULL,
    last_updated TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Constraint to ensure we don't have a cycle in the project tree
ALTER TABLE projects ADD CONSTRAINT parent_child_not_equal CHECK (id != parent_id);

-- Unique constraint
ALTER TABLE provider_access_tokens ADD CONSTRAINT unique_group_id UNIQUE (group_id);
ALTER TABLE repositories ADD CONSTRAINT unique_repo_id UNIQUE (repo_id);
ALTER TABLE signing_keys ADD CONSTRAINT unique_key_identifier UNIQUE (key_identifier);

-- Indexes
CREATE UNIQUE INDEX organizations_name_lower_idx ON organizations (LOWER(name));
CREATE UNIQUE INDEX organizations_company_lower_idx ON organizations (LOWER(company));
CREATE INDEX idx_users_organization_id ON users(organization_id);
CREATE INDEX idx_groups_organization_id ON groups(organization_id);
CREATE INDEX idx_roles_group_id ON roles(group_id);
CREATE UNIQUE INDEX roles_organization_id_name_lower_idx ON roles (organization_id, LOWER(name));
CREATE INDEX idx_provider_access_tokens_group_id ON provider_access_tokens(group_id);
CREATE UNIQUE INDEX users_organization_id_email_lower_idx ON users (organization_id, LOWER(email));
CREATE UNIQUE INDEX users_organization_id_username_lower_idx ON users (organization_id, LOWER(username));
CREATE UNIQUE INDEX repositories_repo_id_idx ON repositories(repo_id);
CREATE UNIQUE INDEX policies_group_id_policy_name_idx ON policies(provider, group_id, name);
CREATE UNIQUE INDEX rule_type_idx ON rule_type(provider, group_id, name);
CREATE UNIQUE INDEX rule_evaluation_status_results_idx ON rule_evaluation_status(policy_id, repository_id, COALESCE(artifact_id, 0), entity, rule_type_id);
CREATE UNIQUE INDEX artifact_name_lower_idx ON artifacts (repository_id, LOWER(artifact_name));
CREATE UNIQUE INDEX artifact_versions_idx ON artifact_versions (artifact_id, sha);

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

-- Create a default status for a policy when it's created
CREATE OR REPLACE FUNCTION create_default_policy_status() RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO policy_status (policy_id, policy_status, last_updated) VALUES (NEW.id, 'pending', NOW());
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER create_default_policy_status
    AFTER INSERT ON policies
    FOR EACH ROW
    EXECUTE PROCEDURE create_default_policy_status();

-- Update overall policy status if a rule evaluation status is updated
-- error takes precedence over failure, failure takes precedence over success
CREATE OR REPLACE FUNCTION update_policy_status() RETURNS TRIGGER AS $$
BEGIN
    -- keep error if policy had errored
    IF (NEW.eval_status = 'error') THEN
        UPDATE policy_status SET policy_status = 'error', last_updated = NOW() WHERE policy_id = NEW.policy_id;
    -- mark status as successful if all evaluations are successful or skipped
    ELSEIF NOT EXISTS (SELECT * FROM rule_evaluation_status WHERE policy_id = NEW.policy_id AND eval_status != 'success' AND eval_status != 'skipped') THEN
        UPDATE policy_status SET policy_status = 'success', last_updated = NOW() WHERE policy_id = NEW.policy_id;
    -- mark policy as successful if it was pending and the new status is success
    ELSEIF (NEW.eval_status = 'success') THEN
        UPDATE policy_status SET policy_status = 'success', last_updated = NOW() WHERE policy_id = NEW.policy_id AND policy_status = 'pending';
    -- mark status as failed if it was successful or pending and the new status is failure
    -- and there are no errors
    ELSEIF (NEW.eval_status = 'failure') AND NOT EXISTS (SELECT * FROM rule_evaluation_status WHERE policy_id = NEW.policy_id and eval_status = 'error') THEN
        UPDATE policy_status SET policy_status = 'failure', last_updated = NOW()
        WHERE policy_id = NEW.policy_id AND (policy_status = 'success' OR policy_status = 'pending') AND NEW.eval_status = 'failure';
    -- only mark policy run as skipped if every evaluation was skipped
    ELSEIF (NEW.eval_status = 'skipped') THEN
        UPDATE policy_status SET policy_status = 'skipped', last_updated = NOW()
        WHERE policy_id = NEW.policy_id AND NOT EXISTS (SELECT * FROM rule_evaluation_status WHERE policy_id = NEW.policy_id AND eval_status != 'skipped');
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_policy_status
    AFTER INSERT OR UPDATE ON rule_evaluation_status
    FOR EACH ROW
    EXECUTE PROCEDURE update_policy_status();

-- Create default root project
INSERT INTO projects (name, metadata) VALUES ('Root Project', '{}');

-- Create default root organization

INSERT INTO organizations (name, company) 
VALUES ('Root Organization', 'Root Company');

INSERT INTO groups (organization_id, name, is_protected)
VALUES (1, 'Root Group', TRUE);

-- superadmin role
INSERT INTO roles (organization_id, name, is_admin, is_protected)
VALUES (1, 'Superadmin Role', TRUE, TRUE);

INSERT INTO users (organization_id, email, username, password, first_name, last_name, is_protected, needs_password_change)
VALUES (1, 'root@localhost', 'root', '$argon2id$v=19$m=16,t=2,p=1$c2VjcmV0aGFzaA$WP4Vqo6QtHBY+n0x99R81Q', 'Root', 'Admin', TRUE, FALSE);   -- password is P4ssw@rd

INSERT INTO user_groups (user_id, group_id) VALUES (1, 1);
INSERT INTO user_roles (user_id, role_id) VALUES (1, 1);
