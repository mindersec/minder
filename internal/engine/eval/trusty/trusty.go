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
	trusty "github.com/stacklok/trusty-sdk-go/pkg/client"
	trustytypes "github.com/stacklok/trusty-sdk-go/pkg/types"

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
	client   *trusty.Trusty
}

// NewTrustyEvaluator creates a new trusty evaluator
func NewTrustyEvaluator(ctx context.Context, ghcli provifv1.GitHub) (*Evaluator, error) {
	if ghcli == nil {
		return nil, fmt.Errorf("provider builder is nil")
	}

	// Read the trusty endpoint from the environment
	trustyEndpoint := os.Getenv(trustyEndpointEnvVar)

	// If the environment variable is not set, use the default endpoint
	if trustyEndpoint == "" {
		trustyEndpoint = trusty.DefaultOptions.BaseURL
		zerolog.Ctx(ctx).Info().Str("trusty-endpoint", trustyEndpoint).Msg("using default trusty endpoint")
	} else {
		zerolog.Ctx(ctx).Info().Str("trusty-endpoint", trustyEndpoint).Msg("using trusty endpoint from environment")
	}

	trustyClient := trusty.NewWithOptions(trusty.Options{
		HttpClient: trusty.DefaultOptions.HttpClient,
		BaseURL:    trustyEndpoint,
	})

	return &Evaluator{
		cli:      ghcli,
		endpoint: trustyEndpoint,
		client:   trustyClient,
	}, nil
}

// Eval implements the Evaluator interface.
func (e *Evaluator) Eval(ctx context.Context, pol map[string]any, res *engif.Result) error {
	// Extract the dependency list from the PR
	prDependencies, err := readPullRequestDependencies(res)
	if err != nil {
		return fmt.Errorf("reading pull request dependencies: %w", err)
	}
	if len(prDependencies.Deps) == 0 {
		return nil
	}

	logger := zerolog.Ctx(ctx).With().
		Int64("pull-number", prDependencies.Pr.Number).
		Str("repo-owner", prDependencies.Pr.RepoOwner).
		Str("repo-name", prDependencies.Pr.RepoName).Logger()

	// Parse the profile data to get the policy configuration
	ruleConfig, err := parseRuleConfig(pol)
	if err != nil {
		return fmt.Errorf("parsing policy configuration: %w", err)
	}

	prSummaryHandler, err := newSummaryPrHandler(prDependencies.Pr, e.cli, e.endpoint)
	if err != nil {
		return fmt.Errorf("failed to create summary handler: %w", err)
	}

	// Classify all dependencies, tracking all that are malicious or scored low
	for _, dep := range prDependencies.Deps {
		depscore, err := getDependencyScore(ctx, e.client, dep)
		if err != nil {
			logger.Error().Msgf("error fetching trusty data: %s", err)
			return fmt.Errorf("getting dependency score: %w", err)
		}

		if depscore == nil || depscore.PackageName == "" {
			logger.Info().
				Str("dependency", dep.Dep.Name).
				Msgf("no trusty data for dependency, skipping")
			continue
		}

		classifyDependency(ctx, &logger, depscore, ruleConfig, prSummaryHandler, dep)
	}

	// If there are no problematic dependencies, return here
	if len(prSummaryHandler.trackedAlternatives) == 0 {
		logger.Debug().Msgf("no action, no packages tracked")
		return nil
	}

	if err := submitSummary(ctx, prSummaryHandler, ruleConfig); err != nil {
		logger.Err(err).Msgf("Failed generating PR summary: %s", err.Error())
		return fmt.Errorf("submitting pull request summary: %w", err)
	}

	return buildEvalResult(prSummaryHandler)
}

func getEcosystemConfig(
	logger *zerolog.Logger, ruleConfig *config, dep *pb.PrDependencies_ContextualDependency,
) *ecosystemConfig {
	ecoConfig := ruleConfig.getEcosystemConfig(dep.Dep.Ecosystem)
	if ecoConfig == nil {
		logger.Info().
			Str("dependency", dep.Dep.Name).
			Str("ecosystem", dep.Dep.Ecosystem.AsString()).
			Msgf("no config for ecosystem, skipping")
		return nil
	}
	return ecoConfig
}

// readPullRequestDependencies returns the dependencies found in theingestion results
func readPullRequestDependencies(res *engif.Result) (*pb.PrDependencies, error) {
	prdeps, ok := res.Object.(*pb.PrDependencies)
	if !ok {
		return nil, fmt.Errorf("object type incompatible with the Trusty evaluator")
	}

	return prdeps, nil
}

// parseRuleConfig parses the profile configuration to build the policy
func parseRuleConfig(pol map[string]any) (*config, error) {
	ruleConfig, err := parseConfig(pol)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if ruleConfig.Action != pr_actions.ActionSummary && ruleConfig.Action != pr_actions.ActionReviewPr {
		return nil, fmt.Errorf("action %s is not implemented", ruleConfig.Action)
	}

	return ruleConfig, nil
}

// submitSummary submits the pull request summary. It will return an error if
// something fails.
func submitSummary(ctx context.Context, prSummary *summaryPrHandler, ruleConfig *config) error {
	if err := prSummary.submit(ctx, ruleConfig); err != nil {
		return fmt.Errorf("failed to submit summary: %w", err)
	}
	return nil
}

