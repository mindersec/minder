// Copyright 2024 Stacklok, Inc.
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

// Package vulncheck provides the vulnerability check evaluator
package vulncheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	"strings"
	"text/template"
)

type vulnerabilityReport interface {
	render() (string, error)
}

type minderTemplateData struct {
	MagicComment string
	Metadata     string
	Body         string
	Footer       string
}

type reportMetadata struct {
	VulnerabilityCount int    `json:"VulnerabilityCount"`
	RemediationCount   int    `json:"RemediationCount"`
	TrackedDepsCount   int    `json:"TrackedDepsCount"`
	CommitSHA          string `json:"CommitSHA"`
}

const (
	vulnsFoundText = `
	Minder found vulnerable dependencies in this PR. Either push an updated
	version or accept the proposed changes. Note that accepting the changes will
	include Minder as a co-author of this PR.
	`
	vulnsFoundTextShort = `
	Vulnerable dependencies found.
	`
	noVulsFoundText = `
	Minder analyzed this PR and found no vulnerable dependencies.
	`
	reviewBodyDismissCommentText = `
	Previous Minder review was dismissed because the PR was updated.
	`
	vulnFoundWithNoPatch = "Vulnerability found, but no patched version exists yet."
)

const (
	reviewBodyMagicComment = "<!-- minder: pr-review-body -->"
	statusBodyMagicComment = "<!-- minder: pr-status-body -->"

	minderTemplateName   = "minderCommentBody"
	minderTemplateString = "{{ .MagicComment }}\n\n{{ .Body }}{{ .Footer }}"

	minderMetadataTemplateName   = "Metadata"
	minderMetadataTemplateString = "<!--{ {{.}} }-->"
)

const (
	contactString = `
<hr>
&#128236; <i>Have feedback on the report? <a href="mailto:info@stacklok.com">Share it here.</a></i>
`
)

const (
	tableVulnerabilitiesHeaderName = "vulnerabilitiesTableHeader"
	tableVulnerabilitiesHeader     = `<h3>Summary of vulnerabilities found</h3>
Minder found the following vulnerabilities in this PR:
<table>
  <tr>
    <th>Ecosystem</th>
    <th>Name</th>
    <th>Version</th>
    <th>Vulnerability ID</th>
    <th>Summary</th>
    <th>Introduced</th>
    <th>Fixed</th>
  </tr>
`
	tableVulnerabilitiesRowsName = "vulnerabilitiesTableRow"
	tableVulnerabilitiesRows     = `
  {{ range .Vulnerabilities }}
  <tr>
    <td>{{ $.DependencyEcosystem }}</td>
    <td>{{ $.DependencyName }}</td>
    <td>{{ $.DependencyVersion }}</td>
    <td>{{ .ID }}</td>
    <td>{{ .Summary }}</td>
    <td>{{ .Introduced }}</td>
    <td>{{ .Fixed }}</td>
  </tr>
  {{ end }}
`
	tableVulnerabilitiesFooter = "</table>"
)

type reviewReport struct {
	TrackedDependencies []dependencyVulnerabilities
}

func (r *reviewReport) render() (string, error) {
	headerTmpl, err := htmltemplate.New(tableVulnerabilitiesHeaderName).Parse(tableVulnerabilitiesHeader)
	if err != nil {
		return "", fmt.Errorf("could not parse dependency template: %w", err)
	}
	rowsTmpl, err := htmltemplate.New(tableVulnerabilitiesRowsName).Parse(tableVulnerabilitiesRows)
	if err != nil {
		return "", fmt.Errorf("could not parse vulnerability template: %w", err)
	}

	var summary strings.Builder
	if len(r.TrackedDependencies) == 0 {
		summary.WriteString(noVulsFoundText)
		return summary.String(), nil
	}

	var headerBuf bytes.Buffer
	if err := headerTmpl.Execute(&headerBuf, nil); err != nil {
		return "", fmt.Errorf("could not execute template: %w", err)
	}
	summary.WriteString(headerBuf.String())

	for i := range r.TrackedDependencies {
		var rowBuf bytes.Buffer

		if err := rowsTmpl.Execute(&rowBuf, struct {
			DependencyEcosystem string
			DependencyName      string
			DependencyVersion   string
			Vulnerabilities     []Vulnerability
		}{
			DependencyEcosystem: r.TrackedDependencies[i].Dependency.Ecosystem.AsString(),
			DependencyName:      r.TrackedDependencies[i].Dependency.Name,
			DependencyVersion:   r.TrackedDependencies[i].Dependency.Version,
			Vulnerabilities:     r.TrackedDependencies[i].Vulnerabilities,
		}); err != nil {
			return "", fmt.Errorf("could not execute template: %w", err)
		}
		summary.WriteString(rowBuf.String())
	}
	summary.WriteString(tableVulnerabilitiesFooter)

	reviewBody, err := render(minderTemplateName, minderTemplateString, minderTemplateData{
		MagicComment: reviewBodyMagicComment,
		Body:         summary.String(),
		Footer:       contactString,
	})
	if err != nil {
		return "", fmt.Errorf("could not create review body: %w", err)
	}

	return reviewBody, nil
}

const (
	bugHtmlEmoji     = "&#128030;"
	fixHtmlEmoji     = "&#128736;"
	warningHtmlEmoji = "&#9888;&#65039;"
	successHtmlEmoji = "&#9989;"
	reviewHtmlEmoji  = "&#128202;"
)

