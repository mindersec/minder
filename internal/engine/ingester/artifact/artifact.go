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
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/minder/v1"
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

// GetType returns the type of the artifact rule data ingest engine
func (*Ingest) GetType() string {
	return ArtifactRuleDataIngestType
}

// GetConfig returns the config for the artifact rule data ingest engine
func (*Ingest) GetConfig() proto.Message {
	return nil
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

	artifact, ok := ent.(*pb.Artifact)
	if !ok {
		return nil, fmt.Errorf("expected Artifact, got %T", ent)
	}

	// Filter the versions of the artifact that are applicable to this rule
	applicable, err := getApplicableArtifactVersions(artifact, cfg)
	if err != nil {
		return nil, err
	}

	return &engif.Result{
		Object: applicable,
	}, nil
}

func getApplicableArtifactVersions(
	artifact *pb.Artifact,
	cfg *ingesterConfig,
) ([]map[string]any, error) {
	var applicableArtifactVersions []struct {
		Verification   any
		GithubWorkflow any
	}
	// make sure the artifact type matches
	if newArtifactIngestType(artifact.Type) != cfg.Type {
		return nil, evalerrors.NewErrEvaluationSkipSilently("artifact type mismatch")
	}

	// if a name is specified, make sure it matches
	if cfg.Name != "" && cfg.Name != artifact.Name {
		return nil, evalerrors.NewErrEvaluationSkipSilently("artifact name mismatch")
	}

	// get all versions of the artifact that are applicable to this rule
	for _, artifactVersion := range artifact.Versions {
		// skip artifact versions without tags
		if len(artifactVersion.Tags) == 0 || artifactVersion.Tags[0] == "" {
			continue
		}

		// rule without tags is treated as a wildcard and matches all tagged artifacts
		// this might be configurable in the future
		if len(cfg.Tags) == 0 {
			applicableArtifactVersions = append(applicableArtifactVersions, struct {
				Verification   any
				GithubWorkflow any
			}{artifactVersion.SignatureVerification, artifactVersion.GithubWorkflow})
			continue
		}

		// make sure all rule tags are present in the artifact version tags
		haveTags := sets.New(artifactVersion.Tags...)
		tagsOk := haveTags.HasAll(cfg.Tags...)
		if tagsOk {
			applicableArtifactVersions = append(applicableArtifactVersions, struct {
				Verification   any
				GithubWorkflow any
			}{artifactVersion.SignatureVerification, artifactVersion.GithubWorkflow})
		}
	}

	// if no applicable artifact versions were found for this rule, we can go ahead and fail the rule evaluation here
	if len(applicableArtifactVersions) == 0 {
		return nil, evalerrors.NewErrEvaluationFailed("no applicable artifact versions found")
	}

	jsonBytes, err := json.Marshal(applicableArtifactVersions)
	if err != nil {
		return nil, err
	}

	result := make([]map[string]any, 0, len(applicableArtifactVersions))
	err = json.Unmarshal(jsonBytes, &result)
	if err != nil {
		return nil, err
	}
	// return the list of applicable artifact versions
	return result, nil
}
