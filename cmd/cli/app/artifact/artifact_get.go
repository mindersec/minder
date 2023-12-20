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
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/cmd/cli/app"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	"github.com/stacklok/minder/internal/util/cli/table"
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
	tag := viper.GetString("tag")
	artifactID := viper.GetString("id")
	latestVersions := viper.GetInt32("versions")
	format := viper.GetString("output")

	// Ensure provider is supported
	if !app.IsProviderSupported(provider) {
		return cli.MessageAndError(fmt.Sprintf("Provider %s is not supported yet", provider), fmt.Errorf("invalid argument"))
	}

	// Ensure the output format is supported
	if !app.IsOutputFormatSupported(format) {
		return cli.MessageAndError(fmt.Sprintf("Output format %s not supported", format), fmt.Errorf("invalid argument"))
	}

	// check artifact by name
	art, err := client.GetArtifactById(ctx, &minderv1.GetArtifactByIdRequest{
		Context:        &minderv1.Context{Provider: &provider, Project: &project},
		Id:             artifactID,
		LatestVersions: latestVersions,
		Tag:            tag,
	})
	if err != nil {
		return cli.MessageAndError("Error getting artifact by id", err)
	}

	switch format {
	case app.Table:
		ta := table.New(table.Simple, "", []string{"ID", "Type", "Owner", "Name", "Repository", "Visibility", "Creation date"})
		ta.AddRow([]string{
			art.Artifact.ArtifactPk,
			art.Artifact.Type,
			art.Artifact.GetOwner(),
			art.Artifact.GetName(),
			art.Artifact.Repository,
			art.Artifact.Visibility,
			art.Artifact.CreatedAt.AsTime().Format(time.RFC3339),
		})
		ta.Render()

		tv := table.New(table.Simple, "", []string{"ID", "Tags", "Signature", "Identity", "Creation date"})
		for _, version := range art.Versions {
			tv.AddRow([]string{
				fmt.Sprintf("%d", version.VersionId),
				strings.Join(version.Tags, ","),
				getSignatureStatusText(version.SignatureVerification),
				version.GetSignatureVerification().GetCertIdentity(),
				version.CreatedAt.AsTime().Format(time.RFC3339),
			})
		}
		tv.Render()
	case app.JSON:
		out, err := util.GetJsonFromProto(art)
		if err != nil {
			return cli.MessageAndError("Error getting json from proto", err)
		}
		cmd.Println(out)
	case app.YAML:
		out, err := util.GetYamlFromProto(art)
		if err != nil {
			return cli.MessageAndError("Error getting yaml from proto", err)
		}
		cmd.Println(out)
	}

	return nil
}

func getSignatureStatusText(sigVer *minderv1.SignatureVerification) string {
	if !sigVer.IsSigned {
		return "❌ not signed"
	}
	if !sigVer.IsVerified {
		return "❌ signature not verified"
	}
	if !sigVer.IsBundleVerified {
		return "❌ bundle signature not verified"
	}
	return "✅ Success"
}

func init() {
	ArtifactCmd.AddCommand(getCmd)
	// Flags
	getCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
	getCmd.Flags().StringP("id", "i", "", "ID of the artifact to get info from")
	getCmd.Flags().Int32P("versions", "v", 1, "Latest artifact versions to retrieve")
	getCmd.Flags().StringP("tag", "", "", "Specific artifact tag to retrieve")
	// Required
	if err := getCmd.MarkFlagRequired("id"); err != nil {
		getCmd.Printf("Error marking flag as required: %s", err)
		os.Exit(1)
	}
	// Exclusive
	getCmd.MarkFlagsMutuallyExclusive("versions", "tag")
}
