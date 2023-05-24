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

-- organisations table
CREATE TABLE organisations (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    company TEXT NOT NULL UNIQUE,
    root_admin_id INT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- groups table
CREATE TABLE groups (
    id SERIAL PRIMARY KEY,
    organisation_id INTEGER REFERENCES organisations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- roles table
CREATE TABLE roles (
    id SERIAL PRIMARY KEY,
    group_id INTEGER REFERENCES groups(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- users table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    organisation_id INTEGER REFERENCES organisations(id) ON DELETE CASCADE,
    group_id INTEGER REFERENCES groups(id) ON DELETE SET NULL,
    role_id INTEGER REFERENCES roles(id) ON DELETE SET NULL,
    email TEXT NOT NULL UNIQUE,
    username TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- access_tokens table
CREATE TABLE access_tokens (
    id SERIAL PRIMARY KEY,
    organisation_id INTEGER NOT NULL REFERENCES organisations(id) ON DELETE CASCADE,
    encrypted_token TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Unique constraint
ALTER TABLE access_tokens ADD CONSTRAINT unique_organisation_id UNIQUE (organisation_id);
ALTER TABLE organisations ADD CONSTRAINT unique_name UNIQUE (name);

-- Indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_organisation_id ON users(organisation_id);
CREATE INDEX idx_groups_organisation_id ON groups(organisation_id);
CREATE INDEX idx_roles_group_id ON roles(group_id);
CREATE INDEX idx_access_tokens_organisation_id ON access_tokens(organisation_id);

-- Create default root organisation

INSERT INTO organisations (name, company, root_admin_id) 
VALUES ('Root Organization', 'Root Company', 1);

INSERT INTO groups (organisation_id, name)
VALUES (1, 'Root Group');

INSERT INTO roles (group_id, name, is_admin)
VALUES (1, 'Role Role', TRUE);

INSERT INTO users (organisation_id, group_id, role_id, email, username, password, first_name, last_name)
VALUES (1, 1, 1, 'root@localhost', 'root', '$argon2id$v=19$m=0,t=3,p=2$mQDRkaBe7p3pbGvzgFn20Q$GYA0SkpXhVMLwcjRSPKCUpmd4ptMcdUcQ5YTAOnLFKs', 'Root', 'Admin');
