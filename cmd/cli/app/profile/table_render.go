// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profile

import (
	"cmp"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli/table"
	"github.com/mindersec/minder/internal/util/cli/table/layouts"
	"github.com/mindersec/minder/internal/util/cli/types"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func marshalStructOrEmpty(v *structpb.Struct) string {
	if v == nil || len(v.GetFields()) == 0 {
		return ""
	}

	out, err := util.GetYamlFromProto(v)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// NewProfileSettingsTable creates a new table for rendering profile settings
func NewProfileSettingsTable(out io.Writer) table.Table {
	return table.New(table.Simple, layouts.Default, out,
		[]string{"Name", "Description", "Alert", "Remediate"}).
		SetAutoMerge(true).
		SetEqualColumns(false) // Divided equally across terminal width
}

// RenderProfileSettingsTable renders the profile settings table
func RenderProfileSettingsTable(p *minderv1.Profile, t table.Table) {
	t.AddRow(p.GetName(), p.GetDisplayName(), p.GetAlert(), p.GetRemediate())
}

// NewProfileRulesTable creates a new table for rendering profiles
func NewProfileRulesTable(out io.Writer) table.Table {
	return table.New(table.Simple, layouts.Default, out,
		[]string{"Entity", "Rule", "Rule Params", "Rule Definition"}).
		SetAutoMerge(true).
		SetEqualColumns(false) // Divided equally across terminal width
}

// RenderProfileRulesTable renders the profile table
func RenderProfileRulesTable(p *minderv1.Profile, t table.Table) {
	renderProfileRow(minderv1.RepositoryEntity, p.Repository, t)
	renderProfileRow(minderv1.BuildEnvironmentEntity, p.BuildEnvironment, t)
	renderProfileRow(minderv1.ArtifactEntity, p.Artifact, t)
	renderProfileRow(minderv1.PullRequestEntity, p.PullRequest, t)
	renderProfileRow(minderv1.ReleaseEntity, p.Release, t)
}

func renderProfileRow(entType minderv1.EntityType, rs []*minderv1.Profile_Rule, t table.Table) {
	for _, rule := range rs {
		t.AddRow(
			entType.String(),
			rule.Type,
			marshalStructOrEmpty(rule.Params),
			marshalStructOrEmpty(rule.Def),
		)
	}
}

// NewProfileStatusTable creates a new table for rendering profile status
func NewProfileStatusTable(out io.Writer) table.Table {
	// Status tables are usually better "Compact" (default) because they have short fields
	return table.New(table.Simple, layouts.Default, out,
		[]string{"Name", "Status", "Evaluated At"}).
		SetAutoMerge(true).
		SetEqualColumns(false)
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
func NewRuleEvaluationsTable(out io.Writer) table.Table {
	return table.New(table.Simple, layouts.Default, out,
		[]string{"Entity", "Rule", "Result", "Details"}).
		SetAutoMerge(true).
		SetEqualColumns(false)
}

func buildLabeledBlock(label, detail, url string) string {
	detail = strings.TrimSpace(detail)
	url = strings.TrimSpace(url)

	if detail == "" && url == "" {
		return ""
	}

	if detail == "" {
		return fmt.Sprintf("%s: %s", label, url)
	}

	block := fmt.Sprintf("%s: %s", label, detail)
	if url != "" {
		block += fmt.Sprintf("\nURL: %s", url)
	}

	return block
}

func buildDetailSummary(eval *minderv1.RuleEvaluationStatus) string {
	sections := make([]string, 0, 4)

	if alert := eval.GetAlert(); alert != nil {
		if section := buildLabeledBlock("Alert", alert.GetDetails(), alert.GetUrl()); section != "" {
			sections = append(sections, section)
		}
	}

	if section := buildLabeledBlock("Remediation", eval.GetRemediationDetails(), eval.GetRemediationUrl()); section != "" {
		sections = append(sections, section)
	}

	if detail := strings.TrimSpace(eval.GetDetails()); detail != "" {
		sections = append(sections, fmt.Sprintf("Details: %s", detail))
	}

	if guidance := strings.TrimSpace(eval.GetGuidance()); guidance != "" {
		sections = append(sections, fmt.Sprintf("Guidance: %s", guidance))
	}

	return strings.Join(sections, "\n")
}

// RuleDisplayName returns the most user-friendly rule name available.
func RuleDisplayName(eval *minderv1.RuleEvaluationStatus) string {
	return strings.TrimSpace(cmp.Or(eval.GetRuleDescriptionName(), eval.GetRuleDisplayName(), eval.GetRuleTypeName()))
}

// FormatEvaluationReasoning returns a rendered reasoning block for CLI output.
func FormatEvaluationReasoning(eval *minderv1.RuleEvaluationStatus) string {
	return cmp.Or(buildDetailSummary(eval), "-")
}

// RenderRuleEvaluationStatusTable renders the rule evaluations table.
func RenderRuleEvaluationStatusTable(
	statuses []*minderv1.RuleEvaluationStatus,
	t table.Table,
	emoji bool,
) {
	slices.SortFunc(statuses, func(a, b *minderv1.RuleEvaluationStatus) int {
		if sort := strings.Compare(a.EntityInfo["name"], b.EntityInfo["name"]); sort != 0 {
			return sort
		}
		return strings.Compare(a.Entity, b.Entity)
	})

	for _, eval := range statuses {
		evalInfo := types.RuleEvalStatus(eval)
		ruleName := RuleDisplayName(eval)
		reasoning := FormatEvaluationReasoning(eval)
		t.AddRowWithColor(
			layouts.NoColor(fmt.Sprintf("%s\n[%s]", eval.EntityInfo["name"], eval.Entity)),
			layouts.NoColor(ruleName),
			table.GetStatusIcon(evalInfo, emoji),
			layouts.NoColor(reasoning),
		)
	}
}
