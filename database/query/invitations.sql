-- ListInvitationsForProject collects the information visible to project
-- administrators after an invitation has been issued.  In particular, it
-- *does not* report the invitation code, which is a secret intended for
-- the invitee.

-- name: ListInvitationsForProject :many
SELECT user_invites.email, role, users.identity_subject, user_invites.created_at, user_invites.updated_at
FROM user_invites
  JOIN users ON user_invites.sponsor = users.id
WHERE project = $1;

-- GetInvitationByEmail retrieves all invitations for a given email address.
-- This is intended to be called by a logged in user with their own email address,
-- to allow them to accept invitations even if email delivery was not working.
-- Note that this requires that the destination email address matches the email
-- address of the logged in user in the external identity service / auth token.

-- name: GetInvitationByEmail :many
SELECT code, role, project, updated_at FROM user_invites WHERE email = $1;