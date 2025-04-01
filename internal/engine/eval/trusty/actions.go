// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package trusty provides an evaluator that uses the trusty API
package trusty

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/url"
	"slices"
	"strings"
	template "text/template"

	"github.com/google/go-github/v63/github"
	"github.com/rs/zerolog"

	"github.com/mindersec/minder/internal/constants"
	"github.com/mindersec/minder/internal/engine/eval/pr_actions"
	pbinternal "github.com/mindersec/minder/internal/proto"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

const (
	// nolint:lll
	commentTemplate = `{{- if .Malicious -}}
### ⚠️ MALICIOUS PACKAGES ⚠️

Minder has detected that this pull request is introducing malicious software dependencies using data from [Stacklok Insight](https://www.trustypkg.dev/):

| Package | Summary | Details |
| --- | --- | --- |
{{ range .Malicious -}}
| [{{ .PackageName }}]({{ .TrustyURL }}) | {{ .Summary }} | {{ .Details }} |
{{ end }}
{{ end }}

{{- if .Dependencies -}}
### <ins>Dependency Information</ins>

Minder analyzed the dependencies introduced in this pull request and detected that some dependencies do not meet your security profile.
{{ range .Dependencies }}

### 📦 Dependency: [{{ .PackageName }}]({{ .TrustyURL }})

{{ if .Archived }}
⚠️ __Archived Package:__ This package is marked as deprecated. Proceed with caution!

Archived packages are no longer updated or maintained. This can lead to security vulnerabilities and compatibility issues.
{{ end }}
{{ if .Deprecated }}
⚠️ __Deprecated Package:__ This package is marked as archived. Proceed with caution!
{{ end }}
{{ if .ScoreComponents }}
<details>
  <summary>Scoring details</summary>

  | Component           | Score |
  | ------------------- | ----: |
  {{ range .ScoreComponents -}}
  | {{ .Label }} | {{ .Value }}  |
{{ end }}
</details>
{{ end }}
{{ if .Provenance }}
<details>
  <summary>Proof of Origin (Provenance)</summary>
  {{ if .Provenance.Historical }}

  __This package can be linked back to its source code using a historical provenance map.__

  We were able to correlate a significant number of git tags and tagged releases in this package’s source code to versions of the published package. This mapping creates a strong link from the package back to its source code repository, verifying proof of origin.

  |              |   |
  | ------------------- | ----: |
  | Published package versions | {{ .Provenance.Historical.NumVersions }} |
  | Number of git tags or releases | {{ .Provenance.Historical.NumTags }}
  | Versions matched to tags or releases | {{ .Provenance.Historical.MatchedVersions }} |

  {{- end }}
  {{ if .Provenance.Sigstore }}

  __This package has been digitally signed using sigtore.__

  |              |   |
  | ------------------- | ----: |
  | Source repository | {{ .Provenance.Sigstore.SourceRepository }} |
  | Cerificate Issuer | {{ .Provenance.Sigstore.Issuer }} |
  | GitHub action workflow | {{ .Provenance.Sigstore.Workflow }} |
  | Rekor (public ledger) entry | {{ .Provenance.Sigstore.RekorURI }} |
  {{- end }}
</details>
{{- end -}}
{{ if .Alternatives }}
<details>
  <summary>Alternatives</summary>

  | Package             | Description |
  | ------------------- | ----------- |
{{ range .Alternatives -}}
  | [{{ .PackageName }}]({{ .TrustyURL }})  | {{ .Summary }} |
{{ end }}
</details>
{{- end -}}
{{- end -}}
{{- end -}}
`
)

// RuleViolationReason are int constants that captures the various
// reasons a package was considered unsafe when compared with trusty data
type RuleViolationReason int

const (
	// TRUSTY_LOW_SCORE Overall score was lower than threshold
	TRUSTY_LOW_SCORE RuleViolationReason = iota + 1

	// TRUSTY_MALICIOUS_PKG Package is marked as malicious
	TRUSTY_MALICIOUS_PKG

	// TRUSTY_LOW_ACTIVITY The package does not have enough activity
	TRUSTY_LOW_ACTIVITY

	// TRUSTY_LOW_PROVENANCE Low trust in proof of origin
	TRUSTY_LOW_PROVENANCE

	// TRUSTY_DEPRECATED means a package was marked upstream as deprecated or archived
	TRUSTY_DEPRECATED
)

type templatePackageData struct {
	Ecosystem   string
	PackageName string
	TrustyURL   string
	Score       float64
}

type maliciousTemplateData struct {
	templatePackageData
	Summary string
	Details string
}

type templatePackage struct {
	templatePackageData
	Deprecated      bool
	Archived        bool
	ScoreComponents []templateScoreComponent
	Alternatives    []templateAlternative
	Provenance      *templateProvenance
}

type templateProvenance struct {
	Historical *templateHistoricalProvenance
	Sigstore   *templateSigstoreProvenance
}

