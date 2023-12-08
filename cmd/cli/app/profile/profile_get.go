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

package profile

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
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var profile_getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details for a profile within a minder control plane",
	Long: `The minder profile get subcommand lets you retrieve details for a profile within a
minder control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	RunE: cli.GRPCClientWrapRunE(func(ctx context.Context, cmd *cobra.Command, conn *grpc.ClientConn) error {
		provider := viper.GetString("provider")
		format := viper.GetString("output")

		if format != app.JSON && format != app.YAML && format != app.Table {
			return fmt.Errorf("error: invalid format: %s", format)
		}

		client := pb.NewProfileServiceClient(conn)

		id := viper.GetString("id")
		profile, err := client.GetProfileById(ctx, &pb.GetProfileByIdRequest{
			Context: &pb.Context{
				Provider: &provider,
				// TODO set up project if specified
				// Currently it's inferred from the authorization token
			},
			Id: id,
		})
		util.ExitNicelyOnError(err, "Error getting profile")

		switch format {
		case app.YAML:
			out, err := util.GetYamlFromProto(profile)
			util.ExitNicelyOnError(err, "Error getting yaml from proto")
			fmt.Println(out)
		case app.JSON:
			out, err := util.GetJsonFromProto(profile)
			util.ExitNicelyOnError(err, "Error getting json from proto")
			fmt.Println(out)
		case app.Table:
			p := profile.GetProfile()
			handleGetTableOutput(cmd, p)
		}

		return nil
	}),
}

func init() {
	ProfileCmd.AddCommand(profile_getCmd)
	profile_getCmd.Flags().StringP("id", "i", "", "ID for the profile to query")
	profile_getCmd.Flags().StringP("output", "o", app.Table, "Output format (json, yaml or table)")
	profile_getCmd.Flags().StringP("provider", "p", "github", "Provider for the profile")
	// TODO set up project if specified

	if err := profile_getCmd.MarkFlagRequired("id"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}

}

func handleGetTableOutput(cmd *cobra.Command, profile *pb.Profile) {
	table := InitializeTable(cmd)

	RenderProfileTable(profile, table)

	table.Render()
}
