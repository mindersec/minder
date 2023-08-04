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

	"github.com/google/go-containerregistry/pkg/name"
	go_github "github.com/google/go-github/v53/github"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
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
		imageRef := fmt.Sprintf("%s/%s/%s@%s", REGISTRY, *pkg.GetOwner().Login, pkg.GetName(), version.GetName())
		baseRef, err := name.ParseReference(imageRef)
		if err != nil {
			// Cannot parse the image reference, continue to the next version
			continue
		}
		manifest, err := container.GetImageManifest(baseRef, *pkg.GetOwner().Login, decryptedToken.AccessToken)
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

		signature_verification := &pb.SignatureVerification{
			IsVerified:       false,
			IsSigned:         false,
			IsBundleVerified: false,
		}

		current_version := &pb.ArtifactVersion{
			VersionId: version.GetID(),
			Tags:      version.Metadata.Container.Tags,
			Sha:       version.GetName(),
			CreatedAt: timestamppb.New(version.GetCreatedAt().Time),
		}

		// get information about signature
		signature, err := container.GetSignatureTag(baseRef)

		// if there is a signature, we can move forward and retrieve details
		if err == nil && signature != nil {
			// we need to extract manifest from the signature
			manifest, err := container.GetImageManifest(signature, *pkg.GetOwner().Login, decryptedToken.AccessToken)
			if err == nil && manifest.Layers != nil {
				signature_verification.IsSigned = true
				identity, issuer, err := container.ExtractIdentityFromCertificate(manifest)
				if err == nil && identity != "" && issuer != "" {
					signature_verification.CertIdentity = &identity
					signature_verification.CertIssuer = &issuer

					// we have issuer and identity, we can verify the image
					imageRef := fmt.Sprintf("%s/%s/%s@%s", REGISTRY, *pkg.GetOwner().Login, pkg.GetName(), version.GetName())
					verified, bundleVerified, imageKeys, err := container.VerifyFromIdentity(ctx, imageRef,
						*pkg.GetOwner().Login, decryptedToken.AccessToken, identity, issuer)
					if err == nil {
						// we can add information for the image
						signature_verification.IsVerified = verified
						signature_verification.IsBundleVerified = bundleVerified
						signature_verification.RekorLogId = proto.String(imageKeys["RekorLogID"].(string))

						log_index := int32(imageKeys["RekorLogIndex"].(int64))
						signature_verification.RekorLogIndex = &log_index

						signature_time := timestamppb.New(time.Unix(imageKeys["SignatureTime"].(int64), 0))
						signature_verification.SignatureTime = signature_time

						current_version.GithubWorkflow = &pb.GithubWorkflow{
							Name:       imageKeys["WorkflowName"].(string),
							Repository: imageKeys["WorkflowRepository"].(string),
							CommitSha:  imageKeys["WorkflowSha"].(string),
							Trigger:    imageKeys["WorkflowTrigger"].(string),
						}
					}
				}
			}
		}
		current_version.SignatureVerification = signature_verification
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
