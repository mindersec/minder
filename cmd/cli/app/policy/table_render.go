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

package policy

import (
	"fmt"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v2"

	"github.com/stacklok/mediator/pkg/entities"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

func initializeTable(cmd *cobra.Command) *tablewriter.Table {
	table := tablewriter.NewWriter(cmd.OutOrStdout())
	table.SetHeader([]string{"Id", "Name", "Provider", "Entity", "Context", "Rule", "Rule Params", "Rule Definition"})
	table.SetRowLine(true)
	table.SetRowSeparator("-")
	table.SetAutoMergeCellsByColumnIndex([]int{0, 1, 2, 3, 4, 5})
	// This is needed for the rule definition and rule parameters
	table.SetAutoWrapText(false)

	return table
}

func renderPolicyTable(
	p *pb.PipelinePolicy,
	table *tablewriter.Table,
) {
	// repositories
	renderEntityRuleSets(p, entities.RepositoryEntity, p.Repository, table)

	// build_environments
	renderEntityRuleSets(p, entities.BuildEnvironmentEntity, p.BuildEnvironment, table)

	// artifacts
	renderEntityRuleSets(p, entities.ArtifactEntity, p.Artifact, table)

	// artifacts
	renderEntityRuleSets(p, entities.PullRequestEntity, p.PullRequest, table)
}

func renderEntityRuleSets(
	p *pb.PipelinePolicy,
	entType entities.EntityType,
	rs []*pb.PipelinePolicy_ContextualRuleSet,
	table *tablewriter.Table,
) {
	for idx := range rs {
		rs := rs[idx]

		renderContextualRuleSetTable(p, entType, rs, table)
	}
}

func renderContextualRuleSetTable(
	p *pb.PipelinePolicy,
	entType entities.EntityType,
	ctxrs *pb.PipelinePolicy_ContextualRuleSet,
	table *tablewriter.Table,
) {
	for idx := range ctxrs.Rules {
		rule := ctxrs.Rules[idx]

		ruleCtx := ""
		if ctxrs.Context != nil {
			ruleCtx = *ctxrs.Context
		}

		renderRuleTable(p, entType, ruleCtx, rule, table)
	}
}

func renderRuleTable(
	p *pb.PipelinePolicy,
	entType entities.EntityType,
	ruleContext string,
	rule *pb.PipelinePolicy_Rule,
	table *tablewriter.Table,
) {

	params := marshalStructOrEmpty(rule.Params)
	def := marshalStructOrEmpty(rule.Def)

	row := []string{
		fmt.Sprintf("%d", *p.Id),
		p.Name,
		p.Context.Provider,
		entType.String(),
		ruleContext,
		rule.Type,
		params,
		def,
	}
	table.Append(row)
}

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
