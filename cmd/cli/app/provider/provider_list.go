// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/mindersec/minder/cmd/cli/app"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/util"
	"github.com/mindersec/minder/pkg/util/cli"
	"github.com/mindersec/minder/pkg/util/cli/table"
	"github.com/mindersec/minder/pkg/util/cli/table/layouts"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List the providers available in a specific project",
	Long:  `The minder provider list command lists the providers available in a specific project.`,
	RunE:  cli.GRPCClientWrapRunE(ListProviderCommand),
}

func init() {
	ProviderCmd.AddCommand(listCmd)

	listCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))

	// TODO: implement pagination in CLI
}

// ListProviderCommand lists the providers available in a specific project
func ListProviderCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {

	client := minderv1.NewProvidersServiceClient(conn)

	project := viper.GetString("project")
	format := viper.GetString("output")

	cursor := ""

	out := &minderv1.ListProvidersResponse{}

	for {
		resp, err := client.ListProviders(ctx, &minderv1.ListProvidersRequest{
			Context: &minderv1.Context{
				Project: &project,
			},
			Cursor: cursor,
		})
		if err != nil {
			return err
		}

		out.Providers = append(out.Providers, resp.Providers...)

		if resp.Cursor == "" {
			break
		}

		cursor = resp.Cursor
	}

	switch format {
	case app.JSON:
		out, err := util.GetJsonFromProto(out)
		if err != nil {
			return err
		}
		cmd.Println(out)
	case app.YAML:
		out, err := util.GetYamlFromProto(out)
		if err != nil {
			return err
		}
		cmd.Println(out)
	case app.Table:
		t := table.New(table.Simple, layouts.ProviderList, nil)
		for _, v := range out.Providers {
			impls := getImplementsAsStrings(v)

			t.AddRow(v.GetName(), v.GetProject(), v.GetVersion(), strings.Join(impls, ", "))
		}
		t.Render()
		return nil
	default:
		return fmt.Errorf("output format %s not supported", format)
	}

	return nil
}
