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

	"github.com/stacklok/minder/internal/constants"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

const (
	// nolint:lll
	noLowScoresText = "Minder analyzed the changes in this pull request with <a href=\"https://www.trustypkg.dev/\">Trusty</a> and found no dependencies scored lower than your profile threshold."

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

{{ if .Alternatives }}
### Summary of Packages With Low Scores

Based on [Trusty](https://www.trustypkg.dev/) dependency data, Minder detected that this PR is introducing software dependencies whose trust score components are lower than some of the configured thresholds. Below is a summary of the packages with low scores and their alternatives.

| Type | Name | Score | Score Components | Alternative Package | Alternative Score |
| --- | --- | --- | --- | --- | --- |
{{ range .Alternatives -}}
| {{ .Ecosystem }} | [{{ .PackageName }}]({{ .TrustyURL }}) | {{ .Score }} | {{ .ScoreComponents }} | [{{ .AlternativeName }}]({{ .AlternativeTrustyURL }}) | {{ .AlternativeScore }} |
{{ end }}
{{ end }}
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

type maliciousTemplateData struct {
	PackageName string
	TrustyURL   string
	Summary     string
	Details     string
}

type lowScoreTemplateData struct {
	Ecosystem            string
	PackageName          string
	Score                float64
	TrustyURL            string
	AlternativeName      string
	AlternativeScore     float64
	AlternativeTrustyURL string
	ScoreComponents      string
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
	var summary strings.Builder

	if len(sph.trackedAlternatives) == 0 {
		summary.WriteString(noLowScoresText)
		return summary.String(), nil
	}
	var malicious = []maliciousTemplateData{}
	var lowScorePackages = []lowScoreTemplateData{}

	// Build the data structure for the template
	for _, alternative := range sph.trackedAlternatives {
		// Build the package trustyURL
		trustyURL := fmt.Sprintf(
			"%s%s/%s", constants.TrustyHttpURL,
			strings.ToLower(alternative.Dependency.Ecosystem.AsString()),
			url.PathEscape(alternative.trustyReply.PackageName),
		)

		var score float64
		if alternative.trustyReply.Summary.Score != nil {
			score = *alternative.trustyReply.Summary.Score
		}

		// If the package is malicious we list it separately
		if slices.Contains(alternative.Reasons, TRUSTY_MALICIOUS_PKG) {
			malicious = append(malicious, maliciousTemplateData{
				PackageName: alternative.trustyReply.PackageName,
				TrustyURL:   trustyURL,
				Summary:     alternative.trustyReply.PackageData.Malicious.Summary,
				Details:     preprocessDetails(alternative.trustyReply.PackageData.Malicious.Details),
			})
			continue
		}

		for _, alt := range alternative.trustyReply.Alternatives.Packages {
			if alt.Score < score {
				continue
			}

			lowScorePkg := lowScoreTemplateData{
				Ecosystem:        alternative.Dependency.Ecosystem.AsString(),
				PackageName:      alternative.trustyReply.PackageName,
				Score:            score,
				TrustyURL:        trustyURL,
				AlternativeName:  alt.PackageName,
				AlternativeScore: alt.Score,
				AlternativeTrustyURL: fmt.Sprintf(
					"%s%s/%s", constants.TrustyHttpURL,
					strings.ToLower(alternative.Dependency.Ecosystem.AsString()),
					url.PathEscape(alt.PackageName),
				),
				ScoreComponents: "-",
			}

			// Add the low score components
			if alternative.trustyReply.Summary.Description != nil {
				sc := ""
				if v, ok := alternative.trustyReply.Summary.Description["activity"]; ok {
					sc = fmt.Sprintf("Activity: %.2f <br>", v.(float64))
				}

				if v, ok := alternative.trustyReply.Summary.Description["provenance"]; ok {
					sc = fmt.Sprintf("Provenance: %.2f <br>", v.(float64))
				}

				if sc != "" {
					lowScorePkg.ScoreComponents = sc
				}
			}
			lowScorePackages = append(lowScorePackages, lowScorePkg)
		}

		// If there are no alternatives, add a single row with no data
		if len(alternative.trustyReply.Alternatives.Packages) == 0 {
			lowScorePackages = append(lowScorePackages, lowScoreTemplateData{
				PackageName:     alternative.trustyReply.PackageName,
				Score:           score,
				TrustyURL:       trustyURL,
				AlternativeName: "N/A",
			})
		}
	}

	var headerBuf bytes.Buffer
	if err := sph.commentTemplate.Execute(&headerBuf, struct {
		Malicious    []maliciousTemplateData
		Alternatives []lowScoreTemplateData
	}{
		Malicious:    malicious,
		Alternatives: lowScorePackages,
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
