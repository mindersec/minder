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

// Package app provides the cli subcommands for managing a minder control plane
package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/minder/internal/auth"
	"github.com/stacklok/minder/internal/config"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/util/cli"
)

var (
	// RootCmd represents the base command when called without any subcommands
	RootCmd = &cobra.Command{
		Use:   "minder-server",
		Short: "Minder control plane server",
		Long:  ``,
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	RootCmd.SetOut(os.Stdout)
	RootCmd.SetErr(os.Stderr)
	err := RootCmd.Execute()
	cli.ExitNicelyOnError(err, "Error executing root command")
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().String("config", "", "config file (default is $PWD/server-config.yaml)")
	if err := config.RegisterDatabaseFlags(viper.GetViper(), RootCmd.PersistentFlags()); err != nil {
		log.Fatal().Err(err).Msg("Error registering database flags")
	}
	if err := auth.RegisterOAuthFlags(viper.GetViper(), RootCmd.PersistentFlags()); err != nil {
		log.Fatal().Err(err).Msg("Error registering oauth flags")
	}
	if err := serverconfig.RegisterIdentityFlags(viper.GetViper(), RootCmd.PersistentFlags()); err != nil {
		log.Fatal().Err(err).Msg("Error registering identity flags")
	}
	if err := viper.BindPFlag("config", RootCmd.PersistentFlags().Lookup("config")); err != nil {
		RootCmd.Printf("error: %s", err)
		os.Exit(1)
	}
}

func initConfig() {
	serverconfig.SetViperDefaults(viper.GetViper())

	cfgFile := viper.GetString("config")
	cfgFileData, err := config.GetConfigFileData(cfgFile, filepath.Join(".", "server-config.yaml"))
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

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// use defaults
		viper.SetConfigName("server-config")
		viper.AddConfigPath(".")
	}
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()

	if err = viper.ReadInConfig(); err != nil {
		fmt.Println("Error reading config file:", err)
	}
}
