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

package auth

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// authWhoamiCmd represents the whoami command
var authWhoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "whoami for current user",
	Long:  `whoami gets information about the current user from the minder server`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := util.GetAppContext()
		defer cancel()

		conn, err := util.GrpcForCommand(cmd, viper.GetViper())
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewUserServiceClient(conn)

		userInfo, err := client.GetUser(ctx, &pb.GetUserRequest{})
		util.ExitNicelyOnError(err, "Error getting information for user")

		cli.PrintCmd(cmd, cli.Header.Render("Here are your details:"))
		renderUserInfoWhoami(cmd, conn, userInfo)
	},
}

func init() {
	AuthCmd.AddCommand(authWhoamiCmd)
}

func renderUserInfoWhoami(cmd *cobra.Command, conn *grpc.ClientConn, user *pb.GetUserResponse) {
	projects := []string{}
	for _, project := range user.Projects {
		projects = append(projects, fmt.Sprintf("%s:%s", project.GetName(), project.GetProjectId()))
	}

	subjectKey := "Subject"
	createdKey := "Created At"
	updatedKey := "Updated At"
	minderSrvKey := "Minder Server"
	projectKey := "Projects"
	if len(projects) > 1 {
		projectKey += "s"
	}
	rows := []table.Row{
		{
			subjectKey, user.GetUser().GetIdentitySubject(),
		},
		{
			createdKey, user.GetUser().GetCreatedAt().AsTime().String(),
		},
		{
			updatedKey, user.GetUser().GetUpdatedAt().AsTime().String(),
		},
		{
			projectKey, strings.Join(projects, ", "),
		},
		{
			minderSrvKey, conn.Target(),
		},
	}

	renderUserToTable(cmd, rows)
}
