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
	"context"

	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/internal/util/cli"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// authWhoamiCmd represents the whoami command
var authWhoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "whoami for current user",
	Long:  `whoami gets information about the current user from the minder server`,
	RunE: cli.GRPCClientWrapRunE(func(ctx context.Context, cmd *cobra.Command, conn *grpc.ClientConn) error {
		client := pb.NewUserServiceClient(conn)

		userInfo, err := client.GetUser(ctx, &pb.GetUserRequest{})
		if err != nil {
			return cli.MessageAndError(cmd, "Error getting information for user", err)
		}

		cli.PrintCmd(cmd, cli.Header.Render("Here are your details:"))
		renderUserInfoWhoami(cmd, conn, userInfo)
		return nil
	}),
}

func init() {
	AuthCmd.AddCommand(authWhoamiCmd)
}

func renderUserInfoWhoami(cmd *cobra.Command, conn *grpc.ClientConn, user *pb.GetUserResponse) {
	subjectKey := "Subject"
	createdKey := "Created At"
	updatedKey := "Updated At"
	minderSrvKey := "Minder Server"
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
			minderSrvKey, conn.Target(),
		},
	}

	rows = append(rows, getProjectTableRows(user.Projects)...)

	renderUserToTable(cmd, rows)
}
