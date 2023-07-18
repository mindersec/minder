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

package pkg

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/stacklok/mediator/internal/util"
	"github.com/stacklok/mediator/pkg/auth"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

// repo_listCmd represents the list command to list repos with the
// mediator control plane
var pkg_listCmd = &cobra.Command{
	Use:   "list",
	Short: "Get repository in the mediator control plane",
	Long:  `Repo get is used to get a repo with the mediator control plane`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "error binding flags: %s", err)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {

		grpc_host := util.GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := util.GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)
		format := viper.GetString("output")

		provider := util.GetConfigValue("provider", "provider", cmd, "").(string)
		if provider != auth.Github {
			return fmt.Errorf("only %s is supported at this time", auth.Github)
		}
		groupID := viper.GetInt32("group-id")

		switch format {
		case "json":
		case "yaml":
		case "table":
		case "":
		default:
			return fmt.Errorf("invalid output format: %s", format)
		}

		conn, err := util.GetGrpcConnection(grpc_host, grpc_port)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewPackageServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		packages, err := client.ListPackages(
			ctx,
			&pb.ListPackagesRequest{
				Provider: provider,
				GroupId:  groupID,
				Limit:    1,
				Offset:   0,
			},
		)

		if err != nil {
			return fmt.Errorf("error getting packages: %s", err)
		}

		switch format {
		case "", "table":
			table := tablewriter.NewWriter(os.Stdout)

			table.SetHeader([]string{"Package ID", "Name", "Latest version ID", "Latest tag",
				"Latest tag is signed", "Latest tag creation date"})

			for _, pkg_item := range packages.Results {
				pkgURI := fmt.Sprintf("ghcr.io/%s/%s", pkg_item.GetOwner(), pkg_item.GetName())
				pkgID := fmt.Sprintf("%d", pkg_item.PkgId)
				versionId := fmt.Sprintf("%d", pkg_item.LastVersion.VersionId)
				signed := fmt.Sprintf("%t", pkg_item.LastVersion.IsSigned)
				table.Append([]string{pkgID, pkgURI, versionId, pkg_item.LastVersion.Tag, signed,
					pkg_item.LastVersion.CreatedAt.AsTime().Format(time.RFC3339)})
			}

			table.Render()
		// }
		case "json":
			output, err := json.MarshalIndent(packages.GetResults(), "", "  ")
			util.ExitNicelyOnError(err, "Error marshalling json")
			fmt.Println(string(output))
		case "yaml":
			yamlData, err := yaml.Marshal(packages.GetResults())
			util.ExitNicelyOnError(err, "Error marshalling yaml")
			fmt.Println(string(yamlData))
		}

		return nil
	},
}

func init() {
	PkgCmd.AddCommand(pkg_listCmd)
	pkg_listCmd.Flags().StringP("output", "f", "", "Output format (json or yaml)")
	pkg_listCmd.Flags().StringP("provider", "n", "", "Name for the provider to enroll")
	pkg_listCmd.Flags().Int32P("group-id", "g", 0, "ID of the group for repo registration")
	if err := pkg_listCmd.MarkFlagRequired("provider"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
	}
}
