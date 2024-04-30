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
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/projects/features"
	"github.com/stacklok/minder/internal/providers"
	"github.com/stacklok/minder/internal/providers/github"
	"github.com/stacklok/minder/internal/repositories"
	ghrepo "github.com/stacklok/minder/internal/repositories/github"
	"github.com/stacklok/minder/internal/util"
	cursorutil "github.com/stacklok/minder/internal/util/cursor"
	"github.com/stacklok/minder/internal/util/ptr"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	v1 "github.com/stacklok/minder/pkg/providers/v1"
)

// maxFetchLimit is the maximum number of repositories that can be fetched from the database in one call
const maxFetchLimit = 100

// RegisterRepository adds repositories to the database and registers a webhook
// Once a user had enrolled in a project (they have a valid token), they can register
// repositories to be monitored by the minder by provisioning a webhook on the
// repository(ies).
func (s *Server) RegisterRepository(
	ctx context.Context,
	in *pb.RegisterRepositoryRequest,
) (*pb.RegisterRepositoryResponse, error) {
	projectID := getProjectID(ctx)
	providerName := getProviderName(ctx)

	// Validate that the Repository struct in the request
	githubRepo := in.GetRepository()
	// If the repo owner is missing, GitHub will assume a default value based
	// on the user's credentials. An explicit check for owner is left out to
	// avoid breaking backwards compatibility.
	if githubRepo.GetName() == "" {
		return nil, util.UserVisibleError(codes.InvalidArgument, "missing repository name")
	}

	l := zerolog.Ctx(ctx).With().
		Str("repoName", githubRepo.GetName()).
		Str("repoOwner", githubRepo.GetOwner()).
		Str("projectID", projectID.String()).
		Logger()
	ctx = l.WithContext(ctx)

	provider, err := s.inferProviderByOwner(ctx, githubRepo.GetOwner(), projectID, providerName)
	if err != nil {
		pErr := providers.ErrProviderNotFoundBy{}
		if errors.As(err, &pErr) {
			return nil, util.UserVisibleError(codes.NotFound, "no suitable provider found, please enroll a provider")
		}
		return nil, status.Errorf(codes.Internal, "cannot get provider: %v", err)
	}

	newRepo, err := s.repos.CreateRepository(ctx, provider, projectID, githubRepo.GetOwner(), githubRepo.GetName())
	if err != nil {
		if errors.Is(err, ghrepo.ErrPrivateRepoForbidden) || errors.Is(err, ghrepo.ErrArchivedRepoForbidden) {
			return nil, util.UserVisibleError(codes.InvalidArgument, err.Error())
		}
		return nil, util.UserVisibleError(codes.Internal, "unable to register repository: %v", err)
	}

	return &pb.RegisterRepositoryResponse{
		Result: &pb.RegisterRepoResult{
			Status: &pb.RegisterRepoResult_Status{
				Success: true,
			},
			Repository: newRepo,
		},
	}, nil
}

// ListRepositories returns a list of repositories for a given project
// This function will typically be called by the client to get a list of
// repositories that are registered present in the minder database
func (s *Server) ListRepositories(ctx context.Context,
	in *pb.ListRepositoriesRequest) (*pb.ListRepositoriesResponse, error) {
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID
	providerName := entityCtx.Provider.Name

	logger.BusinessRecord(ctx).Provider = providerName
	logger.BusinessRecord(ctx).Project = projectID

	providerFilter := getNameFilterParam(providerName)

	reqRepoCursor, err := cursorutil.NewRepoCursor(in.GetCursor())
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, err.Error())
	}

	repoId := sql.NullInt64{}
	if reqRepoCursor.ProjectId == projectID.String() && reqRepoCursor.Provider == providerName {
		repoId = sql.NullInt64{Valid: true, Int64: reqRepoCursor.RepoId}
	}

	limit := sql.NullInt64{Valid: false, Int64: 0}
	reqLimit := in.GetLimit()
	if reqLimit > 0 {
		if reqLimit > maxFetchLimit {
			return nil, util.UserVisibleError(codes.InvalidArgument, "limit too high, max is %d", maxFetchLimit)
		}
		limit = sql.NullInt64{Valid: true, Int64: reqLimit + 1}
	}

	repos, err := s.store.ListRepositoriesByProjectID(ctx, db.ListRepositoriesByProjectIDParams{
		Provider:  providerFilter,
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
		projID := repo.ProjectID.String()
		r := repositories.PBRepositoryFromDB(repo)
		r.Context = &pb.Context{
			Project:  &projID,
			Provider: &repo.Provider,
		}
		results = append(results, r)
	}

	var respRepoCursor *cursorutil.RepoCursor
	if limit.Valid && int64(len(repos)) == limit.Int64 {
		lastRepo := repos[len(repos)-1]
		respRepoCursor = &cursorutil.RepoCursor{
			ProjectId: projectID.String(),
			Provider:  providerName,
			RepoId:    lastRepo.RepoID,
		}

		// remove the (limit + 1)th element from the results
		results = results[:len(results)-1]
	}

	resp.Results = results
	resp.Cursor = respRepoCursor.String()

	return &resp, nil
}

