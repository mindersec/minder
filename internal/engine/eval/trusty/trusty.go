// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package trusty provides an evaluator that uses the trusty API
package trusty

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/rs/zerolog"
	trusty "github.com/stacklok/trusty-sdk-go/pkg/v2/client"
	trustytypes "github.com/stacklok/trusty-sdk-go/pkg/v2/types"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/internal/constants"
	evalerrors "github.com/mindersec/minder/internal/engine/errors"
	"github.com/mindersec/minder/internal/engine/eval/pr_actions"
	"github.com/mindersec/minder/internal/engine/eval/templates"
	pbinternal "github.com/mindersec/minder/internal/proto"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

const (
	// TrustyEvalType is the type of the trusty evaluator
	TrustyEvalType       = "trusty"
	trustyEndpointURL    = "https://api.trustypkg.dev"
	trustyEndpointEnvVar = "MINDER_UNSTABLE_TRUSTY_ENDPOINT"
)

// Evaluator is the trusty evaluator
type Evaluator struct {
	cli      interfaces.GitHubIssuePRClient
	endpoint string
	client   trusty.Trusty
}

// NewTrustyEvaluator creates a new trusty evaluator
func NewTrustyEvaluator(
	ctx context.Context,
	ghcli interfaces.GitHubIssuePRClient,
	opts ...interfaces.Option,
) (*Evaluator, error) {
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
		BaseURL: trustyEndpoint,
	})

	evaluator := &Evaluator{
		cli:      ghcli,
		endpoint: trustyEndpoint,
		client:   trustyClient,
	}

	for _, opt := range opts {
		if err := opt(evaluator); err != nil {
			return nil, err
		}
	}

	return evaluator, nil
}

