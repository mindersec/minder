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

// Package set_project provides the version command for the minder CLI
package set_project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/stacklok/minder/cmd/cli/app"
	"github.com/stacklok/minder/internal/config"
	clientconfig "github.com/stacklok/minder/internal/config/client"
	"github.com/stacklok/minder/internal/util/cli"
)

// SetProjectCmd is the cd command
var SetProjectCmd = &cobra.Command{
	Use:     "set-project",
	Aliases: []string{"sp", "cd"},
	Short:   "Move the current context to another project",
	Long: `The minder set-project command moves the current context to another project.
Passing a UUID will move the context to the project with that UUID. This is akin to
using an absolute path in a filesystem.`,
	RunE: spCommand,
}

// spCommand is the command for changing the current project
func spCommand(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return cmd.Usage()
	}

	project := args[0]

	_, err := uuid.Parse(project)
	// TODO: Implement `cd` to a project name
	if err != nil {
		return cli.MessageAndError("Error parsing project ID", err)
	}

	cfgp := cli.GetRelevantCLIConfigPath(viper.GetViper())
	if cfgp == "" {
		// There is no config file at the moment. Let's create one.
		cfgp, err = persistEmptyDefaultConfig()
		if err != nil {
			return cli.MessageAndError("Error creating config file", err)
		}
	}

	viper.SetConfigFile(cfgp)
	if err := viper.ReadInConfig(); err != nil {
		return cli.MessageAndError("Error reading config file", err)
	}

	cfg, err := config.ReadConfigFromViper[clientconfig.Config](viper.GetViper())
	if err != nil {
		return fmt.Errorf("unable to read config: %w", err)
	}

	cfg.Project = project

	w, err := os.OpenFile(filepath.Clean(cfgp), os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return cli.MessageAndError("Error opening config file for writing", err)
	}

	defer func() {
		//nolint:errcheck // leaking file handle is not a concern here
		_ = w.Close()
	}()

	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)

	defer enc.Close()

	if err := enc.Encode(cfg); err != nil {
		return cli.MessageAndError("Error encoding config to file", err)
	}

	return nil
}

func persistEmptyDefaultConfig() (string, error) {
	cfgp := cli.GetDefaultCLIConfigPath()
	if cfgp == "" {
		return "", errors.New("no default config path found")
	}
	f, err := os.Create(filepath.Clean(cfgp))
	if err != nil {
		if !errors.Is(err, os.ErrExist) {
			return "", err
		}

		// File already exists, no need to write the default config
		return cfgp, nil
	}
	// Ensure we've written the default config to the file
	if err := f.Sync(); err != nil {
		return "", err
	}

	//nolint:errcheck // leaking file handle is not a concern here
	_ = f.Close()

	return cfgp, nil
}

func init() {
	app.RootCmd.AddCommand(SetProjectCmd)
}
