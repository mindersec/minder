// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli"
	"github.com/mindersec/minder/internal/util/cli/table"
	"github.com/mindersec/minder/internal/util/cli/table/layouts"
	"github.com/mindersec/minder/internal/util/cli/types"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List history",
	Long:  `The history list subcommand lets you list history within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(listCommand),
}

const (
	defaultPageSize = 25
)

// listCommand is the profile "list" subcommand
func listCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewEvalResultsServiceClient(conn)

	project := viper.GetString("project")
	profileName := viper.GetStringSlice("profile-name")
	entityName := viper.GetStringSlice("entity-name")
	entityType := viper.GetStringSlice("entity-type")
	evalStatus := viper.GetStringSlice("eval-status")
	remediationStatus := viper.GetStringSlice("remediation-status")
	alertStatus := viper.GetStringSlice("alert-status")
	labels := viper.GetStringSlice("label")

	// time range
	from := viper.GetTime("from")
	to := viper.GetTime("to")

	// page options
	cursorStr := viper.GetString("cursor")
	size := viper.GetUint32("size")

	format := viper.GetString("output")

	// Ensure the output format is supported
	if !app.IsOutputFormatSupported(format) {
		return cli.MessageAndError(fmt.Sprintf("Output format %s not supported", format), fmt.Errorf("invalid argument"))
	}

	// validate the filters which need validation
	if err := validatedFilter(evalStatus, evalStatuses); err != nil {
		return err
	}

	if err := validatedFilter(remediationStatus, remediationStatuses); err != nil {
		return err
	}

	if err := validatedFilter(alertStatus, alertStatuses); err != nil {
		return err
	}

	if err := validatedFilter(entityType, entityTypes); err != nil {
		return err
	}

	// list all the things
	req := &minderv1.ListEvaluationHistoryRequest{
		Context:     &minderv1.Context{Project: &project},
		EntityType:  entityType,
		EntityName:  entityName,
		ProfileName: profileName,
		Status:      evalStatus,
		Remediation: remediationStatus,
		Alert:       alertStatus,
		LabelFilter: labels,
		From:        nil,
		To:          nil,
		Cursor:      cursorFromOptions(cursorStr, size),
	}

	// Viper returns time.Time rather than a pointer to it, so we
	// have to check whether from and/or to were specified by
	// other means.
	if cmd.Flags().Lookup("from").Changed {
		req.From = timestamppb.New(from)
	}
	if cmd.Flags().Lookup("to").Changed {
		req.To = timestamppb.New(to)
	}

	resp, err := client.ListEvaluationHistory(ctx, req)
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
		printTable(cmd.OutOrStderr(), resp, viper.GetBool("emoji"))
	}

	return nil
}

func cursorFromOptions(cursorStr string, size uint32) *minderv1.Cursor {
	var cursor *minderv1.Cursor
	if cursorStr != "" || size != 0 {
		cursor = &minderv1.Cursor{}
	}
	if cursorStr != "" {
		cursor.Cursor = cursorStr
	}
	if size != 0 {
		cursor.Size = size
	}
	return cursor
}

func printTable(w io.Writer, resp *minderv1.ListEvaluationHistoryResponse, emoji bool) {
	historyTable := table.New(table.Simple, layouts.Default,
		[]string{"Time", "Entity", "Rule", "Status"})
	// TODO: add automerge common cells
	renderRuleEvaluationStatusTable(resp.Data, historyTable, emoji)
	historyTable.Render()
	fmt.Println("")
	if next := getNext(resp); next != nil {
		// Ordering is fixed for evaluation history
		// log and the next page points to older
		// records.
		msg := fmt.Sprintf("Older records: %s",
			cli.CursorStyle.Render(next.Cursor),
		)
		fmt.Fprintln(w, msg)
	}
	if prev := getPrev(resp); prev != nil {
		// Ordering is fixed for evaluation history
		// log and the previous page points to newer
		// records.
		msg := fmt.Sprintf("Newer records: %s",
			cli.CursorStyle.Render(prev.Cursor),
		)
		fmt.Fprintln(w, msg)
	}
}

func getNext(resp *minderv1.ListEvaluationHistoryResponse) *minderv1.Cursor {
	if resp.Page != nil && resp.Page.Next != nil {
		return resp.Page.Next
	}
	return nil
}

func getPrev(resp *minderv1.ListEvaluationHistoryResponse) *minderv1.Cursor {
	if resp.Page != nil && resp.Page.Prev != nil {
		return resp.Page.Prev
	}
	return nil
}

func validatedFilter(filters []string, acceptedValues []string) error {
	for _, filter := range filters {
		if !slices.Contains(acceptedValues, filter) {
			return fmt.Errorf("unexpected filter value %s expected one of %s", filter, strings.Join(acceptedValues, ", "))
		}
	}
	return nil
}

func renderRuleEvaluationStatusTable(
	statuses []*minderv1.EvaluationHistory,
	t table.Table,
	emoji bool,
) {
	for _, eval := range statuses {
		t.AddRowWithColor(
			layouts.NoColor(eval.EvaluatedAt.AsTime().Format(time.DateTime)),
			layouts.NoColor(eval.Entity.Name),
			layouts.NoColor(eval.Rule.Name),
			table.GetStatusIcon(types.HistoryStatus(eval), emoji),
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
	listCmd.Flags().StringSlice("profile-name", nil, "Filter evaluation history list by profile name")
	listCmd.Flags().StringSlice("entity-name", nil, "Filter evaluation history list by entity name")
	listCmd.Flags().StringSlice("entity-type", nil, entityTypesMsg)
	listCmd.Flags().StringSlice("eval-status", nil, evalFilterMsg)
	listCmd.Flags().StringSlice("remediation-status", nil, remediationFilterMsg)
	listCmd.Flags().StringSlice("alert-status", nil, alertFilterMsg)
	listCmd.Flags().StringSliceP("label", "l", nil, "Filter evaluation history list by label")
	if err := listCmd.Flags().MarkHidden("label"); err != nil {
		listCmd.Printf("Error hiding flag: %s", err)
		os.Exit(1)
	}
	listCmd.Flags().String("from", "", "Filter evaluation history list by time")
	listCmd.Flags().String("to", "", "Filter evaluation history list by time")
	listCmd.Flags().StringP("cursor", "c", "", "Fetch previous or next page from the list")
	listCmd.Flags().Uint64P("size", "s", defaultPageSize, "Change the number of items fetched")
	listCmd.Flags().Bool("emoji", true, "Use emojis in the output")
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
