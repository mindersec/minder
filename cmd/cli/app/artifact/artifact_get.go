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
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/minder/v1"
)

var artifact_getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get artifact details",
	Long:  `Artifact get will get artifact details from an artifact, for a given ID`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "error binding flags: %s", err)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		tag := util.GetConfigValue("tag", "tag", cmd, "").(string)
		artifactID := viper.GetString("id")
		latest_versions := viper.GetInt32("latest-versions")

		// tag and latest versions cannot be set at same time
		if tag != "" && latest_versions != 1 {
			fmt.Fprintf(os.Stderr, "tag and latest versions cannot be set at the same time")
			os.Exit(1)
		}

		conn, err := util.GrpcForCommand(cmd)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewArtifactServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		// check artifact by name
		art, err := client.GetArtifactById(ctx, &pb.GetArtifactByIdRequest{
			Id:             artifactID,
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
	artifact_getCmd.Flags().StringP("id", "i", "", "ID of the artifact to get info from")
	artifact_getCmd.Flags().Int32P("latest-versions", "v", 1, "Latest artifact versions to retrieve")
	artifact_getCmd.Flags().StringP("tag", "", "", "Specific artifact tag to retrieve")
	if err := artifact_getCmd.MarkFlagRequired("id"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
	}
}
