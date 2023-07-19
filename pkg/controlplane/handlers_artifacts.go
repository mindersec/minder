// Copyright 2023 Stacklok, Inc
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

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/mediator/pkg/auth"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	ghclient "github.com/stacklok/mediator/pkg/providers/github"
)

// ARTIFACT_TYPES is a list of supported artifact types
var ARTIFACT_TYPES = []string{"npm", "maven", "rubygems", "docker", "nuget", "container"}

// ListArtifacts lists all artifacts for a given group and provider
// nolint:gocyclo
func (s *Server) ListArtifacts(ctx context.Context, in *pb.ListArtifactsRequest) (*pb.ListArtifactsResponse, error) {
	if in.Provider != auth.Github {
		return nil, status.Errorf(codes.InvalidArgument, "provider not supported: %v", in.Provider)
	}

	// validate artifact
	valid := false
	for _, s := range ARTIFACT_TYPES {
		if in.ArtifactType == s {
			valid = true
			break
		}
	}
	if !valid {
		return nil, status.Errorf(codes.InvalidArgument, "artifact type not supported: %v", in.ArtifactType)
	}

	// if we do not have a group, check if we can infer it
	if in.GroupId == 0 {
		group, err := auth.GetDefaultGroup(ctx)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "cannot infer group id")
		}
		in.GroupId = group
	}

	// define default values for limit and offset
	if in.Limit == -1 {
		in.Limit = PaginationLimit
	}

	// check if user is authorized
	if !IsRequestAuthorized(ctx, in.GroupId) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	// Check if needs github authorization
	isGithubAuthorized := IsProviderCallAuthorized(ctx, s.store, in.Provider, in.GroupId)
	if !isGithubAuthorized {
		return nil, status.Errorf(codes.PermissionDenied, "user not authorized to interact with provider")
	}

	decryptedToken, err := GetProviderAccessToken(ctx, s.store, in.Provider, in.GroupId, true)
	if err != nil {
		return nil, err
	}

	// call github api to get list of packages
	client, err := ghclient.NewRestClient(ctx, ghclient.GitHubConfig{
		Token: decryptedToken.AccessToken,
	})
	if err != nil {
		return nil, err
	}

	var results []*pb.Artifact

	user, err := client.GetAuthenticatedUser(ctx)
	if err != nil {
		return nil, err
	}
	isOrg := (*user.Type == "Organization")
	pkgList, err := client.ListAllPackages(ctx, isOrg, in.ArtifactType)
	if err != nil {
		return nil, err
	}

	for _, pkg := range pkgList.Packages {
		results = append(results, &pb.Artifact{
			ArtifactId: pkg.GetID(),
			Owner:      *pkg.GetOwner().Login,
			Name:       pkg.GetName(),
			Type:       pkg.GetPackageType(),
			Repository: pkg.GetRepository().GetFullName(),
			Visibility: pkg.GetVisibility(),
			CreatedAt:  timestamppb.New(pkg.GetCreatedAt().Time),
			UpdatedAt:  timestamppb.New(pkg.GetUpdatedAt().Time),
		})
	}

	// slice the results according to start and limit
	if in.Offset > int32(len(results)) {
		return &pb.ListArtifactsResponse{}, nil
	}
	limit := in.Offset + in.Limit
	if limit > int32(len(results)) {
		limit = int32(len(results)) - in.Offset
	}
	results = results[in.Offset:limit]
	return &pb.ListArtifactsResponse{
		Results: results,
	}, nil
}
