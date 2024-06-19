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

// Package app provides the root command for the minder CLI
package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/minder/internal/config"
	clientconfig "github.com/stacklok/minder/internal/config/client"
	"github.com/stacklok/minder/internal/constants"
	"github.com/stacklok/minder/internal/util/cli"
)

var (
	// RootCmd represents the base command when called without any subcommands
	RootCmd = &cobra.Command{
		Use:   "minder",
		Short: "Minder controls the hosted minder service",
		Long: `For more information about minder, please visit:
https://docs.stacklok.com/minder`,
		SilenceErrors: true, // don't print errors twice, we handle them in cli.ExitNicelyOnError
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// Print a warning if the build is not pointing to the production environment
			cfg, _ := config.ReadConfigFromViper[clientconfig.Config](viper.GetViper())

			if cfg == nil || cfg.GRPCClientConfig.Host != constants.MinderGRPCHost {
				fmt.Fprintf(
					cmd.ErrOrStderr(),
					"WARNING: Running against a test environment (%s) and may not be stable\n",
					cfg.GRPCClientConfig.Host)
			}
			return nil
		},
	}

	// ConfigHelpCmd is a "help topic", which is represented as a command with no "Run" function.
	// See https://github.com/spf13/cobra/issues/393#issuecomment-282741924 and
	// https://pkg.go.dev/github.com/spf13/cobra#Command.IsAdditionalHelpTopicCommand
	//nolint:lll
	ConfigHelpCmd = &cobra.Command{
		Use:    "config",
		Short:  "How to manage minder CLI configuration",
		Hidden: true,
		Long: `In addition to the command-line flags, many minder options can be set via a configuration file in the YAML format.

Configuration options include:
- provider
- project
- output
- grpc_server.host
- grpc_server.port
- grpc_server.insecure
- identity.cli.issuer_url
- identity.cli.client_id

By default, we look for the file as $PWD/config.yaml and $XDG_CONFIG_PATH/minder/config.yaml. You can specify a custom path via the --config flag, or by setting the MINDER_CONFIG environment variable.`,
	}
)

const (
	// JSON is the json format for output
	JSON = "json"
	// YAML is the yaml format for output
	YAML = "yaml"
	// Table is the table format for output
	Table = "table"
)

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	RootCmd.SetOut(os.Stdout)
	RootCmd.SetErr(os.Stderr)
	err := RootCmd.Execute()
	cli.ExitNicelyOnError(err, "")
}

func init() {
	cobra.OnInitialize(initConfig)

	// Register minder cli flags - gRPC client config and identity config
	if err := clientconfig.RegisterMinderClientFlags(viper.GetViper(), RootCmd.PersistentFlags()); err != nil {
		RootCmd.Printf("error: %s", err)
		os.Exit(1)
	}

	RootCmd.AddCommand(ConfigHelpCmd)
	RootCmd.PersistentFlags().String("config", "", "Config file (default is $PWD/config.yaml)")
	if err := viper.BindPFlag("config", RootCmd.PersistentFlags().Lookup("config")); err != nil {
		RootCmd.Printf("error: %s", err)
		os.Exit(1)
	}
	viper.AutomaticEnv()
}

func initConfig() {
	viper.SetEnvPrefix("minder")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	cfgFilePath := cli.GetRelevantCLIConfigPath(viper.GetViper())
	if cfgFilePath != "" {
		cfgFileData, err := config.GetConfigFileData(cfgFilePath)
		if err != nil {
			RootCmd.PrintErrln(err)
			os.Exit(1)
		}

		keysWithNullValue := config.GetKeysWithNullValueFromYAML(cfgFileData, "")
		if len(keysWithNullValue) > 0 {
			RootCmd.PrintErrln("Error: The following configuration keys are missing values:")
			for _, key := range keysWithNullValue {
				RootCmd.PrintErrln("Null Value at: " + key)
			}
			os.Exit(1)
		}

		viper.SetConfigFile(cfgFilePath)
	} else {
		// use defaults
		viper.SetConfigName("config")
		viper.AddConfigPath(".")
		if cfgDirPath := cli.GetDefaultCLIConfigPath(); cfgDirPath != "" {
			viper.AddConfigPath(cfgDirPath)
		}
	}
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; use default values
			RootCmd.PrintErrln("No config file present, using default values.")
		} else {
			// Some other error occurred
			RootCmd.Printf("Error reading config file: %s", err)
		}
	}
}

// IsOutputFormatSupported returns true if the output format is supported
func IsOutputFormatSupported(output string) bool {
	for _, format := range SupportedOutputFormats() {
		if output == format {
			return true
		}
	}
	return false
}

// SupportedOutputFormats returns the supported output formats
func SupportedOutputFormats() []string {
	return []string{JSON, YAML, Table}
}
