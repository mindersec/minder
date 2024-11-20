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
	trusty "github.com/stacklok/trusty-sdk-go/pkg/v1/client"
	trustytypes "github.com/stacklok/trusty-sdk-go/pkg/v1/types"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/internal/constants"
	evalerrors "github.com/mindersec/minder/internal/engine/errors"
	"github.com/mindersec/minder/internal/engine/eval/pr_actions"
	"github.com/mindersec/minder/internal/engine/eval/templates"
	eoptions "github.com/mindersec/minder/internal/engine/options"
	pbinternal "github.com/mindersec/minder/internal/proto"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
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
func NewTrustyEvaluator(
	ctx context.Context,
	ghcli provifv1.GitHub,
	opts ...eoptions.Option,
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
	res *interfaces.Result,
) error {
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
			logger.Error().
				Err(err).
				Str("dependency_name", dep.Dep.Name).
				Str("dependency_version", dep.Dep.Version).
				Msg("error fetching trusty data")
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
func readPullRequestDependencies(res *interfaces.Result) (*pbinternal.PrDependencies, error) {
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
	trustyClient *trusty.Trusty,
	dep *pbinternal.PrDependencies_ContextualDependency,
) (*trustyReport, error) {
	// Call the Trusty API
	resp, err := trustyClient.Report(ctx, &trustytypes.Dependency{
		Name:      dep.Dep.Name,
		Version:   dep.Dep.Version,
		Ecosystem: trustytypes.Ecosystem(dep.Dep.Ecosystem),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	res := makeTrustyReport(dep, resp)

	return res, nil
}

func makeTrustyReport(
	dep *pbinternal.PrDependencies_ContextualDependency,
	resp *trustytypes.Reply,
) *trustyReport {
	res := &trustyReport{
		PackageName:     dep.Dep.Name,
		PackageVersion:  dep.Dep.Version,
		PackageType:     dep.Dep.Ecosystem.AsString(),
		TrustyURL:       makeTrustyURL(dep.Dep.Name, strings.ToLower(dep.Dep.Ecosystem.AsString())),
		Score:           resp.Summary.Score,
		IsDeprecated:    resp.PackageData.Deprecated,
		IsArchived:      resp.PackageData.Archived,
		ActivityScore:   getValueFromMap[float64](resp.Summary.Description, "activity"),
		ProvenanceScore: getValueFromMap[float64](resp.Summary.Description, "provenance"),
	}

	res.ScoreComponents = makeScoreComponents(resp.Summary.Description)
	res.Alternatives = makeAlternatives(dep.Dep.Ecosystem.AsString(), resp.Alternatives.Packages)

	if getValueFromMap[bool](resp.Summary.Description, "malicious") {
		res.Malicious = &malicious{
			Summary: resp.PackageData.Malicious.Summary,
			Details: preprocessDetails(resp.PackageData.Malicious.Details),
		}
	}

	res.Provenance = makeProvenance(resp.Provenance)

	return res
}

func makeScoreComponents(descr map[string]any) []scoreComponent {
	scoreComponents := make([]scoreComponent, 0)

	if descr == nil {
		return scoreComponents
	}

	caser := cases.Title(language.Und, cases.NoLower)
	for l, v := range descr {
		switch l {
		case "activity":
			l = "Package activity"
		case "activity_repo":
			l = "Repository activity"
		case "activity_user":
			l = "User activity"
		case "provenance_type":
			l = "Provenance"
		case "typosquatting":
			if f, ok := v.(float64); ok && f > 5.0 {
				// skip typosquatting entry
				continue
			}
			l = "Typosquatting"
			v = "⚠️ Dependency may be trying to impersonate a well known package"
		}

		// Note: if none of the cases above match, we still
		// add the value to the list along with its
		// capitalized label.

		scoreComponents = append(scoreComponents, scoreComponent{
			Label: fmt.Sprintf("%s%s", caser.String(l[0:1]), l[1:]),
			Value: v,
		})
	}

	return scoreComponents
}

func makeAlternatives(
	ecosystem string,
	trustyAlternatives []trustytypes.Alternative,
) []alternative {
	alternatives := []alternative{}
	for _, alt := range trustyAlternatives {
		alternatives = append(alternatives, alternative{
			PackageName: alt.PackageName,
			PackageType: ecosystem,
			Score:       &alt.Score,
			TrustyURL:   makeTrustyURL(alt.PackageName, ecosystem),
		})
	}

	return alternatives
}

func makeProvenance(
	trustyProvenance *trustytypes.Provenance,
) *provenance {
	if trustyProvenance == nil {
		return nil
	}

	prov := &provenance{}
	if trustyProvenance.Description.Historical.Overlap != 0 {
		prov.Historical = &historicalProvenance{
			Versions: int(trustyProvenance.Description.Historical.Versions),
			Tags:     int(trustyProvenance.Description.Historical.Tags),
			Common:   int(trustyProvenance.Description.Historical.Common),
			Overlap:  trustyProvenance.Description.Historical.Overlap,
		}
	}

	if trustyProvenance.Description.Sigstore.Issuer != "" {
		prov.Sigstore = &sigstoreProvenance{
			SourceRepository: trustyProvenance.Description.Sigstore.SourceRepository,
			Workflow:         trustyProvenance.Description.Sigstore.Workflow,
			Issuer:           trustyProvenance.Description.Sigstore.Issuer,
			RekorURI:         trustyProvenance.Description.Sigstore.Transparency,
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

func getValueFromMap[T any](coll map[string]any, field string) T {
	var t T
	v, ok := coll[field]
	if !ok {
		return t
	}
	res, ok := v.(T)
	if !ok {
		return t
	}
	return res
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

	if ecoConfig.Score > packageScore {
		reasons = append(reasons, TRUSTY_LOW_SCORE)
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
