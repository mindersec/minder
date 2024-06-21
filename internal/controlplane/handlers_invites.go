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
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

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

	// Get the sponsor's user information
	sponsorUser, err := s.store.GetUserByID(ctx, retInvite.Sponsor)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %s", err)
	}

	targetProject, err := s.store.GetProjectByID(ctx, retInvite.Project)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get project: %s", err)
	}

	// Resolve the sponsor's identity and display name
	identity, err := s.idClient.Resolve(ctx, sponsorUser.IdentitySubject)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("error resolving identity")
		return nil, util.UserVisibleError(codes.NotFound, "could not find identity %q", sponsorUser.IdentitySubject)
	}

	return &pb.GetInviteDetailsResponse{
		ProjectDisplay: targetProject.Name,
		SponsorDisplay: identity.Human(),
		ExpiresAt:      timestamppb.New(retInvite.UpdatedAt.Add(7 * 24 * time.Hour)),
		Expired:        time.Now().After(retInvite.UpdatedAt.Add(7 * 24 * time.Hour)),
	}, nil
}
