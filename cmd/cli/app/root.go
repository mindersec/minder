// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package app provides the root command for the minder CLI
package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/internal/constants"
	"github.com/mindersec/minder/internal/util/cli"
	"github.com/mindersec/minder/pkg/config"
	clientconfig "github.com/mindersec/minder/pkg/config/client"
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
	RootCmd.PersistentFlags().BoolP("verbose", "v", false, "Output additional messages to STDERR")
	viper.AutomaticEnv()
}

func initConfig() {
	viper.SetEnvPrefix("minder")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	// Get the config flag value directly to ensure we catch explicitly specified configs
	configFlag := viper.GetString("config")
	if configFlag != "" {
		// User explicitly specified a config file via --config flag
		if _, err := os.Stat(configFlag); err != nil {
			cwd, err := os.Getwd()
			if err != nil {
				cwd = err.Error()
			}
			RootCmd.PrintErrln(fmt.Sprintf("Cannot find specified config file: %s (%s)", configFlag, cwd))
			os.Exit(1)
		}
		viper.SetConfigFile(configFlag)
	} else {
		// No config file specified, use defaults
		viper.SetConfigName("config")
		viper.AddConfigPath(".")
		if cfgDirPath := cli.GetDefaultCLIConfigPath(); cfgDirPath != "" {
			viper.AddConfigPath(cfgDirPath)
		}
	}

	viper.SetConfigType("yaml")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if configFlag != "" {
			// If there's any error, we should fail
			RootCmd.PrintErrln(fmt.Sprintf("Error reading config file %s: %v", configFlag, err))
			os.Exit(1)
		}

		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Only allow "config not found" when no config was explicitly specified
			RootCmd.PrintErrln("No config file present, using default values.")
		} else {
			RootCmd.PrintErrln(fmt.Sprintf("Error reading config file: %s", err))
			os.Exit(1)
		}
	}

	// If we successfully read a config file, check for null values
	if configFlag != "" {
		cfgFileData, err := config.GetConfigFileData(configFlag)
		if err != nil {
			RootCmd.PrintErrln(fmt.Sprintf("Error reading config file data: %v", err))
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