// buildEvalResult returns nil or an EvaluationError with details about the
// bad dependencies found by Trusty if any are found.
func buildEvalResult(prSummary *summaryPrHandler) error {
	// If we have malicious or lowscored packages, the evaluation fails.
	// Craft an evaluation failed error with the dependency data:
	var lowScoringPackages, maliciousPackages []string
	for _, d := range prSummary.trackedAlternatives {
		if d.trustyReply.PackageData.Malicious != nil &&
			d.trustyReply.PackageData.Malicious.Published != nil &&
			d.trustyReply.PackageData.Malicious.Published.String() != "" {
			maliciousPackages = append(maliciousPackages, d.trustyReply.PackageName)
		} else {
			lowScoringPackages = append(lowScoringPackages, d.trustyReply.PackageName)
		}
	}

	failedEvalMsg := ""

	if len(maliciousPackages) > 0 {
		failedEvalMsg = fmt.Sprintf(
			"%d malicious packages: %s",
			len(maliciousPackages), strings.Join(maliciousPackages, ","),
		)
	}

	if len(lowScoringPackages) > 0 {
		if failedEvalMsg != "" {
			failedEvalMsg += " and "
		}
		failedEvalMsg += fmt.Sprintf(
			"%d packages with low score: %s",
			len(lowScoringPackages), strings.Join(lowScoringPackages, ","),
		)
	}

	if failedEvalMsg != "" {
		return evalerrors.NewErrEvaluationFailed(failedEvalMsg)
	}

	return nil
}

func getDependencyScore(
	ctx context.Context, trustyClient *trusty.Trusty, dep *pb.PrDependencies_ContextualDependency,
) (*trustytypes.Reply, error) {
	// Call the Trusty API
	resp, err := trustyClient.Report(ctx, &trustytypes.Dependency{
		Name:      dep.Dep.Name,
		Version:   dep.Dep.Version,
		Ecosystem: trustytypes.Ecosystem(dep.Dep.Ecosystem),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	return resp, nil
}

// classifyDependency checks the dependencies from the PR for maliciousness or
// low scores and adds them to the summary if needed
func classifyDependency(
	_ context.Context, logger *zerolog.Logger, resp *trustytypes.Reply, ruleConfig *config,
	prSummary *summaryPrHandler, dep *pb.PrDependencies_ContextualDependency,
) {
	// Check all the policy violations
	reasons := []RuleViolationReason{}

	// shouldBlockPR indicates if the PR should beblocked based on this dep
	shouldBlockPR := false

	ecoConfig := getEcosystemConfig(logger, ruleConfig, dep)
	if ecoConfig == nil {
		return
	}

	// If the package is malicious, ensure that the score is 0 to avoid it
	// getting ignored from the report
	if resp.PackageData.Malicious != nil && resp.PackageData.Malicious.Summary != "" {
		logger.Debug().
			Str("dependency", fmt.Sprintf("%s@%s", dep.Dep.Name, dep.Dep.Version)).
			Str("malicious", "true").
			Msgf("malicious dependency")

		if !ecoConfig.AllowMalicious {
			shouldBlockPR = true
		}

		reasons = append(reasons, TRUSTY_MALICIOUS_PKG)
	}

	// Note if the packages is deprecated
	if resp.PackageData.Deprecated {
		logger.Debug().
			Str("dependency", fmt.Sprintf("%s@%s", dep.Dep.Name, dep.Dep.Version)).
			Str("deprecated", "true").
			Msgf("deprecated dependency")

		if !ecoConfig.AllowDeprecated {
			shouldBlockPR = true
		}

		reasons = append(reasons, TRUSTY_DEPRECATED)
	}

	packageScore := float64(0)
	if resp.Summary.Score != nil {
		packageScore = *resp.Summary.Score
	}

	descr := readPackageDescription(resp)

	if ecoConfig.Score > packageScore {
		reasons = append(reasons, TRUSTY_LOW_SCORE)
	}

	if ecoConfig.Provenance > descr["provenance"].(float64) && descr["provenance"].(float64) > 0 {
		reasons = append(reasons, TRUSTY_LOW_PROVENANCE)
	}

	if ecoConfig.Activity > descr["activity"].(float64) && descr["activity"].(float64) > 0 {
		reasons = append(reasons, TRUSTY_LOW_ACTIVITY)
	}

	if len(reasons) > 0 {
		logger.Debug().
			Str("dependency", dep.Dep.Name).
			Float64("score", packageScore).
			Float64("threshold", ecoConfig.Score).
			Msgf("the dependency has lower score than threshold or is malicious, tracking")

		prSummary.trackAlternatives(dependencyAlternatives{
			Dependency:  dep.Dep,
			Reasons:     reasons,
			BlockPR:     shouldBlockPR,
			trustyReply: resp,
		})
	} else {
		logger.Debug().
			Str("dependency", dep.Dep.Name).
			Float64("score", *resp.Summary.Score).
			Float64("threshold", ecoConfig.Score).
			Msgf("dependency ok")
	}
}

// readPackageDescription reads the description from the package summary and
// normlizes the required values when missing from a partial Trusty response
func readPackageDescription(resp *trustytypes.Reply) map[string]any {
	descr := map[string]any{}
	if resp == nil {
		resp = &trustytypes.Reply{}
	}
	if resp.Summary.Description != nil {
		descr = resp.Summary.Description
	}

	// Ensure don't panic checking all fields are there
	for _, fld := range []string{"activity", "provenance"} {
		if _, ok := descr[fld]; !ok || descr[fld] == nil {
			descr[fld] = float64(0)
		}
	}
	return descr
}
