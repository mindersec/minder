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
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/providers"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/verifier/verifyif"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// ListArtifacts lists all artifacts for a given project and provider
// nolint:gocyclo
func (s *Server) ListArtifacts(ctx context.Context, in *pb.ListArtifactsRequest) (*pb.ListArtifactsResponse, error) {
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	provider, err := getProviderFromRequestOrDefault(ctx, s.store, in, projectID)
	if err != nil {
		return nil, providerError(err)
	}
	res := &pb.ListArtifactsResponse{}

	if in.From != "" {
		artifactFilter, err := parseArtifactListFrom(s.store, in.From)
		if err != nil {
			return nil, fmt.Errorf("failed to parse artifact list from: %w", err)
		}

		results, err := artifactFilter.listArtifacts(ctx, provider.Name, projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to list artifacts: %w", err)
		}

		res.Results = results
	} else {
		artifacts, err := s.store.ListArtifactsByProjectID(ctx, projectID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, status.Errorf(codes.NotFound, "artifacts not found")
			}
			return nil, status.Errorf(codes.Unknown, "failed to get artifacts: %s", err)
		}

		for _, artifact := range artifacts {
			res.Results = append(res.Results, &pb.Artifact{
				ArtifactPk: artifact.ID.String(),
				// TODO: add owner and repo to artifact
				Owner:      "",
				Repository: "",
				Name:       artifact.ArtifactName,
				Type:       artifact.ArtifactType,
				Visibility: artifact.ArtifactVisibility,
				CreatedAt:  timestamppb.New(artifact.CreatedAt),
			})
		}
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = provider.Name
	logger.BusinessRecord(ctx).Project = projectID

	return res, nil
}

// GetArtifactByName gets an artifact by name
// nolint:gocyclo
func (s *Server) GetArtifactByName(ctx context.Context, in *pb.GetArtifactByNameRequest) (*pb.GetArtifactByNameResponse, error) {
	// tag and latest versions cannot be set at same time
	nameParts := strings.Split(in.Name, "/")
	if len(nameParts) < 3 {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid artifact name user repoOwner/repoName/artifactName")
	}

	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	// the artifact name is the rest of the parts
	artifactName := strings.Join(nameParts[2:], "/")
	artifact, err := s.store.GetArtifactByName(ctx, db.GetArtifactByNameParams{
		ProjectID:    entityCtx.Project.ID,
		ArtifactName: artifactName,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "artifact not found")
		}
		return nil, status.Errorf(codes.Unknown, "failed to get artifact: %s", err)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = artifact.ProviderName
	logger.BusinessRecord(ctx).Project = artifact.ProjectID
	logger.BusinessRecord(ctx).Artifact = artifact.ID

	var repoOwner string
	var repoName string
	if artifact.RepositoryID.Valid {
		repo, err := s.store.GetRepositoryByIDAndProject(ctx, db.GetRepositoryByIDAndProjectParams{
			ID:        artifact.RepositoryID.UUID,
			ProjectID: projectID,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, util.UserVisibleError(codes.NotFound, "repository not found")
			}
			return nil, status.Errorf(codes.Unknown, "failed to get repository: %s", err)
		}

		logger.BusinessRecord(ctx).Repository = artifact.RepositoryID.UUID

		repoOwner = repo.RepoOwner
		repoName = repo.RepoName
	}

	return &pb.GetArtifactByNameResponse{Artifact: &pb.Artifact{
		ArtifactPk: artifact.ID.String(),
		Owner:      repoOwner,
		Name:       artifact.ArtifactName,
		Type:       artifact.ArtifactType,
		Visibility: artifact.ArtifactVisibility,
		Repository: repoName,
		CreatedAt:  timestamppb.New(artifact.CreatedAt),
	},
		Versions: nil, // explicitly nil, will probably deprecate that field later
	}, nil
}

