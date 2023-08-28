// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.role/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Package rule provides the CLI subcommand for managing rules

// Package artifact provides the artifact ingestion engine
package artifact

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/proto"
	"k8s.io/apimachinery/pkg/util/sets"

	evalerrors "github.com/stacklok/mediator/internal/engine/errors"
	engif "github.com/stacklok/mediator/internal/engine/interfaces"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

const (
	// ArtifactRuleDataIngestType is the type of the artifact rule data ingest engine
	ArtifactRuleDataIngestType = "artifact"
)

// Ingest is the engine for a rule type that uses artifact data ingest
// Implements enginer.ingester.Ingester
type Ingest struct {
}

// NewArtifactDataIngest creates a new artifact rule data ingest engine
func NewArtifactDataIngest(
	_ *pb.ArtifactType,
) (*Ingest, error) {
	return &Ingest{}, nil
}

// Ingest checks the passed in artifact, makes sure it is applicable to the current rule
// and if it is, returns the appropriately marshalled data.
func (_ *Ingest) Ingest(
	_ context.Context,
	ent proto.Message,
	params map[string]any,
) (*engif.Result, error) {
	cfg, err := configFromParams(params)
	if err != nil {
		return nil, err
	}

	versionedArtifact, ok := ent.(*pb.VersionedArtifact)
	if !ok {
		return nil, fmt.Errorf("expected VersionedArtifact, got %T", ent)
	}

	if !isApplicableArtifact(versionedArtifact, cfg) {
		return nil, evalerrors.ErrEvaluationSkipSilently
	}

	result := struct {
		Verification   any
		GithubWorkflow any
	}{
		versionedArtifact.Version.SignatureVerification,
		versionedArtifact.Version.GithubWorkflow}
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	out := make(map[string]any)
	err = json.Unmarshal(jsonBytes, &out)
	if err != nil {
		return nil, err
	}

	return &engif.Result{
		Object: out,
	}, nil
}

func isApplicableArtifact(
	versionedArtifact *pb.VersionedArtifact,
	cfg *ingesterConfig,
) bool {
	if newArtifactIngestType(versionedArtifact.Artifact.Type) != cfg.Type {
		// not interested in this type of artifact
		return false
	}

	if cfg.Name != versionedArtifact.Artifact.Name {
		// not interested in this artifact
		return false
	}

	// no tags is treated as a wildcard and matches any container. This might be configurable in the future
	if len(cfg.Tags) == 0 {
		return true
	}

	haveTags := sets.New(versionedArtifact.Version.Tags...)
	return haveTags.HasAny(cfg.Tags...)
}
