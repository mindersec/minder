// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/engcontext"
	"github.com/mindersec/minder/internal/logger"
	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/ptr"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// ListArtifacts lists all artifacts for a given project and provider
// nolint:gocyclo
func (s *Server) ListArtifacts(ctx context.Context, in *pb.ListArtifactsRequest) (*pb.ListArtifactsResponse, error) {
	entityCtx := engcontext.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID
	providerName := entityCtx.Provider.Name

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = providerName
	logger.BusinessRecord(ctx).Project = projectID

	artifactFilter, err := parseArtifactListFrom(s.store, in.From)
	if err != nil {
		return nil, fmt.Errorf("failed to parse artifact list from: %w", err)
	}

	results, err := artifactFilter.listArtifacts(ctx, providerName, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list artifacts: %w", err)
	}

	return &pb.ListArtifactsResponse{Results: results}, nil
}

// GetArtifactByName gets an artifact by name
// nolint:gocyclo
func (s *Server) GetArtifactByName(ctx context.Context, in *pb.GetArtifactByNameRequest) (*pb.GetArtifactByNameResponse, error) {
	// tag and latest versions cannot be set at same time
	nameParts := strings.Split(in.Name, "/")
	if len(nameParts) < 3 {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid artifact name user repoOwner/repoName/artifactName")
	}

	entityCtx := engcontext.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID
	providerName := entityCtx.Provider.Name
	providerFilter := getNameFilterParam(providerName)

	logger.BusinessRecord(ctx).Provider = providerName

	// the artifact name is the rest of the parts
	artifactName := strings.Join(nameParts[2:], "/")
	artifact, err := s.store.GetArtifactByName(ctx, db.GetArtifactByNameParams{
		ArtifactName: artifactName,
		ProjectID:    projectID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "artifact not found")
		}
		return nil, status.Errorf(codes.Unknown, "failed to get artifact: %s", err)
	}

	var repoName, repoOwner string
	// TODO look for repo if any
	if artifact.RepositoryID.Valid {
		repo, err := s.store.GetRepositoryByRepoName(ctx, db.GetRepositoryByRepoNameParams{
			Provider:  providerFilter,
			RepoOwner: nameParts[0],
			RepoName:  nameParts[1],
			ProjectID: projectID,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, util.UserVisibleError(codes.NotFound, "repository not found")
			}
			return nil, status.Errorf(codes.Unknown, "failed to get repository: %s", err)
		}

		logger.BusinessRecord(ctx).Repository = repo.ID

		repoName = repo.RepoName
		repoOwner = repo.RepoOwner
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).ProviderID = artifact.ProviderID
	logger.BusinessRecord(ctx).Project = artifact.ProjectID
	logger.BusinessRecord(ctx).Artifact = artifact.ID

	return &pb.GetArtifactByNameResponse{Artifact: &pb.Artifact{
		ArtifactPk: artifact.ID.String(),
		Context: &pb.Context{
			Provider: ptr.Ptr(artifact.ProviderName),
			Project:  ptr.Ptr(artifact.ProjectID.String()),
		},
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
	entityCtx := engcontext.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	// tag and latest versions cannot be set at same time
	parsedArtifactID, err := uuid.Parse(in.Id)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid artifact ID")
	}

	// retrieve artifact details
	artifact, err := s.store.GetArtifactByID(ctx, db.GetArtifactByIDParams{
		ID:        parsedArtifactID,
		ProjectID: projectID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "artifact not found")
		}
		return nil, status.Errorf(codes.Unknown, "failed to get artifact: %s", err)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).ProviderID = artifact.ProviderID
	logger.BusinessRecord(ctx).Project = artifact.ProjectID
	logger.BusinessRecord(ctx).Artifact = artifact.ID
	if artifact.RepositoryID.Valid {
		logger.BusinessRecord(ctx).Repository = artifact.RepositoryID.UUID
	}

	return &pb.GetArtifactByIdResponse{Artifact: &pb.Artifact{
		ArtifactPk: artifact.ID.String(),
		Context: &pb.Context{
			Provider: ptr.Ptr(artifact.ProviderName),
			Project:  ptr.Ptr(artifact.ProjectID.String()),
		},
		Owner:      "", // TODO re-add
		Name:       artifact.ArtifactName,
		Type:       artifact.ArtifactType,
		Visibility: artifact.ArtifactVisibility,
		Repository: "", // TODO re-add
		CreatedAt:  timestamppb.New(artifact.CreatedAt),
	},
		Versions: nil, // explicitly nil, will probably deprecate that field later
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
				Context: &pb.Context{
					Provider: ptr.Ptr(artifact.ProviderName),
					Project:  ptr.Ptr(artifact.ProjectID.String()),
				},
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
	providerFilter := getNameFilterParam(provider)

	repositories, err := store.ListRegisteredRepositoriesByProjectIDAndProvider(ctx,
		db.ListRegisteredRepositoriesByProjectIDAndProviderParams{Provider: providerFilter, ProjectID: projectID})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "repositories not found")
		}
		return nil, status.Errorf(codes.Unknown, "failed to get repositories: %s", err)
	}

	var filterRepositories []*db.Repository
	for _, repo := range repositories {
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
