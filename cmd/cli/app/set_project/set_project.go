// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package set_project provides the version command for the minder CLI
package set_project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
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
	Args: cobra.ExactArgs(1),
	RunE: spCommand,
}

// spCommand is the command for changing the current project
func spCommand(cmd *cobra.Command, args []string) error {
	project := args[0]

	projectID, err := uuid.Parse(project)
	if err != nil {
		projectID, err = resolveProjectByName(cmd, project)
		if err != nil {
			return err
		}
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

	cfg.Project = projectID.String()

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

// resolveProjectByName is the CLI entry point, creates a real gRPC client
// and delegates to resolveProjectByNameFromClient
func resolveProjectByName(cmd *cobra.Command, name string) (uuid.UUID, error) {
	client, cleanup, err := cli.GetCLIClient(cmd, minderv1.NewProjectsServiceClient)
	if err != nil {
		return uuid.UUID{}, cli.MessageAndError("Error creating client", err)
	}
	defer cleanup()

	return resolveProjectByNameFromClient(cmd.Context(), client, name)
}

// resolveProjectByNameFromClient contains the pure resolution logic.
// It is split out from resolveProjectByName so it can be tested with a mock client.
func resolveProjectByNameFromClient(ctx context.Context, client minderv1.ProjectsServiceClient, name string) (uuid.UUID, error) {
	resp, err := client.ListProjects(ctx, &minderv1.ListProjectsRequest{})
	if err != nil {
		return uuid.UUID{}, cli.MessageAndError("Error listing projects", err)
	}

	var matches []*minderv1.Project
	for _, p := range resp.Projects {
		if p.Name == name || p.DisplayName == name {
			matches = append(matches, p)
		}
	}

	if len(matches) == 0 {
		return uuid.UUID{}, fmt.Errorf("no project found with name %q", name)
	}
	id, err := uuid.Parse(matches[0].ProjectId)
	if err != nil {
		return uuid.UUID{}, cli.MessageAndError("Error parsing project ID from server", err)
	}

	return id, nil
}

func init() {
	app.RootCmd.AddCommand(SetProjectCmd)
}
