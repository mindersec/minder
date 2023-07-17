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

	"github.com/stacklok/mediator/pkg/auth"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/oci"
	ghclient "github.com/stacklok/mediator/pkg/providers/github"
)

// ImageRef is a reference to an image
type ImageRef struct {
	URI string
}

// ListPackages lists all packages
func (s *Server) ListPackages(ctx context.Context, in *pb.ListPackagesRequest) (*pb.ListPackagesResponse, error) {
	if in.Provider != auth.Github {
		return nil, status.Errorf(codes.InvalidArgument, "provider not supported: %v", in.Provider)
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
		return nil, err
	}

	// call github api to get list of packages
	client, err := ghclient.NewRestClient(ctx, ghclient.GitHubConfig{
		Token: decryptedToken.AccessToken,
	})
	if err != nil {
		return nil, err
	}

	var results []*pb.Packages

	pkgList, err := client.ListAllContainers(ctx, false)
	if err != nil {
		return nil, err
	}

	user, err := client.GetAuthenticatedUser(ctx)
	if err != nil {
		return nil, err
	}

	for _, pkg := range pkgList.Packages {
		tags := pkg.LastVersion.Metadata.Container.Tags
		if len(tags) == 0 {
			// we cannot check images without tags
			continue
		}
		manifest, err := oci.GetImageManifest(*pkg.Package.Owner.Login, *pkg.Package.Name, tags, *user.Login, decryptedToken.AccessToken)
		if err != nil {
			return nil, err
		}

		// check if singed
		for _, layer := range manifest.Layers {
			fmt.Println(layer)
		}

		latest_tag := ""
		if pkg.LastVersion.Metadata.Container.Tags != nil && len(pkg.LastVersion.Metadata.Container.Tags) > 0 {
			latest_tag = pkg.LastVersion.Metadata.Container.Tags[0]
		}

		createdAt := pkg.LastVersion.CreatedAt

		results = append(results, &pb.Packages{
			Owner: *pkg.Package.Owner.Login,
			Name:  *pkg.Package.Name,
			PkgId: *pkg.Package.ID,
			LastVersion: &pb.PackageVersion{
				VersionId: *pkg.LastVersion.Version,
				Tag:       latest_tag,
				Sha256:    *pkg.LastVersion.Name,
				IsSigned:  false,
				CreatedAt: timestamppb.New(createdAt.Time),
			},
		})
	}

	return &pb.ListPackagesResponse{
		Results: results,
	}, nil
}
