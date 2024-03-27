//
// Copyright 2024 Stacklok, Inc.
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

// Package app provides the cli subcommands for managing the reminder service
package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/minder/internal/config"
	reminderconfig "github.com/stacklok/minder/internal/config/reminder"
	"github.com/stacklok/minder/internal/util/cli"
)

var (
	// RootCmd represents the base command when called without any subcommands
	RootCmd = &cobra.Command{
		Use:   "reminder",
		Short: "reminder controls the reminder service",
		Long:  `reminder sends entity reconciliation requests to the minder server`,
	}
)

const configFileName = "reminder-config.yaml"

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	RootCmd.SetOut(os.Stdout)
	RootCmd.SetErr(os.Stderr)
	err := RootCmd.ExecuteContext(context.Background())
	cli.ExitNicelyOnError(err, "Error executing root command")
}

func init() {
	cobra.OnInitialize(initConfig)
	reminderconfig.SetViperDefaults(viper.GetViper())
	RootCmd.PersistentFlags().String("config", "", fmt.Sprintf("config file (default is $PWD/%s)", configFileName))

	if err := reminderconfig.RegisterReminderFlags(viper.GetViper(), RootCmd.PersistentFlags()); err != nil {
		log.Fatal().Err(err).Msg("Error registering reminder flags")
	}

	if err := viper.BindPFlag("config", RootCmd.PersistentFlags().Lookup("config")); err != nil {
		log.Fatal().Err(err).Msg("Error binding config flag")
	}
}

func initConfig() {
	cfgFile := viper.GetString("config")
	cfgFileData, err := config.GetConfigFileData(cfgFile, filepath.Join(".", configFileName))
	if err != nil {
		log.Fatal().Err(err).Msg("Error reading config file")
	}

	keysWithNullValue := config.GetKeysWithNullValueFromYAML(cfgFileData, "")
	if len(keysWithNullValue) > 0 {
		RootCmd.PrintErrln("Error: The following configuration keys are missing values:")
		for _, key := range keysWithNullValue {
			RootCmd.PrintErrln("Null Value at: " + key)
		}
		os.Exit(1)
	}

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// use defaults
		viper.SetConfigName(strings.TrimSuffix(configFileName, filepath.Ext(configFileName)))
		viper.AddConfigPath(".")
	}
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()

	if err = viper.ReadInConfig(); err != nil {
		fmt.Println("Error reading config file:", err)
	}
}
