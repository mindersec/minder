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

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/stacklok/mediator/pkg/auth"
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
		return nil, err
	}
	isOrg := (*user.Type == "Organization")
	pkg, versions, err := client.GetPackageByName(ctx, isOrg, in.ArtifactType, in.Name, int(in.LatestVersions))
	if err != nil {
		return nil, err
	}

	// TODO: we only cover versions for container at the moment
	art_versions := make([]*pb.ArtifactVersion, len(versions))
	if in.ArtifactType == "container" {
		for _, version := range versions {
			tags := version.Metadata.GetContainer().Tags
			fmt.Println(tags)

			//tag := ""
			/*if len(tags) > 0 {
				tag = tags[0]

				// having the tag, we can get more information about this image
				manifest, err := oci.GetImageManifest(*pkg.GetOwner().Login, pkg.GetName(), tag, user.GetLogin(), decryptedToken.AccessToken)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "cannot get image manifest")
				}

				// try to find the cosign layer
				is_signed := false
				signature := ""
				signature_digest := ""
				log_id := int32(0)
				for _, layer := range manifest.Layers {
					result, _ := json.MarshalIndent(layer, "", "  ")

					if layer.MediaType == "application/vnd.dev.cosign.simplesigning.v1+json" {

						is_signed = true
						signature_digest = layer.Digest.String()
						signature = layer.Annotations["dev.cosignproject.cosign/signature"]

						rawMessage := layer.Annotations["dev.sigstore.cosign/bundle"]
						var bundleData map[string]interface{}
						rawData := json.RawMessage(rawMessage)
						err := json.Unmarshal(rawData, &bundleData)
						if err == nil {
							payload := bundleData["Payload"].(map[string]interface{})
							if payload != nil {
								current_log, ok := payload["logIndex"]
								if ok {
									log_id = int32(current_log.(float64))
								}
							}
						}

						cert := layer.Annotations["dev.sigstore.cosign/certificate"]
						chain := layer.Annotations["dev.sigstore.cosign/chain"]
						is_verified, err := oci.CheckSignatureVerification(*pkg.GetOwner().Login, pkg.GetName(), tag, cert, chain)
						fmt.Println(is_verified)

						break
					}
				}
				art_versions[i] = &pb.ArtifactVersion{
					VersionId:       version.GetID(),
					Tag:             tag,
					Sha:             *version.Name,
					CreatedAt:       timestamppb.New(version.CreatedAt.Time),
					UpdatedAt:       timestamppb.New(version.UpdatedAt.Time),
					IsSigned:        is_signed,
					Signature:       signature,
					SignatureDigest: signature_digest,
					RekorLogId:      log_id,
				}
			}*/
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
		Versions: art_versions,
	}, nil
}
