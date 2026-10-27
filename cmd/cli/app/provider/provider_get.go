// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
	"github.com/mindersec/minder/internal/util/cli/table"
	"github.com/mindersec/minder/internal/util/cli/table/layouts"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a given provider available in a specific project",
	Long:  `The minder provider get command gets a given provider available in a specific project.`,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error binding flags: %w", err)
		}

		format := viper.GetString("output")

		return app.IsOutputFormatSupported(format)
	},
	RunE: getProviderCommand,
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

// getProviderCommand gets a given provider available in a specific project
func getProviderCommand(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	client, cleanup, err := cli.GetCLIClient(cmd, minderv1.NewProvidersServiceClient)
	if err != nil {
		return err
	}
	defer cleanup()

	project := viper.GetString("project")
	format := viper.GetString("output")
	name := viper.GetString("name")

	out, err := client.GetProvider(cmd.Context(), &minderv1.GetProviderRequest{
		Context: &minderv1.Context{
			Project: &project,
		},
		Name: name,
	})
	if err != nil {
		return cli.MessageAndError("Failed to get provider", err)
	}

	switch format {
	case app.JSON, app.YAML:
		return cli.PrintProto(cmd, format, out.GetProvider())
	case app.Table:
		t := table.New(table.Simple, layouts.Default, cmd.OutOrStdout(), []string{"Key", "Value"})
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
	}
	return nil
}

func mapToKvPairs(m map[string]any, parentKey string, result *[]string, nesting int) {
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
