// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package invites

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/mindersec/minder/internal/auth"
	"github.com/mindersec/minder/internal/authz"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/email"
	"github.com/mindersec/minder/internal/projects"
	"github.com/mindersec/minder/internal/util"
	minder "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/eventer/interfaces"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

// InviteService encapsulates the methods to manage user invites to a project
type InviteService interface {
	// CreateInvite creates a new user invite
	CreateInvite(ctx context.Context, qtx db.Querier, eventsPub interfaces.Publisher,
		emailConfig serverconfig.EmailConfig, targetProject uuid.UUID, authzRole authz.Role, inviteeEmail string,
	) (*minder.Invitation, error)

	// UpdateInvite updates the invite status
	UpdateInvite(ctx context.Context, qtx db.Querier, eventsPub interfaces.Publisher,
		emailConfig serverconfig.EmailConfig, targetProject uuid.UUID, authzRole authz.Role, inviteeEmail string,
	) (*minder.Invitation, error)

	// RemoveInvite removes the user invite
	RemoveInvite(ctx context.Context, qtx db.Querier, idClient auth.Resolver, targetProject uuid.UUID,
		authzRole authz.Role, inviteeEmail string,
	) (*minder.Invitation, error)
}

type inviteService struct {
}

// NewInviteService creates a new instance of InviteService
func NewInviteService() InviteService {
	return &inviteService{}
}

func (_ *inviteService) UpdateInvite(ctx context.Context, qtx db.Querier, eventsPub interfaces.Publisher,
	emailConfig serverconfig.EmailConfig, targetProject uuid.UUID, authzRole authz.Role, inviteeEmail string,
) (*minder.Invitation, error) {
	var userInvite db.UserInvite
	// Get the sponsor's user information (current user)
	identity := auth.IdentityFromContext(ctx)
	if identity.String() == "" {
		return nil, status.Errorf(codes.Internal, "failed to get user")
	}

	// Get all invitations for this email and project
	existingInvites, err := qtx.GetInvitationsByEmailAndProject(ctx, db.GetInvitationsByEmailAndProjectParams{
		Email:   inviteeEmail,
		Project: targetProject,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting invitations: %v", err)
	}

	// Exit early if there are no or multiple existing invitations for this email and project
	if len(existingInvites) == 0 {
		return nil, util.UserVisibleError(codes.NotFound, "no invitations found for this email and project")
	} else if len(existingInvites) > 1 {
		return nil, status.Errorf(codes.Internal, "multiple invitations found for this email and project")
	}

	// At this point, there should be exactly 1 invitation.
	// Depending on the role from the request, we can either update the role and its expiration
	// or just bump the expiration date.
	// In both cases, we can use the same query.
	userInvite, err = qtx.UpdateInvitationRole(ctx, db.UpdateInvitationRoleParams{
		Code: existingInvites[0].Code,
		Role: authzRole.String(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error updating invitation: %v", err)
	}

	// Resolve the project's display name
	prj, err := qtx.GetProjectByID(ctx, userInvite.Project)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get project: %s", err)
	}

	// Parse the project metadata, so we can get the display name set by project owner
	meta, err := projects.ParseMetadata(&prj)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error parsing project metadata: %v", err)
	}

	inviteURL, err := getInviteUrl(emailConfig, userInvite)
	if err != nil {
		return nil, fmt.Errorf("error getting invite URL: %w", err)
	}

	emailSkipped := false
	// Publish the event for sending the invitation email
	// This will happen only if the role is updated (existingInvites[0].Role != authzRole.String())
	// or the role stayed the same, but the last invite update was more than a day ago
	if existingInvites[0].Role != authzRole.String() || userInvite.UpdatedAt.Sub(existingInvites[0].UpdatedAt) > 24*time.Hour {
		msg, err := email.NewMessage(
			ctx,
			userInvite.Email,
			inviteURL,
			emailConfig.MinderURLBase,
			userInvite.Role,
			meta.Public.DisplayName,
			identity.Human(),
		)
		if err != nil {
			if errors.Is(err, email.ErrValidationFailed) {
				return nil, util.UserVisibleError(codes.InvalidArgument, "error creating email message: %v", err)
			}
			return nil, status.Errorf(codes.Internal, "error creating email message: %v", err)
		}
		err = eventsPub.Publish(email.TopicQueueInviteEmail, msg)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "error publishing event: %v", err)
		}
	} else {
		zerolog.Ctx(ctx).Info().Msg("skipping sending email, role stayed the same and last update was less than a day ago")
		emailSkipped = true
	}

	return &minder.Invitation{
		Role:           userInvite.Role,
		Email:          userInvite.Email,
		Project:        userInvite.Project.String(),
		ProjectDisplay: prj.Name,
		Code:           userInvite.Code,
		InviteUrl:      inviteURL,
		Sponsor:        identity.String(),
		SponsorDisplay: identity.Human(),
		CreatedAt:      timestamppb.New(userInvite.CreatedAt),
		ExpiresAt:      GetExpireIn7Days(userInvite.UpdatedAt),
		Expired:        IsExpired(userInvite.UpdatedAt),
		EmailSkipped:   emailSkipped,
	}, nil

}

