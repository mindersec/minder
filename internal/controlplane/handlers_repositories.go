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

	"github.com/stacklok/mediator/internal/db"
	"github.com/stacklok/mediator/internal/providers"
	github "github.com/stacklok/mediator/internal/providers/github"
	"github.com/stacklok/mediator/internal/reconcilers"
	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/minder/v1"
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
	projectID, err := getProjectFromRequestOrDefault(ctx, in)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, err.Error())
	}

	// check if user is authorized
	if err := AuthorizedOnProject(ctx, projectID); err != nil {
		return nil, err
	}

	provider, err := s.store.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name:      in.GetProvider(),
		ProjectID: projectID})
	if err != nil {
		return nil, providerError(fmt.Errorf("provider error: %w", err))
	}

	p, err := providers.GetProviderBuilder(ctx, provider, projectID, s.store, s.cryptoEngine)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get provider builder: %v", err)
	}

	// Unmarshal the in.GetRepositories() into a struct Repository
	var upstreamRepos []UpstreamRepositoryReference
	if in.GetRepositories() == nil || len(in.GetRepositories()) <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "no repositories provided")
	}

	for _, repository := range in.GetRepositories() {
		upstreamRepos = append(upstreamRepos, UpstreamRepositoryReference{
			Owner:      repository.GetOwner(),
			Name:       repository.GetName(),
			UpstreamID: repository.GetRepoId(), // Handle the RepoID here.
		})
	}

	allEvents := []string{"*"}
	resultData, err := s.registerWebhookForRepository(
		ctx, p, projectID, upstreamRepos, allEvents)
	if err != nil {
		return nil, err
	}

	for idx := range resultData {
		result := resultData[idx]
		r := result.Repository

		// Convert each result to a pb.Repository object
		if result.Status.Error != nil {
			continue
		}

		// update the database
		dbRepo, err := s.store.CreateRepository(ctx, db.CreateRepositoryParams{
			Provider:  provider.Name,
			ProjectID: projectID,
			RepoOwner: r.Owner,
			RepoName:  r.Name,
			RepoID:    r.RepoId,
			IsPrivate: r.IsPrivate,
			IsFork:    r.IsFork,
			WebhookID: sql.NullInt32{
				Int32: int32(r.HookId),
				Valid: true,
			},
			CloneUrl:   r.CloneUrl,
			WebhookUrl: r.HookUrl,
			DeployUrl:  r.DeployUrl,
		})
		// even if we set the webhook, if we couldn't create it in the database, we'll return an error
		if err != nil {
			log.Printf("error creating repository '%s/%s' in database: %v", r.Owner, r.Name, err)

			result.Status.Success = false
			errorStr := "error creating repository in database"
			result.Status.Error = &errorStr
			continue
		}

		repoDBID := dbRepo.ID.String()
		r.Id = &repoDBID

		// publish a reconcile event for the registered repositories
		log.Printf("publishing register event for repository: %s/%s", r.Owner, r.Name)

		msg, err := reconcilers.NewRepoReconcilerMessage(in.Provider, r.RepoId, projectID)
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
		Results: resultData,
	}

	return response, nil
}

