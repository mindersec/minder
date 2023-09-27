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
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/mediator/internal/auth"
	"github.com/stacklok/mediator/internal/db"
	"github.com/stacklok/mediator/internal/gh/queries"
	github "github.com/stacklok/mediator/internal/providers/github"
	"github.com/stacklok/mediator/internal/reconcilers"
	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
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
//
// nolint: gocyclo
func (s *Server) RegisterRepository(ctx context.Context,
	in *pb.RegisterRepositoryRequest) (*pb.RegisterRepositoryResponse, error) {
	// if we have set no events, give an error
	if len(in.Events) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "no events provided")
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
	if err := AuthorizedOnGroup(ctx, in.GroupId); err != nil {
		return nil, err
	}

	provider, err := s.store.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name:    in.GetProvider(),
		GroupID: in.GetGroupId()})
	if err != nil {
		return nil, providerError(fmt.Errorf("provider error: %w", err))
	}

	// Check if needs github authorization
	isGithubAuthorized := s.IsProviderCallAuthorized(ctx, provider, in.GroupId)
	if !isGithubAuthorized {
		return nil, status.Errorf(codes.PermissionDenied, "user not authorized to interact with provider")
	}

	decryptedToken, _, err := s.GetProviderAccessToken(ctx, provider.Name, in.GroupId, true)

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
			Provider:   provider.Name,
			GroupID:    in.GroupId,
			RepoOwner:  result.Owner,
			RepoName:   result.Repository,
			RepoID:     result.RepoID,
			DeployUrl:  result.DeployURL,
		})
		if err != nil {
			return nil, err
		}

		// publish a reconcile event for the registered repositories
		log.Printf("publishing register event for repository: %s", result.Repository)

		msg, err := reconcilers.NewRepoReconcilerMessage(in.Provider, result.RepoID, in.GroupId)
		if err != nil {
			log.Printf("error creating reconciler event: %v", err)
			continue
		}

		// This is a non-fatal error, so we'll just log it and continue with the next ones
		if err := s.evt.Publish(reconcilers.InternalReconcilerEventTopic, msg); err != nil {
			log.Printf("error publishing reconciler event: %v", err)
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
	// if we do not have a group, check if we can infer it
	if in.GroupId == 0 {
		group, err := auth.GetDefaultGroup(ctx)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "cannot infer group id")
		}
		in.GroupId = group
	}

	// check if user is authorized
	if err := AuthorizedOnGroup(ctx, in.GroupId); err != nil {
		return nil, err
	}

	provider, err := s.store.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name:    in.GetProvider(),
		GroupID: in.GetGroupId()})
	if err != nil {
		return nil, providerError(fmt.Errorf("provider error: %w", err))
	}

	repos, err := s.store.ListRepositoriesByGroupID(ctx, db.ListRepositoriesByGroupIDParams{
		Provider: provider.Name,
		GroupID:  in.GroupId,
		Limit:    in.Limit,
		Offset:   in.Offset,
	})

	if err != nil {
		return nil, err
	}

	var resp pb.ListRepositoriesResponse
	var results []*pb.RepositoryRecord

	var filterCondition func(*db.Repository) bool

	switch in.Filter {
	case pb.RepoFilter_REPO_FILTER_SHOW_UNSPECIFIED:
		return nil, status.Errorf(codes.InvalidArgument, "filter not specified")
	case pb.RepoFilter_REPO_FILTER_SHOW_ALL:
		filterCondition = func(_ *db.Repository) bool {
			return true
		}
	case pb.RepoFilter_REPO_FILTER_SHOW_NOT_REGISTERED_ONLY:
		filterCondition = func(repo *db.Repository) bool {
			return repo.WebhookUrl == ""
		}
	case pb.RepoFilter_REPO_FILTER_SHOW_REGISTERED_ONLY:
		filterCondition = func(repo *db.Repository) bool {
			return repo.WebhookUrl != ""
		}
	}

	for _, repo := range repos {
		repo := repo

		if filterCondition(&repo) {
			results = append(results, &pb.RepositoryRecord{
				Id:        repo.ID.String(),
				Provider:  provider.Name,
				GroupId:   repo.GroupID,
				Owner:     repo.RepoOwner,
				Name:      repo.RepoName,
				RepoId:    repo.RepoID,
				IsPrivate: repo.IsPrivate,
				IsFork:    repo.IsFork,
				HookUrl:   repo.WebhookUrl,
				DeployUrl: repo.DeployUrl,
				CloneUrl:  repo.CloneUrl,
				CreatedAt: timestamppb.New(repo.CreatedAt),
				UpdatedAt: timestamppb.New(repo.UpdatedAt),
			})
		}
	}

	resp.Results = results

	return &resp, nil
}

