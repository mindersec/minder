// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package set_project_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/mindersec/minder/cmd/cli/app"
	_ "github.com/mindersec/minder/cmd/cli/app/set_project"
	"github.com/mindersec/minder/internal/util/cli"
	clientconfig "github.com/mindersec/minder/pkg/config/client"
)

//nolint:paralleltest // Cannot run in parallel because tests modify global env, viper, and root command state
func TestSetProjectCommand(t *testing.T) {
	t.Run("successfully sets valid project ID and creates config file", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tempDir)

		targetProjectID := uuid.New().String()

		rootCmd := app.RootCmd
		cli.ResetEntireTree(rootCmd)
		viper.Reset()

		stdoutBuf := new(bytes.Buffer)
		stderrBuf := new(bytes.Buffer)
		rootCmd.SetOut(stdoutBuf)
		rootCmd.SetErr(stderrBuf)
		rootCmd.SetArgs([]string{"set-project", targetProjectID})

		err := rootCmd.Execute()
		require.NoError(t, err)

		configPath := filepath.Join(tempDir, "minder", "config.yaml")
		assert.FileExists(t, configPath)

		content, err := os.ReadFile(configPath)
		require.NoError(t, err)

		var cfg clientconfig.Config
		err = yaml.Unmarshal(content, &cfg)
		require.NoError(t, err)

		assert.Equal(t, targetProjectID, cfg.Project)
	})

	t.Run("successfully updates existing config file preserving other fields", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tempDir)

		configDir := filepath.Join(tempDir, "minder")
		require.NoError(t, os.MkdirAll(configDir, 0o700))

		configPath := filepath.Join(configDir, "config.yaml")
		initialConfig := clientconfig.Config{
			Project: uuid.New().String(),
			GRPCClientConfig: clientconfig.GRPCClientConfig{
				Host:     "custom.minder.host",
				Port:     8443,
				Insecure: true,
			},
		}
		data, err := yaml.Marshal(initialConfig)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(configPath, data, 0o600))

		newProjectID := uuid.New().String()

		rootCmd := app.RootCmd
		cli.ResetEntireTree(rootCmd)
		viper.Reset()

		stdoutBuf := new(bytes.Buffer)
		stderrBuf := new(bytes.Buffer)
		rootCmd.SetOut(stdoutBuf)
		rootCmd.SetErr(stderrBuf)
		rootCmd.SetArgs([]string{"set-project", newProjectID})

		err = rootCmd.Execute()
		require.NoError(t, err)

		content, err := os.ReadFile(configPath)
		require.NoError(t, err)

		var updatedCfg clientconfig.Config
		err = yaml.Unmarshal(content, &updatedCfg)
		require.NoError(t, err)

		assert.Equal(t, newProjectID, updatedCfg.Project)
		assert.Equal(t, "custom.minder.host", updatedCfg.GRPCClientConfig.Host)
		assert.Equal(t, 8443, updatedCfg.GRPCClientConfig.Port)
		assert.True(t, updatedCfg.GRPCClientConfig.Insecure)
	})

	t.Run("works using aliases cd and sp", func(t *testing.T) {
		for _, alias := range []string{"cd", "sp"} {
			tempDir := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", tempDir)

			projectID := uuid.New().String()

			rootCmd := app.RootCmd
			cli.ResetEntireTree(rootCmd)
			viper.Reset()

			rootCmd.SetOut(new(bytes.Buffer))
			rootCmd.SetErr(new(bytes.Buffer))
			rootCmd.SetArgs([]string{alias, projectID})

			err := rootCmd.Execute()
			require.NoError(t, err)

			configPath := filepath.Join(tempDir, "minder", "config.yaml")
			content, err := os.ReadFile(configPath)
			require.NoError(t, err)

			var cfg clientconfig.Config
			err = yaml.Unmarshal(content, &cfg)
			require.NoError(t, err)

			assert.Equal(t, projectID, cfg.Project)
		}
	})

	t.Run("returns error when project ID is not a valid UUID", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tempDir)

		rootCmd := app.RootCmd
		cli.ResetEntireTree(rootCmd)
		viper.Reset()

		rootCmd.SetOut(new(bytes.Buffer))
		rootCmd.SetErr(new(bytes.Buffer))
		rootCmd.SetArgs([]string{"set-project", "invalid-uuid-string"})

		err := rootCmd.Execute()
		require.Error(t, err)

		var wrappedErr *cli.ErrWrappedCLIError
		require.ErrorAs(t, err, &wrappedErr)
		assert.Equal(t, "Error parsing project ID", wrappedErr.Message)
		assert.Contains(t, wrappedErr.Err.Error(), "invalid UUID")
	})

	t.Run("returns usage error when no argument is provided", func(t *testing.T) {
		rootCmd := app.RootCmd
		cli.ResetEntireTree(rootCmd)
		viper.Reset()

		stdoutBuf := new(bytes.Buffer)
		stderrBuf := new(bytes.Buffer)
		rootCmd.SetOut(stdoutBuf)
		rootCmd.SetErr(stderrBuf)
		rootCmd.SetArgs([]string{"set-project"})

		err := rootCmd.Execute()
		require.NoError(t, err)
	})

	t.Run("returns usage error when more than one argument is provided", func(t *testing.T) {
		rootCmd := app.RootCmd
		cli.ResetEntireTree(rootCmd)
		viper.Reset()

		stdoutBuf := new(bytes.Buffer)
		stderrBuf := new(bytes.Buffer)
		rootCmd.SetOut(stdoutBuf)
		rootCmd.SetErr(stderrBuf)
		rootCmd.SetArgs([]string{"set-project", uuid.New().String(), "extra-arg"})

		err := rootCmd.Execute()
		require.NoError(t, err)
	})
}