// Eval implements the Evaluator interface.
func (e *Evaluator) Eval(
	ctx context.Context,
	pol map[string]any,
	_ protoreflect.ProtoMessage,
	res *interfaces.Ingested,
) (*interfaces.EvaluationResult, error) {
	// Extract the dependency list from the PR
	prDependencies, err := readPullRequestDependencies(res)
	if err != nil {
		return nil, fmt.Errorf("reading pull request dependencies: %w", err)
	}
	if len(prDependencies.Deps) == 0 {
		return &interfaces.EvaluationResult{}, nil
	}

	logger := zerolog.Ctx(ctx).With().
		Int64("pull-number", prDependencies.Pr.Number).
		Str("repo-owner", prDependencies.Pr.RepoOwner).
		Str("repo-name", prDependencies.Pr.RepoName).Logger()

	// Parse the profile data to get the policy configuration
	ruleConfig, err := parseRuleConfig(pol)
	if err != nil {
		return nil, fmt.Errorf("parsing policy configuration: %w", err)
	}

	prSummaryHandler, err := newSummaryPrHandler(prDependencies.Pr, e.cli, e.endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create summary handler: %w", err)
	}

	// Classify all dependencies, tracking all that are malicious or scored low
	for _, dep := range prDependencies.Deps {
		depscore, err := getDependencyScore(ctx, e.client, dep)
		if err != nil {
			logger.Error().
				Err(err).
				Str("dependency_name", dep.Dep.Name).
				Str("dependency_version", dep.Dep.Version).
				Msg("error fetching trusty data")
			return nil, fmt.Errorf("getting dependency score: %w", err)
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
		return &interfaces.EvaluationResult{}, nil
	}

	if err := submitSummary(ctx, prSummaryHandler, ruleConfig); err != nil {
		logger.Err(err).Msgf("Failed generating PR summary: %s", err.Error())
		return nil, fmt.Errorf("submitting pull request summary: %w", err)
	}

	err = buildEvalResult(prSummaryHandler)
	if err != nil {
		return nil, err
	}

	return &interfaces.EvaluationResult{}, nil
}

func getEcosystemConfig(
	logger *zerolog.Logger, ruleConfig *config, dep *pbinternal.PrDependencies_ContextualDependency,
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
func readPullRequestDependencies(res *interfaces.Ingested) (*pbinternal.PrDependencies, error) {
	prdeps, ok := res.Object.(*pbinternal.PrDependencies)
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
		if d.trustyReply.Malicious != nil {
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

	if len(maliciousPackages) > 0 || len(lowScoringPackages) > 0 {
		return evalerrors.NewDetailedErrEvaluationFailed(
			templates.TrustyTemplate,
			map[string]any{"maliciousPackages": maliciousPackages, "lowScoringPackages": lowScoringPackages},
			"%s",
			failedEvalMsg,
		)
	}

	return nil
}

type trustyReport struct {
	PackageName     string
	PackageType     string
	PackageVersion  string
	TrustyURL       string
	IsDeprecated    bool
	IsArchived      bool
	Score           *float64
	ActivityScore   float64
	ProvenanceScore float64
	ScoreComponents []scoreComponent
	Alternatives    []alternative
	Provenance      *provenance
	Malicious       *malicious
}

type scoreComponent struct {
	Label string
	Value any
}

type provenance struct {
	Historical *historicalProvenance
	Sigstore   *sigstoreProvenance
}

type historicalProvenance struct {
	Versions int
	Tags     int
	Common   int
	Overlap  float64
}

type sigstoreProvenance struct {
	SourceRepository string
	Workflow         string
	Issuer           string
	RekorURI         string
}

type malicious struct {
	Summary string
	Details string
}

type alternative struct {
	PackageName string
	PackageType string
	Score       *float64
	TrustyURL   string
}

func getDependencyScore(
	ctx context.Context,
	trustyClient trusty.Trusty,
	dep *pbinternal.PrDependencies_ContextualDependency,
) (*trustyReport, error) {
	// Call the Trusty API
	packageType := dep.Dep.Ecosystem.AsString()
	input := &trustytypes.Dependency{
		PackageName:    dep.Dep.Name,
		PackageType:    packageType,
		PackageVersion: &dep.Dep.Version,
	}

	summary := make(chan *trustytypes.PackageSummaryAnnotation, 1)
	metadata := make(chan *trustytypes.TrustyPackageData, 1)
	alternatives := make(chan *trustytypes.PackageAlternatives, 1)
	provenance := make(chan *trustytypes.Provenance, 1)
	errors := make(chan error)

	defer func() {
		close(summary)
		close(metadata)
		close(alternatives)
		close(provenance)
		close(errors)
	}()

	go func() {
		resp, err := trustyClient.Summary(ctx, input)
		errors <- err
		summary <- resp
	}()

	go func() {
		resp, err := trustyClient.PackageMetadata(ctx, input)
		errors <- err
		metadata <- resp
	}()

	go func() {
		resp, err := trustyClient.Alternatives(ctx, input)
		errors <- err
		alternatives <- resp
	}()

	go func() {
		resp, err := trustyClient.Provenance(ctx, input)
		errors <- err
		provenance <- resp
	}()

	// Beware of the magic number 4, which is the number of
	// asynchronous calls fired in the previous lines. This must
	// be kept in sync.
	for i := 0; i < 4; i++ {
		err := <-errors
		if err != nil {
			return nil, fmt.Errorf("trusty call failed: %w", err)
		}
	}
	respSummary := <-summary
	respPkg := <-metadata
	respAlternatives := <-alternatives
	respProvenance := <-provenance

	res := makeTrustyReport(dep,
		*respSummary,
		*respPkg,
		*respAlternatives,
		*respProvenance,
	)

	return res, nil
}

func makeTrustyReport(
	dep *pbinternal.PrDependencies_ContextualDependency,
	respSummary trustytypes.PackageSummaryAnnotation,
	respPkg trustytypes.TrustyPackageData,
	respAlternatives trustytypes.PackageAlternatives,
	respProvenance trustytypes.Provenance,
) *trustyReport {
	res := &trustyReport{
		PackageName:    dep.Dep.Name,
		PackageVersion: dep.Dep.Version,
		PackageType:    dep.Dep.Ecosystem.AsString(),
		TrustyURL:      makeTrustyURL(dep.Dep.Name, strings.ToLower(dep.Dep.Ecosystem.AsString())),
	}

	addSummaryDetails(res, respSummary)
	addMetadataDetails(res, respPkg)

	res.ScoreComponents = makeScoreComponents(respSummary.Description)
	res.Alternatives = makeAlternatives(dep.Dep.Ecosystem.AsString(), respAlternatives.Packages)

	if respSummary.Description.Malicious {
		res.Malicious = makeMaliciousDetails(respPkg.Malicious)
	}

	res.Provenance = makeProvenance(respProvenance)

	return res
}

func addSummaryDetails(res *trustyReport, resp trustytypes.PackageSummaryAnnotation) {
	res.Score = resp.Score
	res.ActivityScore = resp.Description.Activity
	res.ProvenanceScore = resp.Description.Provenance
}

func addMetadataDetails(res *trustyReport, resp trustytypes.TrustyPackageData) {
	res.IsDeprecated = resp.IsDeprecated != nil && *resp.IsDeprecated
	res.IsArchived = resp.Archived != nil && *resp.Archived
}

func makeScoreComponents(resp trustytypes.SummaryDescription) []scoreComponent {
	scoreComponents := make([]scoreComponent, 0)

	// activity scores
	if resp.Activity != 0 {
		scoreComponents = append(scoreComponents, scoreComponent{
			Label: "Package activity",
			Value: resp.Activity,
		})
	}
	if resp.ActivityRepo != 0 {
		scoreComponents = append(scoreComponents, scoreComponent{
			Label: "Repository activity",
			Value: resp.ActivityRepo,
		})
	}
	if resp.ActivityUser != 0 {
		scoreComponents = append(scoreComponents, scoreComponent{
			Label: "User activity",
			Value: resp.ActivityUser,
		})
	}

	// provenance information
	if resp.ProvenanceType != nil {
		scoreComponents = append(scoreComponents, scoreComponent{
			Label: "Provenance",
			Value: string(*resp.ProvenanceType),
		})
	}

	// typosquatting information
	if resp.TypoSquatting != 0 && resp.TypoSquatting <= 5.0 {
		scoreComponents = append(scoreComponents, scoreComponent{
			Label: "Typosquatting",
			Value: "⚠️ Dependency may be trying to impersonate a well known package",
		})
	}

	// Note: in the previous implementation based on Trusty v1
	// API, if new fields were added to the `"description"` field
	// of a package they were implicitly added to the table of
	// score components.
	//
	// This was possible because the `Description` field of the go
	// struct was defined as `map[string]any`.
	//
	// This is not the case with v2 API, so we need to keep track
	// of new measures being added to the API.

	return scoreComponents
}

func makeAlternatives(
	ecosystem string,
	trustyAlternatives []*trustytypes.PackageBasicInfo,
) []alternative {
	alternatives := []alternative{}
	for _, alt := range trustyAlternatives {
		alternatives = append(alternatives, alternative{
			PackageName: alt.PackageName,
			PackageType: ecosystem,
			Score:       alt.Score,
			TrustyURL:   makeTrustyURL(alt.PackageName, ecosystem),
		})
	}

	return alternatives
}

func makeMaliciousDetails(
	maliciousInfo *trustytypes.PackageMaliciousPayload,
) *malicious {
	return &malicious{
		Summary: maliciousInfo.Summary,
		Details: preprocessDetails(maliciousInfo.Details),
	}
}

func makeProvenance(
	resp trustytypes.Provenance,
) *provenance {
	prov := &provenance{}
	if resp.Historical.Overlap != 0 {
		prov.Historical = &historicalProvenance{
			Versions: int(resp.Historical.Versions),
			Tags:     int(resp.Historical.Tags),
			Common:   int(resp.Historical.Common),
			Overlap:  resp.Historical.Overlap,
		}
	}

	if resp.Sigstore.Issuer != "" {
		prov.Sigstore = &sigstoreProvenance{
			SourceRepository: resp.Sigstore.SourceRepo,
			Workflow:         resp.Sigstore.Workflow,
			Issuer:           resp.Sigstore.Issuer,
			RekorURI:         resp.Sigstore.Transparency,
		}
	}

	return prov
}

func makeTrustyURL(packageName string, ecosystem string) string {
	trustyURL, _ := url.JoinPath(
		constants.TrustyHttpURL,
		"report",
		strings.ToLower(ecosystem),
		url.PathEscape(packageName))
	return trustyURL
}

// classifyDependency checks the dependencies from the PR for maliciousness or
// low scores and adds them to the summary if needed
func classifyDependency(
	_ context.Context,
	logger *zerolog.Logger,
	resp *trustyReport,
	ruleConfig *config,
	prSummary *summaryPrHandler,
	dep *pbinternal.PrDependencies_ContextualDependency,
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
	if resp.Malicious != nil && resp.Malicious.Summary != "" {
		logger.Debug().
			Str("dependency", fmt.Sprintf("%s@%s", dep.Dep.Name, dep.Dep.Version)).
			Str("malicious", "true").
			Msgf("malicious dependency")

		if !ecoConfig.AllowMalicious {
			shouldBlockPR = true
		}

		reasons = append(reasons, TRUSTY_MALICIOUS_PKG)
	}

	// Note if the packages is deprecated or archived
	if resp.IsDeprecated || resp.IsArchived {
		logger.Debug().
			Str("dependency", fmt.Sprintf("%s@%s", dep.Dep.Name, dep.Dep.Version)).
			Bool("deprecated", resp.IsDeprecated).
			Bool("archived", resp.IsArchived).
			Msgf("deprecated dependency")

		if !ecoConfig.AllowDeprecated {
			shouldBlockPR = true
		}

		reasons = append(reasons, TRUSTY_DEPRECATED)
	} else {
		logger.Debug().
			Str("dependency", fmt.Sprintf("%s@%s", dep.Dep.Name, dep.Dep.Version)).
			Bool("deprecated", resp.IsDeprecated).
			Msgf("not deprecated dependency")
	}

	packageScore := float64(0)
	if resp.Score != nil {
		packageScore = *resp.Score
	}

	if ecoConfig.Provenance > resp.ProvenanceScore && resp.ProvenanceScore > 0 {
		reasons = append(reasons, TRUSTY_LOW_PROVENANCE)
	}

	if ecoConfig.Activity > resp.ActivityScore && resp.ActivityScore > 0 {
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
			Float64("threshold", ecoConfig.Score).
			Msgf("dependency ok")
	}
}
