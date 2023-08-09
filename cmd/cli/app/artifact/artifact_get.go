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
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

// repo_listCmd represents the list command to list repos with the
// mediator control plane
var artifact_getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get artifact details",
	Long:  `Artifact get will get artifact details from an artifact, for a given type and name`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "error binding flags: %s", err)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {

		grpc_host := util.GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := util.GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)

		provider := util.GetConfigValue("provider", "provider", cmd, "").(string)
		artifact_type := util.GetConfigValue("type", "type", cmd, "").(string)
		name := util.GetConfigValue("name", "name", cmd, "").(string)
		tag := util.GetConfigValue("tag", "tag", cmd, "").(string)
		groupID := viper.GetInt32("group-id")
		latest_versions := viper.GetInt32("latest-versions")

		// tag and latest versions cannot be set at same time
		if tag != "" && latest_versions != 1 {
			fmt.Fprintf(os.Stderr, "tag and latest versions cannot be set at the same time")
			os.Exit(1)
		}

		conn, err := util.GetGrpcConnection(grpc_host, grpc_port)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewArtifactServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		// check artifact by name
		art, err := client.GetArtifactByName(ctx, &pb.GetArtifactByNameRequest{
			Provider:       provider,
			GroupId:        groupID,
			ArtifactType:   artifact_type,
			Name:           name,
			LatestVersions: latest_versions,
			Tag:            tag,
		})
		util.ExitNicelyOnError(err, "Error getting repo by id")
		out, err := util.GetJsonFromProto(art)
		util.ExitNicelyOnError(err, "Error getting json from proto")
		fmt.Println(out)
		return nil
	},
}

func init() {
	ArtifactCmd.AddCommand(artifact_getCmd)
	artifact_getCmd.Flags().StringP("provider", "p", "", "Name for the provider to enroll")
	artifact_getCmd.Flags().Int32P("group-id", "g", 0, "ID of the group for repo registration")
	artifact_getCmd.Flags().StringP("type", "t", "",
		"Type of the artifact to get info from (npm, maven, rubygems, docker, nuget, container)")
	artifact_getCmd.Flags().StringP("name", "n", "", "Name of the artifact to get info from")
	artifact_getCmd.Flags().Int32P("latest-versions", "v", 1, "Latest artifact versions to retrieve")
	artifact_getCmd.Flags().StringP("tag", "", "", "Specific artifact tag to retrieve")
	if err := artifact_getCmd.MarkFlagRequired("provider"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
	}
	if err := artifact_getCmd.MarkFlagRequired("type"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
	}
	if err := artifact_getCmd.MarkFlagRequired("name"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
	}

}