// ListRepositories returns a list of repositories for a given group
// This function will typically be called by the client to get a list of
// repositories that are registered present in the mediator database
// The API is called with a group id, limit and offset
func (s *Server) ListRepositories(ctx context.Context,
	in *pb.ListRepositoriesRequest) (*pb.ListRepositoriesResponse, error) {
	projectID, err := getProjectFromRequestOrDefault(ctx, in)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, err.Error())
	}

	// check if user is authorized
	if err := AuthorizedOnProject(ctx, projectID); err != nil {
		return nil, err
	}

	provider, err := s.store.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name:      in.GetProvider(),
		ProjectID: projectID})
	if err != nil {
		return nil, providerError(fmt.Errorf("provider error: %w", err))
	}

	repos, err := s.store.ListRepositoriesByProjectID(ctx, db.ListRepositoriesByProjectIDParams{
		Provider:  provider.Name,
		ProjectID: projectID,
	})

	if err != nil {
		return nil, err
	}

	var resp pb.ListRepositoriesResponse
	var results []*pb.Repository

	for _, repo := range repos {
		repo := repo

		id := repo.ID.String()
		projID := repo.ProjectID.String()
		results = append(results, &pb.Repository{
			Id: &id,
			Context: &pb.Context{
				Project:  &projID,
				Provider: repo.Provider,
			},
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
	if err := AuthorizedOnProject(ctx, repo.ProjectID); err != nil {
		return nil, err
	}

	createdAt := timestamppb.New(repo.CreatedAt)
	updatedat := timestamppb.New(repo.UpdatedAt)

	id := repo.ID.String()
	projID := repo.ProjectID.String()
	return &pb.GetRepositoryByIdResponse{Repository: &pb.Repository{
		Id: &id,
		Context: &pb.Context{
			Project:  &projID,
			Provider: repo.Provider,
		},
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

	projectID, err := getProjectFromRequestOrDefault(ctx, in)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, err.Error())
	}

	// check if user is authorized
	if err := AuthorizedOnProject(ctx, projectID); err != nil {
		return nil, err
	}

	provider, err := s.store.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name:      in.Provider,
		ProjectID: projectID,
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
	if err := AuthorizedOnProject(ctx, repo.ProjectID); err != nil {
		return nil, err
	}

	createdAt := timestamppb.New(repo.CreatedAt)
	updatedat := timestamppb.New(repo.UpdatedAt)

	id := repo.ID.String()
	projID := repo.ProjectID.String()
	return &pb.GetRepositoryByNameResponse{Repository: &pb.Repository{
		Id: &id,
		Context: &pb.Context{
			Project:  &projID,
			Provider: repo.Provider,
		},
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

// DeleteRepositoryById deletes a repository by name
func (s *Server) DeleteRepositoryById(ctx context.Context,
	in *pb.DeleteRepositoryByIdRequest) (*pb.DeleteRepositoryByIdResponse, error) {
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
	if err := AuthorizedOnProject(ctx, repo.ProjectID); err != nil {
		return nil, err
	}

	// delete the repository
	if err := s.store.DeleteRepository(ctx, repo.ID); err != nil {
		return nil, err
	}

	// return the response with the id of the deleted repository
	return &pb.DeleteRepositoryByIdResponse{
		RepositoryId: in.RepositoryId,
	}, nil
}

// DeleteRepositoryByName deletes a repository by name
func (s *Server) DeleteRepositoryByName(ctx context.Context,
	in *pb.DeleteRepositoryByNameRequest) (*pb.DeleteRepositoryByNameResponse, error) {
	// split repo name in owner and name
	fragments := strings.Split(in.Name, "/")
	if len(fragments) != 2 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid repository name, needs to have the format: owner/name")
	}

	projectID, err := getProjectFromRequestOrDefault(ctx, in)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, err.Error())
	}

	// check if user is authorized
	if err := AuthorizedOnProject(ctx, projectID); err != nil {
		return nil, err
	}

	provider, err := s.store.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name:      in.Provider,
		ProjectID: projectID,
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
	if err := AuthorizedOnProject(ctx, repo.ProjectID); err != nil {
		return nil, err
	}

	// delete the repository
	if err := s.store.DeleteRepository(ctx, repo.ID); err != nil {
		return nil, err
	}

	// return the response with the name of the deleted repository
	return &pb.DeleteRepositoryByNameResponse{
		Name: in.Name,
	}, nil
}

// ListRemoteRepositoriesFromProvider returns a list of repositories from a provider
func (s *Server) ListRemoteRepositoriesFromProvider(
	ctx context.Context,
	in *pb.ListRemoteRepositoriesFromProviderRequest,
) (*pb.ListRemoteRepositoriesFromProviderResponse, error) {
	projectID, err := getProjectFromRequestOrDefault(ctx, in)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, err.Error())
	}

	// check if user is authorized
	if err := AuthorizedOnProject(ctx, projectID); err != nil {
		return nil, err
	}

	provider, err := s.store.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name:      in.Provider,
		ProjectID: projectID,
	})
	if err != nil {
		return nil, providerError(fmt.Errorf("provider error: %w", err))
	}

	// FIXME: this is a hack to get the owner filter from the request
	_, owner_filter, err := s.GetProviderAccessToken(ctx, provider.Name, projectID, true)

	if err != nil {
		return nil, status.Errorf(codes.PermissionDenied, "cannot get access token for provider")
	}

	p, err := providers.GetProviderBuilder(ctx, provider, projectID, s.store, s.cryptoEngine)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get provider builder: %v", err)
	}

	if !p.Implements(db.ProviderTypeRepoLister) {
		return nil, util.UserVisibleError(codes.Unimplemented, "provider does not implement repository listing")
	}

	client, err := p.GetRepoLister(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot create github client: %v", err)
	}

	tmoutCtx, cancel := context.WithTimeout(ctx, github.ExpensiveRestCallTimeout)
	defer cancel()

	var remoteRepos []*pb.Repository
	isOrg := (owner_filter != "")
	if isOrg {
		remoteRepos, err = client.ListOrganizationRepsitories(tmoutCtx, owner_filter)
		if err != nil {
			return nil, util.UserVisibleError(codes.Internal, "cannot list repositories: %v", err)
		}
	} else {
		remoteRepos, err = client.ListUserRepositories(tmoutCtx, owner_filter)
		if err != nil {
			return nil, util.UserVisibleError(codes.Internal, "cannot list repositories: %v", err)
		}
	}

	out := &pb.ListRemoteRepositoriesFromProviderResponse{
		Results: make([]*pb.UpstreamRepositoryRef, 0, len(remoteRepos)),
	}

	for idx, rem := range remoteRepos {
		// Skip private repositories
		if rem.IsPrivate && !projectAllowsPrivateRepos(ctx, s.store, projectID) {
			continue
		}
		remoteRepo := remoteRepos[idx]
		repo := &pb.UpstreamRepositoryRef{
			Owner:  remoteRepo.Owner,
			Name:   remoteRepo.Name,
			RepoId: remoteRepo.RepoId,
		}
		out.Results = append(out.Results, repo)
	}

	return out, nil
}
