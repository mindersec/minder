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

// Package trusty provides an evaluator that uses the trusty API
package trusty

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"

	evalerrors "github.com/stacklok/minder/internal/engine/errors"
	"github.com/stacklok/minder/internal/engine/eval/pr_actions"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

const (
	// TrustyEvalType is the type of the trusty evaluator
	TrustyEvalType       = "trusty"
	trustyEndpointURL    = "https://api.trustypkg.dev"
	trustyEndpointEnvVar = "MINDER_UNSTABLE_TRUSTY_ENDPOINT"
)

// Evaluator is the trusty evaluator
type Evaluator struct {
	cli      provifv1.GitHub
	endpoint string
}

// NewTrustyEvaluator creates a new trusty evaluator
func NewTrustyEvaluator(ctx context.Context, provider provifv1.Provider) (*Evaluator, error) {
	if provider == nil {
		return nil, fmt.Errorf("provider builder is nil")
	}

	// Read the trusty endpoint from the environment
	trustyEndpoint := os.Getenv(trustyEndpointEnvVar)
	// If the environment variable is not set, use the default endpoint
	if trustyEndpoint == "" {
		trustyEndpoint = trustyEndpointURL
		zerolog.Ctx(ctx).Info().Str("trusty-endpoint", trustyEndpoint).Msg("using default trusty endpoint")
	} else {
		zerolog.Ctx(ctx).Info().Str("trusty-endpoint", trustyEndpoint).Msg("using trusty endpoint from environment")
	}

	ghcli, err := provifv1.As[provifv1.GitHub](provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get github client: %w", err)
	}

	return &Evaluator{
		cli:      ghcli,
		endpoint: trustyEndpoint,
	}, nil
}

// Eval implements the Evaluator interface.
//
//nolint:gocyclo
func (e *Evaluator) Eval(ctx context.Context, pol map[string]any, res *engif.Result) error {
	var lowScoringPackages []string

	//nolint:govet
	prdeps, ok := res.Object.(*pb.PrDependencies)
	if !ok {
		return fmt.Errorf("invalid object type for vulncheck evaluator")
	}

	if len(prdeps.Deps) == 0 {
		return nil
	}

	ruleConfig, err := parseConfig(pol)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if !isActionImplemented(ruleConfig.Action) {
		return fmt.Errorf("action %s is not implemented", ruleConfig.Action)
	}

	logger := zerolog.Ctx(ctx).With().
		Int64("pull-number", prdeps.Pr.Number).
		Str("repo-owner", prdeps.Pr.RepoOwner).
		Str("repo-name", prdeps.Pr.RepoName).
		Logger()

	prSummaryHandler, err := newSummaryPrHandler(prdeps.Pr, e.cli, e.endpoint)
	if err != nil {
		return fmt.Errorf("failed to create summary handler: %w", err)
	}

	piCli := newPiClient(e.endpoint)
	if piCli == nil {
		return fmt.Errorf("failed to create pi client")
	}

	for _, dep := range prdeps.Deps {
		ecoConfig := ruleConfig.getEcosystemConfig(dep.Dep.Ecosystem)
		if ecoConfig == nil {
			logger.Info().
				Str("dependency", dep.Dep.Name).
				Str("ecosystem", dep.Dep.Ecosystem.AsString()).
				Msgf("no config for ecosystem, skipping")
			continue
		}

		resp, err := piCli.SendRecvRequest(ctx, dep.Dep)
		if err != nil {
			return fmt.Errorf("failed to send request: %w", err)
		}

		if resp == nil || resp.PackageName == "" {
			logger.Info().
				Str("dependency", dep.Dep.Name).
				Msgf("no trusty data for dependency, skipping")
			continue
		}

		if resp.Summary.Score == 0 {
			logger.Info().
				Str("dependency", dep.Dep.Name).
				Msgf("the dependency has no score, skipping")
			continue
		}

		s, err := ecoConfig.getScore(resp.Summary)
		if err != nil {
			return fmt.Errorf("failed to get score: %w", err)
		}

		if s >= ecoConfig.Score {
			logger.Debug().
				Str("dependency", dep.Dep.Name).
				Str("score-source", ecoConfig.getScoreSource()).
				Float64("score", s).
				Float64("threshold", ecoConfig.Score).
				Msgf("the dependency has higher score than threshold, skipping")
			continue
		}

		logger.Debug().
			Str("dependency", dep.Dep.Name).
			Str("score-source", ecoConfig.getScoreSource()).
			Float64("score", s).
			Float64("threshold", ecoConfig.Score).
			Msgf("the dependency has lower score than threshold, tracking")

		lowScoringPackages = append(lowScoringPackages, dep.Dep.Name)

		prSummaryHandler.trackAlternatives(dep, resp)
	}

	if err := prSummaryHandler.submit(ctx); err != nil {
		return fmt.Errorf("failed to submit summary: %w", err)
	}

	if len(lowScoringPackages) > 0 {
		return evalerrors.NewErrEvaluationFailed(fmt.Sprintf("packages with low score: %s", strings.Join(lowScoringPackages, ",")))
	}
	return nil
}

func isActionImplemented(action pr_actions.Action) bool {
	return action == pr_actions.ActionSummary
}
