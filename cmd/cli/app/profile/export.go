// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profile

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v3"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/fileconvert"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export profile and associated resources",
	Long:  `The profile export subcommand lets you retrieve the definition of a profile and its associated resources.`,
	RunE:  cli.GRPCClientWrapRunE(exportCommand),
}

// getCommand is the profile "get" subcommand
func exportCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	output, closer, err := getOutput(cmd)
	if err != nil {
		return cli.MessageAndError("Error opening output", err)
	}
	defer closer()

	profileClient := minderv1.NewProfileServiceClient(conn)

	project := viper.GetString("project")
	id := viper.GetString("id")
	name := viper.GetString("name")

	if id == "" && name == "" {
		return cli.MessageAndError("Error getting profile", fmt.Errorf("id or name required"))
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	prof, err := getProfileByNameOrId(ctx, profileClient, project, id, name)
	if err != nil {
		return cli.MessageAndError("Error getting profile", err)
	}
	if err := fileconvert.WriteResource(output, prof); err != nil {
		return cli.MessageAndError("Error encoding profile", err)
	}

	// Fetch associated resources
	rulesClient := minderv1.NewRuleTypeServiceClient(conn)

	// TODO: it would be nice if this were just a list of rules...
	rules := slices.Concat(
		prof.GetRepository(),
		prof.GetBuildEnvironment(),
		prof.GetArtifact(),
		prof.GetPullRequest(),
		prof.GetRelease(),
		prof.GetPipelineRun(),
		prof.GetTaskRun(),
		prof.GetBuild(),
	)
	ruletypes := make([]string, 0, len(rules))
	for _, res := range rules {
		ruletypes = append(ruletypes, res.GetType())
	}
	slices.Sort(ruletypes)
	ruletypes = slices.Compact(ruletypes)

	// Collect the referenced datasources from the rule types as we process them.
	datasources := make([]string, 0)
	for _, ruletype := range ruletypes {
		resp, err := rulesClient.GetRuleTypeByName(ctx, &minderv1.GetRuleTypeByNameRequest{
			Context: &minderv1.Context{Project: &project},
			Name:    ruletype,
		})
		if err != nil {
			return cli.MessageAndError(fmt.Sprintf("Error getting rule type %q", ruletype), err)
		}
		ruletype := resp.GetRuleType()
		for _, datasource := range ruletype.GetDef().GetEval().GetDataSources() {
			datasources = append(datasources, datasource.GetName())
		}
		if err := fileconvert.WriteResource(output, ruletype); err != nil {
			return cli.MessageAndError(fmt.Sprintf("Error encoding rule type %q", ruletype.GetName()), err)
		}
	}

	// Remove duplicates from the datasource list
	slices.Sort(datasources)
	datasources = slices.Compact(datasources)
	datasourceClient := minderv1.NewDataSourceServiceClient(conn)
	for _, datasource := range datasources {
		resp, err := datasourceClient.GetDataSourceByName(ctx, &minderv1.GetDataSourceByNameRequest{
			Context: &minderv1.ContextV2{ProjectId: project},
			Name:    datasource,
		})
		if err != nil {
			return cli.MessageAndError(fmt.Sprintf("Error getting datasource %q", datasource), err)
		}
		if err := fileconvert.WriteResource(output, resp.GetDataSource()); err != nil {
			return cli.MessageAndError(fmt.Sprintf("Error encoding datasource %q", datasource), err)
		}
	}

	return nil
}

func getOutput(cmd *cobra.Command) (*yaml.Encoder, func(), error) {
	outputFlag := viper.GetString("output")

	var outFile io.Writer
	closer := func() {}

	if outputFlag == "" || outputFlag == "-" {
		outFile = cmd.OutOrStdout()
	} else {
		file, err := os.Create(filepath.Clean(outputFlag))
		if err != nil {
			return nil, closer, cli.MessageAndError("Error opening file", err)
		}
		outFile = file
		closer = func() {
			_ = file.Close()
		}
	}
	output := yaml.NewEncoder(outFile)
	yamlClose := func() {
		_ = output.Close()
		closer()
	}

	return output, yamlClose, nil
}

func getProfileByNameOrId(
	ctx context.Context, client minderv1.ProfileServiceClient, project string, id string, name string,
) (*minderv1.Profile, error) {
	if id != "" {
		p, err := client.GetProfileById(ctx, &minderv1.GetProfileByIdRequest{
			Context: &minderv1.Context{Project: &project},
			Id:      id,
		})
		if err != nil {
			return nil, err
		}
		return p.GetProfile(), nil
	}
	p, err := client.GetProfileByName(ctx, &minderv1.GetProfileByNameRequest{
		Context: &minderv1.Context{Project: &project},
		Name:    name,
	})
	if err != nil {
		return nil, err
	}
	return p.GetProfile(), nil
}

func init() {
	ProfileCmd.AddCommand(exportCmd)
	// Flags
	exportCmd.Flags().StringP("id", "i", "", "ID for the profile to query")
	exportCmd.Flags().StringP("name", "n", "", "Name for the profile to query")
	exportCmd.Flags().StringP("output", "o", "-", "Output file (or stdout)")
	exportCmd.MarkFlagsMutuallyExclusive("id", "name")
}
