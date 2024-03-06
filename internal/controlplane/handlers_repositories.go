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
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/providers"
	github "github.com/stacklok/minder/internal/providers/github"
	"github.com/stacklok/minder/internal/reconcilers"
	"github.com/stacklok/minder/internal/util"
	cursorutil "github.com/stacklok/minder/internal/util/cursor"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// maxFetchLimit is the maximum number of repositories that can be fetched from the database in one call
const maxFetchLimit = 100

// RegisterRepository adds repositories to the database and registers a webhook
// Once a user had enrolled in a project (they have a valid token), they can register
// repositories to be monitored by the minder by provisioning a webhook on the
// repository(ies).
func (s *Server) RegisterRepository(ctx context.Context,
	in *pb.RegisterRepositoryRequest) (*pb.RegisterRepositoryResponse, error) {
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	provider, err := getProviderFromRequestOrDefault(ctx, s.store, in, projectID)
	if err != nil {
		return nil, providerError(err)
	}

	pbOpts := []providers.ProviderBuilderOption{
		providers.WithProviderMetrics(s.provMt),
		providers.WithRestClientCache(s.restClientCache),
	}
	p, err := providers.GetProviderBuilder(ctx, provider, projectID, s.store, s.cryptoEngine, pbOpts...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get provider builder: %v", err)
	}

	// Unmarshal the in.GetRepositories() into a struct Repository
	if in.GetRepository() == nil || in.GetRepository().Name == "" {
		return nil, util.UserVisibleError(codes.InvalidArgument, "no repository provided")
	}

	repo := in.GetRepository()

	result, err := s.registerWebhookForRepository(ctx, p, projectID, repo)
	if err != nil {
		return nil, util.UserVisibleError(codes.Internal, "cannot register webhook: %v", err)
	}

	r := result.Repository

	response := &pb.RegisterRepositoryResponse{
		Result: result,
	}

	// Convert each result to a pb.Repository object
	if result.Status.Error != nil {
		return response, nil
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
		WebhookID: sql.NullInt64{
			Int64: r.HookId,
			Valid: true,
		},
		CloneUrl:   r.CloneUrl,
		WebhookUrl: r.HookUrl,
		DeployUrl:  r.DeployUrl,
		DefaultBranch: sql.NullString{
			String: r.DefaultBranch,
			Valid:  true,
		},
	})
	// even if we set the webhook, if we couldn't create it in the database, we'll return an error
	if err != nil {
		log.Printf("error creating repository '%s/%s' in database: %v", r.Owner, r.Name, err)

		result.Status.Success = false
		errorStr := "error creating repository in database"
		result.Status.Error = &errorStr
		return response, nil
	}

	repoDBID := dbRepo.ID.String()
	r.Id = &repoDBID

	// publish a reconciling event for the registered repositories
	log.Printf("publishing register event for repository: %s/%s", r.Owner, r.Name)

	msg, err := reconcilers.NewRepoReconcilerMessage(provider.Name, r.RepoId, projectID)
	if err != nil {
		log.Printf("error creating reconciler event: %v", err)
		return response, nil
	}

	// This is a non-fatal error, so we'll just log it and continue with the next ones
	if err := s.evt.Publish(reconcilers.InternalReconcilerEventTopic, msg); err != nil {
		log.Printf("error publishing reconciler event: %v", err)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = provider.Name
	logger.BusinessRecord(ctx).Project = projectID
	logger.BusinessRecord(ctx).Repository = dbRepo.ID

	return response, nil
}

