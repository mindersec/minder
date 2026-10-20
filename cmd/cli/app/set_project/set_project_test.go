// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package set_project_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
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

func runSetProject(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	rootCmd := app.RootCmd
	cli.ResetEntireTree(rootCmd)
	viper.Reset()

	stdoutBuf := new(bytes.Buffer)
	stderrBuf := new(bytes.Buffer)
	rootCmd.SetOut(stdoutBuf)
	rootCmd.SetErr(stderrBuf)
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	return stdoutBuf.String(), stderrBuf.String(), err
}

//nolint:paralleltest // Cannot run in parallel because tests modify global env, viper, and root command state
func TestSetProjectCommand(t *testing.T) {
	t.Run("successfully sets valid project ID and creates config file", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tempDir)

		targetProjectID := uuid.New().String()
		stdout, stderr, err := runSetProject(t, "set-project", targetProjectID)
		require.NoError(t, err)

		assert.Empty(t, stdout, "stdout should be empty on successful set-project")
		// On first run without existing config, initConfig creates empty config and prints a notice to stderr
		assert.Contains(t, stderr, "creating")

		configPath := filepath.Join(tempDir, "minder", "config.yaml")
		assert.FileExists(t, configPath)

		if runtime.GOOS != "windows" {
			info, statErr := os.Stat(configPath)
			require.NoError(t, statErr)
			assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "config file should have 0600 permissions")
		}

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
		stdout, stderr, err := runSetProject(t, "set-project", newProjectID)
		require.NoError(t, err)

		assert.Empty(t, stdout, "stdout should be empty on successful update")
		assert.Empty(t, stderr, "stderr should be empty on successful update when config file exists")

		if runtime.GOOS != "windows" {
			info, statErr := os.Stat(configPath)
			require.NoError(t, statErr)
			assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "config file should have 0600 permissions")
		}

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
			t.Run(alias, func(t *testing.T) {
				tempDir := t.TempDir()
				t.Setenv("XDG_CONFIG_HOME", tempDir)

				// Pre-create the config directory and config file so initConfig does not print creation notice
				configDir := filepath.Join(tempDir, "minder")
				require.NoError(t, os.MkdirAll(configDir, 0o700))
				configPath := filepath.Join(configDir, "config.yaml")
				require.NoError(t, os.WriteFile(configPath, []byte{}, 0o600))

				projectID := uuid.New().String()
				stdout, stderr, err := runSetProject(t, alias, projectID)
				require.NoError(t, err)

				assert.Empty(t, stdout)
				assert.Empty(t, stderr)

				content, err := os.ReadFile(configPath)
				require.NoError(t, err)

				var cfg clientconfig.Config
				err = yaml.Unmarshal(content, &cfg)
				require.NoError(t, err)

				assert.Equal(t, projectID, cfg.Project)
			})
		}
	})

	t.Run("returns error when project ID is not a valid UUID", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tempDir)

		_, _, err := runSetProject(t, "set-project", "invalid-uuid-string")
		require.Error(t, err)

		var wrappedErr *cli.ErrWrappedCLIError
		require.ErrorAs(t, err, &wrappedErr)
		assert.Equal(t, "Error parsing project ID", wrappedErr.Message)
		assert.Contains(t, wrappedErr.Err.Error(), "invalid UUID")
	})

	t.Run("returns error when no argument is provided", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tempDir)

		stdout, stderr, err := runSetProject(t, "set-project")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "accepts 1 arg(s), received 0")
		assert.Contains(t, stdout+stderr, "Usage:")
	})

	t.Run("returns error when more than one argument is provided", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tempDir)

		stdout, stderr, err := runSetProject(t, "set-project", uuid.New().String(), "extra-arg")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "accepts 1 arg(s), received 2")
		assert.Contains(t, stdout+stderr, "Usage:")
	})
}
