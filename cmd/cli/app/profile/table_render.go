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
	"time"

	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v2"

	"github.com/stacklok/minder/cmd/cli/app/common"
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
		common.GetEvalStatusColor(ps.ProfileStatus),
		layouts.NoColor(ps.LastUpdated.AsTime().Format(time.RFC3339)),
	)
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
			layouts.NoColor(eval.RuleDescriptionName),
			layouts.NoColor(eval.RuleDisplayName),
			layouts.NoColor(eval.Entity),
			common.GetEvalStatusColor(eval.Status),
			common.GetRemediateStatusColor(eval.RemediationStatus),
			layouts.NoColor(mapToYAMLOrEmpty(eval.EntityInfo)),
		)
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