// ListRepositories returns a list of repositories for a given project
// This function will typically be called by the client to get a list of
// repositories that are registered present in the minder database
func (s *Server) ListRepositories(ctx context.Context,
	in *pb.ListRepositoriesRequest) (*pb.ListRepositoriesResponse, error) {
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	provider, err := getProviderFromRequestOrDefault(ctx, s.store, in, projectID)
	if err != nil {
		return nil, providerError(err)
	}

	reqRepoCursor, err := cursorutil.NewRepoCursor(in.GetCursor())
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, err.Error())
	}

	repoId := sql.NullInt64{}
	if reqRepoCursor.ProjectId == projectID.String() && reqRepoCursor.Provider == provider.Name {
		repoId = sql.NullInt64{Valid: true, Int64: reqRepoCursor.RepoId}
	}

	limit := sql.NullInt32{Valid: false, Int32: 0}
	reqLimit := in.GetLimit()
	if reqLimit > 0 {
		if reqLimit > maxFetchLimit {
			return nil, util.UserVisibleError(codes.InvalidArgument, "limit too high, max is %d", maxFetchLimit)
		}
		limit = sql.NullInt32{Valid: true, Int32: reqLimit + 1}
	}

	repos, err := s.store.ListRepositoriesByProjectID(ctx, db.ListRepositoriesByProjectIDParams{
		Provider:  provider.Name,
		ProjectID: projectID,
		RepoID:    repoId,
		Limit:     limit,
	})

	if err != nil {
		return nil, err
	}

	var resp pb.ListRepositoriesResponse
	var results []*pb.Repository

	for _, repo := range repos {
		repo := repo

		projID := repo.ProjectID.String()
		r := util.PBRepositoryFromDB(repo)
		r.Context = &pb.Context{
			Project:  &projID,
			Provider: &repo.Provider,
		}
		results = append(results, r)
	}

	var respRepoCursor *cursorutil.RepoCursor
	if limit.Valid && len(repos) == int(limit.Int32) {
		lastRepo := repos[len(repos)-1]
		respRepoCursor = &cursorutil.RepoCursor{
			ProjectId: projectID.String(),
			Provider:  provider.Name,
			RepoId:    lastRepo.RepoID,
		}

		// remove the (limit + 1)th element from the results
		results = results[:len(results)-1]
	}

	resp.Results = results
	resp.Cursor = respRepoCursor.String()

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = provider.Name
	logger.BusinessRecord(ctx).Project = projectID

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

	projID := repo.ProjectID.String()
	r := util.PBRepositoryFromDB(repo)
	r.Context = &pb.Context{
		Project:  &projID,
		Provider: &repo.Provider,
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = repo.Provider
	logger.BusinessRecord(ctx).Project = repo.ProjectID
	logger.BusinessRecord(ctx).Repository = repo.ID

	return &pb.GetRepositoryByIdResponse{Repository: r}, nil
}

