// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package set_project provides the version command for the minder CLI
package set_project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
	"github.com/mindersec/minder/pkg/config"
	clientconfig "github.com/mindersec/minder/pkg/config/client"
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

	configFile := viper.GetViper().ConfigFileUsed()
	if configFile == "" {
		cfgDir, err := cli.GetConfigDirPath()
		if err != nil {
			cfgDir = "."
		}
		configFile = filepath.Join(cfgDir, "config.yaml")
		if err := os.MkdirAll(cfgDir, 0700); err != nil {
			return cli.MessageAndError("Error creating config directory", err)
		}
	}

	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		return cli.MessageAndError("Error reading config file", err)
	}

	cfg, err := config.ReadConfigFromViper[clientconfig.Config](viper.GetViper())
	if err != nil {
		return fmt.Errorf("unable to read config: %w", err)
	}

	cfg.Project = project

	w, err := os.OpenFile(filepath.Clean(configFile), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
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

func init() {
	app.RootCmd.AddCommand(SetProjectCmd)
}
