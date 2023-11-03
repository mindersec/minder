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

// ListArtifacts lists all artifacts for a given group and provider
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

	provider, err := s.store.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name:      in.Provider,
		ProjectID: projectID,
	})
	if err != nil {
		return nil, providerError(err)
	}

	// first read all the repositories for provider and group
	repositories, err := s.store.ListRegisteredRepositoriesByProjectIDAndProvider(ctx,
		db.ListRegisteredRepositoriesByProjectIDAndProviderParams{Provider: provider.Name, ProjectID: projectID})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "repositories not found")
		}
		return nil, status.Errorf(codes.Unknown, "failed to get repositories: %s", err)
	}

	results := []*pb.Artifact{}
	for _, repository := range repositories {
		artifacts, err := s.store.ListArtifactsByRepoID(ctx, repository.ID)
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
