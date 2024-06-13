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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/url"
	"slices"
	"strings"
	template "text/template"
	"unicode"

	"github.com/google/go-github/v61/github"
	"github.com/rs/zerolog"
	trustytypes "github.com/stacklok/trusty-sdk-go/pkg/types"

	"github.com/stacklok/minder/internal/constants"
	"github.com/stacklok/minder/internal/engine/eval/pr_actions"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

const (
	// nolint:lll
	commentTemplate = `{{- if .Malicious -}}
### ⚠️ MALICIOUS PACKAGES ⚠️

Minder has detected that this pull request is introducing malicious software dependencies using data from [Trusty](https://www.trustypkg.dev/):

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
{{ end }}
{{ if .Deprecated }}
⚠️ __Deprecated Package:__ This package is marked as archived. Proceed with caution!
{{ end }}
#### Trusty Score: {{ .Score }}
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

  {{- end -}}
  {{ if .Provenance.Sigstore }}

  __This package has been digitally signed using sigtore.__

  |              |   |
  | ------------------- | ----: |
  | Source repository | {{ .Provenance.Sigstore.SourceRepository }} |
  | Cerificate Issuer | {{ .Provenance.Sigstore.Issuer }} |
  | GitHub action workflow | {{ .Provenance.Sigstore.Workflow }} |
  | Rekor (public ledger) entry | {{ .Provenance.Sigstore.RekorURI }} |
  {{- end -}}
  </details>
{{- end -}}
{{ if .Alternatives }}
<details>
  <summary>Alternatives</summary>

  | Package             | Score | Description |
  | ------------------- | ----: | ----------- |
{{ range .Alternatives -}}
  | [{{ .PackageName }}]({{ .TrustyURL }})  | {{ .Score }} | {{ .Summary }} |
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

	// TRUSTY_DEPRECATED means a package was marked upstream as deprecated
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
	Dependency *pb.Dependency

	// Reason captures the reason why a package was flagged
	Reasons []RuleViolationReason

	// BlockPR will cause the PR to be blocked as requesting changes when true
	BlockPR bool

	// trustyReply is the complete response from trusty for this package
	trustyReply *trustytypes.Reply
}

// summaryPrHandler is a prStatusHandler that adds a summary text to the PR as a comment.
type summaryPrHandler struct {
	cli       provifv1.GitHub
	pr        *pb.PullRequest
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
	var reviewAction string = "COMMENT"
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
			if alternative.trustyReply.Summary.Score != nil {
				score = *alternative.trustyReply.Summary.Score
			}
			packageData := templatePackageData{
				Ecosystem:   alternative.Dependency.Ecosystem.AsString(),
				PackageName: alternative.Dependency.Name,
				TrustyURL: fmt.Sprintf(
					"%s%s/%s", constants.TrustyHttpURL,
					strings.ToLower(alternative.Dependency.Ecosystem.AsString()),
					url.PathEscape(alternative.trustyReply.PackageName),
				),
				Score: score,
			}

			// If the package is malicious we list it separately
			if slices.Contains(alternative.Reasons, TRUSTY_MALICIOUS_PKG) {
				malicious = append(malicious, maliciousTemplateData{
					templatePackageData: packageData,
					Summary:             alternative.trustyReply.PackageData.Malicious.Summary,
					Details:             preprocessDetails(alternative.trustyReply.PackageData.Malicious.Details),
				})
				continue
			}

			lowScorePackages[alternative.Dependency.Name] = templatePackage{
				templatePackageData: packageData,
				Deprecated:          alternative.trustyReply.PackageData.Deprecated,
				Archived:            alternative.trustyReply.PackageData.Archived,
				ScoreComponents:     buildScoreMatrix(alternative),
				Alternatives:        []templateAlternative{},
				Provenance:          buildProvenanceStruct(alternative.trustyReply),
			}
		}

		for _, altData := range alternative.trustyReply.Alternatives.Packages {
			if altData.Score <= lowScorePackages[alternative.Dependency.Name].Score {
				continue
			}

			altPackageData := templateAlternative{
				templatePackageData: templatePackageData{
					Ecosystem:   alternative.Dependency.Ecosystem.AsString(),
					PackageName: altData.PackageName,
					TrustyURL: fmt.Sprintf(
						"%s%s/%s", constants.TrustyHttpURL,
						strings.ToLower(alternative.Dependency.Ecosystem.AsString()),
						url.PathEscape(altData.PackageName),
					),
					Score: altData.Score,
				},
			}

			dep := lowScorePackages[alternative.Dependency.Name]
			dep.Alternatives = append(dep.Alternatives, altPackageData)
			lowScorePackages[alternative.Dependency.Name] = dep
		}
	}

	return sph.compileTemplate(malicious, lowScorePackages)
}

// buildProvenanceStruct builds the provenance data structure for the PR template
func buildProvenanceStruct(r *trustytypes.Reply) *templateProvenance {
	if r == nil || r.Provenance == nil {
		return nil
	}
	var provenance *templateProvenance
	if r.Provenance != nil {
		provenance = &templateProvenance{}
		if r.Provenance.Description.Historical.Overlap != 0 {
			provenance.Historical = &templateHistoricalProvenance{
				NumVersions:     int(r.Provenance.Description.Historical.Versions),
				NumTags:         int(r.Provenance.Description.Historical.Tags),
				MatchedVersions: int(r.Provenance.Description.Historical.Common),
			}
		}

		if r.Provenance.Description.Sigstore.Issuer != "" {
			provenance.Sigstore = &templateSigstoreProvenance{
				SourceRepository: r.Provenance.Description.Sigstore.SourceRepository,
				Workflow:         r.Provenance.Description.Sigstore.Workflow,
				Issuer:           r.Provenance.Description.Sigstore.Issuer,
				RekorURI:         r.Provenance.Description.Sigstore.Transparency,
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
func buildScoreMatrix(alternative dependencyAlternatives) []templateScoreComponent {
	scoreComp := []templateScoreComponent{}
	if alternative.trustyReply.Summary.Description != nil {
		for l, v := range alternative.trustyReply.Summary.Description {
			switch l {
			case "activity":
				l = "Package activity"
			case "activity_user":
				l = "User activity"
			case "provenance":
				l = "Provenance"
			case "typosquatting":
				if v.(float64) > 5.00 {
					continue
				}
				v = "⚠️ Dependency may be trying to impersonate a well known package"
				l = "Typosquatting"
			case "activity_repo":
				l = "Repository activity"
			default:
				if len(l) > 1 {
					l = string(unicode.ToUpper([]rune(l)[0])) + l[1:]
				}
			}
			scoreComp = append(scoreComp, templateScoreComponent{
				Label: l,
				Value: v,
			})
		}
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
	pr *pb.PullRequest,
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

func preprocessDetails(s string) string {
	scanner := bufio.NewScanner(strings.NewReader(s))
	text := ""
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "#") {
			continue
		}
		text += scanner.Text() + "<br>"
	}
	return strings.ReplaceAll(text, "|", "")
}
