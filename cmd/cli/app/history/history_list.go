// Copyright 2024 Stacklok, Inc.
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

package history

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/cmd/cli/app"
	"github.com/stacklok/minder/cmd/cli/app/common"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	"github.com/stacklok/minder/internal/util/cli/table"
	"github.com/stacklok/minder/internal/util/cli/table/layouts"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List history",
	Long:  `The history list subcommand lets you list history within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(listCommand),
}

// listCommand is the profile "list" subcommand
func listCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewEvalResultsServiceClient(conn)
	//client := minderv1.NewProfileServiceClient(conn)

	project := viper.GetString("project")
	profileName := viper.GetString("profileName")
	entityName := viper.GetString("entityName")
	entityType := viper.GetString("entityType")
	evalStatus := viper.GetString("evalStatus")
	remediationStatus := viper.GetString("remediationStatus")
	alertStatus := viper.GetString("alertStatus")
	format := viper.GetString("output")

	// Ensure the output format is supported
	if !app.IsOutputFormatSupported(format) {
		return cli.MessageAndError(fmt.Sprintf("Output format %s not supported", format), fmt.Errorf("invalid argument"))
	}

	// validate the filters which need validation
	evalStatusFilter, err := valitedFilter(evalStatus, evalStatuses)
	if err != nil {
		return err
	}

	remediationStatusFilter, err := valitedFilter(remediationStatus, remediationStatuses)
	if err != nil {
		return err
	}

	alertStatusFilter, err := valitedFilter(alertStatus, alertStatuses)
	if err != nil {
		return err
	}

	entityTypeFilter, err := valitedFilter(entityType, entityTypes)
	if err != nil {
		return err
	}

	// list all the things
	resp, err := client.ListEvaluationHistory(ctx, &minderv1.ListEvaluationHistoryRequest{
		Context:     &minderv1.Context{Project: &project},
		EntityType:  entityTypeFilter,
		EntityName:  asFilter(entityName),
		ProfileName: asFilter(profileName),
		Status:      evalStatusFilter,
		Remediation: remediationStatusFilter,
		Alert:       alertStatusFilter,
		From:        nil,
		To:          nil,
		Cursor:      nil,
	})
	if err != nil {
		return cli.MessageAndError("Error getting profile status", err)
	}

	switch format {
	case app.JSON:
		out, err := util.GetJsonFromProto(resp)
		if err != nil {
			return cli.MessageAndError("Error getting json from proto", err)
		}
		cmd.Println(out)
	case app.YAML:
		out, err := util.GetYamlFromProto(resp)
		if err != nil {
			return cli.MessageAndError("Error getting yaml from proto", err)
		}
		cmd.Println(out)
	case app.Table:
		historyTable := table.New(table.Simple, layouts.EvaluationHistory, nil)
		renderRuleEvaluationStatusTable(resp.Data, historyTable)
		historyTable.Render()
	}

	return nil
}

func asFilter(filter string) []string {
	if filter == "" {
		return nil
	}
	return []string{filter}
}

func valitedFilter(filter string, acceptedValues []string) ([]string, error) {
	if filter == "" {
		return nil, nil
	}
	// since the CLI uses capitalized statuses, lowercase to be safe
	filter = strings.ToLower(filter)
	if !slices.Contains(acceptedValues, filter) {
		return nil, fmt.Errorf("unexpected filter value %s expected one of %s", filter, strings.Join(acceptedValues, ", "))
	}
	return []string{filter}, nil
}

func renderRuleEvaluationStatusTable(
	statuses []*minderv1.EvaluationHistory,
	t table.Table,
) {
	for _, eval := range statuses {
		t.AddRowWithColor(
			layouts.NoColor(eval.EvaluatedAt.AsTime().Format(time.DateTime)),
			layouts.NoColor(eval.Rule.Name),
			layouts.NoColor(eval.Entity.Name),
			common.GetEvalStatusColor(eval.Status.Status),
			common.GetRemediateStatusColor(eval.Remediation.Status),
			common.GetAlertStatusColor(eval.Alert.Status),
		)
	}
}

func init() {
	historyCmd.AddCommand(listCmd)

	basicMsg := "Filter evaluation history list by %s - one of %s"
	evalFilterMsg := fmt.Sprintf(basicMsg, "evaluation status", strings.Join(evalStatuses, ", "))
	remediationFilterMsg := fmt.Sprintf(basicMsg, "remediation status", strings.Join(remediationStatuses, ", "))
	alertFilterMsg := fmt.Sprintf(basicMsg, "alert status", strings.Join(alertStatuses, ", "))
	entityTypesMsg := fmt.Sprintf(basicMsg, "entity type", strings.Join(entityTypes, ", "))

	// Flags
	listCmd.Flags().String("ruleName", "", "Filter evaluation history list by rule name")
	listCmd.Flags().String("profileName", "", "Filter evaluation history list by profile name")
	listCmd.Flags().String("entityName", "", "Filter evaluation history list by entity name")
	listCmd.Flags().String("entityType", "", entityTypesMsg)
	listCmd.Flags().String("evalStatus", "", evalFilterMsg)
	listCmd.Flags().String("remediationStatus", "", remediationFilterMsg)
	listCmd.Flags().String("alertStatus", "", alertFilterMsg)
}

// TODO: we should have a common set of enums and validators in `internal`

var evalStatuses = []string{
	string(db.EvalStatusTypesPending),
	string(db.EvalStatusTypesFailure),
	string(db.EvalStatusTypesError),
	string(db.EvalStatusTypesSuccess),
	string(db.EvalStatusTypesSkipped),
}

var remediationStatuses = []string{
	string(db.RemediationStatusTypesFailure),
	string(db.RemediationStatusTypesFailure),
	string(db.RemediationStatusTypesError),
	string(db.RemediationStatusTypesSuccess),
	string(db.RemediationStatusTypesSkipped),
	string(db.RemediationStatusTypesNotAvailable),
}

var alertStatuses = []string{
	string(db.AlertStatusTypesOff),
	string(db.AlertStatusTypesOn),
	string(db.AlertStatusTypesError),
	string(db.AlertStatusTypesSkipped),
	string(db.AlertStatusTypesNotAvailable),
}

var entityTypes = []string{
	string(db.EntitiesRepository),
	string(db.EntitiesArtifact),
	string(db.EntitiesPullRequest),
}