type templateHistoricalProvenance struct {
	NumVersions     int
	NumTags         int
	MatchedVersions int
}

type templateSigstoreProvenance struct {
	SourceRepository string
	Workflow         string
	Issuer           string
	RekorURI         string
}

type templateAlternative struct {
	templatePackageData
	Summary string
}

type templateScoreComponent struct {
	Label string
	Value any
}

type dependencyAlternatives struct {
	Dependency *pbinternal.Dependency

	// Reason captures the reason why a package was flagged
	Reasons []RuleViolationReason

	// BlockPR will cause the PR to be blocked as requesting changes when true
	BlockPR bool

	// trustyReply is the complete response from trusty for this package
	trustyReply *trustyReport
}

// summaryPrHandler is a prStatusHandler that adds a summary text to the PR as a comment.
type summaryPrHandler struct {
	cli       provifv1.GitHub
	pr        *pbinternal.PullRequest
	trustyUrl string

	trackedAlternatives []dependencyAlternatives
	commentTemplate     *template.Template
}

func (sph *summaryPrHandler) trackAlternatives(dep dependencyAlternatives) {
	sph.trackedAlternatives = append(sph.trackedAlternatives, dep)
}

func (sph *summaryPrHandler) submit(ctx context.Context, ruleConfig *config) error {
	if len(sph.trackedAlternatives) == 0 {
		zerolog.Ctx(ctx).Info().Msgf(
			"trusty flagged no dependencies in pull request %s/%s#%d",
			sph.pr.RepoOwner, sph.pr.RepoName, sph.pr.Number,
		)
		return nil
	}

	zerolog.Ctx(ctx).Debug().Msgf(
		"trusty flagged %d dependencies in pull request %s/%s#%d",
		len(sph.trackedAlternatives), sph.pr.RepoOwner, sph.pr.RepoName, sph.pr.Number,
	)

	summary, err := sph.generateSummary()
	if err != nil {
		return fmt.Errorf("could not generate summary: %w", err)
	}

	action := ruleConfig.Action

	// Check all the tracked dependencies. If any of them call for the PR
	// to be blocked, set the review action to REQUEST_CHANGES
	var reviewAction = "COMMENT"
	for _, d := range sph.trackedAlternatives {
		if d.BlockPR {
			reviewAction = "REQUEST_CHANGES"
			break
		}
	}

	switch action {
	case pr_actions.ActionReviewPr:
		_, err = sph.cli.CreateReview(
			ctx, sph.pr.GetRepoOwner(), sph.pr.GetRepoName(), int(sph.pr.GetNumber()),
			&github.PullRequestReviewRequest{
				NodeID:   new(string),
				CommitID: &sph.pr.CommitSha,
				Body:     &summary,
				Event:    github.String(reviewAction),
				Comments: []*github.DraftReviewComment{},
			},
		)
		if err != nil {
			return fmt.Errorf("submitting pr summary: %w", err)
		}
	case pr_actions.ActionSummary:
		_, err = sph.cli.CreateIssueComment(ctx, sph.pr.GetRepoOwner(), sph.pr.GetRepoName(), int(sph.pr.GetNumber()), summary)
		if err != nil {
			return fmt.Errorf("could not create comment: %w", err)
		}
	case pr_actions.ActionComment, pr_actions.ActionCommitStatus, pr_actions.ActionProfileOnly:
		return fmt.Errorf("pull request action not supported")
	}
	return nil
}

func (sph *summaryPrHandler) generateSummary() (string, error) {
	var malicious = []maliciousTemplateData{}
	var lowScorePackages = map[string]templatePackage{}

	// Build the data structure for the template
	for _, alternative := range sph.trackedAlternatives {
		if _, ok := lowScorePackages[alternative.Dependency.Name]; !ok {
			var score float64
			if alternative.trustyReply.Score != nil {
				score = *alternative.trustyReply.Score
			}

			packageUIURL, err := url.JoinPath(
				constants.TrustyHttpURL,
				"report",
				strings.ToLower(alternative.Dependency.Ecosystem.AsString()),
				url.PathEscape(alternative.Dependency.Name))
			if err != nil {
				// This is unlikely to happen, but if it does, we skip the package
				continue
			}
			packageData := templatePackageData{
				Ecosystem:   alternative.Dependency.Ecosystem.AsString(),
				PackageName: alternative.Dependency.Name,
				TrustyURL:   packageUIURL,
				Score:       score,
			}

			// If the package is malicious we list it separately
			if slices.Contains(alternative.Reasons, TRUSTY_MALICIOUS_PKG) {
				malicious = append(malicious, maliciousTemplateData{
					templatePackageData: packageData,
					Summary:             alternative.trustyReply.Malicious.Summary,
					Details:             alternative.trustyReply.Malicious.Details,
				})
				continue
			}

			lowScorePackages[alternative.Dependency.Name] = templatePackage{
				templatePackageData: packageData,
				Deprecated:          alternative.trustyReply.IsDeprecated,
				Archived:            alternative.trustyReply.IsArchived,
				ScoreComponents:     buildScoreMatrix(alternative.trustyReply.ScoreComponents),
				Alternatives:        []templateAlternative{},
				Provenance:          buildProvenanceStruct(alternative.trustyReply),
			}
		}

		for _, altData := range alternative.trustyReply.Alternatives {
			// Note: now that the score is deprecated and
			// effectively `nil` for all packages, this
			// loop will always discard all alternatives,
			// rendering the whole block dead code.
			//
			// Since (1) we don't have score anymore, and
			// (2) we don't suggest malicious packages, I
			// suggest getting rid of this check
			// altogether and always report all available
			// alternatives.
			if comparePackages(altData, lowScorePackages[alternative.Dependency.Name]) == worse {
				continue
			}

			altPackageData := templateAlternative{
				templatePackageData: templatePackageData{
					Ecosystem:   altData.PackageType,
					PackageName: altData.PackageName,
					TrustyURL:   altData.TrustyURL,
				},
			}
			if altData.Score != nil {
				altPackageData.Score = *altData.Score
			}

			dep := lowScorePackages[alternative.Dependency.Name]
			dep.Alternatives = append(dep.Alternatives, altPackageData)
			lowScorePackages[alternative.Dependency.Name] = dep
		}
	}

	return sph.compileTemplate(malicious, lowScorePackages)
}

