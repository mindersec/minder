-- Organisations

-- CreateOrganisation creates a new organisation
INSERT INTO organisations (name) VALUES ($1) RETURNING *;

-- GetOrganisationByID retrieves an organisation by ID
SELECT * FROM organisations WHERE id = $1;

-- UpdateOrganisation updates an organisation's name
UPDATE organisations SET name = $2, updated_at = NOW() WHERE id = $1 RETURNING *;

-- DeleteOrganisation deletes an organisation by ID
DELETE FROM organisations WHERE id = $1;

-- Users

-- CreateUser creates a new user
INSERT INTO users (organisation_id, group_id, email, password, first_name, last_name, is_admin, is_super_admin) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING *;

-- GetUserByID retrieves a user by ID
SELECT * FROM users WHERE id = $1;

-- GetUserByEmail retrieves a user by email
SELECT * FROM users WHERE email = $1;

-- UpdateUser updates a user's information
UPDATE users SET organisation_id = $2, group_id = $3, email = $4, password = $5, first_name = $6, last_name = $7, is_admin = $8, is_super_admin = $9, updated_at = NOW() WHERE id = $1 RETURNING *;

-- DeleteUser deletes a user by ID
DELETE FROM users WHERE id = $1;

-- Groups

-- CreateGroup creates a new group
INSERT INTO groups (organisation_id, name) VALUES ($1, $2) RETURNING *;

-- GetGroupByID retrieves a group by ID
SELECT * FROM groups WHERE id = $1;

-- UpdateGroup updates a group's name
UPDATE groups SET name = $2, updated_at = NOW() WHERE id = $1 RETURNING *;

-- DeleteGroup deletes a group by ID
DELETE FROM groups WHERE id = $1;

-- Roles

-- CreateRole creates a new role
INSERT INTO roles (organisation_id, name) VALUES ($1, $2) RETURNING *;

-- GetRoleByID retrieves a role by ID
SELECT * FROM roles WHERE id = $1;

-- UpdateRole updates a role's name
UPDATE roles SET name = $2, updated_at = NOW() WHERE id = $1 RETURNING *;

-- DeleteRole deletes a role by ID
DELETE FROM roles WHERE id = $1;

-- GroupRoles

-- AddRoleToGroup adds a role to a group
INSERT INTO group_roles (group_id, role_id) VALUES ($1, $2) RETURNING *;

-- GetGroupRoles retrieves all roles in a group
SELECT * FROM group_roles WHERE group_id = $1;

-- RemoveRoleFromGroup removes a role from a group
DELETE FROM group_roles WHERE group_id = $1 AND role_id = $2;

-- UserRoles

-- AssignRoleToUser assigns a role to a user
INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2) RETURNING *;

-- GetUserRoles retrieves all roles assigned to a user
SELECT * FROM user_roles WHERE user_id = $1;

-- RevokeRoleFromUser revokes a role from a user
DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2;

-- AccessTokens

-- CreateAccessToken creates a new access token for an organisation
INSERT INTO access_tokens (organisation_id, encrypted_token) VALUES ($1, $2) RETURNING *;

-- GetAccessTokenByOrganisationID retrieves an access token by organisation ID
SELECT * FROM access_tokens WHERE organisation_id = $1;

-- UpdateAccessToken updates an organisation's access token
UPDATE access_tokens SET encrypted_token = $2, updated_at = NOW() WHERE organisation_id = $1 RETURNING *;

-- DeleteAccessToken deletes an access token by organisation ID
DELETE FROM access_tokens WHERE organisation_id = $1;