func (_ *inviteService) RemoveInvite(ctx context.Context, qtx db.Querier, idClient auth.Resolver, targetProject uuid.UUID,
	authzRole authz.Role, inviteeEmail string) (*minder.Invitation, error) {
	// Get all invitations for this email and project
	invitesToRemove, err := qtx.GetInvitationsByEmailAndProject(ctx, db.GetInvitationsByEmailAndProjectParams{
		Email:   inviteeEmail,
		Project: targetProject,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting invitation: %v", err)
	}

	// If there are no invitations for this email, return an error
	if len(invitesToRemove) == 0 {
		return nil, util.UserVisibleError(codes.NotFound, "no invitations found for this email and project")
	}

	// Find the invitation to remove. There should be only one invitation for the given role and email in the project.
	var inviteToRemove *db.GetInvitationsByEmailAndProjectRow
	for _, i := range invitesToRemove {
		if i.Role == authzRole.String() {
			inviteToRemove = &i
			break
		}
	}
	// If there's no invitation to remove, return an error
	if inviteToRemove == nil {
		return nil, util.UserVisibleError(codes.NotFound, "no invitation found for this role and email in the project")
	}
	// Delete the invitation
	ret, err := qtx.DeleteInvitation(ctx, inviteToRemove.Code)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error deleting invitation: %v", err)
	}

	// Resolve the project's display name
	prj, err := qtx.GetProjectByID(ctx, ret.Project)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get project: %s", err)
	}

	// Get the sponsor's user information (current user)
	sponsorUser, err := qtx.GetUserByID(ctx, ret.Sponsor)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %s", err)
	}

	// Resolve the sponsor's identity and display name
	sponsorDisplay := sponsorUser.IdentitySubject
	identity, err := idClient.Resolve(ctx, sponsorUser.IdentitySubject)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("error resolving identity")
	} else {
		sponsorDisplay = identity.Human()
	}

	return &minder.Invitation{
		Role:           ret.Role,
		Email:          ret.Email,
		Project:        ret.Project.String(),
		Code:           ret.Code,
		CreatedAt:      timestamppb.New(ret.CreatedAt),
		ExpiresAt:      GetExpireIn7Days(ret.UpdatedAt),
		Expired:        IsExpired(ret.UpdatedAt),
		Sponsor:        sponsorUser.IdentitySubject,
		SponsorDisplay: sponsorDisplay,
		ProjectDisplay: prj.Name,
	}, nil
}

func (_ *inviteService) CreateInvite(ctx context.Context, qtx db.Querier, eventsPub interfaces.Publisher,
	emailConfig serverconfig.EmailConfig, targetProject uuid.UUID, authzRole authz.Role, inviteeEmail string,
) (*minder.Invitation, error) {
	identity := auth.IdentityFromContext(ctx)
	// Slight hack -- only the null/default provider has String == UserID
	if identity == nil || identity.String() != identity.UserID {
		return nil, util.UserVisibleError(codes.PermissionDenied, "only human users can create invites")
	}
	// Get the sponsor's user information (current user)
	currentUser, err := qtx.GetUserBySubject(ctx, identity.String())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %s", err)
	}

	// Check if the user is already invited
	existingInvites, err := qtx.GetInvitationsByEmailAndProject(ctx, db.GetInvitationsByEmailAndProjectParams{
		Email:   inviteeEmail,
		Project: targetProject,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting invitations: %v", err)
	}

	if len(existingInvites) != 0 {
		return nil, util.UserVisibleError(
			codes.AlreadyExists,
			"invitation for this email and project already exists, use update instead",
		)
	}

	// If there are no invitations for this email, great, we should create one

	// Resolve the target project's display name
	prj, err := qtx.GetProjectByID(ctx, targetProject)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get target project: %s", err)
	}

	// Parse the project metadata, so we can get the display name set by project owner
	meta, err := projects.ParseMetadata(&prj)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error parsing project metadata: %v", err)
	}

	// Create the invitation
	userInvite, err := qtx.CreateInvitation(ctx, db.CreateInvitationParams{
		Code:    GenerateCode(),
		Email:   inviteeEmail,
		Role:    authzRole.String(),
		Project: targetProject,
		Sponsor: currentUser.ID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error creating invitation: %v", err)
	}

	inviteURL, err := getInviteUrl(emailConfig, userInvite)
	if err != nil {
		return nil, fmt.Errorf("error getting invite URL: %w", err)
	}

	// Publish the event for sending the invitation email
	msg, err := email.NewMessage(
		ctx,
		userInvite.Email,
		inviteURL,
		emailConfig.MinderURLBase,
		userInvite.Role,
		meta.Public.DisplayName,
		identity.Human(),
	)
	if err != nil {
		if errors.Is(err, email.ErrValidationFailed) {
			return nil, util.UserVisibleError(codes.InvalidArgument, "error creating email message: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "error creating email message: %v", err)
	}

	err = eventsPub.Publish(email.TopicQueueInviteEmail, msg)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error publishing event: %v", err)
	}

	return &minder.Invitation{
		Role:           userInvite.Role,
		Email:          userInvite.Email,
		Project:        userInvite.Project.String(),
		ProjectDisplay: prj.Name,
		Code:           userInvite.Code,
		InviteUrl:      inviteURL,
		Sponsor:        identity.String(),
		SponsorDisplay: identity.Human(),
		CreatedAt:      timestamppb.New(userInvite.CreatedAt),
		ExpiresAt:      GetExpireIn7Days(userInvite.UpdatedAt),
		Expired:        IsExpired(userInvite.UpdatedAt),
	}, nil
}

func getInviteUrl(emailCfg serverconfig.EmailConfig, userInvite db.UserInvite) (string, error) {
	inviteURL := ""
	if emailCfg.MinderURLBase != "" {
		baseUrl, err := url.Parse(emailCfg.MinderURLBase)
		if err != nil {
			return "", fmt.Errorf("error parsing base URL: %w", err)
		}
		inviteURL, err = url.JoinPath(baseUrl.String(), "join", userInvite.Code)
		if err != nil {
			return "", fmt.Errorf("error joining URL path: %w", err)
		}
	}
	return inviteURL, nil
}
