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

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new data source",
	Long:  `The datasource create subcommand lets you create a new data source within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(createCommand),
}

// createCommand is the datasource create subcommand
func createCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewDataSourceServiceClient(conn)

	project := viper.GetString("project")
	format := viper.GetString("output")
	//name := viper.GetString("name")
	//endpoint := viper.GetString("endpoint")
	//version := viper.GetString("version")

	// No longer print usage on returned error, since we've parsed our inputs
	cmd.SilenceUsage = true

	// Create the REST data source definition
	// restDef := &minderv1.RestDataSource{
	// 	Def: map[string]*minderv1.RestDataSource_Def{
	// 		"default": {
	// 			Endpoint: endpoint,
	// 		},
	// 	},
	// }

	// Create the data source
	// dataSource := &minderv1.DataSource{
	// 	Version: version,
	// 	Type:    "data-source",
	// 	Name:    name,
	// 	Context: &minderv1.ContextV2{
	// 		ProjectId: project,
	// 	},
	// 	Driver: &minderv1.DataSource_Rest{
	// 		Rest: restDef,
	// 	},
	// }

	resp, err := client.CreateDataSource(ctx, &minderv1.CreateDataSourceRequest{
		Context: &minderv1.ContextV2{
			ProjectId: project,
		},
		//DataSource: dataSource,
	})
	if err != nil {
		return cli.MessageAndError("Failed to create data source", err)
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
	DataSourceCmd.AddCommand(createCmd)

	createCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
	createCmd.Flags().StringP("name", "n", "", "Name of the data source")
	createCmd.Flags().StringP("endpoint", "e", "", "Endpoint URL for the REST data source")
	createCmd.Flags().StringP("version", "v", "v1", "Version of the data source API")

	if err := createCmd.MarkFlagRequired("name"); err != nil {
		panic(err)
	}
	if err := createCmd.MarkFlagRequired("endpoint"); err != nil {
		panic(err)
	}
}
