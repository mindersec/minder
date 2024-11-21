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

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a data source",
	Long:  `The datasource update subcommand lets you update an existing data source within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(updateCommand),
}

// updateCommand is the datasource update subcommand
func updateCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewDataSourceServiceClient(conn)

	project := viper.GetString("project")
	format := viper.GetString("output")
	id := viper.GetString("id")
	// name := viper.GetString("name")
	// endpoint := viper.GetString("endpoint")
	// version := viper.GetString("version")

	// No longer print usage on returned error
	cmd.SilenceUsage = true

	// Create the REST data source definition
	// restDef := &minderv1.RestDataSource{
	// 	Def: map[string]*minderv1.RestDataSource_Def{
	// 		"default": {
	// 			Endpoint: endpoint,
	// 		},
	// 	},
	// }

	// Create the data source update
	// dataSource := &minderv1.DataSource{
	// 	Version: version,
	// 	Type:    "data-source",
	// 	Name:    name,
	// 	Id:      id,
	// 	Context: &minderv1.ContextV2{
	// 		ProjectId: project,
	// 	},
	// 	Driver: &minderv1.DataSource_Rest{
	// 		Rest: restDef,
	// 	},
	// }

	resp, err := client.UpdateDataSource(ctx, &minderv1.UpdateDataSourceRequest{
		Context: &minderv1.ContextV2{
			ProjectId: project,
		},
		Id: id,
		//DataSource: dataSource,
	})
	if err != nil {
		return cli.MessageAndError("Failed to update data source", err)
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
	DataSourceCmd.AddCommand(updateCmd)

	updateCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
	updateCmd.Flags().StringP("id", "i", "", "ID of the data source to update")
	updateCmd.Flags().StringP("name", "n", "", "New name for the data source")
	updateCmd.Flags().StringP("endpoint", "e", "", "New endpoint URL for the REST data source")
	updateCmd.Flags().StringP("version", "v", "", "New version of the data source API")

	if err := updateCmd.MarkFlagRequired("id"); err != nil {
		panic(err)
	}
}
