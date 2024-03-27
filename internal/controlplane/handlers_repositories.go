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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/projects/features"
	"github.com/stacklok/minder/internal/providers"
	"github.com/stacklok/minder/internal/providers/github"
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
	provider, client, err := s.getProviderAndClient(ctx, projectID, in)
	if err != nil {
		return nil, err
	}

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

	newRepo, err := s.repos.CreateRepository(ctx, client, provider, projectID, githubRepo.GetOwner(), githubRepo.GetName())
	if err != nil {
		if errors.Is(err, ghrepo.ErrPrivateRepoForbidden) {
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

	limit := sql.NullInt64{Valid: false, Int64: 0}
	reqLimit := in.GetLimit()
	if reqLimit > 0 {
		if reqLimit > maxFetchLimit {
			return nil, util.UserVisibleError(codes.InvalidArgument, "limit too high, max is %d", maxFetchLimit)
		}
		limit = sql.NullInt64{Valid: true, Int64: reqLimit + 1}
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
	if limit.Valid && int64(len(repos)) == limit.Int64 {
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

// DeleteRepositoryById deletes a repository by its UUID
func (s *Server) DeleteRepositoryById(
	ctx context.Context,
	in *pb.DeleteRepositoryByIdRequest,
) (*pb.DeleteRepositoryByIdResponse, error) {
	parsedRepositoryID, err := uuid.Parse(in.RepositoryId)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid repository ID")
	}

	err = s.deleteRepository(ctx, in, func(client v1.GitHub, projectID uuid.UUID, _ string) error {
		return s.repos.DeleteRepositoryByID(ctx, client, projectID, parsedRepositoryID)
	})
	if err != nil {
		return nil, err
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

	err := s.deleteRepository(ctx, in, func(client v1.GitHub, projectID uuid.UUID, providerName string) error {
		return s.repos.DeleteRepositoryByName(ctx, client, projectID, providerName, fragments[0], fragments[1])
	})
	if err != nil {
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
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	// Telemetry logging
	logger.BusinessRecord(ctx).Project = projectID

	provs, err := getProvidersByTrait(ctx, s.store, in, projectID, db.ProviderTypeRepoLister)
	if err != nil {
		return nil, providerError(err)
	}

	out := &pb.ListRemoteRepositoriesFromProviderResponse{
		Results: []*pb.UpstreamRepositoryRef{},
	}

	for _, provider := range provs {
		zerolog.Ctx(ctx).Trace().
			Str("provider", provider.Name).
			Str("project_id", projectID.String()).
			Msg("listing repositories")

		pbOpts := []providers.ProviderBuilderOption{
			providers.WithProviderMetrics(s.provMt),
			providers.WithRestClientCache(s.restClientCache),
		}
		p, err := providers.GetProviderBuilder(ctx, provider, s.store, s.cryptoEngine, &s.cfg.Provider, pbOpts...)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "cannot get provider builder: %v", err)
		}

		results, err := s.listRemoteRepositoriesForProvider(ctx, provider.Name, p, projectID)
		if err != nil {
			return nil, err
		}

		out.Results = append(out.Results, results...)
	}

	return out, nil
}

func (s *Server) listRemoteRepositoriesForProvider(
	ctx context.Context,
	provName string,
	p *providers.ProviderBuilder,
	projectID uuid.UUID,
) ([]*pb.UpstreamRepositoryRef, error) {
	// by now we've already checked that repo listed is implemented
	client, err := p.GetRepoLister()
	if err != nil {
		if errors.Is(err, providers.ErrInvalidCredential) {
			return nil, util.UserVisibleError(codes.PermissionDenied,
				"cannot get credential for provider: did you run `minder provider enroll`?")
		}
		return nil, status.Errorf(codes.Internal, "cannot create github client: %v", err)
	}

	tmoutCtx, cancel := context.WithTimeout(ctx, github.ExpensiveRestCallTimeout)
	defer cancel()

	remoteRepos, err := client.ListAllRepositories(tmoutCtx)
	if err != nil {
		return nil, util.UserVisibleError(codes.Internal, "cannot list repositories: %v", err)
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

// TODO: this probably can probably be used elsewhere
// returns project ID and github client - this flow of code exists in multiple
// places
func (s *Server) getProviderAndClient(
	ctx context.Context,
	projectID uuid.UUID,
	request HasProtoContext,
) (*db.Provider, v1.GitHub, error) {
	provider, err := getProviderFromRequestOrDefault(ctx, s.store, request, projectID)
	if err != nil {
		return nil, nil, providerError(err)
	}

	pbOpts := []providers.ProviderBuilderOption{
		providers.WithProviderMetrics(s.provMt),
		providers.WithRestClientCache(s.restClientCache),
	}

	p, err := providers.GetProviderBuilder(ctx, provider, s.store, s.cryptoEngine, &s.cfg.Provider, pbOpts...)
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "cannot get provider builder: %v", err)
	}

	client, err := p.GetGitHub()
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "error creating github provider: %v", err)
	}

	return &provider, client, nil
}

// covers the common logic for the two varieties of repo deletion
func (s *Server) deleteRepository(
	ctx context.Context,
	request HasProtoContext,
	deletionMethod func(v1.GitHub, uuid.UUID, string) error,
) error {
	projectID := getProjectID(ctx)
	provider, client, err := s.getProviderAndClient(ctx, projectID, request)
	if err != nil {
		return err
	}

	err = deletionMethod(client, projectID, provider.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return status.Errorf(codes.NotFound, "repository not found")
	} else if err != nil {
		return status.Errorf(codes.Internal, "unexpected error deleting repo: %v", err)
	}

	return nil
}

func getProjectID(ctx context.Context) uuid.UUID {
	entityCtx := engine.EntityFromContext(ctx)
	return entityCtx.Project.ID
}
