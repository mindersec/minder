// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"database/sql"
	"errors"

	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/invites"
	"github.com/mindersec/minder/internal/projects"
	"github.com/mindersec/minder/internal/util"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// GetInviteDetails returns the details of an invitation
func (s *Server) GetInviteDetails(ctx context.Context, req *pb.GetInviteDetailsRequest) (*pb.GetInviteDetailsResponse, error) {
	// Get the invitation code from the request
	code := req.GetCode()
	if code == "" {
		return nil, util.UserVisibleError(codes.InvalidArgument, "code is required")
	}

	retInvite, err := s.store.GetInvitationByCode(ctx, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "invitation not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get invitation: %s", err)
	}

	targetProject, err := s.store.GetProjectByID(ctx, retInvite.Project)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get project: %s", err)
	}

	// Parse the project metadata, so we can get the display name set by project owner
	meta, err := projects.ParseMetadata(&targetProject)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error parsing project metadata: %v", err)
	}

	// Resolve the sponsor's identity and display name
	identity, err := s.idClient.Resolve(ctx, retInvite.IdentitySubject)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("error resolving identity")
		return nil, util.UserVisibleError(codes.NotFound, "could not find identity %q", retInvite.IdentitySubject)
	}

	return &pb.GetInviteDetailsResponse{
		ProjectDisplay: meta.Public.DisplayName,
		SponsorDisplay: identity.Human(),
		ExpiresAt:      invites.GetExpireIn7Days(retInvite.UpdatedAt),
		Expired:        invites.IsExpired(retInvite.UpdatedAt),
	}, nil
}