// GetRepositoryByName returns information about a repository.
// This function will typically be called by the client to get a
// repository which is already registered and present in the minder database
// The API is called with a project id
func (s *Server) GetRepositoryByName(ctx context.Context,
	in *pb.GetRepositoryByNameRequest) (*pb.GetRepositoryByNameResponse, error) {
	// split repo name in owner and name
	fragments := strings.Split(in.Name, "/")
	if len(fragments) != 2 {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid repository name, needs to have the format: owner/name")
	}

	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	provider, err := getProviderFromRequestOrDefault(ctx, s.store, in, projectID)
	if err != nil {
		return nil, providerError(err)
	}

	repo, err := s.store.GetRepositoryByRepoName(ctx, db.GetRepositoryByRepoNameParams{
		Provider:  provider.Name,
		RepoOwner: fragments[0],
		RepoName:  fragments[1],
		ProjectID: projectID,
	})

	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Errorf(codes.NotFound, "repository not found")
	} else if err != nil {
		return nil, err
	}

	projID := repo.ProjectID.String()
	r := util.PBRepositoryFromDB(repo)
	r.Context = &pb.Context{
		Project:  &projID,
		Provider: &repo.Provider,
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = repo.Provider
	logger.BusinessRecord(ctx).Project = repo.ProjectID
	logger.BusinessRecord(ctx).Repository = repo.ID

	return &pb.GetRepositoryByNameResponse{Repository: r}, nil
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

	provider, err := getProviderFromRequestOrDefault(ctx, s.store, in, repo.ProjectID)
	if err != nil {
		return nil, providerError(err)
	}

	err = s.deleteRepositoryAndWebhook(ctx, repo, repo.ProjectID, provider)
	if err != nil {
		return nil, err
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = repo.Provider
	logger.BusinessRecord(ctx).Project = repo.ProjectID
	logger.BusinessRecord(ctx).Repository = repo.ID

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
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid repository name, needs to have the format: owner/name")
	}

	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	provider, err := getProviderFromRequestOrDefault(ctx, s.store, in, projectID)
	if err != nil {
		return nil, providerError(err)
	}

	repo, err := s.store.GetRepositoryByRepoName(ctx, db.GetRepositoryByRepoNameParams{
		Provider:  provider.Name,
		RepoOwner: fragments[0],
		RepoName:  fragments[1],
		ProjectID: projectID,
	})

	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Errorf(codes.NotFound, "repository not found")
	} else if err != nil {
		return nil, err
	}
	err = s.deleteRepositoryAndWebhook(ctx, repo, projectID, provider)
	if err != nil {
		return nil, err
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = repo.Provider
	logger.BusinessRecord(ctx).Project = repo.ProjectID
	logger.BusinessRecord(ctx).Repository = repo.ID

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
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	provider, err := getProviderFromRequestOrDefault(ctx, s.store, in, projectID)
	if err != nil {
		return nil, providerError(err)
	}

	zerolog.Ctx(ctx).Debug().
		Str("provider", provider.Name).
		Str("projectID", projectID.String()).
		Msg("listing repositories")

	// FIXME: this is a hack to get the owner filter from the request
	_, owner_filter, err := s.getProviderAccessToken(ctx, provider.Name, projectID)

	if err != nil {
		return nil, util.UserVisibleError(codes.PermissionDenied,
			"cannot get access token for provider: did you run `minder provider enroll`?")
	}

	pbOpts := []providers.ProviderBuilderOption{
		providers.WithProviderMetrics(s.provMt),
		providers.WithRestClientCache(s.restClientCache),
	}
	p, err := providers.GetProviderBuilder(ctx, provider, projectID, s.store, s.cryptoEngine, pbOpts...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get provider builder: %v", err)
	}

	if !p.Implements(db.ProviderTypeRepoLister) {
		return nil, util.UserVisibleError(codes.Unimplemented, "provider does not implement repository listing")
	}

	client, err := p.GetRepoLister()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot create github client: %v", err)
	}

	tmoutCtx, cancel := context.WithTimeout(ctx, github.ExpensiveRestCallTimeout)
	defer cancel()

	var remoteRepos []*pb.Repository
	isOrg := (owner_filter != "")
	if isOrg {
		zerolog.Ctx(ctx).Debug().Msgf("listing repositories for organization")
		remoteRepos, err = client.ListOrganizationRepsitories(tmoutCtx, owner_filter)
		if err != nil {
			return nil, util.UserVisibleError(codes.Internal, "cannot list repositories: %v", err)
		}
	} else {
		zerolog.Ctx(ctx).Debug().Msgf("listing repositories for the user")
		remoteRepos, err = client.ListUserRepositories(tmoutCtx, owner_filter)
		if err != nil {
			return nil, util.UserVisibleError(codes.Internal, "cannot list repositories: %v", err)
		}
	}

	out := &pb.ListRemoteRepositoriesFromProviderResponse{
		Results: make([]*pb.UpstreamRepositoryRef, 0, len(remoteRepos)),
	}

	allowsPrivateRepos := projectAllowsPrivateRepos(ctx, s.store, projectID)
	if !allowsPrivateRepos {
		zerolog.Ctx(ctx).Info().Msg("filtering out private repositories")
	} else {
		zerolog.Ctx(ctx).Info().Msg("including private repositories")
	}

	for idx, rem := range remoteRepos {
		// Skip private repositories
		if rem.IsPrivate && !allowsPrivateRepos {
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

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = provider.Name
	logger.BusinessRecord(ctx).Project = projectID

	return out, nil
}

func (s *Server) deleteRepositoryAndWebhook(
	ctx context.Context,
	repo db.Repository,
	projectID uuid.UUID,
	provider db.Provider,
) error {
	tx, err := s.store.BeginTransaction()
	if err != nil {
		return status.Errorf(codes.Internal, "error deleting repository")
	}
	defer s.store.Rollback(tx)

	qtx := s.store.GetQuerierWithTransaction(tx)
	if err := qtx.DeleteRepository(ctx, repo.ID); err != nil {
		return status.Errorf(codes.Internal, "error deleting repository: %v", err)
	}

	if err := s.deleteWebhookFromRepository(ctx, provider, projectID, repo); err != nil {
		return err
	}

	if err := s.store.Commit(tx); err != nil {
		return status.Errorf(codes.Internal, "error deleting repository")
	}

	return nil
}
