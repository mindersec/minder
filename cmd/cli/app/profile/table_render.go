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

	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v2"

	"github.com/stacklok/minder/internal/util/cli/table"
	"github.com/stacklok/minder/internal/util/cli/table/layouts"
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

// NewProfileSettingsTable creates a new table for rendering profile settings
func NewProfileSettingsTable() table.Table {
	return table.New(table.Simple, layouts.ProfileSettings, nil)
}

// RenderProfileSettingsTable renders the profile settings table
func RenderProfileSettingsTable(p *minderv1.Profile, t table.Table) {
	t.AddRow(p.GetId(), p.GetName(), p.GetAlert(), p.GetRemediate())
}

// NewProfileTable creates a new table for rendering profiles
func NewProfileTable() table.Table {
	return table.New(table.Simple, layouts.Profile, nil)
}

// RenderProfileTable renders the profile table
func RenderProfileTable(p *minderv1.Profile, t table.Table) {
	// repositories
	renderProfileRow(minderv1.RepositoryEntity, p.Repository, t)

	// build_environments
	renderProfileRow(minderv1.BuildEnvironmentEntity, p.BuildEnvironment, t)

	// artifacts
	renderProfileRow(minderv1.ArtifactEntity, p.Artifact, t)

	// pull request
	renderProfileRow(minderv1.PullRequestEntity, p.PullRequest, t)
}

func renderProfileRow(entType minderv1.EntityType, rs []*minderv1.Profile_Rule, t table.Table) {
	for idx := range rs {
		rule := rs[idx]
		params := marshalStructOrEmpty(rule.Params)
		def := marshalStructOrEmpty(rule.Def)

		t.AddRow(
			entType.String(),
			rule.Type,
			params,
			def,
		)
	}
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
	for _, eval := range statuses {
		t.AddRowWithColor(
			layouts.NoColor(eval.RuleId),
			layouts.NoColor(eval.RuleDescriptionName),
			layouts.NoColor(eval.Entity),
			getColoredEvalStatus(eval.Status),
			getRemediateStatusColor(eval.RemediationStatus),
			layouts.NoColor(mapToYAMLOrEmpty(eval.EntityInfo)),
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
	case notAvailableStatus, skippedStatus:
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
		return "Success"
	case failureStatus:
		return "Failure"
	case errorStatus:
		return "Error"
	case skippedStatus:
		return "Skipped"
	case pendingStatus:
		return "Pending"
	default:
		return "Unknown"
	}
}

// Gets a friendly status text with an emoji
func getRemediationStatusText(status string) string {
	// remediation statuses can be 'success', 'failure', 'error', 'skipped', 'pending' or 'not supported'
	switch strings.ToLower(status) {
	case successStatus:
		return "Success"
	case failureStatus:
		return "Failure"
	case errorStatus:
		return "Error"
	case skippedStatus:
		return "Skipped" // visually empty as we didn't have to remediate
	case pendingStatus:
		return "Pending"
	case notAvailableStatus:
		return "Not Available"
	default:
		return "Unknown"
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
