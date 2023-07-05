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

// Package app provides the cli subcommands for managing a mediator control plane
package app

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stacklok/mediator/pkg/db"
	"github.com/stacklok/mediator/pkg/util"
)

var (
	cfgFile string // config file (default is $PWD/config.yaml)
	// RootCmd represents the base command when called without any subcommands
	RootCmd = &cobra.Command{
		Use:   "mediator-server",
		Short: "Mediator control plane server",
		Long:  ``,
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := RootCmd.Execute()
	util.ExitNicelyOnError(err, "Error executing root command")
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $PWD/config.yaml)")
	if err := db.RegisterFlags(viper.GetViper(), RootCmd.PersistentFlags()); err != nil {
		log.Fatal(err)
	}
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// use defaults
		viper.SetConfigName("config")
		viper.AddConfigPath(".")
	}
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Error reading config file:", err)
	}
}
