package datasource

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli"
	"github.com/mindersec/minder/internal/util/cli/table"
	"github.com/mindersec/minder/internal/util/cli/table/layouts"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get data source details",
	Long:  `The datasource get subcommand lets you retrieve details for a data source within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(getCommand),
}

// getCommand is the datasource get subcommand
func getCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewDataSourceServiceClient(conn)

	project := viper.GetString("project")
	format := viper.GetString("output")
	id := viper.GetString("id")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	resp, err := client.GetDataSourceById(ctx, &minderv1.GetDataSourceByIdRequest{
		Context: &minderv1.ContextV2{
			ProjectId: project,
		},
		Id: id,
	})
	if err != nil {
		return cli.MessageAndError("Failed to get data source", err)
	}

	switch format {
	case app.JSON:
		out, err := util.GetJsonFromProto(resp)
		if err != nil {
			return cli.MessageAndError("Error getting json from proto", err)
		}
		cmd.Println(out)
	case app.YAML:
		out, err := util.GetYamlFromProto(resp)
		if err != nil {
			return cli.MessageAndError("Error getting yaml from proto", err)
		}
		cmd.Println(out)
	case app.Table:
		t := table.New(table.Simple, layouts.Default, []string{"ID", "Name", "Type"})
		ds := resp.GetDataSource()
		t.AddRow(ds.Id, ds.Name, getDataSourceType(ds))
		t.Render()
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}

	return nil
}

func init() {
	DataSourceCmd.AddCommand(getCmd)

	getCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
	getCmd.Flags().StringP("id", "i", "", "ID of the data source to get info from")

	if err := getCmd.MarkFlagRequired("id"); err != nil {
		panic(err)
	}
}
