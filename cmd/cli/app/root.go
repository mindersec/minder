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
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	clientconfig "github.com/stacklok/minder/internal/config/client"
	ghclient "github.com/stacklok/minder/internal/providers/github"
	"github.com/stacklok/minder/internal/util/cli"
)

var (
	cfgFile string // config file (default is $PWD/config.yaml)

	// RootCmd represents the base command when called without any subcommands
	RootCmd = &cobra.Command{
		Use:   "minder",
		Short: "Minder controls the hosted minder service",
		Long: `For more information about minder, please visit:
https://docs.stacklok.com/minder`,
		SilenceErrors: true, // don't print errors twice, we handle them in cli.ExitNicelyOnError
	}

	// This is a "help topic", which is represented as a command with no "Run" function.
	// See https://github.com/spf13/cobra/issues/393#issuecomment-282741924 and
	// https://pkg.go.dev/github.com/spf13/cobra#Command.IsAdditionalHelpTopicCommand
	configHelpCmd = &cobra.Command{
		Use:   "config",
		Short: "How to manage minder CLI configuration",
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

By default, we look for the file as $PWD/config.yaml. You can specify a custom path via the --config flag, or by setting the MINDER_CONFIG environment variable.`,
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

	RootCmd.AddCommand(configHelpCmd)
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file (default is $PWD/config.yaml)")
}

func initConfig() {
	viper.SetEnvPrefix("minder")
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else if os.Getenv("MINDER_CONFIG") != "" {
		viper.SetConfigFile(os.Getenv("MINDER_CONFIG"))
	} else {
		// use defaults
		viper.SetConfigName("config")
		viper.AddConfigPath(".")
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

// IsProviderSupported returns true if the provider is supported
func IsProviderSupported(provider string) bool {
	for _, p := range SupportedProviders() {
		if provider == p {
			return true
		}
	}
	return false
}

// SupportedProviders returns the supported providers list
func SupportedProviders() []string {
	return []string{ghclient.Github}
}
