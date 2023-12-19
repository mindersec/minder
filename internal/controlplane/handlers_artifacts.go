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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/util"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// ListArtifacts lists all artifacts for a given project and provider
// nolint:gocyclo
func (s *Server) ListArtifacts(ctx context.Context, in *pb.ListArtifactsRequest) (*pb.ListArtifactsResponse, error) {
	projectID, err := getProjectFromRequestOrDefault(ctx, in)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, err.Error())
	}

	// check if user is authorized
	if err := AuthorizedOnProject(ctx, projectID); err != nil {
		return nil, err
	}

	provider, err := getProviderFromRequestOrDefault(ctx, s.store, in, projectID)
	if err != nil {
		return nil, providerError(err)
	}

	artifactFilter, err := parseArtifactListFrom(s.store, in.From)
	if err != nil {
		return nil, fmt.Errorf("failed to parse artifact list from: %w", err)
	}

	results, err := artifactFilter.listArtifacts(ctx, provider.Name, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list artifacts: %w", err)
	}

	return &pb.ListArtifactsResponse{Results: results}, nil
}

// GetArtifactById gets an artifact by id
// nolint:gocyclo
func (s *Server) GetArtifactById(ctx context.Context, in *pb.GetArtifactByIdRequest) (*pb.GetArtifactByIdResponse, error) {
	// tag and latest versions cannot be set at same time
	if in.Tag != "" && in.LatestVersions > 1 {
		return nil, status.Errorf(codes.InvalidArgument, "tag and latest versions cannot be set at same time")
	}

	if in.LatestVersions < 1 || in.LatestVersions > 10 {
		return nil, status.Errorf(codes.InvalidArgument, "latest versions must be between 1 and 10")
	}

	parsedArtifactID, err := uuid.Parse(in.Id)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid artifact ID")
	}

	// retrieve artifact details
	artifact, err := s.store.GetArtifactByID(ctx, parsedArtifactID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "artifact not found")
		}
		return nil, status.Errorf(codes.Unknown, "failed to get artifact: %s", err)
	}

	// check if user is authorized
	if err := AuthorizedOnProject(ctx, artifact.ProjectID); err != nil {
		return nil, err
	}

	// get artifact versions
	if in.LatestVersions <= 0 {
		in.LatestVersions = 10
	}

	var versions []db.ArtifactVersion
	if in.Tag != "" {
		versions, err = s.store.ListArtifactVersionsByArtifactIDAndTag(ctx,
			db.ListArtifactVersionsByArtifactIDAndTagParams{ArtifactID: parsedArtifactID,
				Tags:  sql.NullString{Valid: true, String: in.Tag},
				Limit: sql.NullInt32{Valid: true, Int32: in.LatestVersions}})

	} else {
		versions, err = s.store.ListArtifactVersionsByArtifactID(ctx,
			db.ListArtifactVersionsByArtifactIDParams{ArtifactID: parsedArtifactID,
				Limit: sql.NullInt32{Valid: true, Int32: in.LatestVersions}})
	}

	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get artifact versions: %s", err)
	}

	final_versions := []*pb.ArtifactVersion{}
	for _, version := range versions {
		tags := []string{}
		if version.Tags.Valid {
			tags = strings.Split(version.Tags.String, ",")
		}

		sigVerification := &pb.SignatureVerification{}
		if version.SignatureVerification.Valid {
			if err := protojson.Unmarshal(version.SignatureVerification.RawMessage, sigVerification); err != nil {
				return nil, err
			}
		}

		ghWorkflow := &pb.GithubWorkflow{}
		if version.GithubWorkflow.Valid {
			if err := protojson.Unmarshal(version.GithubWorkflow.RawMessage, ghWorkflow); err != nil {
				return nil, err
			}
		}

		final_versions = append(final_versions, &pb.ArtifactVersion{
			VersionId:             version.Version,
			Tags:                  tags,
			Sha:                   version.Sha,
			SignatureVerification: sigVerification,
			GithubWorkflow:        ghWorkflow,
			CreatedAt:             timestamppb.New(version.CreatedAt),
		})

	}

	return &pb.GetArtifactByIdResponse{Artifact: &pb.Artifact{
		ArtifactPk: artifact.ID.String(),
		Owner:      artifact.RepoOwner,
		Name:       artifact.ArtifactName,
		Type:       artifact.ArtifactType,
		Visibility: artifact.ArtifactVisibility,
		Repository: artifact.RepoName,
		CreatedAt:  timestamppb.New(artifact.CreatedAt),
	},
		Versions: final_versions,
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
		artifacts, err := filter.store.ListArtifactsByRepoID(ctx, repository.ID)
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
