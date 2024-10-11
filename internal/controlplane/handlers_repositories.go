// Copyright 2024 Stacklok, Inc
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
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/engcontext"
	"github.com/stacklok/minder/internal/entities/properties"
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/projects/features"
	"github.com/stacklok/minder/internal/providers"
	"github.com/stacklok/minder/internal/providers/github"
	"github.com/stacklok/minder/internal/repositories"
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
	projectID := GetProjectID(ctx)
	providerName := GetProviderName(ctx)

	var fetchByProps *properties.Properties
	var provider *db.Provider
	var err error
	if in.GetEntity() != nil {
		fetchByProps, provider, err = s.repoCreateInfoFromUpstreamEntityRef(
			ctx, projectID, providerName, in.GetEntity())
	} else if in.GetRepository() != nil {
		fetchByProps, provider, err = s.repoCreateInfoFromUpstreamRepositoryRef(
			ctx, projectID, providerName, in.GetRepository())
	} else {
		return nil, util.UserVisibleError(codes.InvalidArgument, "missing entity or repository field")
	}

	if err != nil {
		return nil, err
	}

	l := zerolog.Ctx(ctx).With().
		Dict("properties", fetchByProps.ToLogDict()).
		Str("projectID", projectID.String()).
		Logger()
	ctx = l.WithContext(ctx)

	newRepo, err := s.repos.CreateRepository(ctx, provider, projectID, fetchByProps)
	if err != nil {
		if errors.Is(err, repositories.ErrPrivateRepoForbidden) || errors.Is(err, repositories.ErrArchivedRepoForbidden) {
			return nil, util.UserVisibleError(codes.InvalidArgument, "%s", err.Error())
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

func (s *Server) repoCreateInfoFromUpstreamRepositoryRef(
	ctx context.Context,
	projectID uuid.UUID,
	providerName string,
	rep *pb.UpstreamRepositoryRef,
) (*properties.Properties, *db.Provider, error) {
	// If the repo owner is missing, GitHub will assume a default value based
	// on the user's credentials. An explicit check for owner is left out to
	// avoid breaking backwards compatibility.
	if rep.GetName() == "" {
		return nil, nil, util.UserVisibleError(codes.InvalidArgument, "missing repository name")
	}

	fetchByProps, err := properties.NewProperties(map[string]any{
		properties.PropertyUpstreamID: fmt.Sprintf("%d", rep.GetRepoId()),
		properties.PropertyName:       fmt.Sprintf("%s/%s", rep.GetOwner(), rep.GetName()),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("error creating properties: %w", err)
	}

	provider, err := s.inferProviderByOwner(ctx, rep.GetOwner(), projectID, providerName)
	if err != nil {
		pErr := providers.ErrProviderNotFoundBy{}
		if errors.As(err, &pErr) {
			return nil, nil, util.UserVisibleError(codes.NotFound, "no suitable provider found, please enroll a provider")
		}
		return nil, nil, status.Errorf(codes.Internal, "cannot get provider: %v", err)
	}

	return fetchByProps, provider, nil
}

func (s *Server) repoCreateInfoFromUpstreamEntityRef(
	ctx context.Context,
	projectID uuid.UUID,
	providerName string,
	entity *pb.UpstreamEntityRef,
) (*properties.Properties, *db.Provider, error) {
	inPropsMap := entity.GetProperties().AsMap()
	fetchByProps, err := properties.NewProperties(inPropsMap)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating properties: %w", err)
	}

	provider, err := s.providerStore.GetByName(ctx, projectID, providerName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, util.UserVisibleError(codes.NotFound, "provider not found")
		}
		return nil, nil, status.Errorf(codes.Internal, "cannot get provider: %v", err)
	}

	return fetchByProps, provider, nil
}

// ListRepositories returns a list of repositories for a given project
// This function will typically be called by the client to get a list of
// repositories that are registered present in the minder database
func (s *Server) ListRepositories(ctx context.Context,
	in *pb.ListRepositoriesRequest) (*pb.ListRepositoriesResponse, error) {
	entityCtx := engcontext.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID
	providerName := entityCtx.Provider.Name

	logger.BusinessRecord(ctx).Provider = providerName
	logger.BusinessRecord(ctx).Project = projectID

	providerFilter := getNameFilterParam(providerName)

	reqRepoCursor, err := cursorutil.NewRepoCursor(in.GetCursor())
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "%s", err.Error())
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
	projectID := GetProjectID(ctx)

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

	entityCtx := engcontext.EntityFromContext(ctx)
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

	projectID := GetProjectID(ctx)

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

	projectID := GetProjectID(ctx)
	providerName := GetProviderName(ctx)

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
	entityCtx := engcontext.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	// Telemetry logging
	logger.BusinessRecord(ctx).Project = projectID

	providerName := in.GetContext().GetProvider()
	provs, errorProvs, err := s.providerManager.BulkInstantiateByTrait(
		ctx, projectID, db.ProviderTypeRepoLister, providerName)
	if err != nil {
		pErr := providers.ErrProviderNotFoundBy{}
		if errors.As(err, &pErr) {
			return nil, util.UserVisibleError(codes.NotFound, "no suitable provider found, please enroll a provider")
		}
		return nil, providerError(err)
	}

	out := &pb.ListRemoteRepositoriesFromProviderResponse{
		Results:  []*pb.UpstreamRepositoryRef{},
		Entities: []*pb.RegistrableUpstreamEntityRef{},
	}

	for providerID, providerT := range provs {
		results, err := s.fetchRepositoriesForProvider(
			ctx, projectID, providerID, providerT.Name, providerT.Provider)
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).
				Msgf("error listing repositories for provider %s in project %s", providerT.Name, projectID)
			errorProvs = append(errorProvs, providerT.Name)
			continue
		}
		for _, result := range results {
			out.Results = append(out.Results, result.Repo)
			out.Entities = append(out.Entities, result.Entity)
		}
	}

	// If all providers failed, return an error
	if len(errorProvs) > 0 && len(out.Results) == 0 {
		return nil, util.UserVisibleError(codes.Internal, "cannot list repositories for providers: %v", errorProvs)
	}

	return out, nil
}

