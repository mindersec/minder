-- ListInvitationsForProject collects the information visible to project
-- administrators after an invitation has been issued.  In particular, it
-- *does not* report the invitation code, which is a secret intended for
-- the invitee.

-- name: ListInvitationsForProject :many
SELECT user_invites.email, role, users.identity_subject, user_invites.created_at, user_invites.updated_at
FROM user_invites
  JOIN users ON user_invites.sponsor = users.id
WHERE project = $1;

-- GetInvitationsByEmail retrieves all invitations for a given email address.
-- This is intended to be called by a logged in user with their own email address,
-- to allow them to accept invitations even if email delivery was not working.
-- Note that this requires that the destination email address matches the email
-- address of the logged in user in the external identity service / auth token.

-- name: GetInvitationsByEmail :many
SELECT * FROM user_invites WHERE email = $1;

-- GetInvitationByEmailAndProjectAndRole retrieves an invitation by email, project,
-- and role.

-- name: GetInvitationByEmailAndProjectAndRole :one
SELECT * FROM user_invites WHERE email = $1 AND project = $2 AND role = $3;

-- GetInvitationByCode retrieves an invitation by its code. This is intended to
-- be called by a user who has received an invitation email and is following the
-- link to accept the invitation or when querying for additional info about the
-- invitation.

-- name: GetInvitationByCode :one
SELECT * FROM user_invites WHERE code = $1;

-- CreateInvitation creates a new invitation. The code is a secret that is sent
-- to the invitee, and the email is the address to which the invitation will be
-- sent. The role is the role that the invitee will have when they accept the
-- invitation. The project is the project to which the invitee will be invited.
-- The sponsor is the user who is inviting the invitee.

-- name: CreateInvitation :one
INSERT INTO user_invites (code, email, role, project, sponsor) VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- DeleteInvitation deletes an invitation by its code. This is intended to be
-- called by a user who has issued an invitation and then accepted it, declined
-- it or the sponsor has decided to revoke it.

-- name: DeleteInvitation :one
DELETE FROM user_invites WHERE code = $1 RETURNING *;

-- UpdateInvitation updates an invitation by its code. This is intended to be
-- called by a user who has issued an invitation and then decided to bump its
-- expiration.

-- name: UpdateInvitation :one
UPDATE user_invites SET updated_at = NOW() WHERE code = $1 RETURNING *;