type packageComparison int

const (
	better packageComparison = iota
	worse
)

// comparePackages compares two packages to determine whether the
// first argument is better or worse than the second one. It does so
// by checking Trusty scores.
func comparePackages(alt alternative, examined templatePackage) packageComparison {
	if alt.Score != nil && *alt.Score != 0 && *alt.Score <= examined.Score {
		return worse
	}
	return better
}

// buildProvenanceStruct builds the provenance data structure for the PR template
func buildProvenanceStruct(r *trustyReport) *templateProvenance {
	if r == nil || r.Provenance == nil {
		return nil
	}
	var provenance *templateProvenance
	if r.Provenance != nil {
		provenance = &templateProvenance{}
		if r.Provenance.Historical != nil && r.Provenance.Historical.Overlap != 0 {
			provenance.Historical = &templateHistoricalProvenance{
				NumVersions:     int(r.Provenance.Historical.Versions),
				NumTags:         int(r.Provenance.Historical.Tags),
				MatchedVersions: int(r.Provenance.Historical.Common),
			}
		}

		if r.Provenance.Sigstore != nil && r.Provenance.Sigstore.Issuer != "" {
			provenance.Sigstore = &templateSigstoreProvenance{
				SourceRepository: r.Provenance.Sigstore.SourceRepository,
				Workflow:         r.Provenance.Sigstore.Workflow,
				Issuer:           r.Provenance.Sigstore.Issuer,
				RekorURI:         r.Provenance.Sigstore.RekorURI,
			}
		}

		if provenance.Historical == nil && provenance.Sigstore == nil {
			provenance = nil
		}
	}
	return provenance
}

// buildScoreMatrix builds the score components matrix that populates
// the score table in the PR comment template
//
//nolint:gosimple // This code is legacy and should be removed
func buildScoreMatrix(components []scoreComponent) []templateScoreComponent {
	scoreComp := []templateScoreComponent{}
	for _, component := range components {
		scoreComp = append(scoreComp, templateScoreComponent(component))
	}
	return scoreComp
}

func (sph *summaryPrHandler) compileTemplate(malicious []maliciousTemplateData, deps map[string]templatePackage) (string, error) {
	var summary strings.Builder
	var headerBuf bytes.Buffer
	if err := sph.commentTemplate.Execute(&headerBuf, struct {
		Malicious    []maliciousTemplateData
		Dependencies map[string]templatePackage
	}{
		Malicious:    malicious,
		Dependencies: deps,
	}); err != nil {
		return "", fmt.Errorf("could not execute template: %w", err)
	}
	if _, err := summary.WriteString(headerBuf.String()); err != nil {
		return "", fmt.Errorf("writing to string buffer: %w", err)
	}

	return summary.String(), nil
}

func newSummaryPrHandler(
	pr *pbinternal.PullRequest,
	cli provifv1.GitHub,
	trustyUrl string,
) (*summaryPrHandler, error) {
	tmpl, err := template.New("comment").Parse(commentTemplate)
	if err != nil {
		return nil, fmt.Errorf("could not parse dependency template: %w", err)
	}

	return &summaryPrHandler{
		cli:                 cli,
		pr:                  pr,
		trustyUrl:           trustyUrl,
		commentTemplate:     tmpl,
		trackedAlternatives: make([]dependencyAlternatives, 0),
	}, nil
}

func preprocessDetails(s *string) string {
	if s == nil {
		return ""
	}

	scanner := bufio.NewScanner(strings.NewReader(*s))
	text := ""
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "#") {
			continue
		}
		text += scanner.Text() + "<br>"
	}
	return strings.ReplaceAll(text, "|", "")
}
