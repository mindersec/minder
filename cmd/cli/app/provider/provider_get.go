//
// Copyright 2024 Stacklok, Inc.
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

package provider

import (
	"context"
	"fmt"
	"strings"

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

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a given provider available in a specific project",
	Long:  `The minder provider get command gets a given provider available in a specific project.`,
	RunE:  cli.GRPCClientWrapRunE(GetProviderCommand),
}

func init() {
	ProviderCmd.AddCommand(getCmd)

	getCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
	getCmd.Flags().StringP("name", "n", "", "Name of the provider to get")
	if err := getCmd.MarkFlagRequired("name"); err != nil {
		panic(err)
	}
}

// GetProviderCommand lists the providers available in a specific project
func GetProviderCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewProvidersServiceClient(conn)

	project := viper.GetString("project")
	format := viper.GetString("output")
	name := viper.GetString("name")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	out, err := client.GetProvider(ctx, &minderv1.GetProviderRequest{
		Context: &minderv1.Context{
			Project: &project,
		},
		Name: name,
	})
	if err != nil {
		return cli.MessageAndError("Failed to get provider", err)
	}

	switch format {
	case app.JSON:
		out, err := util.GetJsonFromProto(out.GetProvider())
		if err != nil {
			return err
		}
		fmt.Println(out)
	case app.YAML:
		out, err := util.GetYamlFromProto(out.GetProvider())
		if err != nil {
			return err
		}
		fmt.Println(out)
	case app.Table:
		t := table.New(table.Simple, layouts.Default, []string{"Key", "Value"})
		p := out.GetProvider()

		impls := getImplementsAsStrings(p)
		afs := getAuthFlowsAsStrings(p)

		t.AddRow("Name", p.GetName())
		t.AddRow("Project", p.GetProject())
		t.AddRow("Version", p.GetVersion())
		t.AddRow("Implements", strings.Join(impls, ", "))
		t.AddRow("Auth Flows", strings.Join(afs, ", "))

		t.Render()
		return nil
	default:
		return fmt.Errorf("output format %s not supported", format)
	}

	return nil
}
