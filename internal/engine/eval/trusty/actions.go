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
	"bytes"
	"context"
	"fmt"
	htmltemplate "html/template"
	"net/url"
	"strings"

	"github.com/stacklok/minder/internal/constants"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

const (
	// nolint:lll
	noLowScoresText = "Minder analyzed this PR with <a href=\"https://www.trustypkg.dev/\">Trusty</a> and found no dependencies scored lower than your profile threshold."

	tableHeaderTmplName = "alternativesTableHeader"
	tableTemplateHeader = `### Summary of packages with low scores
Minder detected that this PR is adding dependencies whose score is lower than the threshold configured with
Minder profiles. Below is a summary of the packages with low scores and their alternatives.

<table>
  <tr>
    <td> Pkg </td>
    <td> Name </td>
    <td> Score </td>
    <td> Alternative Name </td>
    <td> Alternative Score </td>
  </tr>
`
	tableFooter       = "</table>"
	tableRowsTmplName = "alternativesTableRow"
	// nolint:lll
	tableTemplateRow = `
{{ range .Alternatives }}
  <tr>
	<td>{{ $.Ecosystem }}</td>
	<td><a href="{{ $.BaseUrl }}/{{ $.Ecosystem }}/{{ $.NameURL }}">{{ $.Name }}</a></td>
	<td>{{ $.Score }}</td>
	<td>{{ if .PackageName }}<a href="{{ $.BaseUrl }}/{{ $.Ecosystem }}/{{ .PackageNameURL }}">{{ .PackageName }}</a>{{ else }}No alternative found{{ end }}</td>
	<td>{{ if .PackageName }}{{ .Score }}{{ else }}-{{ end }}</td>
  </tr>
{{ end }}
`
)

type dependencyAlternatives struct {
	Dependency  *pb.Dependency
	trustyReply *Reply
}

// summaryPrHandler is a prStatusHandler that adds a summary text to the PR as a comment.
type summaryPrHandler struct {
	cli       provifv1.GitHub
	pr        *pb.PullRequest
	trustyUrl string

	trackedAlternatives []dependencyAlternatives
	headerTmpl          *htmltemplate.Template
	rowsTmpl            *htmltemplate.Template
}

func (sph *summaryPrHandler) trackAlternatives(
	dep *pb.PrDependencies_ContextualDependency,
	trustyReply *Reply,
) {
	sph.trackedAlternatives = append(sph.trackedAlternatives, dependencyAlternatives{
		Dependency:  dep.Dep,
		trustyReply: trustyReply,
	})
}

func (sph *summaryPrHandler) submit(ctx context.Context) error {
	summary, err := sph.generateSummary()
	if err != nil {
		return fmt.Errorf("could not generate summary: %w", err)
	}

	_, err = sph.cli.CreateComment(ctx, sph.pr.GetRepoOwner(), sph.pr.GetRepoName(), int(sph.pr.GetNumber()), summary)
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

	var headerBuf bytes.Buffer
	if err := sph.headerTmpl.Execute(&headerBuf, nil); err != nil {
		return "", fmt.Errorf("could not execute template: %w", err)
	}
	summary.WriteString(headerBuf.String())

	for i := range sph.trackedAlternatives {
		var rowBuf bytes.Buffer

		higherScoringAlternatives := make([]Alternative, 0)
		for _, alt := range sph.trackedAlternatives[i].trustyReply.Alternatives.Packages {
			if alt.Score > sph.trackedAlternatives[i].trustyReply.Summary.Score {
				alt.PackageNameURL = url.PathEscape(alt.PackageName)
				higherScoringAlternatives = append(higherScoringAlternatives, alt)
			}
		}
		if len(higherScoringAlternatives) == 0 {
			higherScoringAlternatives = append(higherScoringAlternatives, Alternative{
				PackageName: "", // Note: Set it to empty to indicate no alternative found in the template
				Score:       0,
			})
		}

		// Populate the Trusty alternatives row template
		if err := sph.rowsTmpl.Execute(&rowBuf, struct {
			Ecosystem    string
			Name         string
			NameURL      string
			Score        float64
			Alternatives []Alternative
			BaseUrl      string
		}{
			Ecosystem:    strings.ToLower(sph.trackedAlternatives[i].Dependency.Ecosystem.AsString()),
			Name:         sph.trackedAlternatives[i].Dependency.Name,
			NameURL:      url.PathEscape(sph.trackedAlternatives[i].Dependency.Name),
			Score:        sph.trackedAlternatives[i].trustyReply.Summary.Score,
			Alternatives: higherScoringAlternatives,
			BaseUrl:      constants.TrustyHttpURL,
		}); err != nil {
			return "", fmt.Errorf("could not execute template: %w", err)
		}
		summary.WriteString(rowBuf.String())
	}
	summary.WriteString(tableFooter)

	return summary.String(), nil
}

func newSummaryPrHandler(
	pr *pb.PullRequest,
	cli provifv1.GitHub,
	trustyUrl string,
) (*summaryPrHandler, error) {
	headerTmpl, err := htmltemplate.New(tableHeaderTmplName).Parse(tableTemplateHeader)
	if err != nil {
		return nil, fmt.Errorf("could not parse dependency template: %w", err)
	}
	rowsTmpl, err := htmltemplate.New(tableRowsTmplName).Parse(tableTemplateRow)
	if err != nil {
		return nil, fmt.Errorf("could not parse vulnerability template: %w", err)
	}

	return &summaryPrHandler{
		cli:                 cli,
		pr:                  pr,
		trustyUrl:           trustyUrl,
		headerTmpl:          headerTmpl,
		rowsTmpl:            rowsTmpl,
		trackedAlternatives: make([]dependencyAlternatives, 0),
	}, nil
}
