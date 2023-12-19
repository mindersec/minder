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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/cmd/cli/app"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
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

	// Ensure provider is supported
	if !app.IsProviderSupported(provider) {
		return cli.MessageAndError(fmt.Sprintf("Provider %s is not supported yet", provider), fmt.Errorf("invalid argument"))
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

	out, err := util.GetJsonFromProto(art)
	if err != nil {
		return cli.MessageAndError("Error getting json from proto", err)
	}
	cmd.Println(out)
	return nil
}

func init() {
	ArtifactCmd.AddCommand(getCmd)
	// Flags
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
