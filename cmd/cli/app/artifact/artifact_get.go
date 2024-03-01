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

package artifact

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/stacklok/minder/cmd/cli/app"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	"github.com/stacklok/minder/internal/util/cli/table"
	"github.com/stacklok/minder/internal/util/cli/table/layouts"
	"github.com/stacklok/minder/internal/util/jsonyaml"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get artifact details",
	Long:  `The artifact get subcommand will get artifact details from an artifact, for a given ID.`,
	RunE:  cli.GRPCClientWrapRunE(getCommand),
}

// getCommand is the artifact get subcommand
func getCommand(ctx context.Context, cmd *cobra.Command, conn *grpc.ClientConn) error {
	client := minderv1.NewArtifactServiceClient(conn)

	provider := viper.GetString("provider")
	project := viper.GetString("project")
	artifactID := viper.GetString("id")
	artifactName := viper.GetString("name")
	format := viper.GetString("output")

	// Ensure provider is supported
	if !app.IsProviderSupported(provider) {
		return cli.MessageAndError(fmt.Sprintf("Provider %s is not supported yet", provider), fmt.Errorf("invalid argument"))
	}

	// Ensure the output format is supported
	if !app.IsOutputFormatSupported(format) {
		return cli.MessageAndError(fmt.Sprintf("Output format %s not supported", format), fmt.Errorf("invalid argument"))
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	pbArt, art, err := artifactGet(ctx, client, provider, project, artifactID, artifactName)
	if err != nil {
		return cli.MessageAndError("Error getting artifact", err)
	}

	if err := printArtifact(cmd, pbArt, art, format); err != nil {
		return cli.MessageAndError("Error printing artifact", err)
	}

	evalStatus, err := artifactEvalStatus(ctx, conn, art, provider, project)
	if err != nil {
		return cli.MessageAndError("Error getting artifact evaluation status", err)
	}

	if err := printEvalStatus(cmd, evalStatus, format); err != nil {
		return cli.MessageAndError("Error printing artifact evaluation status", err)
	}

	return nil
}

func artifactGet(
	ctx context.Context,
	client minderv1.ArtifactServiceClient,
	provider string, project string,
	artifactID string, artifactName string,
) (pbArt protoreflect.ProtoMessage, art *minderv1.Artifact, err error) {

	if artifactName != "" {
		// check artifact by Name
		artByName, errGet := client.GetArtifactByName(ctx, &minderv1.GetArtifactByNameRequest{
			Context: &minderv1.Context{Provider: &provider, Project: &project},
			Name:    artifactName,
		})
		if errGet != nil {
			err = fmt.Errorf("error getting artifact by name: %w", errGet)
			return
		}
		pbArt = artByName
		art = artByName.GetArtifact()
		return
	} else if artifactID != "" {
		// check artifact by ID
		artById, errGet := client.GetArtifactById(ctx, &minderv1.GetArtifactByIdRequest{
			Context: &minderv1.Context{Provider: &provider, Project: &project},
			Id:      artifactID,
		})
		if errGet != nil {
			err = fmt.Errorf("error getting artifact by id: %w", errGet)
			return
		}
		pbArt = artById
		art = artById.GetArtifact()
		return
	}

	err = errors.New("neither name nor ID set")
	return
}

func artifactEvalStatus(
	ctx context.Context, conn *grpc.ClientConn,
	artifact *minderv1.Artifact,
	provider, project string,
) ([]*minderv1.RuleEvaluationStatus, error) {
	profClient := minderv1.NewProfileServiceClient(conn)
	profiles, err := profClient.ListProfiles(ctx, &minderv1.ListProfilesRequest{
		Context: &minderv1.Context{
			Provider: &provider,
			Project:  &project,
		},
	})
	if err != nil {
		return nil, cli.MessageAndError("Error getting profiles", err)
	}

	var respList []*minderv1.RuleEvaluationStatus

	for _, profile := range profiles.Profiles {
		req := &minderv1.GetProfileStatusByNameRequest{
			Context: &minderv1.Context{
				Provider: &provider,
				Project:  &project,
			},
			Name: profile.GetName(),
			Entity: &minderv1.EntityTypedId{
				Id:   artifact.ArtifactPk,
				Type: minderv1.Entity_ENTITY_ARTIFACTS,
			},
		}

		resp, err := profClient.GetProfileStatusByName(ctx, req)
		if err != nil {
			return nil, cli.MessageAndError("Error getting profile status", err)
		}
		respList = append(respList, resp.GetRuleEvaluationStatus()...)
	}

	return respList, nil
}

func printArtifact(
	cmd *cobra.Command, pbArt protoreflect.ProtoMessage, art *minderv1.Artifact, format string,
) error {
	switch format {
	case app.Table:
		ta := table.New(table.Simple, layouts.Default,
			[]string{"ID", "Type", "Owner", "Name", "Repository", "Visibility", "Creation date"})
		ta.AddRow(
			art.ArtifactPk,
			art.Type,
			art.GetOwner(),
			art.GetName(),
			art.Repository,
			art.Visibility,
			art.CreatedAt.AsTime().Format(time.RFC3339),
		)
		ta.Render()
	case app.JSON:
		out, err := util.GetJsonFromProto(pbArt)
		if err != nil {
			return cli.MessageAndError("Error getting json from proto", err)
		}
		cmd.Println(out)
	case app.YAML:
		out, err := util.GetYamlFromProto(pbArt)
		if err != nil {
			return cli.MessageAndError("Error getting yaml from proto", err)
		}
		cmd.Println(out)
	}

	return nil
}

func printEvalStatus(
	cmd *cobra.Command, evalStatus []*minderv1.RuleEvaluationStatus, format string,
) error {
	switch format {
	case app.Table:
		ta := table.New(table.Simple, layouts.Default,
			[]string{"Profile", "Rule", "Status", "Message"})
		for _, status := range evalStatus {
			ta.AddRow(
				status.ProfileId,
				status.RuleTypeName,
				status.Status,
				status.Details,
			)
		}
		ta.Render()
	case app.JSON, app.YAML:
		jsonList := make([]string, 0)
		for _, es := range evalStatus {
			jsonEs, err := util.GetJsonFromProto(es)
			if err != nil {
				return cli.MessageAndError("Error getting json from proto", err)
			}
			jsonList = append(jsonList, jsonEs)
		}

		jsonArray := "[" + strings.Join(jsonList, ",") + "]"
		if format == app.JSON {
			var prettyBuffer bytes.Buffer
			err := json.Indent(&prettyBuffer, []byte(jsonArray), "", "  ")
			if err != nil {
				return cli.MessageAndError("Error indenting json", err)
			}
			cmd.Println(prettyBuffer.String())
		} else {
			raw := json.RawMessage(jsonArray)
			yamlResult, err := jsonyaml.ConvertJsonToYaml(raw)
			if err != nil {
				return cli.MessageAndError("Error converting json to yaml", err)
			}
			cmd.Println(yamlResult)
		}
	}

	return nil
}

func init() {
	ArtifactCmd.AddCommand(getCmd)
	// Flags
	getCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
	getCmd.Flags().StringP("name", "n", "", "name of the artifact to get info from in the form repoOwner/repoName/artifactName")
	getCmd.Flags().StringP("id", "i", "", "ID of the artifact to get info from")
	// We allow searching by name or ID but not both. One of them must be specified.
	getCmd.MarkFlagsMutuallyExclusive("name", "id")
	getCmd.MarkFlagsOneRequired("name", "id")
}
