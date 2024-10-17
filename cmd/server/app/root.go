// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package app provides the cli subcommands for managing a minder control plane
package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/internal/config"
	serverconfig "github.com/mindersec/minder/internal/config/server"
	"github.com/mindersec/minder/internal/util/cli"
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
	cfgFilePath := config.GetRelevantCfgPath(append([]string{cfgFile},
		filepath.Join(".", "server-config.yaml"),
	))
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
		viper.SetConfigName("server-config")
		viper.AddConfigPath(".")
	}
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Error reading config file:", err)
	}
}
