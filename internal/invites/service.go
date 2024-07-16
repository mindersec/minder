//
// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package invites

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/internal/auth"
	"github.com/stacklok/minder/internal/auth/jwt"
	"github.com/stacklok/minder/internal/authz"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/email"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/projects"
	"github.com/stacklok/minder/internal/util"
	minder "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

// InviteService encapsulates the methods to manage user invites to a project
type InviteService interface {
	// UpdateInvite updates the invite status
	UpdateInvite(ctx context.Context, qtx db.Querier, idClient auth.Resolver, eventsPub events.Publisher,
		emailConfig serverconfig.EmailConfig, targetProject uuid.UUID, authzRole authz.Role, inviteeEmail string,
	) (*minder.Invitation, error)
}

type inviteService struct {
}

// NewInviteService creates a new instance of InviteService
func NewInviteService() InviteService {
	return &inviteService{}
}

func (_ *inviteService) UpdateInvite(ctx context.Context, qtx db.Querier, idClient auth.Resolver, eventsPub events.Publisher,
	emailConfig serverconfig.EmailConfig, targetProject uuid.UUID, authzRole authz.Role, inviteeEmail string,
) (*minder.Invitation, error) {
	var userInvite db.UserInvite
	// Get the sponsor's user information (current user)
	currentUser, err := qtx.GetUserBySubject(ctx, jwt.GetUserSubjectFromContext(ctx))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %s", err)
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

	// Resolve the sponsor's identity and display name
	identity, err := idClient.Resolve(ctx, currentUser.IdentitySubject)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("error resolving identity")
		return nil, util.UserVisibleError(codes.NotFound, "could not find identity %q", currentUser.IdentitySubject)
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
			return nil, fmt.Errorf("error generating UUID: %w", err)
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