// GetArtifactById gets an artifact by id
// nolint:gocyclo
func (s *Server) GetArtifactById(ctx context.Context, in *pb.GetArtifactByIdRequest) (*pb.GetArtifactByIdResponse, error) {
	// tag and latest versions cannot be set at same time
	parsedArtifactID, err := uuid.Parse(in.Id)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid artifact ID")
	}

	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	// retrieve artifact details
	artifact, err := s.store.GetArtifactByID(ctx, db.GetArtifactByIDParams{
		ProjectID: projectID,
		ID:        parsedArtifactID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "artifact not found")
		}
		return nil, status.Errorf(codes.Unknown, "failed to get artifact: %s", err)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = artifact.ProviderName
	logger.BusinessRecord(ctx).Project = artifact.ProjectID
	logger.BusinessRecord(ctx).Artifact = artifact.ID

	var repoOwner string
	var repoName string
	if artifact.RepositoryID.Valid {
		repo, err := s.store.GetRepositoryByIDAndProject(ctx, db.GetRepositoryByIDAndProjectParams{
			ID:        artifact.RepositoryID.UUID,
			ProjectID: projectID,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, util.UserVisibleError(codes.NotFound, "repository not found")
			}
			return nil, status.Errorf(codes.Unknown, "failed to get repository: %s", err)
		}

		logger.BusinessRecord(ctx).Repository = artifact.RepositoryID.UUID

		repoOwner = repo.RepoOwner
		repoName = repo.RepoName
	}

	return &pb.GetArtifactByIdResponse{Artifact: &pb.Artifact{
		ArtifactPk: artifact.ID.String(),
		Owner:      repoOwner,
		Name:       artifact.ArtifactName,
		Type:       artifact.ArtifactType,
		Visibility: artifact.ArtifactVisibility,
		Repository: repoName,
		CreatedAt:  timestamppb.New(artifact.CreatedAt),
	},
		Versions: nil, // explicitly nil, will probably deprecate that field later
	}, nil
}

// RegisterArtifact registers an artifact
func (s *Server) RegisterArtifact(ctx context.Context, in *pb.RegisterArtifactRequest) (*pb.RegisterArtifactResponse, error) {
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID
	prov, err := getProviderFromRequestOrDefault(ctx, s.store, in, entityCtx.Project.ID)
	if err != nil {
		return nil, providerError(err)
	}

	ref := in.GetArtifact()

	// name and type cannot be empty
	if ref.GetName() == "" || ref.GetType() == "" {
		return nil, util.UserVisibleError(codes.InvalidArgument, "name and type cannot be empty")
	}

	typ := ref.GetType()
	// TODO make this a switch and move out of the verifyif package
	if typ != string(verifyif.ArtifactTypeContainer) {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid artifact type")
	}

	// check if artifact already exists
	_, err = s.store.GetArtifactByName(ctx, db.GetArtifactByNameParams{
		ProjectID:    projectID,
		ArtifactName: ref.GetName(),
	})
	if err == nil {
		return nil, util.UserVisibleError(codes.AlreadyExists, "artifact already exists")
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, status.Errorf(codes.Unknown, "failed to get artifact: %s", err)
	}

	// get OCI registry provider builder
	pbOpts := []providers.ProviderBuilderOption{
		providers.WithProviderMetrics(s.provMt),
		providers.WithRestClientCache(s.restClientCache),
	}
	provBuilder, err := providers.GetProviderBuilder(ctx, prov, s.store, s.cryptoEngine, &s.cfg.Provider, pbOpts...)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get provider builder: %s", err)
	}

	ociprov, err := provBuilder.GetOCI()
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get OCI client: %s", err)
	}

	// we verify that the artifact exists in the OCI registry
	// by listing the tags for the given artifact
	_, err = ociprov.ListTags(ctx, ref.GetName())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "artifact not found in registry: %s", err)
	}

	artifact, err := s.store.CreateArtifact(ctx, db.CreateArtifactParams{
		ProjectID:    projectID,
		RepositoryID: uuid.NullUUID{},
		ArtifactName: ref.GetName(),
		ArtifactType: ref.GetType(),
		// TODO get this from the provider
		ArtifactVisibility: "public",
	})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to create artifact: %s", err)
	}

	pbart := &pb.Artifact{
		ArtifactPk: artifact.ID.String(),
		Owner:      "",
		Name:       artifact.ArtifactName,
		Type:       artifact.ArtifactType,
		Visibility: artifact.ArtifactVisibility,
		Repository: "",
		CreatedAt:  timestamppb.New(artifact.CreatedAt),
	}

	// build entity info and publish
	if err := entities.NewEntityInfoWrapper().
		WithProvider(prov.Name).
		WithArtifact(pbart).
		WithArtifactID(artifact.ID).
		WithProjectID(projectID).
		Publish(s.evt); err != nil {
		zerolog.Ctx(ctx).Error().Err(err).
			Str("artifact_id", artifact.ID.String()).
			Msg("failed to publish artifact event")
	}

	return &pb.RegisterArtifactResponse{
		Artifact: pbart,
	}, nil
}

