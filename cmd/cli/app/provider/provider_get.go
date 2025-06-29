// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"
	"reflect"
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
		cmd.Println(out)
	case app.YAML:
		out, err := util.GetYamlFromProto(out.GetProvider())
		if err != nil {
			return err
		}
		cmd.Println(out)
	case app.Table:
		t := table.New(table.Simple, layouts.Default, []string{"Key", "Value"})
		p := out.GetProvider()

		impls := getImplementsAsStrings(p)
		afs := getAuthFlowsAsStrings(p)

		t.AddRow("ID", p.GetId())
		t.AddRow("Name", p.GetName())
		t.AddRow("Class", p.GetClass())
		t.AddRow("Project", p.GetProject())
		t.AddRow("Version", p.GetVersion())
		t.AddRow("Implements", strings.Join(impls, ", "))
		t.AddRow("Auth Flows", strings.Join(afs, ", "))
		config := configAsKeyValues(p)
		if config != "" {
			t.AddRow("Config", config)
		}

		t.Render()
		return nil
	default:
		return fmt.Errorf("output format %s not supported", format)
	}

	return nil
}

// mapToKvPairs converts a map to a list of key-value pairs
// TODO(jakub): This works OK now that we have a low-number of config options
// if we have more elaborate configs, we might want to just dump the config as YAML, but for the usual
// case now (1 option..) that would not be very readable
func mapToKvPairs(m map[string]any, parentKey string, result *[]string, nesting int) {
	// just in case
	if nesting > 10 {
		return
	}

	for key, value := range m {
		fullKey := key
		if parentKey != "" {
			fullKey = parentKey + "." + key
		}

		v := reflect.ValueOf(value)
		switch v.Kind() { // nolint:exhaustive
		case reflect.Map:
			nestedMap := value.(map[string]any)
			mapToKvPairs(nestedMap, fullKey, result, nesting+1)
		default:
			// this should work for most types, if not, we'll likely just switch to printing YAML
			*result = append(*result, fmt.Sprintf("%s=%v", fullKey, value))
		}
	}
}

func configAsKeyValues(p *minderv1.Provider) string {
	if p == nil {
		return ""
	}

	conf := p.GetConfig().AsMap()
	if conf == nil {
		return ""
	}

	var result []string
	mapToKvPairs(conf, "", &result, 0)

	return strings.Join(result, "\n")
}
