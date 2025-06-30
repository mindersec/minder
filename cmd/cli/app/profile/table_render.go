// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profile

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v2"

	"github.com/mindersec/minder/internal/util/cli/table"
	"github.com/mindersec/minder/internal/util/cli/table/layouts"
	"github.com/mindersec/minder/internal/util/cli/types"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func marshalStructOrEmpty(v *structpb.Struct) string {
	if v == nil {
		return ""
	}

	m := v.AsMap()

	if len(m) == 0 {
		return ""
	}

	// marhsal as YAML
	out, err := yaml.Marshal(m)
	if err != nil {
		return ""
	}

	return string(out)
}

// NewProfileSettingsTable creates a new table for rendering profile settings
func NewProfileSettingsTable() table.Table {
	return table.New(table.Simple, layouts.Default,
		[]string{"Name", "Description", "Alert", "Remediate"})
}

// RenderProfileSettingsTable renders the profile settings table
func RenderProfileSettingsTable(p *minderv1.Profile, t table.Table) {
	t.AddRow(p.GetName(), p.GetDisplayName(), p.GetAlert(), p.GetRemediate())
}

// NewProfileRulesTable creates a new table for rendering profiles
func NewProfileRulesTable() table.Table {
	return table.New(table.Simple, layouts.Default,
		[]string{"Entity", "Rule", "Rule Params", "Rule Definition"})
}

// RenderProfileRulesTable renders the profile table
func RenderProfileRulesTable(p *minderv1.Profile, t table.Table) {
	// repositories
	renderProfileRow(minderv1.RepositoryEntity, p.Repository, t)

	// build_environments
	renderProfileRow(minderv1.BuildEnvironmentEntity, p.BuildEnvironment, t)

	// artifacts
	renderProfileRow(minderv1.ArtifactEntity, p.Artifact, t)

	// pull request
	renderProfileRow(minderv1.PullRequestEntity, p.PullRequest, t)

	// release
	renderProfileRow(minderv1.ReleaseEntity, p.Release, t)
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
	return table.New(table.Simple, layouts.Default,
		[]string{"Name", "Status", "Evaluated At"})
}

// RenderProfileStatusTable renders the profile status table
func RenderProfileStatusTable(ps *minderv1.ProfileStatus, t table.Table, emoji bool) {
	t.AddRowWithColor(
		layouts.NoColor(ps.ProfileName),
		table.GetStatusIcon(types.ProfileStatus(ps), emoji),
		layouts.NoColor(ps.LastUpdated.AsTime().Format(time.RFC3339)),
	)
}

// NewRuleEvaluationsTable creates a new table for rendering rule evaluations
func NewRuleEvaluationsTable() table.Table {
	return table.New(table.Simple, layouts.Default,
		[]string{"Entity", "Rule Name", "Status", "Details"})
	// TODO: add automerge common cells (name/entity)
}

// RenderRuleEvaluationStatusTable renders the rule evaluations table
func RenderRuleEvaluationStatusTable(
	statuses []*minderv1.RuleEvaluationStatus,
	t table.Table,
	emoji bool,
) {
	// sort by entity
	slices.SortFunc(statuses, func(a *minderv1.RuleEvaluationStatus, b *minderv1.RuleEvaluationStatus) int {
		return strings.Compare(a.EntityInfo["name"], b.EntityInfo["name"])
	})

	for _, eval := range statuses {
		evalInfo := types.RuleEvalStatus(eval)
		t.AddRowWithColor(
			layouts.NoColor(fmt.Sprintf("%s\n[%s]", eval.EntityInfo["name"], eval.Entity)),
			layouts.NoColor(fmt.Sprintf("%s\n[%s]", eval.RuleDescriptionName, eval.RuleTypeName)),
			table.GetStatusIcon(evalInfo, emoji),
			table.BestDetail(evalInfo),
		)
	}
}
