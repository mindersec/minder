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
	"fmt"

	go_github "github.com/google/go-github/v53/github"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/stacklok/mediator/pkg/auth"
	"github.com/stacklok/mediator/pkg/container"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	ghclient "github.com/stacklok/mediator/pkg/providers/github"
)

// ARTIFACT_TYPES is a list of supported artifact types
var ARTIFACT_TYPES = sets.New[string]("npm", "maven", "rubygems", "docker", "nuget", "container")

// ListArtifacts lists all artifacts for a given group and provider
// nolint:gocyclo
func (s *Server) ListArtifacts(ctx context.Context, in *pb.ListArtifactsRequest) (*pb.ListArtifactsResponse, error) {
	if in.Provider != auth.Github {
		return nil, status.Errorf(codes.InvalidArgument, "provider not supported: %v", in.Provider)
	}

	// validate artifact
	if !ARTIFACT_TYPES.Has(in.ArtifactType) {
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
	if in.Limit <= 0 {
		in.Limit = PaginationLimit
	}
	// GitHub API only works on offsets of whole page sizes
	pageNumber := (in.Offset / in.Limit) + 1
	itemsPerPage := in.Limit

	// check if user is authorized
	if !IsRequestAuthorized(ctx, in.GroupId) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	// Check if needs github authorization
	isGithubAuthorized := IsProviderCallAuthorized(ctx, s.store, in.Provider, in.GroupId)
	if !isGithubAuthorized {
		return nil, status.Errorf(codes.PermissionDenied, "user not authorized to interact with provider")
	}

	decryptedToken, owner_filter, err := GetProviderAccessToken(ctx, s.store, in.Provider, in.GroupId, true)
	if err != nil {
		return nil, status.Errorf(codes.PermissionDenied, "user not authorized to interact with provider")
	}
	isOrg := (owner_filter != "")

	// call github api to get list of packages
	client, err := ghclient.NewRestClient(ctx, ghclient.GitHubConfig{
		Token: decryptedToken.AccessToken,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot create github client")
	}

	var results []*pb.Artifact

	pkgList, err := client.ListAllPackages(ctx, isOrg, owner_filter, in.ArtifactType, int(pageNumber), int(itemsPerPage))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot list packages")
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

	return &pb.ListArtifactsResponse{
		Results: results,
	}, nil
}

// GetArtifactByName gets an artifact by type and name
// nolint:gocyclo
func (s *Server) GetArtifactByName(ctx context.Context, in *pb.GetArtifactByNameRequest) (*pb.GetArtifactByNameResponse, error) {
	if in.Provider != auth.Github {
		return nil, status.Errorf(codes.InvalidArgument, "provider not supported: %v", in.Provider)
	}

	// tag and latest versions cannot be set at same time
	if in.Tag != "" && in.LatestVersions > 1 {
		return nil, status.Errorf(codes.InvalidArgument, "tag and latest versions cannot be set at same time")
	}

	if in.LatestVersions < 1 || in.LatestVersions > 10 {
		return nil, status.Errorf(codes.InvalidArgument, "latest versions must be between 1 and 10")
	}

	// if we do not have a group, check if we can infer it
	if in.GroupId == 0 {
		group, err := auth.GetDefaultGroup(ctx)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "cannot infer group id")
		}
		in.GroupId = group
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

	decryptedToken, owner_filter, err := GetProviderAccessToken(ctx, s.store, in.Provider, in.GroupId, true)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting provider access token")
	}
	isOrg := (owner_filter != "")

	// call github api to get detail for an artifact
	client, err := ghclient.NewRestClient(ctx, ghclient.GitHubConfig{
		Token: decryptedToken.AccessToken,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot create github client")
	}

	pkg, err := client.GetPackageByName(ctx, isOrg, owner_filter, in.ArtifactType, in.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get package")
	}

	// get versions
	var versions []*go_github.PackageVersion
	if in.Tag != "" {
		version, err := client.GetPackageVersionByTag(ctx, isOrg, owner_filter, in.ArtifactType, in.Name, in.Tag)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "cannot get package version")
		}
		if version == nil {
			return nil, status.Errorf(codes.NotFound, "package version not found")
		}
		versions = append(versions, version)
	} else {
		versions, err = client.GetPackageVersions(ctx, isOrg, owner_filter, in.ArtifactType, in.Name)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "cannot get package versions")
		}
	}

	final_versions := []*pb.ArtifactVersion{}
	for _, version := range versions {
		// first try to read the manifest
		imageRef := fmt.Sprintf("%s/%s/%s@%s", container.REGISTRY, *pkg.GetOwner().Login, pkg.GetName(), version.GetName())
		current_version := &pb.ArtifactVersion{
			VersionId: version.GetID(),
			Tags:      version.Metadata.Container.Tags,
			Sha:       version.GetName(),
			CreatedAt: timestamppb.New(version.GetCreatedAt().Time),
		}

		signature_verification, github_workflow, err := container.ValidateSignature(ctx,
			decryptedToken.AccessToken, *pkg.GetOwner().Login, imageRef)
		if err == nil {
			current_version.SignatureVerification = signature_verification
			current_version.GithubWorkflow = github_workflow
		}
		final_versions = append(final_versions, current_version)
		if len(final_versions) == int(in.LatestVersions) {
			break
		}
	}

	return &pb.GetArtifactByNameResponse{Artifact: &pb.Artifact{
		ArtifactId: pkg.GetID(),
		Owner:      *pkg.GetOwner().Login,
		Name:       pkg.GetName(),
		Type:       pkg.GetPackageType(),
		Visibility: pkg.GetVisibility(),
		Repository: pkg.GetRepository().GetFullName(),
		CreatedAt:  timestamppb.New(pkg.GetCreatedAt().Time),
		UpdatedAt:  timestamppb.New(pkg.GetUpdatedAt().Time),
	},
		Versions: final_versions,
	}, nil
}
