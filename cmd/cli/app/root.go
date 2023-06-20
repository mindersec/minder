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

// Package app provides the root command for the medctl CLI
package app

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stacklok/mediator/pkg/util"
)

var (
	cfgFile string // config file (default is $PWD/config.yaml)

	// RootCmd represents the base command when called without any subcommands
	RootCmd = &cobra.Command{
		Use:   "medctl",
		Short: "medctl controls mediator via the control plane",
		Long: `For more information about mediator, please visit:
https://docs.stacklok.com/mediator/medctl/overview.html`,
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := RootCmd.Execute()
	util.ExitNicelyOnError(err, "Error on execute")
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().String("grpc-host", "localhost", "Server host")
	RootCmd.PersistentFlags().Int("grpc-port", 8090, "Server port")
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $PWD/config.yaml)")
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
