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

	"github.com/stacklok/minder/internal/constants"
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
	ScoreComponents []templateScoreComponent
	Alternatives    []templateAlternative
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
	Reasons     []RuleViolationReason
	trustyReply *Reply
}

// summaryPrHandler is a prStatusHandler that adds a summary text to the PR as a comment.
type summaryPrHandler struct {
	cli       provifv1.GitHub
	pr        *pb.PullRequest
	trustyUrl string

	trackedAlternatives []dependencyAlternatives
	commentTemplate     *template.Template
}

func (sph *summaryPrHandler) trackAlternatives(
	dep *pb.PrDependencies_ContextualDependency,
	violationReasons []RuleViolationReason,
	trustyReply *Reply,
) {
	sph.trackedAlternatives = append(sph.trackedAlternatives, dependencyAlternatives{
		Dependency:  dep.Dep,
		Reasons:     violationReasons,
		trustyReply: trustyReply,
	})
}

func (sph *summaryPrHandler) submit(ctx context.Context) error {
	if len(sph.trackedAlternatives) == 0 {
		return nil
	}

	summary, err := sph.generateSummary()
	if err != nil {
		return fmt.Errorf("could not generate summary: %w", err)
	}

	_, err = sph.cli.CreateIssueComment(ctx, sph.pr.GetRepoOwner(), sph.pr.GetRepoName(), int(sph.pr.GetNumber()), summary)
	if err != nil {
		return fmt.Errorf("could not create comment: %w", err)
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
				ScoreComponents:     scoreComp,
				Alternatives:        []templateAlternative{},
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
	summary.WriteString(headerBuf.String())

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