// ListRemoteArtifactsFromProvider lists all artifacts for a given project and provider
func (s *Server) ListRemoteArtifactsFromProvider(ctx context.Context, in *pb.ListRemoteArtifactsFromProviderRequest) (*pb.ListRemoteArtifactsFromProviderResponse, error) {
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	prov, err := getProviderFromRequestOrDefault(ctx, s.store, in, projectID)
	if err != nil {
		return nil, providerError(err)
	}

	if in.GetType() != "container" {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid artifact type")
	}

	// get OCI registry provider builder
	pbOpts := []providers.ProviderBuilderOption{
		providers.WithProviderMetrics(s.provMt),
		providers.WithRestClientCache(s.restClientCache),
	}
	provBuilder, err := providers.GetProviderBuilder(ctx, prov, s.store, s.cryptoEngine, &s.cfg.Provider, pbOpts...)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get provider builder: %s", err)
	}

	ociprov, err := provBuilder.GetImageLister()
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get OCI client: %s", err)
	}

	// list all artifacts in the OCI registry
	refs, err := ociprov.ListImages(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to list artifacts: %s", err)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = prov.Name
	logger.BusinessRecord(ctx).Project = projectID

	upstreamRefs := make([]*pb.UpstreamArtifactRef, 0, len(refs))
	for _, ref := range refs {
		upstreamRefs = append(upstreamRefs, &pb.UpstreamArtifactRef{
			Name: ref,
			Type: "container",
		})
	}

	return &pb.ListRemoteArtifactsFromProviderResponse{
		Results: upstreamRefs,
	}, nil
}

type artifactSource string

const (
	artifactSourceRepo artifactSource = "repository"
)

type artifactListFilter struct {
	store db.Store

	repoSlubList []string
	source       artifactSource
	filter       string
}

func parseArtifactListFrom(store db.Store, from string) (*artifactListFilter, error) {
	if from == "" {
		return &artifactListFilter{
			store:  store,
			source: artifactSourceRepo,
		}, nil
	}

	parts := strings.Split(from, "=")
	if len(parts) != 2 {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid filter, use format: <source>=<filter>")
	}

	source := parts[0]
	filter := parts[1]

	var repoSlubList []string

	switch source {
	case string(artifactSourceRepo):
		repoSlubList = strings.Split(filter, ",")
	default:
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid filter source, only repository is supported")
	}

	return &artifactListFilter{
		store:        store,
		source:       artifactSource(source),
		filter:       filter,
		repoSlubList: repoSlubList,
	}, nil
}

func (filter *artifactListFilter) listArtifacts(ctx context.Context, provider string, project uuid.UUID) ([]*pb.Artifact, error) {
	if filter.source != artifactSourceRepo {
		// just repos are supported now and we should never get here
		// when we support more, we turn this into an if-else or a switch
		return []*pb.Artifact{}, nil
	}

	repositories, err := artifactListRepoFilter(ctx, filter.store, provider, project, filter.repoSlubList)
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories: %w", err)
	}

	results := []*pb.Artifact{}
	for _, repository := range repositories {
		artifacts, err := filter.store.ListArtifactsByRepoID(ctx, uuid.NullUUID{
			UUID:  repository.ID,
			Valid: true,
		})
		if err != nil {
			return nil, status.Errorf(codes.Unknown, "failed to get artifacts: %s", err)
		}

		for _, artifact := range artifacts {
			results = append(results, &pb.Artifact{
				ArtifactPk: artifact.ID.String(),
				Owner:      repository.RepoOwner,
				Name:       artifact.ArtifactName,
				Type:       artifact.ArtifactType,
				Visibility: artifact.ArtifactVisibility,
				Repository: repository.RepoName,
				CreatedAt:  timestamppb.New(artifact.CreatedAt),
			})
		}
	}

	return results, nil
}

func artifactListRepoFilter(
	ctx context.Context, store db.Store, provider string, projectID uuid.UUID, repoSlubList []string,
) ([]*db.Repository, error) {
	repositories, err := store.ListRegisteredRepositoriesByProjectIDAndProvider(ctx,
		db.ListRegisteredRepositoriesByProjectIDAndProviderParams{Provider: provider, ProjectID: projectID})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "repositories not found")
		}
		return nil, status.Errorf(codes.Unknown, "failed to get repositories: %s", err)
	}

	var filterRepositories []*db.Repository
	for _, repo := range repositories {
		repo := repo
		if repoInSlubList(&repo, repoSlubList) {
			filterRepositories = append(filterRepositories, &repo)
		}
	}

	return filterRepositories, nil
}

func repoInSlubList(repo *db.Repository, slubList []string) bool {
	if len(slubList) == 0 {
		return true
	}

	// we might want to save the repoSlub in the future into the db..
	repoSlub := fmt.Sprintf("%s/%s", repo.RepoOwner, repo.RepoName)
	return slices.Contains(slubList, repoSlub)
}
