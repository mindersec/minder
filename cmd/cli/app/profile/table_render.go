//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package profile

import (
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"gopkg.in/yaml.v2"

	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli/table"
	"github.com/stacklok/minder/internal/util/cli/table/layouts"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

const (
	successStatus      = "success"
	failureStatus      = "failure"
	errorStatus        = "error"
	skippedStatus      = "skipped"
	pendingStatus      = "pending"
	notAvailableStatus = "not_available"
)

// NewProfileSettingsTable creates a new table for rendering profile settings
func NewProfileSettingsTable() table.Table {
	return table.New(table.Simple, layouts.ProfileSettings, nil)
}

// RenderProfileSettingsTable renders the profile settings table
func RenderProfileSettingsTable(p *minderv1.Profile, t table.Table) {
	// if alert is not set in the profile definition, default to its minder behaviour which is "on"
	alert := p.GetAlert()
	if alert == "" {
		alert = "on"
	}
	// if remediation is not set in the profile definition, default to its minder behaviour which is "off"
	remediate := p.GetRemediate()
	if remediate == "" {
		remediate = "off"
	}
	t.AddRow(p.GetId(), p.GetName(), p.GetContext().GetProvider(), alert, remediate)
}

// NewProfileTable creates a new table for rendering profiles
func NewProfileTable() table.Table {
	return table.New(table.Simple, layouts.Profile, nil)
}

// RenderProfileTable renders the profile table
func RenderProfileTable(p *minderv1.Profile, t table.Table) {
	// repositories
	renderEntityRuleSets(minderv1.RepositoryEntity, p.Repository, t)

	// build_environments
	renderEntityRuleSets(minderv1.BuildEnvironmentEntity, p.BuildEnvironment, t)

	// artifacts
	renderEntityRuleSets(minderv1.ArtifactEntity, p.Artifact, t)

	// pull request
	renderEntityRuleSets(minderv1.PullRequestEntity, p.PullRequest, t)
}

func renderEntityRuleSets(entType minderv1.EntityType, rs []*minderv1.Profile_Rule, t table.Table) {
	for idx := range rs {
		rule := rs[idx]

		renderRuleTable(entType, rule, t)
	}
}

func renderRuleTable(entType minderv1.EntityType, rule *minderv1.Profile_Rule, t table.Table) {
	params := util.MarshalStructOrEmpty(rule.Params)
	def := util.MarshalStructOrEmpty(rule.Def)

	t.AddRow(
		entType.String(),
		rule.Type,
		params,
		def,
	)
}

// NewProfileStatusTable creates a new table for rendering profile status
func NewProfileStatusTable() table.Table {
	return table.New(table.Simple, layouts.ProfileStatus, nil)
}

// RenderProfileStatusTable renders the profile status table
func RenderProfileStatusTable(ps *minderv1.ProfileStatus, t table.Table) {
	t.AddRowWithColor(
		layouts.NoColor(ps.ProfileId),
		layouts.NoColor(ps.ProfileName),
		getColoredEvalStatus(ps.ProfileStatus),
		layouts.NoColor(ps.LastUpdated.AsTime().Format(time.RFC3339)),
	)
}

func getColoredEvalStatus(status string) layouts.ColoredColumn {
	txt := getEvalStatusText(status)
	// eval statuses can be 'success', 'failure', 'error', 'skipped', 'pending'
	switch strings.ToLower(status) {
	case successStatus:
		return layouts.GreenColumn(txt)
	case failureStatus:
		return layouts.RedColumn(txt)
	case errorStatus:
		return layouts.RedColumn(txt)
	case skippedStatus:
		return layouts.YellowColumn(txt)
	default:
		return layouts.NoColor(txt)
	}
}

// NewRuleEvaluationsTable creates a new table for rendering rule evaluations
func NewRuleEvaluationsTable() table.Table {
	return table.New(table.Simple, layouts.RuleEvaluations, nil)
}

// RenderRuleEvaluationStatusTable renders the rule evaluations table
func RenderRuleEvaluationStatusTable(
	statuses []*minderv1.RuleEvaluationStatus,
	t table.Table,
) {
	ruleTypeNameCount := make(map[string]int)
	for _, eval := range statuses {
		ruleTypeNameCount[eval.RuleTypeName]++
	}

	for _, eval := range statuses {
		ruleName := eval.RuleTypeName
		if ruleTypeNameCount[eval.RuleTypeName] > 1 {
			ruleName = eval.RuleTypeName + "\n(" + eval.RuleHash + ")"
		}

		t.AddRowWithColor(
			layouts.NoColor(eval.RuleId),
			layouts.NoColor(ruleName),
			layouts.NoColor(eval.Entity),
			getColoredEvalStatus(eval.Status),
			getRemediateStatusColor(eval.RemediationStatus),
			layouts.NoColor(mapToYAMLOrEmpty(eval.EntityInfo)),
			layouts.NoColor(guidanceOrEncouragement(eval.Status, eval.Guidance)),
		)
	}
}

func getRemediateStatusColor(status string) layouts.ColoredColumn {
	txt := getRemediationStatusText(status)
	// remediation statuses can be 'success', 'failure', 'error', 'skipped', 'not supported'
	switch strings.ToLower(status) {
	case successStatus:
		return layouts.GreenColumn(txt)
	case failureStatus:
		return layouts.RedColumn(txt)
	case errorStatus:
		return layouts.RedColumn(txt)
	case notAvailableStatus:
		return layouts.YellowColumn(txt)
	default:
		return layouts.NoColor(txt)
	}
}

// Gets a friendly status text with an emoji
func getEvalStatusText(status string) string {
	// eval statuses can be 'success', 'failure', 'error', 'skipped', 'pending'
	switch strings.ToLower(status) {
	case successStatus:
		return "✅ Success"
	case failureStatus:
		return "❌ Failure"
	case errorStatus:
		return "❌ Error"
	case skippedStatus:
		return "⏹ Skipped"
	case pendingStatus:
		return "⏳ Pending"
	default:
		return "⚠️ Unknown"
	}
}

// Gets a friendly status text with an emoji
func getRemediationStatusText(status string) string {
	// remediation statuses can be 'success', 'failure', 'error', 'skipped', 'not supported'
	switch strings.ToLower(status) {
	case successStatus:
		return "✅ Success"
	case failureStatus:
		return "❌ Failure"
	case errorStatus:
		return "❌ Error"
	case skippedStatus:
		return "" // visually empty as we didn't have to remediate
	case notAvailableStatus:
		return "🚫 Not Available"
	default:
		return "⚠️ Unknown"
	}
}

func mapToYAMLOrEmpty(m map[string]string) string {
	if m == nil {
		return ""
	}

	yamlText, err := yaml.Marshal(m)
	if err != nil {
		return ""
	}

	return string(yamlText)
}

func guidanceOrEncouragement(status, guidance string) string {
	if status == successStatus && guidance == "" {
		return "👍"
	}

	if guidance == "" {
		return "No guidance available for this rule 😞"
	}

	// TODO: use a color scheme for minder instead of a pre-defined one.
	// Related-to: https://github.com/stacklok/minder/issues/1006
	renderedGuidance, err := glamour.Render(guidance, "dark")
	if err != nil {
		return guidance
	}

	return renderedGuidance
}