// GetRepositoryById returns a repository for a given repository id
func (s *Server) GetRepositoryById(ctx context.Context,
	in *pb.GetRepositoryByIdRequest) (*pb.GetRepositoryByIdResponse, error) {
	parsedRepositoryID, err := uuid.Parse(in.RepositoryId)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid repository ID")
	}

	// read the repository
	repo, err := s.store.GetRepositoryByID(ctx, parsedRepositoryID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Errorf(codes.NotFound, "repository not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot read repository: %v", err)
	}

	// check if user is authorized
	if err := AuthorizedOnGroup(ctx, repo.GroupID); err != nil {
		return nil, err
	}

	provider, err := s.store.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name:    repo.Provider,
		GroupID: repo.GroupID,
	})
	if err != nil {
		return nil, providerError(fmt.Errorf("provider error: %w", err))
	}

	createdAt := timestamppb.New(repo.CreatedAt)
	updatedat := timestamppb.New(repo.UpdatedAt)

	return &pb.GetRepositoryByIdResponse{Repository: &pb.RepositoryRecord{
		Id:        repo.ID.String(),
		Provider:  provider.Name,
		GroupId:   repo.GroupID,
		Owner:     repo.RepoOwner,
		Name:      repo.RepoName,
		RepoId:    repo.RepoID,
		IsPrivate: repo.IsPrivate,
		IsFork:    repo.IsFork,
		HookUrl:   repo.WebhookUrl,
		DeployUrl: repo.DeployUrl,
		CloneUrl:  repo.CloneUrl,
		CreatedAt: createdAt,
		UpdatedAt: updatedat,
	}}, nil
}

// GetRepositoryByName returns information about a repository.
// This function will typically be called by the client to get a
// repository which is already registered and present in the mediator database
// The API is called with a group id
func (s *Server) GetRepositoryByName(ctx context.Context,
	in *pb.GetRepositoryByNameRequest) (*pb.GetRepositoryByNameResponse, error) {
	// split repo name in owner and name
	fragments := strings.Split(in.Name, "/")
	if len(fragments) != 2 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid repository name, needs to have the format: owner/name")
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
	if err := AuthorizedOnGroup(ctx, in.GroupId); err != nil {
		return nil, err
	}

	provider, err := s.store.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name:    in.Provider,
		GroupID: in.GroupId,
	})
	if err != nil {
		return nil, providerError(fmt.Errorf("provider error: %w", err))
	}

	repo, err := s.store.GetRepositoryByRepoName(ctx,
		db.GetRepositoryByRepoNameParams{Provider: provider.Name, RepoOwner: fragments[0], RepoName: fragments[1]})

	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Errorf(codes.NotFound, "repository not found")
	} else if err != nil {
		return nil, err
	}
	// check if user is authorized
	if err := AuthorizedOnGroup(ctx, repo.GroupID); err != nil {
		return nil, err
	}

	createdAt := timestamppb.New(repo.CreatedAt)
	updatedat := timestamppb.New(repo.UpdatedAt)

	return &pb.GetRepositoryByNameResponse{Repository: &pb.RepositoryRecord{
		Id:        repo.ID.String(),
		Provider:  provider.Name,
		GroupId:   repo.GroupID,
		Owner:     repo.RepoOwner,
		Name:      repo.RepoName,
		RepoId:    repo.RepoID,
		IsPrivate: repo.IsPrivate,
		IsFork:    repo.IsFork,
		HookUrl:   repo.WebhookUrl,
		DeployUrl: repo.DeployUrl,
		CloneUrl:  repo.CloneUrl,
		CreatedAt: createdAt,
		UpdatedAt: updatedat,
	}}, nil
}

// SyncRepositories synchronizes the repositories for a given provider and group
func (s *Server) SyncRepositories(ctx context.Context, in *pb.SyncRepositoriesRequest) (*pb.SyncRepositoriesResponse, error) {
	// if we do not have a group, check if we can infer it
	if in.GroupId == 0 {
		group, err := auth.GetDefaultGroup(ctx)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "cannot infer group id")
		}
		in.GroupId = group
	}

	// check if user is authorized
	if err := AuthorizedOnGroup(ctx, in.GroupId); err != nil {
		return nil, err
	}

	provider, err := s.store.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name:    in.Provider,
		GroupID: in.GroupId,
	})
	if err != nil {
		return nil, providerError(fmt.Errorf("provider error: %w", err))
	}

	// Check if needs github authorization
	isGithubAuthorized := s.IsProviderCallAuthorized(ctx, provider, in.GroupId)
	if !isGithubAuthorized {
		return nil, status.Errorf(codes.PermissionDenied, "user not authorized to interact with provider")
	}

	token, owner_filter, err := s.GetProviderAccessToken(ctx, provider.Name, in.GroupId, true)

	if err != nil {
		return nil, status.Errorf(codes.PermissionDenied, "cannot get access token for provider")
	}

	// Populate the database with the repositories using the GraphQL API
	client, err := github.NewRestClient(ctx, &pb.GitHubProviderConfig{}, token.AccessToken, owner_filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot create github client: %v", err)
	}

	tmoutCtx, cancel := context.WithTimeout(ctx, github.ExpensiveRestCallTimeout)
	defer cancel()

	isOrg := (owner_filter != "")
	repos, err := client.ListAllRepositories(tmoutCtx, isOrg, owner_filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot list repositories: %v", err)
	}

	// // Insert the repositories into the database
	// This uses the context with the extended timeout to allow for the
	// database to be populated with the repositories. Otherwise the original context
	// expires and the database insertions are cancelled.
	err = queries.SyncRepositoriesWithDB(tmoutCtx, s.store, repos, provider.Name, in.GroupId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot sync repositories: %v", err)
	}

	return &pb.SyncRepositoriesResponse{}, nil
}