// GetRepositoryById returns a repository for a given repository id
func (s *Server) GetRepositoryById(ctx context.Context,
	in *pb.GetRepositoryByIdRequest) (*pb.GetRepositoryByIdResponse, error) {
	parsedRepositoryID, err := uuid.Parse(in.RepositoryId)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid repository ID")
	}
	projectID := getProjectID(ctx)

	// read the repository
	repo, err := s.store.GetRepositoryByIDAndProject(ctx, db.GetRepositoryByIDAndProjectParams{
		ID:        parsedRepositoryID,
		ProjectID: projectID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Errorf(codes.NotFound, "repository not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot read repository: %v", err)
	}

	projID := repo.ProjectID.String()
	r := repositories.PBRepositoryFromDB(repo)
	r.Context = &pb.Context{
		Project:  &projID,
		Provider: &repo.Provider,
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).ProviderID = repo.ProviderID
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

	// TODO: move this lookup logic out of the controlplane
	providerFilter := getNameFilterParam(entityCtx.Provider.Name)
	repo, err := s.store.GetRepositoryByRepoName(ctx, db.GetRepositoryByRepoNameParams{
		Provider:  providerFilter,
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
	r := repositories.PBRepositoryFromDB(repo)
	r.Context = &pb.Context{
		Project:  &projID,
		Provider: &repo.Provider,
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).ProviderID = repo.ProviderID
	logger.BusinessRecord(ctx).Project = repo.ProjectID
	logger.BusinessRecord(ctx).Repository = repo.ID

	return &pb.GetRepositoryByNameResponse{Repository: r}, nil
}

// DeleteRepositoryById deletes a repository by its UUID
func (s *Server) DeleteRepositoryById(
	ctx context.Context,
	in *pb.DeleteRepositoryByIdRequest,
) (*pb.DeleteRepositoryByIdResponse, error) {
	parsedRepositoryID, err := uuid.Parse(in.RepositoryId)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid repository ID")
	}

	projectID := getProjectID(ctx)

	err = s.repos.DeleteByID(ctx, parsedRepositoryID, projectID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Errorf(codes.NotFound, "repository not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "unexpected error deleting repo: %v", err)
	}

	// return the response with the id of the deleted repository
	return &pb.DeleteRepositoryByIdResponse{
		RepositoryId: in.RepositoryId,
	}, nil
}

// DeleteRepositoryByName deletes a repository by name
func (s *Server) DeleteRepositoryByName(
	ctx context.Context,
	in *pb.DeleteRepositoryByNameRequest,
) (*pb.DeleteRepositoryByNameResponse, error) {
	// split repo name in owner and name
	fragments := strings.Split(in.Name, "/")
	if len(fragments) != 2 {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid repository name, needs to have the format: owner/name")
	}

	projectID := getProjectID(ctx)
	providerName := getProviderName(ctx)

	err := s.repos.DeleteByName(ctx, fragments[0], fragments[1], projectID, providerName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Errorf(codes.NotFound, "repository not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "unexpected error deleting repo: %v", err)
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
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	// Telemetry logging
	logger.BusinessRecord(ctx).Project = projectID

	providerName := in.GetContext().GetProvider()
	provs, errorProvs, err := s.providerManager.BulkInstantiateByTrait(ctx, projectID, db.ProviderTypeRepoLister, providerName)
	if err != nil {
		pErr := providers.ErrProviderNotFoundBy{}
		if errors.As(err, &pErr) {
			return nil, util.UserVisibleError(codes.NotFound, "no suitable provider found, please enroll a provider")
		}
		return nil, providerError(err)
	}

	out := &pb.ListRemoteRepositoriesFromProviderResponse{
		Results: []*pb.UpstreamRepositoryRef{},
	}

	for providerName, provider := range provs {
		zerolog.Ctx(ctx).Trace().
			Str("provider", providerName).
			Str("project_id", projectID.String()).
			Msg("listing repositories")

		repoLister, err := v1.As[v1.RepoLister](provider)
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("error instantiating repo lister")
			errorProvs = append(errorProvs, providerName)
			continue
		}

		results, err := s.listRemoteRepositoriesForProvider(ctx, providerName, repoLister, projectID)
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("cannot list repositories for provider")
			errorProvs = append(errorProvs, providerName)

			// skip over this provider
			continue
		}

		registeredRepos, err := s.store.ListRepositoriesByProjectID(ctx, db.ListRepositoriesByProjectIDParams{
			ProjectID: projectID,
			Provider:  getNameFilterParam(provider.Name),
		})
		if err != nil {
			// Here we try to handle errors gracefully by
			// simply not filtering.
			zerolog.Ctx(ctx).Error().Err(err).Msg("cannot get registered repos for provider")
			erroringProviders = append(erroringProviders, provider.Name)
		}

		registered := make(map[int64]bool)
		for _, repo := range registeredRepos {
			registered[repo.RepoID] = true
		}

		for _, result := range results {
			result.Registered = registered[result.RepoId]
		}
		out.Results = append(out.Results, results...)
	}

	// If all providers failed, return an error
	if len(errorProvs) > 0 && len(out.Results) == 0 {
		return nil, util.UserVisibleError(codes.Internal, "cannot list repositories for providers: %v", errorProvs)
	}

	return out, nil
}

func (s *Server) listRemoteRepositoriesForProvider(
	ctx context.Context,
	provName string,
	repoLister v1.RepoLister,
	projectID uuid.UUID,
) ([]*pb.UpstreamRepositoryRef, error) {
	tmoutCtx, cancel := context.WithTimeout(ctx, github.ExpensiveRestCallTimeout)
	defer cancel()

	remoteRepos, err := repoLister.ListAllRepositories(tmoutCtx)
	if err != nil {
		return nil, fmt.Errorf("cannot list repositories: %v", err)
	}

	allowsPrivateRepos := features.ProjectAllowsPrivateRepos(ctx, s.store, projectID)
	if !allowsPrivateRepos {
		zerolog.Ctx(ctx).Info().Msg("filtering out private repositories")
	} else {
		zerolog.Ctx(ctx).Info().Msg("including private repositories")
	}

	results := make([]*pb.UpstreamRepositoryRef, 0, len(remoteRepos))

	for idx, rem := range remoteRepos {
		// Skip private repositories
		if rem.IsPrivate && !allowsPrivateRepos {
			continue
		}
		remoteRepo := remoteRepos[idx]
		repo := &pb.UpstreamRepositoryRef{
			Context: &pb.Context{
				Provider: &provName,
				Project:  ptr.Ptr(projectID.String()),
			},
			Owner:  remoteRepo.Owner,
			Name:   remoteRepo.Name,
			RepoId: remoteRepo.RepoId,
		}
		results = append(results, repo)
	}

	return results, nil
}

// TODO: move out of controlplane
// inferProviderByOwner returns the provider to use for a given repo owner
func (s *Server) inferProviderByOwner(ctx context.Context, owner string, projectID uuid.UUID, providerName string,
) (*db.Provider, error) {
	if providerName != "" {
		return s.providerStore.GetByName(ctx, projectID, providerName)
	}
	opts, err := s.providerStore.GetByTraitInHierarchy(ctx, projectID, providerName, db.ProviderTypeGithub)
	if err != nil {
		return nil, fmt.Errorf("error getting providers: %v", err)
	}

	slices.SortFunc(opts, func(a, b db.Provider) int {
		// Sort GitHub OAuth provider after all GitHub App providers
		if a.Class == db.ProviderClassGithub && b.Class == db.ProviderClassGithubApp {
			return 1
		}
		if a.Class == db.ProviderClassGithubApp && b.Class == db.ProviderClassGithub {
			return -1
		}
		return 0
	})

	for _, prov := range opts {
		if github.CanHandleOwner(ctx, prov, owner) {
			return &prov, nil
		}
	}

	return nil, fmt.Errorf("no providers can handle repo owned by %s", owner)
}

func getProjectID(ctx context.Context) uuid.UUID {
	entityCtx := engine.EntityFromContext(ctx)
	return entityCtx.Project.ID
}

func getProviderName(ctx context.Context) string {
	entityCtx := engine.EntityFromContext(ctx)
	return entityCtx.Provider.Name
}
