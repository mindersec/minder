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

	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"

	evalerrors "github.com/stacklok/minder/internal/engine/errors"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/providers"
	"github.com/stacklok/minder/internal/verifier"
	"github.com/stacklok/minder/internal/verifier/sigstore"
	"github.com/stacklok/minder/internal/verifier/sigstore/container"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

const (
	// ArtifactRuleDataIngestType is the type of the artifact rule data ingest engine
	ArtifactRuleDataIngestType = "artifact"
)

// Ingest is the engine for a rule type that uses artifact data ingest
// Implements enginer.ingester.Ingester
type Ingest struct {
	ghCli provifv1.GitHub
}

// NewArtifactDataIngest creates a new artifact rule data ingest engine
func NewArtifactDataIngest(
	_ *pb.ArtifactType,
	pbuild *providers.ProviderBuilder,
) (*Ingest, error) {

	ghCli, err := pbuild.GetGitHub(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get github client: %w", err)
	}

	return &Ingest{
		ghCli: ghCli,
	}, nil
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
func (i *Ingest) Ingest(
	ctx context.Context,
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
	applicable, err := i.getApplicableArtifactVersions(ctx, artifact, cfg)
	if err != nil {
		return nil, err
	}

	return &engif.Result{
		Object: applicable,
	}, nil
}

func (i *Ingest) getApplicableArtifactVersions(
	ctx context.Context,
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

	// Build a tag matcher based on the configuration
	tagMatcher, err := buildTagMatcher(cfg.Tags, cfg.TagRegex)
	if err != nil {
		return nil, err
	}

	// get all versions of the artifact that are applicable to this rule
	for _, artifactVersion := range artifact.Versions {
		if !isProcessable(artifactVersion.Tags) {
			continue
		}

		if tagMatcher.MatchTag(artifactVersion.Tags...) {
			sig, wflow, err := getSignatureAndWorkflowInVersion(
				ctx, i.ghCli, artifact.Owner, artifact.Name, artifactVersion.Sha)
			if err != nil {
				return nil, err
			}
			applicableArtifactVersions = append(applicableArtifactVersions, struct {
				Verification   any
				GithubWorkflow any
			}{sig, wflow})
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

	zerolog.Ctx(ctx).Debug().Any("result", result).Msg("ingestion result")

	if err != nil {
		return nil, err
	}
	// return the list of applicable artifact versions
	return result, nil
}

func isProcessable(tags []string) bool {
	if len(tags) == 0 {
		return false
	}

	for _, tag := range tags {
		if tag == "" {
			return false
		}
	}

	return true
}

func getSignatureAndWorkflowInVersion(
	ctx context.Context,
	client provifv1.GitHub,
	artifactOwnerLogin, artifactName, packageVersionName string,
) (*pb.SignatureVerification, *pb.GithubWorkflow, error) {
	// get the verifier for sigstore
	artifactVerifier, err := verifier.NewVerifier(
		verifier.VerifierSigstore,
		sigstore.SigstorePublicTrustedRootRepo,
		container.WithAccessToken(client.GetToken()), container.WithGitHubClient(client))
	if err != nil {
		return nil, nil, fmt.Errorf("error getting sigstore verifier: %w", err)
	}
	defer artifactVerifier.ClearCache()

	// now get information for signature and workflow
	res, err := artifactVerifier.Verify(ctx, verifier.ArtifactTypeContainer, "",
		artifactOwnerLogin, artifactName, packageVersionName)
	if err != nil {
		zerolog.Ctx(ctx).Debug().Err(err).Str("URI", res.URI).Msg("no signature information found")
	}

	return res.SignatureInfoProto(), res.WorkflowInfoProto(), nil
}