// fetchRepositoriesForProvider fetches repositories for a given provider
//
// Returns a list of repositories that with an up-to-date status of whether they are registered
func (s *Server) fetchRepositoriesForProvider(
	ctx context.Context,
	projectID uuid.UUID,
	providerID uuid.UUID,
	providerName string,
	provider v1.Provider,
) ([]*UpstreamRepoAndEntityRef, error) {
	zerolog.Ctx(ctx).Trace().
		Str("provider_id", providerID.String()).
		Str("project_id", projectID.String()).
		Msg("listing repositories")

	repoLister, err := v1.As[v1.RepoLister](provider)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("error instantiating repo lister")
		return nil, err
	}

	results, err := s.listRemoteRepositoriesForProvider(ctx, providerName, repoLister, projectID)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("cannot list repositories for provider")
		return nil, err
	}

	registeredRepos, err := s.repos.ListRepositories(
		ctx,
		projectID,
		providerID,
	)
	if err != nil {
		zerolog.Ctx(ctx).Error().
			Str("project_id", projectID.String()).
			Str("provider_id", providerID.String()).
			Err(err).Msg("cannot list registered repositories")
		return nil, util.UserVisibleError(
			codes.Internal,
			"cannot list registered repositories",
		)
	}

	registered := make(map[string]bool)
	for _, repo := range registeredRepos {
		uidP := repo.Properties.GetProperty(properties.PropertyUpstreamID)
		if uidP == nil {
			zerolog.Ctx(ctx).Warn().
				Str("entity_id", repo.Entity.ID.String()).
				Str("entity_name", repo.Entity.Name).
				Str("provider_id", providerID.String()).
				Str("project_id", projectID.String()).
				Msg("repository has no upstream ID")
			continue
		}
		registered[uidP.GetString()] = true
	}

	for _, result := range results {
		uprops := result.Entity.GetEntity().GetProperties()
		upropsMap := uprops.AsMap()
		if upropsMap == nil {
			zerolog.Ctx(ctx).Warn().
				Str("provider_id", providerID.String()).
				Str("project_id", projectID.String()).
				Msg("upstream repository entry has no properties")
			continue
		}
		uidAny, ok := upropsMap[properties.PropertyUpstreamID]
		if !ok {
			zerolog.Ctx(ctx).Warn().
				Str("provider_id", providerID.String()).
				Str("project_id", projectID.String()).
				Msg("upstream repository entry has no upstream ID")
			continue
		}

		uid, ok := uidAny.(string)
		if !ok {
			zerolog.Ctx(ctx).Warn().
				Str("provider_id", providerID.String()).
				Str("project_id", projectID.String()).
				Msg("upstream repository entry has invalid upstream ID")
			continue
		}

		result.Repo.Registered = registered[uid]
		result.Entity.Registered = registered[uid]
	}

	return results, nil
}

func (s *Server) listRemoteRepositoriesForProvider(
	ctx context.Context,
	provName string,
	repoLister v1.RepoLister,
	projectID uuid.UUID,
) ([]*UpstreamRepoAndEntityRef, error) {
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

	results := make([]*UpstreamRepoAndEntityRef, 0, len(remoteRepos))

	for idx, rem := range remoteRepos {
		// Skip private repositories
		if rem.IsPrivate && !allowsPrivateRepos {
			continue
		}
		remoteRepo := remoteRepos[idx]

		var props *structpb.Struct
		if remoteRepo.Properties != nil {
			props = remoteRepo.Properties
		}

		repo := &UpstreamRepoAndEntityRef{
			Repo: &pb.UpstreamRepositoryRef{
				Context: &pb.Context{
					Provider: &provName,
					Project:  ptr.Ptr(projectID.String()),
				},
				Owner:  remoteRepo.Owner,
				Name:   remoteRepo.Name,
				RepoId: remoteRepo.RepoId,
			},
			Entity: &pb.RegistrableUpstreamEntityRef{
				Entity: &pb.UpstreamEntityRef{
					Context: &pb.ContextV2{
						Provider:  provName,
						ProjectId: projectID.String(),
					},
					Type:       pb.Entity_ENTITY_REPOSITORIES,
					Properties: props,
				},
			},
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

// UpstreamRepoAndEntityRef is a pair of upstream repository and entity references
type UpstreamRepoAndEntityRef struct {
	Repo   *pb.UpstreamRepositoryRef
	Entity *pb.RegistrableUpstreamEntityRef
}
