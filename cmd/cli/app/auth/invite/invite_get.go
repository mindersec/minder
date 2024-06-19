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

// Package invite provides the auth invite command for the minder CLI.
package invite

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/cmd/cli/app"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	"github.com/stacklok/minder/internal/util/cli/table"
	"github.com/stacklok/minder/internal/util/cli/table/layouts"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// inviteGetCmd represents the list command
var inviteGetCmd = &cobra.Command{
	Hidden: true, // TODO: This hides the command, remove it once it's implemented
	Use:    "get",
	Short:  "Get info about pending invitations",
	Long:   `Get shows additional information about a pending invitation`,
	RunE:   cli.GRPCClientWrapRunE(inviteGetCommand),
	Args:   cobra.ExactArgs(1),
}

// inviteGetCommand is the invite get subcommand
func inviteGetCommand(ctx context.Context, cmd *cobra.Command, args []string, conn *grpc.ClientConn) error {
	client := minderv1.NewInviteServiceClient(conn)

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true
	format := viper.GetString("output")

	res, err := client.GetInviteDetails(ctx, &minderv1.GetInviteDetailsRequest{
		Code: args[0],
	})
	if err != nil {
		return cli.MessageAndError("Error getting info for invitation", err)
	}

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
	case app.Table:
		t := table.New(table.Simple, layouts.Default, []string{"Sponsor", "Project", "Expires"})
		t.AddRow(res.SponsorDisplay, res.ProjectDisplay, res.ExpiresAt.AsTime().Format(time.RFC3339))
		t.Render()
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
	return nil
}

func init() {
	inviteCmd.AddCommand(inviteGetCmd)
	inviteGetCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
}
