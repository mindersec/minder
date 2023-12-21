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
	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v2"

	"github.com/stacklok/minder/internal/util/cli/table"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func marshalStructOrEmpty(v *structpb.Struct) string {
	if v == nil {
		return ""
	}

	m := v.AsMap()

	// marhsal as YAML
	out, err := yaml.Marshal(m)
	if err != nil {
		return ""
	}

	return string(out)
}

const (
	successStatus      = "success"
	failureStatus      = "failure"
	errorStatus        = "error"
	skippedStatus      = "skipped"
	pendingStatus      = "pending"
	notAvailableStatus = "not_available"
)

func NewProfileSettingsTable() table.Table {
	return table.New(table.Simple, "keyvalue", nil)
}

func RenderProfileSettingsTable(p *minderv1.Profile, t table.Table) {
	t.AddRow([]string{"ID", p.GetId()})
	t.AddRow([]string{"Name", p.GetName()})
	t.AddRow([]string{"Provider", p.GetContext().GetProvider()})
	t.AddRow([]string{"Alert", p.GetAlert()})
	t.AddRow([]string{"Remediate", p.GetRemediate()})
}

// NewProfileTable creates a new table for rendering profiles
func NewProfileTable() table.Table {
	return table.New(table.Simple, "profile", nil)
}

// RenderProfileTable renders the profile table
func RenderProfileTable(p *minderv1.Profile, t table.Table) {
	// repositories
	renderEntityRuleSets(p, minderv1.RepositoryEntity, p.Repository, t)

	// build_environments
	renderEntityRuleSets(p, minderv1.BuildEnvironmentEntity, p.BuildEnvironment, t)

	// artifacts
	renderEntityRuleSets(p, minderv1.ArtifactEntity, p.Artifact, t)

	// artifacts
	renderEntityRuleSets(p, minderv1.PullRequestEntity, p.PullRequest, t)
}

func renderEntityRuleSets(p *minderv1.Profile, entType minderv1.EntityType, rs []*minderv1.Profile_Rule, t table.Table) {
	for idx := range rs {
		rule := rs[idx]

		renderRuleTable(p, entType, rule, t)
	}
}

func renderRuleTable(p *minderv1.Profile, entType minderv1.EntityType, rule *minderv1.Profile_Rule, t table.Table) {
	params := marshalStructOrEmpty(rule.Params)
	def := marshalStructOrEmpty(rule.Def)

	row := []string{
		entType.String(),
		rule.Type,
		params,
		def,
	}
	t.AddRow(row)
}

// NewProfileStatusTable creates a new table for rendering profile status
func NewProfileStatusTable() table.Table {
	return table.New(table.Simple, "profile_status", nil)
}

// RenderProfileStatusTable renders the profile status table
func RenderProfileStatusTable(ps *minderv1.ProfileStatus, t table.Table) {
	row := []string{
		ps.ProfileId,
		ps.ProfileName,
		getEvalStatusText(ps.ProfileStatus),
		ps.LastUpdated.AsTime().Format(time.RFC3339),
	}
	t.AddRowWithColor(row, []string{
		"",
		"",
		getEvalStatusColor(ps.ProfileStatus),
		"",
	})
}

func getEvalStatusColor(status string) string {
	// eval statuses can be 'success', 'failure', 'error', 'skipped', 'pending'
	switch strings.ToLower(status) {
	case successStatus:
		return table.ColorGreen
	case failureStatus:
		return table.ColorRed
	case errorStatus:
		return table.ColorRed
	case skippedStatus:
		return table.ColorYellow
	default:
		return ""
	}
}

// NewRuleEvaluationsTable creates a new table for rendering rule evaluations
func NewRuleEvaluationsTable() table.Table {
	return table.New(table.Simple, "rule_evaluations", nil)
}

// RenderRuleEvaluationStatusTable renders the rule evaluations table
func RenderRuleEvaluationStatusTable(
	statuses []*minderv1.RuleEvaluationStatus,
	t table.Table,
) {
	for _, eval := range statuses {
		row := []string{
			eval.RuleId,
			eval.RuleName,
			eval.Entity,
			getEvalStatusText(eval.Status),
			getRemediationStatusText(eval.RemediationStatus),
			mapToYAMLOrEmpty(eval.EntityInfo),
			guidanceOrEncouragement(eval.Status, eval.Guidance),
		}

		t.AddRowWithColor(row, []string{
			"",
			"",
			"",
			"",
			getEvalStatusColor(eval.Status),
			getRemediateStatusColor(eval.RemediationStatus),
			"",
			"",
		})
	}
}

func getRemediateStatusColor(status string) string {
	// remediation statuses can be 'success', 'failure', 'error', 'skipped', 'not supported'
	switch strings.ToLower(status) {
	case successStatus:
		return table.ColorGreen
	case failureStatus:
		return table.ColorRed
	case errorStatus:
		return table.ColorRed
	case notAvailableStatus:
		return table.ColorYellow
	default:
		return ""
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
