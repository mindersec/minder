// Copyright 2023 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/stacklok/mediator/pkg/auth"
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RegisterRepository adds repositories to the database and registers a webhook
// Once a user had enrolled in a group (they have a valid token), they can register
// repositories to be monitored by the mediator by provisioning a webhook on the
// repositor(ies).
// The API is called with a slice of repositories to register and a slice of events
// e.g.
//
//	grpcurl -plaintext -d '{
//		"repositories": [
//			{ "owner": "acme", "name": "widgets" },
//			{ "owner": "acme", "name": "gadgets" }
//		  ],
//		  "events": [ "push", "issues" ]
//	}' 127.0.0.1:8090 mediator.v1.RepositoryService/RegisterRepository
func (s *Server) RegisterRepository(ctx context.Context,
	in *pb.RegisterRepositoryRequest) (*pb.RegisterRepositoryResponse, error) {
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

	// Unmarshal the in.GetRepositories() into a struct Repository
	var repositories []Repository
	if in.GetRepositories() == nil || len(in.GetRepositories()) <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "no repositories provided")
	}

	for _, repository := range in.GetRepositories() {
		repositories = append(repositories, Repository{
			Owner:  repository.GetOwner(),
			Repo:   repository.GetName(),
			RepoID: repository.GetRepoId(), // Handle the RepoID here.
		})
	}

	registerData, err := RegisterWebHook(ctx, decryptedToken, repositories, in.Events)
	if err != nil {
		return nil, err
	}

	var results []*pb.RepositoryResult

	for _, result := range registerData {
		// Convert each result to a pb.RepositoryResult object
		pbResult := &pb.RepositoryResult{
			Owner:      result.Owner,
			Repository: result.Repository,
			RepoId:     result.RepoID,
			HookId:     result.HookID,
			HookUrl:    result.HookURL,
			HookName:   result.HookName,
			DeployUrl:  result.DeployURL,
			Success:    result.Success,
			Uuid:       result.HookUUID,
		}
		results = append(results, pbResult)

		// update the database
		_, err = s.store.UpdateRepositoryByID(ctx, db.UpdateRepositoryByIDParams{
			WebhookID:  sql.NullInt32{Int32: int32(result.HookID), Valid: true},
			WebhookUrl: result.HookURL,
			Provider:   in.Provider,
			GroupID:    in.GroupId,
			RepoOwner:  result.Owner,
			RepoName:   result.Repository,
			RepoID:     result.RepoID,
			DeployUrl:  result.DeployURL,
		})
		if err != nil {
			return nil, err
		}
	}

	response := &pb.RegisterRepositoryResponse{
		Results: results,
	}

	return response, nil
}

// ListRepositories returns a list of repositories for a given group
// This function will typically be called by the client to get a list of
// repositories that are registered present in the mediator database
// The API is called with a group id, limit and offset
func (s *Server) ListRepositories(ctx context.Context,
	in *pb.ListRepositoriesRequest) (*pb.ListRepositoriesResponse, error) {
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

	repos, err := s.store.ListRepositoriesByGroupID(ctx, db.ListRepositoriesByGroupIDParams{
		Provider: in.Provider,
		GroupID:  in.GroupId,
		Limit:    in.Limit,
		Offset:   in.Offset,
	})

	if err != nil {
		return nil, err
	}

	var resp pb.ListRepositoriesResponse
	var results []*pb.Repositories

	// Do not return results containing the webhook (e.g. registered), if the
	// client is not interested
	if in.FilterRegistered {
		for _, repo := range repos {
			if repo.WebhookUrl == "" {
				results = append(results, &pb.Repositories{
					Owner:  repo.RepoOwner,
					Name:   repo.RepoName,
					RepoId: repo.RepoID,
				})
			}
		}
	} else {
		for _, repo := range repos {
			results = append(results, &pb.Repositories{
				Owner:  repo.RepoOwner,
				Name:   repo.RepoName,
				RepoId: repo.RepoID,
			})
		}
	}

	resp.Results = results

	return &resp, nil
}
