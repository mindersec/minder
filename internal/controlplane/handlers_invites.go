// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controlplane

import (
	"context"
	"database/sql"
	"errors"

	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/internal/invites"
	"github.com/stacklok/minder/internal/projects"
	"github.com/stacklok/minder/internal/util"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
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
