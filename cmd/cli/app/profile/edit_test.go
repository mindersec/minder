// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profile_test

import (
	"bytes"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
)

//nolint:paralleltest // Mutates global viper and CLI state
func TestEditCommand(t *testing.T) {
	t.Run("rejects unexpected positional arguments", func(t *testing.T) {
		rootCmd := app.RootCmd
		cli.ResetEntireTree(rootCmd)
		viper.Reset()

		stdoutBuf := new(bytes.Buffer)
		stderrBuf := new(bytes.Buffer)
		rootCmd.SetOut(stdoutBuf)
		rootCmd.SetErr(stderrBuf)
		rootCmd.SetArgs([]string{"profile", "edit", "--name", "my-profile", "unexpected-arg"})

		err := rootCmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown command "unexpected-arg" for "minder profile edit"`)
	})
}
