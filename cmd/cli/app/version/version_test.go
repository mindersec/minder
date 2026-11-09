// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/constants"
	"github.com/mindersec/minder/internal/util/cli"
	"github.com/mindersec/minder/internal/util/cli/useragent"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global RootCmd / Viper state
func TestVersionCommand_Streams(t *testing.T) {
	rootCmd := app.RootCmd
	cli.ResetEntireTree(rootCmd)
	viper.Reset()
	// RunCmdTests is not used here because version output includes dynamic build/commit metadata,
	// and we want direct assertion on stdout and stderr without needing golden files.
	// Point to an isolated config with a non-localhost host to prevent initConfig creating notices
	// or PersistentPreRunE test environment warnings on stderr.
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	minderDir := filepath.Join(tempDir, "minder")
	require.NoError(t, os.MkdirAll(minderDir, 0o700))
	require.NoError(t, os.WriteFile(
		filepath.Join(minderDir, "config.yaml"),
		[]byte("grpc_server:\n  host: api.stacklok.com\n"),
		0o600,
	))

	stdoutBuf := new(bytes.Buffer)
	stderrBuf := new(bytes.Buffer)
	rootCmd.SetOut(stdoutBuf)
	rootCmd.SetErr(stderrBuf)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	expectedStdout := fmt.Sprintf("%s\nUser Agent: %s\n", constants.VerboseCLIVersion, useragent.GetUserAgent())
	assert.Equal(t, expectedStdout, stdoutBuf.String(), "version output must be written to stdout")
	assert.Empty(t, stderrBuf.String(), "stderr must be empty when executing version command")
}