const (
	reportTemplateName   = "reportBody"
	reportTemplateString = `
{{- $vulnerabilityCount := .Metadata.VulnerabilityCount -}}
{{- $remediationCount := .Metadata.RemediationCount -}}
{{- $successSymbol := .Symbols.success -}}
{{- $warningSymbol := .Symbols.warning -}}
{{- $countVulnsWithFix := .CountFixedVulnerabilities -}}
<h2>Minder Vulnerability Report {{ if gt $vulnerabilityCount 0 }}{{$warningSymbol}}{{ else }}{{$successSymbol}}{{ end }}</h2>
<p>{{ .Report.StatusText }}</p>
<blockquote>
<h3>Vulnerability scan of <code>{{slice .Report.CommitSHA 0 8 }}:</code></h3>
{{- if .Report.ReviewUrl }}<a href=" {{ .Report.ReviewUrl }}">{{ .Symbols.review }} <b>View Full Review</b></a>{{ end -}}
<ul>
	<li>{{ .Symbols.bug }} <b>vulnerabilities:</b> <code>{{ $vulnerabilityCount }}</code></li>
	<li>{{ .Symbols.fix }} <b>fixes:</b> <code>{{ $remediationCount }}</code></li>
</ul>
</blockquote>
{{- if .Report.TrackedDependencies }}
<table>
	<th>
		<tr>
			<th>Package</th>
			<th>Version</th>
			<th>#Vulnerabilities</th>
			<th>#Fixes</th>
			<th>Patch</th>
	  	</tr>
	</th>
	{{- range .Report.TrackedDependencies }}
	<tr>
		<td>{{.Dependency.Name}}</td>
		<td>{{.Dependency.Version}}</td>
		<td>{{len .Vulnerabilities}}</td>
		<td>{{ (call $countVulnsWithFix .Vulnerabilities) }}</td>
		<td>{{- if .PatchVersion }}{{.PatchVersion}}{{- else }}{{$warningSymbol}}{{- end }}</td>
	</tr>
	{{ end }}
</table>
{{ end -}}
`
)

type statusReport struct {
	StatusText          string
	CommitSHA           string
	ReviewUrl           string
	TrackedDependencies []dependencyVulnerabilities
}

func counter(condition func(dep dependencyVulnerabilities) bool) func(deps []dependencyVulnerabilities) int {
	return func(deps []dependencyVulnerabilities) int {
		count := 0
		for _, dep := range deps {
			if condition(dep) {
				count++
			}
		}
		return count
	}
}

func (s *statusReport) render() (string, error) {

	countVulnsWithFix := func(vulns []Vulnerability) int {
		count := 0
		for _, vuln := range vulns {
			if vuln.Fixed != "" {
				count++
			}
		}
		return count
	}

	reportSymbols := map[string]interface{}{
		"bug":     bugHtmlEmoji,
		"fix":     fixHtmlEmoji,
		"warning": warningHtmlEmoji,
		"success": successHtmlEmoji,
		"review":  reviewHtmlEmoji,
	}

	wrappedReport := struct {
		Report                    *statusReport
		Metadata                  reportMetadata
		Symbols                   map[string]interface{}
		CountFixedVulnerabilities func(vulns []Vulnerability) int
	}{
		Report:                    s,
		Metadata:                  s.generateMetadata(),
		Symbols:                   reportSymbols,
		CountFixedVulnerabilities: countVulnsWithFix,
	}

	status, err := render(reportTemplateName, reportTemplateString, wrappedReport)
	if err != nil {
		return "", fmt.Errorf("could not create report body: %w", err)
	}

	// Convert the object to JSON
	metadata, err := json.Marshal(wrappedReport.Metadata)
	if err != nil {
		return "", fmt.Errorf("could not convert Metadata to JSON: %w", err)
	}

	MetadataText, err := render(minderMetadataTemplateName, minderMetadataTemplateString, string(metadata))
	if err != nil {
		return "", fmt.Errorf("could not create Metadata text: %w", err)
	}

	reviewBody, err := render(minderTemplateName, minderTemplateString, minderTemplateData{
		MagicComment: statusBodyMagicComment,
		Body:         status,
		Metadata:     MetadataText,
		Footer:       contactString,
	})
	if err != nil {
		return "", fmt.Errorf("could not create review body: %w", err)
	}

	return reviewBody, nil
}

func (s *statusReport) generateMetadata() reportMetadata {
	countVulns := counter(func(dep dependencyVulnerabilities) bool { return len(dep.Vulnerabilities) > 0 })
	countFixes := counter(func(dep dependencyVulnerabilities) bool { return dep.PatchVersion != "" })

	return reportMetadata{
		VulnerabilityCount: countVulns(s.TrackedDependencies),
		RemediationCount:   countFixes(s.TrackedDependencies),
		TrackedDepsCount:   len(s.TrackedDependencies),
		CommitSHA:          s.CommitSHA,
	}

}

func render(templateName, templateString string, object interface{}) (string, error) {
	tmpl, err := template.New(templateName).Option("missingkey=error").Parse(templateString)
	if err != nil {
		return "", err
	}

	// Execute the template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, object); err != nil {
		return "", err
	}

	return buf.String(), nil
}
