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
	"time"

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

// REGISTRY is the default registry to use
var REGISTRY = "ghcr.io"

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
	if in.Limit == -1 {
		in.Limit = PaginationLimit
	}
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

	decryptedToken, err := GetProviderAccessToken(ctx, s.store, in.Provider, in.GroupId, true)
	if err != nil {
		return nil, status.Errorf(codes.PermissionDenied, "user not authorized to interact with provider")
	}

	// call github api to get list of packages
	client, err := ghclient.NewRestClient(ctx, ghclient.GitHubConfig{
		Token: decryptedToken.AccessToken,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot create github client")
	}

	var results []*pb.Artifact

	user, err := client.GetAuthenticatedUser(ctx)
	if err != nil {
		return nil, err
	}
	isOrg := (*user.Type == "Organization")
	pkgList, err := client.ListAllPackages(ctx, isOrg, in.ArtifactType, int(pageNumber), int(itemsPerPage))
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

	if in.LatestVersions < 0 || in.LatestVersions > 10 {
		return nil, status.Errorf(codes.InvalidArgument, "latest versions must be between 0 and 10")
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

	decryptedToken, err := GetProviderAccessToken(ctx, s.store, in.Provider, in.GroupId, true)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting provider access token")
	}

	// call github api to get detail for an artifact
	client, err := ghclient.NewRestClient(ctx, ghclient.GitHubConfig{
		Token: decryptedToken.AccessToken,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot create github client")
	}

	user, err := client.GetAuthenticatedUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get authenticated user")
	}
	isOrg := (*user.Type == "Organization")
	pkg, err := client.GetPackageByName(ctx, isOrg, in.ArtifactType, in.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get package")
	}

	// get versions
	versions, err := client.GetPackageVersions(ctx, isOrg, in.ArtifactType, in.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get package versions")
	}

	final_versions := []*pb.ArtifactVersion{}
	for _, version := range versions {
		// first try to read the manifest
		imageRef := fmt.Sprintf("%s/%s/%s@%s", REGISTRY, *pkg.GetOwner().Login, pkg.GetName(), version.GetName())
		manifest, err := container.GetImageManifest(imageRef, user.GetLogin(), decryptedToken.AccessToken)
		if err != nil {
			// we do not have a manifest, so we cannot add it to versions
			continue
		}

		// need to check if we have a cosign layer, then we skip it
		is_signature := false
		if manifest.Layers != nil {
			for _, layer := range manifest.Layers {
				if layer.MediaType == "application/vnd.dev.cosign.simplesigning.v1+json" {
					is_signature = true
					break
				}
			}
		}
		if is_signature {
			continue
		}

		current_version := &pb.ArtifactVersion{}
		current_version.IsVerified = false
		current_version.IsBundleVerified = false
		current_version.VersionId = version.GetID()
		current_version.Tags = version.Metadata.Container.Tags
		current_version.Sha = version.GetName()
		current_version.CreatedAt = timestamppb.New(version.GetCreatedAt().Time)

		// get information about signature
		signature, err := container.GetSignatureTag(REGISTRY, *pkg.GetOwner().Login, pkg.GetName(), version.GetName())

		// an image is signed if we can find a signature tag for it
		current_version.IsSigned = (err == nil && signature != "")

		// if there is a signature, we can move forward and retrieve details
		if current_version.IsSigned {
			// we need to extract manifest from the signature
			manifest, err := container.GetImageManifest(signature, user.GetLogin(), decryptedToken.AccessToken)
			if err == nil && manifest.Layers != nil {
				identity, issuer, err := container.ExtractIdentityFromCertificate(manifest)
				if err == nil && identity != "" && issuer != "" {
					current_version.CertIdentity = &identity
					current_version.CertIssuer = &issuer

					// we have issuer and identity, we can verify the image
					verified, bundleVerified, imageKeys, err := container.VerifyFromIdentity(ctx, REGISTRY,
						*pkg.GetOwner().Login, pkg.GetName(), version.GetName(), identity, issuer)
					if err == nil {
						// we can add information for the image
						current_version.IsVerified = verified
						current_version.IsBundleVerified = bundleVerified

						log_id := imageKeys["RekorLogID"].(string)
						current_version.RekorLogId = &log_id

						log_index := int32(imageKeys["RekorLogIndex"].(int64))
						current_version.RekorLogIndex = &log_index

						signature_time := timestamppb.New(time.Unix(imageKeys["SignatureTime"].(int64), 0))
						current_version.SignatureTime = signature_time

						workflow_name := imageKeys["WorkflowName"].(string)
						current_version.GithubWorkflowName = &workflow_name

						workflow_repository := imageKeys["WorkflowRepository"].(string)
						current_version.GithubWorkflowRepository = &workflow_repository

						workflow_sha := imageKeys["WorkflowSha"].(string)
						current_version.GithubWorkflowCommitSha = &workflow_sha

						workflow_trigger := imageKeys["WorkflowTrigger"].(string)
						current_version.GithubWorkflowTrigger = &workflow_trigger
					}
				}
			}
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
