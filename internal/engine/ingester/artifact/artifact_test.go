// Copyright 2023 Stacklok, Inc.
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
// Package rule provides the CLI subcommand for managing rules

package artifact_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	evalerrors "github.com/stacklok/mediator/internal/engine/errors"
	"github.com/stacklok/mediator/internal/engine/ingester/artifact"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

func TestArtifactIngestMatchingName(t *testing.T) {
	t.Parallel()

	ing, err := artifact.NewArtifactDataIngest(nil)
	require.NoError(t, err, "expected no error")

	got, err := ing.Ingest(context.Background(), &pb.VersionedArtifact{
		Artifact: &pb.Artifact{
			Type: "container",
			Name: "this-will-match",
		},
		Version: &pb.ArtifactVersion{},
	}, map[string]interface{}{
		"name": "this-will-match",
	})
	require.NoError(t, err, "expected no error")
	require.NotNil(t, got, "expected non-nil result")
}

func TestArtifactIngestNotMatchingName(t *testing.T) {
	t.Parallel()

	ing, err := artifact.NewArtifactDataIngest(nil)
	require.NoError(t, err, "expected no error")

	got, err := ing.Ingest(context.Background(), &pb.VersionedArtifact{
		Artifact: &pb.Artifact{
			Type: "container",
			Name: "this-will-match",
		},
		Version: &pb.ArtifactVersion{},
	}, map[string]interface{}{
		"name": "this-will-NOT-match",
	})
	require.Error(t, err, "expected error")
	require.ErrorIs(t, err, evalerrors.ErrEvaluationSkipSilently, "expected ErrEvaluationSkipSilently")
	require.Nil(t, got, "expected nil result")
}

func TestArtifactIngestMatchAnyName(t *testing.T) {
	t.Parallel()

	ing, err := artifact.NewArtifactDataIngest(nil)
	require.NoError(t, err, "expected no error")

	got, err := ing.Ingest(context.Background(), &pb.VersionedArtifact{
		Artifact: &pb.Artifact{
			Type: "container",
			Name: "surely-noone-will-set-this-name",
		},
		Version: &pb.ArtifactVersion{},
	}, map[string]interface{}{
		"name": "", // empty string means match any name
	})
	require.NoError(t, err, "expected no error")
	require.NotNil(t, got, "expected non-nil result")
}
