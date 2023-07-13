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

	"github.com/stacklok/mediator/pkg/auth"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/oci"
	ghclient "github.com/stacklok/mediator/pkg/providers/github"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ImageRef struct {
	URI string
}

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

	decryptedToken, err := GetProviderAccessToken(ctx, s.store, in.Provider, in.GroupId)
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
	var images []*ImageRef

	pkgList, err := client.ListAllPackages(ctx, false)
	if err != nil {
		return nil, err
	}

	for _, pkg := range pkgList.Packages {
		pkgURI := fmt.Sprintf("ghcr.io/%s/%s", *pkg.Owner.Login, *pkg.Name)

		imageDigest, err := oci.GetImageManifest(pkgURI, "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")
		if err != nil {
			return nil, err
		}

		images = append(images, &ImageRef{
			URI: fmt.Sprintf("%s@%s", pkgURI, imageDigest),
		})

		//
		results = append(results, &pb.Packages{

			Owner: *pkg.Owner.Login,
			Name:  *pkg.Name,
			PkgId: *pkg.ID,
		})
	}

	for _, image := range images {
		// check if image exists in
		fmt.Println(image.URI)
	}

	return &pb.ListPackagesResponse{
		Results: results,
	}, nil
}
