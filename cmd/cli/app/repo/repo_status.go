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

package repo

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/cmd/cli/app"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get repository evaluation status",
	Long:  `The repo status subcommand is used to get the evaluation status for a registered repository within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(statusCommand),
}

// statusCommand is the repo evaluation status subcommand
func statusCommand(ctx context.Context, cmd *cobra.Command, conn *grpc.ClientConn) error {
	client := minderv1.NewEvalResultsServiceClient(conn)

	provider := viper.GetString("provider")
	project := viper.GetString("project")
	format := viper.GetString("output")
	entityId := viper.GetString("entity")
	ruletypes := viper.GetString("ruletypes")
	profile := viper.GetString("profile")
	labels := viper.GetString("labels")

	// Ensure provider is supported
	if !app.IsProviderSupported(provider) {
		return cli.MessageAndError(fmt.Sprintf("Provider %s is not supported yet", provider), fmt.Errorf("invalid argument"))
	}

	// Ensure the output format is supported
	if !app.IsOutputFormatSupported(format) || format == app.Table {
		return cli.MessageAndError(fmt.Sprintf("Output format %s not supported", format), fmt.Errorf("invalid argument"))
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	// Prepare the evaluation results request
	req := &minderv1.ListEvaluationResultsRequest{
		Context:  &minderv1.Context{Provider: &provider, Project: &project},
		Entity:   []*minderv1.EntityTypedId{{Type: minderv1.Entity_ENTITY_REPOSITORIES, Id: entityId}},
		RuleName: strings.Split(ruletypes, ","),
	}

	// Set the profile or labels. Note they are set as mutually exclusive in Cobra too.
	if profile != "" {
		req.ProfileSelector = &minderv1.ListEvaluationResultsRequest_Profile{
			Profile: profile,
		}
	} else if labels != "" {
		req.ProfileSelector = &minderv1.ListEvaluationResultsRequest_LabelFilter{
			LabelFilter: labels,
		}
	}

	// Get the evaluation results
	res, err := client.ListEvaluationResults(ctx, req)
	if err != nil {
		return cli.MessageAndError("Error listing evaluation results", err)
	}

	// Print result in JSON or YAML format
	// TODO: Add table format
	switch format {
	case app.JSON:
		out, err := util.GetJsonFromProto(res)
		if err != nil {
			return cli.MessageAndError("Error getting json from proto", err)
		}
		cmd.Println(out)
	case app.YAML:
		out, err := util.GetYamlFromProto(res)
		if err != nil {
			return cli.MessageAndError("Error getting yaml from proto", err)
		}
		cmd.Println(out)
	}

	return nil
}

func init() {
	RepoCmd.AddCommand(statusCmd)
	// Flags
	statusCmd.Flags().StringP("output", "o", app.JSON,
		fmt.Sprintf("Output format (one of %s)", strings.Join([]string{app.JSON, app.YAML}, ",")))
	statusCmd.Flags().StringP("entity", "e", "", "Entity ID to get evaluation status for")
	statusCmd.Flags().StringP("ruletypes", "r", "", "Query by ruletypes, i.e ruletypes=rule1,rule2")
	statusCmd.Flags().StringP("profile", "n", "", "Query by a profile")
	statusCmd.Flags().StringP("labels", "l", "", "Query by labels")
	// Exclusive flags
	statusCmd.MarkFlagsMutuallyExclusive("profile", "labels")

}
